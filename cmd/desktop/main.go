// Command desktop is the LocalOps desktop application entry point. It
// serves the same LocalOps core (storage, project inspection, Doctor,
// Overview) as the CLI, adapted for the frontend through
// internal/desktop.Service.
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/msiuda/localops/internal/desktop"
	"github.com/msiuda/localops/internal/storage"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	store, err := defaultStore()
	if err != nil {
		log.Fatal(err)
	}

	app := application.New(application.Options{
		Name:        "LocalOps",
		Description: "LocalOps desktop",
		Services: []application.Service{
			application.NewService(desktop.NewService(store)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Dark, neutral background matches the app's default (dark) theme and
	// avoids a white flash before the frontend paints. Frameless is left
	// false: on macOS, Wails only applies Mac.TitleBar settings and only
	// keeps the native traffic-light buttons enabled when Frameless is
	// false (Frameless forces its own borderless/rounded-corner frame and
	// explicitly hides all three title-bar buttons). MacTitleBarHidden
	// gives the desired VS Code-style look instead: the title bar is
	// hidden/transparent, content extends full-size behind it, and the
	// standard native AppKit traffic lights remain at their normal inset.
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "LocalOps",
		Width:            1280,
		Height:           820,
		MinWidth:         1000,
		MinHeight:        650,
		BackgroundColour: application.NewRGB(22, 23, 26),
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHidden,
		},
		URL: "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// defaultStore resolves the same registered-project storage location the
// CLI uses, so the desktop app and CLI share one project registry.
func defaultStore() (*storage.Store, error) {
	path, err := storage.DefaultPath()
	if err != nil {
		return nil, err
	}

	return storage.New(path), nil
}
