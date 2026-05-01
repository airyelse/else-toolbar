package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

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
			AdditionalBrowserArgs:     []string{"--disable-gpu"},
			DisableQuitOnLastWindowClosed: true,
		},
	})

	svc.SetApp(app)

	// System tray
	tray := app.SystemTray.New()
	tray.SetLabel("else-toolbox")
	tray.SetTooltip("Else Toolbox")

	menu := app.NewMenu()
	menu.Add("显示/隐藏窗口")
	menu.AddSeparator()
	menu.AddRole(application.Quit)
	tray.SetIcon(appIcon).SetMenu(menu).Run()

	// Main window
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "else-toolbox",
		Width:            1024,
		Height:           768,
		BackgroundColour: application.NewRGBA(27, 38, 54, 255),
	})

	// Close button hides to tray instead of quitting
	window.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
