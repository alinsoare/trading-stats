package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// kpiCard is a titled panel with a large coloured value label.
type kpiCard struct {
	object *fyne.Container
	value  *canvas.Text
}

func newKpiCard(title string) *kpiCard {
	bg := canvas.NewRectangle(colBGSurface)
	bg.StrokeColor = colBorder
	bg.StrokeWidth = 1
	bg.CornerRadius = 6

	titleText := canvas.NewText(strings.ToUpper(title), colTextSecond)
	titleText.TextSize = 10
	titleText.TextStyle = fyne.TextStyle{Bold: true}

	value := canvas.NewText("—", colTextPrimary)
	value.TextSize = 20
	value.TextStyle = fyne.TextStyle{Bold: true}

	inner := container.NewVBox(titleText, value)
	stack := container.NewStack(bg, container.NewPadded(inner))
	return &kpiCard{object: stack, value: value}
}

// setValue updates the displayed value. positive controls colour:
// true -> green, false -> red, nil -> default.
func (k *kpiCard) setValue(text string, positive *bool) {
	switch {
	case positive == nil:
		k.value.Color = colTextPrimary
	case *positive:
		k.value.Color = colPositive
	default:
		k.value.Color = colNegative
	}
	k.value.Text = text
	k.value.Refresh()
}

func (k *kpiCard) reset() { k.setValue("—", nil) }

func boolPtr(b bool) *bool { return &b }
