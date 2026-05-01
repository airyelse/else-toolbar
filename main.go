package main

import (
	"embed"
	"log"

	"else-toolbox/internal/settings"

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
			AdditionalBrowserArgs:         []string{"--disable-gpu"},
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
	svc.SetMainWindow(window)

	// Close button behavior: backend is the single source of truth.
	// Use RegisterHook (synchronous + cancellable) instead of OnWindowEvent (async)
	// to prevent the framework from destroying the window.
	// If the saved close behavior is unknown OR settings fail to load,
	// cancel and emit "window:close-requested" so the frontend can prompt.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if svc.ShouldBypassCloseConfirm() {
			return
		}

		s, err := settings.Load()
		if err != nil {
			// Settings load failed — ask the frontend instead of guessing
			e.Cancel()
			svc.emitEvent("window:close-requested", nil)
			return
		}

		switch s.CloseBehavior {
		case "quit":
			e.Cancel()
			svc.QuitApp()
			return
		case "minimize":
			e.Cancel()
			window.Hide()
			return
		}

		// Unknown / empty — let the frontend decide
		e.Cancel()
		svc.emitEvent("window:close-requested", nil)
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
