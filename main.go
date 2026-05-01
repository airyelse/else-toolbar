package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

func main() {
	svc := NewApp()

	app := application.New(application.Options{
		Name: "else-toolbox",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Windows: application.WindowsOptions{
			AdditionalBrowserArgs: []string{"--disable-gpu"},
		},
	})

	svc.SetApp(app)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "else-toolbox",
		Width:            1024,
		Height:           768,
		BackgroundColour: application.NewRGBA(27, 38, 54, 255),
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
