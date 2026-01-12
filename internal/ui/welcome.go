package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

	title := widget.NewLabel("Warehouse Management System")
	subtitle := widget.NewLabel("")

	vbox := container.NewVBox(
		container.NewCenter(title),
		container.NewCenter(subtitle),
		addBtn,
		exitBtn,
	)

	return widget.NewSimpleRenderer(container.NewCenter(vbox))
}
