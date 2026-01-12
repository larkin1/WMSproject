package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/larkin1/wmsproject/internal/api"
	"github.com/larkin1/wmsproject/internal/queue"
	"github.com/larkin1/wmsproject/internal/ui"
)

var (
	basePath     string
	settingsPath string
	appAPI       *api.Client
	commitQueue  *queue.Queue
)

func init() {
	if exe, err := os.Executable(); err == nil {
		basePath = filepath.Dir(exe)
	} else {
		basePath, _ = os.Getwd()
	}
	settingsPath = filepath.Join(basePath, "settings.json")
}

func loadSettings() (bool, error) {
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		// Create empty settings
		emptySettings := map[string]string{
			"api_url": "",
			"api_key": "",
		}
		data, _ := json.MarshalIndent(emptySettings, "", "  ")
		os.WriteFile(settingsPath, data, 0644)
		return false, nil
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return false, err
	}

	var settings map[string]string
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return false, err
	}

	if settings["api_url"] == "" || settings["api_key"] == "" {
		return false, nil
	}

	appAPI = api.NewClient(settings["api_url"], settings["api_key"], basePath)
	commitQueue = queue.NewQueue(appAPI, basePath)
	commitQueue.Start()

	return true, nil
}

func main() {
	a := app.New()
	w := a.NewWindow("WMS - Warehouse Management System")
	w.Resize(fyne.NewSize(600, 800))

	hasSettings, _ := loadSettings()

	if !hasSettings {
		// Show settings screen
		settingsUI := ui.NewSettingsUI(func(apiURL, apiKey string) {
			appAPI = api.NewClient(apiURL, apiKey, basePath)
			commitQueue = queue.NewQueue(appAPI, basePath)
			commitQueue.Start()

			// Save settings
			settings := map[string]string{
				"api_url": apiURL,
				"api_key": apiKey,
			}
			data, _ := json.MarshalIndent(settings, "", "  ")
			os.WriteFile(settingsPath, data, 0644)

			// Show welcome screen
			w.SetContent(makeApp())
		}, basePath)

		w.SetContent(settingsUI)
	} else {
		w.SetContent(makeApp())
	}

	w.ShowAndRun()

	if commitQueue != nil {
		commitQueue.Stop()
	}
}

func makeApp() fyne.CanvasObject {
	return container.NewVBox(
		ui.NewWelcomeScreen(func(screen string) {
			// Handle screen switching
		}),
	)
}
