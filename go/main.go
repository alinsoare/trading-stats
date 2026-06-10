// Command trading-stats is a self-contained desktop app for analysing
// MetaTrader 5 deal CSV exports across multiple accounts.
package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/alinsoare/trading-stats/go/internal/ui"
)

func main() {
	a := app.NewWithID("com.alinsoare.tradingstats")
	a.Settings().SetTheme(ui.Theme())
	w := ui.NewWindow(a)
	w.Show()
}
