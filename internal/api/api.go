package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	BaseURL  string
	APIKey   string
	Client   *http.Client
	BasePath string
}

type CommitPayload struct {
	DeviceID string `json:"device_id"`
	Location string `json:"location"`
	Delta    int    `json:"delta"`
	ItemID   int    `json:"item_id"`
}

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Location struct {
	LocationName string `json:"location"`
	Items        []int  `json:"items"`
}

func NewClient(baseURL, apiKey, basePath string) *Client {
	return &Client{
		BaseURL:  strings.TrimSuffix(baseURL, "/"),
		APIKey:   apiKey,
		BasePath: basePath,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) Check() bool {
	req, err := http.NewRequest("GET", c.BaseURL+"/rest/v1/items?limit=1", nil)
	if err != nil {
		return false
	}

	c.setAuthHeaders(req)
	resp, err := c.Client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (c *Client) SendCommit(deviceID, location string, delta, itemID int) (map[string]interface{}, error) {
	payload := CommitPayload{
		DeviceID: deviceID,
		Location: location,
		Delta:    delta,
		ItemID:   itemID,
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.BaseURL+"/rest/v1/commits", bytes.NewBuffer(data))
	c.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return result, nil
}

func (c *Client) FetchItems() ([]Item, error) {
	log.Println("[API] FetchItems() called")
	req, _ := http.NewRequest("GET", c.BaseURL+"/rest/v1/items", nil)
	c.setAuthHeaders(req)

	log.Printf("[API] Making request to: %s\n", c.BaseURL+"/rest/v1/items")
	resp, err := c.Client.Do(req)
	if err != nil {
		log.Printf("[API] Request error: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("[API] Response status: %d\n", resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[API] Response body: %s\n", string(body))

	var items []Item
	err = json.Unmarshal(body, &items)
	if err != nil {
		log.Printf("[API] JSON unmarshal error: %v\n", err)
		return nil, err
	}

	log.Printf("[API] Parsed %d items\n", len(items))
	for i, item := range items {
		log.Printf("[API] Item %d: ID=%d, Name=%s\n", i, item.ID, item.Name)
	}

	return items, nil
}

func (c *Client) FetchLocations() ([]Location, error) {
	req, _ := http.NewRequest("GET", c.BaseURL+"/rest/v1/locations", nil)
	c.setAuthHeaders(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var locations []Location
	json.Unmarshal(body, &locations)

	return locations, nil
}

func (c *Client) ExportItemsToCSV(filePath string) error {
	log.Println("[API] ExportItemsToCSV() called")
	items, err := c.FetchItems()
	if err != nil {
		log.Printf("[API] FetchItems error: %v\n", err)
		return err
	}

	log.Printf("[API] Exporting %d items to CSV\n", len(items))

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write([]string{"id", "name"})

	for _, item := range items {
		writer.Write([]string{fmt.Sprintf("%d", item.ID), item.Name})
	}

	writer.Flush()
	log.Printf("[API] CSV export complete: %s\n", filePath)
	return nil
}

func (c *Client) ExportLocationsToCSV(filePath string) error {
	locations, err := c.FetchLocations()
	if err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write([]string{"location", "items"})

	for _, loc := range locations {
		itemsStr := fmt.Sprintf("%v", loc.Items)
		writer.Write([]string{loc.LocationName, itemsStr})
	}

	writer.Flush()
	return nil
}

func (c *Client) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("apikey", c.APIKey)
}
