package main

import (
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Forensic Disk Imaging Tool",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.OnStartup,
		Bind: []interface{}{
			app, // This exposes methods like GetAvailableDisks() to the frontend
		},
		/*	BackgroundColour: &options.RGBA{R: 26, G: 26, B: 26, A: 1},
			Resizable:        true,
			Fullscreen:       false,
			Debug: options.Debug{
				OpenInspectorOnStartup: false,
			},*/
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
