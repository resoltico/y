package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func (app *OtsuApp) setupMenus() {
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Open Image...", func() {
			app.handleImageLoad()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Save Processed Image...", func() {
			app.handleImageSave()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			app.fyneApp.Quit()
		}),
	)

	debugMenu := fyne.NewMenu("Debug",
		fyne.NewMenuItem("Memory Stats", func() {
			stats := app.debugManager.GetMemoryStats()
			dialog.ShowInformation("Memory Statistics", stats, app.window)
		}),
		fyne.NewMenuItem("Performance Report", func() {
			report := app.debugManager.GetPerformanceReport()
			dialog.ShowInformation("Performance Report", report, app.window)
		}),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, debugMenu)
	app.window.SetMainMenu(mainMenu)
}
