package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed appicon.png
var iconBytes []byte

// AppIcon returns the application icon used for the window, taskbar and dock.
// Setting it via app.SetIcon makes the OS show it on the Linux/Windows taskbar
// (the macOS dock icon comes from the bundled icon.icns).
func AppIcon() fyne.Resource {
	return fyne.NewStaticResource("appicon.png", iconBytes)
}
