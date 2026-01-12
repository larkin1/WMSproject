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
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/larkin1/wmsproject/internal/api"
	"github.com/larkin1/wmsproject/internal/queue"
)

type CommitUI struct {
	widget.BaseWidget

	scannerInput  *widget.Entry
	locationLabel *widget.Label
	deltaInput    *widget.Entry
	toggleBtn     *widget.Button
	commitBtn     *widget.Button
	changeItemBtn *widget.Button
	error         *widget.RichText

	mode      string
	location  string
	itemID    int
	locations map[string][]int
	items     map[string]int
	items_r   map[int]string

	api       *api.Client
	queue     *queue.Queue
	basePath  string
	canvasObj fyne.CanvasObject // Store the canvas to show dialogs
}

func NewCommitUI(apiClient *api.Client, commitQueue *queue.Queue, basePath string) *CommitUI {
	c := &CommitUI{
		api:       apiClient,
		queue:     commitQueue,
		basePath:  basePath,
		mode:      "ADD",
		items:     make(map[string]int),
		items_r:   make(map[int]string),
		locations: make(map[string][]int),
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
	locationsData, err := c.api.FetchLocations()
	if err != nil {
		return
	}

	c.locations = make(map[string][]int)
	for _, loc := range locationsData {
		c.locations[loc.LocationName] = loc.Items
	}
}

func (c *CommitUI) onScanned(text string) {
	c.location = strings.TrimSpace(text)
	c.loadLocations()

	if itemIDs, ok := c.locations[c.location]; ok {
		// Location exists
		if len(itemIDs) == 0 {
			c.setError("Location has no items")
			return
		}

		if len(itemIDs) > 1 {
			c.showItemSelectDialog(itemIDs)
			return
		} else if len(itemIDs) == 1 {
			c.itemID = itemIDs[0]
		}
	} else {
		// Location doesn't exist - show search for new location
		c.setError(fmt.Sprintf("Location '%s' not found. Use Change Item to select an item for new location.", c.location))
		c.itemID = 0
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
		c.setError("")
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
}

func (c *CommitUI) setError(msg string) {
	if msg == "" {
		c.error.ParseMarkdown("")
	} else {
		c.error.ParseMarkdown("**Error:** " + msg)
	}
}

func (c *CommitUI) showItemSelectDialog(itemIDs []int) {
	// Create a dialog to select from multiple items at this location
	items := make([]string, len(itemIDs))
	itemMap := make(map[string]int) // map from display name to ID

	for i, id := range itemIDs {
		name := c.items_r[id]
		if name == "" {
			name = fmt.Sprintf("ID: %d", id)
		}
		items[i] = name
		itemMap[name] = id
	}

	select := widget.NewSelect(items, func(value string) {
		if id, ok := itemMap[value]; ok {
			c.itemID = id
			c.updateLocationLabel()
		}
	})
	select.PlaceHolder = "Select item..."
	if len(items) > 0 {
		select.SetSelected(items[0])
		c.itemID = itemMap[items[0]]
	}

	d := dialog.NewForm(
		[]*widget.FormItem{
			widget.NewFormItem("Items at this location", select),
		},
		"Select",
		"Cancel",
		func(ok bool) {
			if ok {
				c.updateLocationLabel()
			}
		},
		c.canvasObj.(fyne.Window),
	)
	d.Show()
}

func (c *CommitUI) showItemSearch() {
	// Create a search/select dialog for items
	var itemNames []string
	for name := range c.items {
		itemNames = append(itemNames, name)
	}

	select := widget.NewSelect(itemNames, func(value string) {
		if id, ok := c.items[value]; ok {
			c.itemID = id
		}
	})
	select.PlaceHolder = "Search or select item..."

	d := dialog.NewForm(
		[]*widget.FormItem{
			widget.NewFormItem("Item", select),
		},
		"Select",
		"Cancel",
		func(ok bool) {
			if ok && select.Selected != "" {
				c.updateLocationLabel()
			}
		},
		c.canvasObj.(fyne.Window),
	)
	d.Show()
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

	// Store the canvas for dialog rendering
	c.canvasObj = vbox

	return widget.NewSimpleRenderer(vbox)
}
