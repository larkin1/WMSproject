package ui

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/larkin1/wmsproject/internal/api"
	"github.com/larkin1/wmsproject/internal/queue"
)

type CommitUI struct {
	widget.BaseWidget

	scannerInput    *widget.Entry
	locationLabel   *widget.Label
	deltaInput      *widget.Entry
	toggleBtn       *widget.Button
	commitBtn       *widget.Button
	changeItemBtn   *widget.Button
	error           *widget.RichText

	mode        string
	location    string
	itemID      int
	locations   map[string]string
	items       map[string]int
	items_r     map[int]string

	api     *api.Client
	queue   *queue.Queue
	basePath string
}

func NewCommitUI(apiClient *api.Client, commitQueue *queue.Queue, basePath string) *CommitUI {
	c := &CommitUI{
		api:      apiClient,
		queue:    commitQueue,
		basePath: basePath,
		mode:     "ADD",
		items:    make(map[string]int),
		items_r:  make(map[int]string),
		locations: make(map[string]string),
	}

	c.loadItems()
	c.loadLocations()

	return c
}

func (c *CommitUI) loadItems() {
	itemsCSV := filepath.Join(c.basePath, "items.csv")

	// Try to fetch fresh data
	c.api.ExportItemsToCSV(itemsCSV)

	// Load from CSV
	file, err := os.Open(itemsCSV)
	if err != nil {
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return
	}

	for i, record := range records {
		if i == 0 {
			continue // skip header
		}
		if len(record) >= 2 {
			id, _ := strconv.Atoi(record[0])
			name := record[1]
			c.items[name] = id
			c.items_r[id] = name
		}
	}
}

func (c *CommitUI) loadLocations() {
	locationsCSV := filepath.Join(c.basePath, "locations.csv")

	// Try to fetch fresh data
	c.api.ExportLocationsToCSV(locationsCSV)

	// Load from CSV
	file, err := os.Open(locationsCSV)
	if err != nil {
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return
	}

	for i, record := range records {
		if i == 0 {
			continue // skip header
		}
		if len(record) >= 2 {
			c.locations[record[0]] = record[1]
		}
	}
}

func (c *CommitUI) onScanned(text string) {
	c.location = strings.TrimSpace(text)
	c.loadLocations()

	if items, ok := c.locations[c.location]; ok {
		itemsStr := strings.Trim(items, "[]")
		itemIDs := strings.Split(itemsStr, ", ")

		var itemInts []int
		for _, idStr := range itemIDs {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				itemInts = append(itemInts, id)
			}
		}

		if len(itemInts) > 1 {
			c.showItemSelectDialog(itemInts)
			return
		} else if len(itemInts) == 1 {
			c.itemID = itemInts[0]
		}
	} else {
		c.showItemSearch()
		return
	}

	c.updateLocationLabel()
}

func (c *CommitUI) updateLocationLabel() {
	if c.location != "" {
		itemName := c.items_r[c.itemID]
		if itemName == "" {
			itemName = fmt.Sprintf("ID: %d", c.itemID)
		}
		c.locationLabel.SetText(fmt.Sprintf("Location: %s\nItem: %s", c.location, itemName))
	}
}

func (c *CommitUI) toggleMode() {
	if c.mode == "ADD" {
		c.mode = "SUB"
	} else {
		c.mode = "ADD"
	}
	c.toggleBtn.SetText("Mode: " + c.mode)
}

func (c *CommitUI) commit() {
	if c.location == "" || c.itemID == 0 {
		c.setError("No location or item selected")
		return
	}

	qty, err := strconv.Atoi(c.deltaInput.Text)
	if err != nil {
		c.setError("Invalid number")
		return
	}

	if c.mode == "SUB" {
		qty = -qty
	}

	c.queue.SubmitCommit("TOUGHPAD01", c.location, qty, c.itemID)
	c.deltaInput.SetText("")
	c.setError("")
	// Focus removed - Fyne v2 doesn't support Entry.Focus()
}

func (c *CommitUI) setError(msg string) {
	if msg == "" {
		c.error.ParseMarkdown("")
	} else {
		c.error.ParseMarkdown("**Error:** " + msg)
	}
}

func (c *CommitUI) showItemSelectDialog(itemIDs []int) {
	// Simplified: just select first one
	if len(itemIDs) > 0 {
		c.itemID = itemIDs[0]
		c.updateLocationLabel()
	}
}

func (c *CommitUI) showItemSearch() {
	// Basic implementation
	chosenItem := ""
	for name := range c.items {
		chosenItem = name
		break
	}
	if chosenItem != "" {
		c.itemID = c.items[chosenItem]
		c.updateLocationLabel()
	}
}

func (c *CommitUI) CreateRenderer() fyne.WidgetRenderer {
	c.scannerInput = widget.NewEntry()
	c.scannerInput.SetPlaceHolder("Scan location code...")
	c.scannerInput.OnSubmitted = func(s string) {
		c.onScanned(s)
		c.scannerInput.SetText("")
	}

	c.locationLabel = widget.NewLabel("Location: (waiting for scan)")

	c.deltaInput = widget.NewEntry()
	c.deltaInput.SetPlaceHolder("Enter quantity")

	c.toggleBtn = widget.NewButton("Mode: ADD", func() {
		c.toggleMode()
	})

	c.commitBtn = widget.NewButton("Commit", func() {
		c.commit()
	})

	c.changeItemBtn = widget.NewButton("Change Item", func() {
		c.showItemSearch()
	})

	c.error = widget.NewRichTextFromMarkdown("")

	buttons := container.NewHBox(
		c.toggleBtn,
		c.commitBtn,
		c.changeItemBtn,
	)

	vbox := container.NewVBox(
		c.scannerInput,
		c.locationLabel,
		c.deltaInput,
		buttons,
		c.error,
	)

	return widget.NewSimpleRenderer(vbox)
}
