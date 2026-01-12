package ui

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	window    fyne.Window // Store the window for dialogs
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

	return c
}

func (c *CommitUI) loadItems() {
	// Clear old data
	c.items = make(map[string]int)
	c.items_r = make(map[int]string)

	itemsCSV := filepath.Join(c.basePath, "items.csv")

	// Always try to fetch fresh data from API
	err := c.api.ExportItemsToCSV(itemsCSV)
	if err != nil {
		// API call failed, try to use cached CSV
	}

	// Load from CSV (either fresh or cached)
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
			id, err := strconv.Atoi(strings.TrimSpace(record[0]))
			if err != nil {
				continue
			}
			name := strings.TrimSpace(record[1])
			if name != "" {
				c.items[name] = id
				c.items_r[id] = name
			}
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
		// Location doesn't exist - automatically show item picker
		c.setError(fmt.Sprintf("New location '%s' - select an item below:", c.location))
		c.itemID = 0
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
		c.error.ParseMarkdown("**Status:** " + msg)
	}
}

func (c *CommitUI) showItemSelectDialog(itemIDs []int) {
	// Create options for the select widget
	options := make([]string, len(itemIDs))
	itemMap := make(map[string]int)

	for i, id := range itemIDs {
		name := c.items_r[id]
		if name == "" {
			name = fmt.Sprintf("ID: %d", id)
		}
		options[i] = name
		itemMap[name] = id
	}

	// Create the select widget
	selectWidget := widget.NewSelect(options, func(value string) {
		if id, ok := itemMap[value]; ok {
			c.itemID = id
		}
	})
	selectWidget.PlaceHolder = "Select item..."
	if len(options) > 0 {
		selectWidget.SetSelected(options[0])
		c.itemID = itemMap[options[0]]
	}

	// Create form
	form := container.NewVBox(
		widget.NewLabel("Multiple items found at this location. Select one:"),
		selectWidget,
	)

	dlg := dialog.NewCustom("Select Item", "OK", form, c.window)
	dlg.Show()
}

func (c *CommitUI) showItemSearch() {
	// Ensure items are loaded
	c.loadItems()

	// Build sorted list of item names
	var itemNames []string
	for name := range c.items {
		itemNames = append(itemNames, name)
	}
	sort.Strings(itemNames)

	if len(itemNames) == 0 {
		c.setError("No items loaded from database")
		return
	}

	// Create the select widget
	selectWidget := widget.NewSelect(itemNames, func(value string) {
		if id, ok := c.items[value]; ok {
			c.itemID = id
			c.updateLocationLabel()
		}
	})
	selectWidget.PlaceHolder = "Search or select item..."

	// Create form
	form := container.NewVBox(
		widget.NewLabel("Select an item:"),
		selectWidget,
	)

	dlg := dialog.NewCustom("Change Item", "OK", form, c.window)
	dlg.Show()
}

func (c *CommitUI) CreateRenderer() fyne.WidgetRenderer {
	// Load data when renderer is created
	c.loadItems()
	c.loadLocations()

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

// SetWindow allows main to pass the window reference
func (c *CommitUI) SetWindow(w fyne.Window) {
	c.window = w
}
