package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type WelcomeScreen struct {
	widget.BaseWidget
	onScreenChange func(string)
}

func NewWelcomeScreen(onScreenChange func(string)) *WelcomeScreen {
	return &WelcomeScreen{
		onScreenChange: onScreenChange,
	}
}

func (w *WelcomeScreen) CreateRenderer() fyne.WidgetRenderer {
	addBtn := widget.NewButton("Add/Remove Stock", func() {
		w.onScreenChange("commit")
	})
	addBtn.Importance = widget.HighImportance

	exitBtn := widget.NewButton("Exit", func() {
		fyne.CurrentApp().Quit()
	})

	vbox := container.NewVBox(
		widget.NewLabelWithAlignment("Warehouse Management System", fyne.TextAlignCenter),
		widget.NewLabelWithAlignment("", fyne.TextAlignCenter),
		addBtn,
		exitBtn,
	)

	return widget.NewSimpleRenderer(container.NewCenter(vbox))
}
