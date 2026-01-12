package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/larkin1/wmsproject/internal/api"
)

type SettingsUI struct {
	widget.BaseWidget

	urlInput   *widget.Entry
	keyInput   *widget.Entry
	submitBtn  *widget.Button
	errLabel   *widget.RichText

	onSubmit func(url, key string)
	basePath string
}

func NewSettingsUI(onSubmit func(url, key string), basePath string) *SettingsUI {
	return &SettingsUI{
		onSubmit: onSubmit,
		basePath: basePath,
	}
}

func (s *SettingsUI) checkCredentials(url, key string) bool {
	client := api.NewClient(url, key, s.basePath)
	return client.Check()
}

func (s *SettingsUI) submit() {
	url := s.urlInput.Text
	key := s.keyInput.Text

	if url == "" || key == "" {
		s.setError("URL and key cannot be empty")
		return
	}

	// Auto-prefix https if needed
	if !string([]rune(url)[0:4]) == "http" {
		url = "https://" + url
	}

	s.setError("Checking credentials...")

	if !s.checkCredentials(url, key) {
		s.setError("Invalid credentials or cannot connect")
		return
	}

	s.setError("")
	s.onSubmit(url, key)
}

func (s *SettingsUI) setError(msg string) {
	if msg == "" {
		s.errLabel.ParseMarkdown("")
	} else {
		s.errLabel.ParseMarkdown(fmt.Sprintf("**%s**", msg))
	}
}

func (s *SettingsUI) CreateRenderer() fyne.WidgetRenderer {
	s.urlInput = widget.NewEntry()
	s.urlInput.SetPlaceHolder("API Base URL (e.g., https://your-api.example.com)")
	s.urlInput.OnSubmitted = func(text string) {
		s.keyInput.Focus()
	}

	s.keyInput = widget.NewEntry()
	s.keyInput.SetPlaceHolder("API Key")
	s.keyInput.Password = true
	s.keyInput.OnSubmitted = func(text string) {
		s.submit()
	}

	s.submitBtn = widget.NewButton("Submit", func() {
		s.submit()
	})
	s.submitBtn.Importance = widget.HighImportance

	s.errLabel = widget.NewRichTextFromMarkdown("")

	vbox := container.NewVBox(
		widget.NewLabelWithAlignment("Warehouse Management System", fyne.TextAlignCenter),
		widget.NewLabelWithAlignment("Initial Configuration", fyne.TextAlignCenter),
		widget.NewLabel(""),
		widget.NewLabel("API Configuration:"),
		s.urlInput,
		s.keyInput,
		s.submitBtn,
		s.errLabel,
	)

	return widget.NewSimpleRenderer(container.NewCenter(vbox))
}
