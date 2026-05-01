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

	// Left click toggles window, right click shows menu
	tray.OnClick(func() {
		tray.ToggleWindow()
	})

	menu := app.NewMenu()
	menu.Add("显示/隐藏窗口").OnClick(func(_ *application.Context) {
		tray.ToggleWindow()
	})
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

	tray.AttachWindow(window)

	// Close button hides to tray instead of quitting
	// Use RegisterHook (synchronous + cancellable) instead of OnWindowEvent (async)
	// to prevent the framework from destroying the window
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel()
		window.Hide()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
