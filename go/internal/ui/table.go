package ui

import (
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

var cachedRowH float32

// tableRowHeight is the uniform height of one widget.Table row, used as the
// pixel step for re-laying-out the chart while the splitter is dragged.
func tableRowHeight() float32 {
	if cachedRowH == 0 {
		cachedRowH = widget.NewLabel("Ag").MinSize().Height
	}
	return cachedRowH
}

// dataTable wraps a widget.Table with a simple headers + rows model and
// click-to-sort column headers (numeric-aware).
type dataTable struct {
	object  fyne.CanvasObject
	table   *widget.Table
	headers []string
	rows    [][]string // current (possibly sorted) view
	orig    [][]string // original row order

	sortCol int // -1 = unsorted
	sortAsc bool
}

func newDataTable() *dataTable {
	d := &dataTable{sortCol: -1}
	t := widget.NewTable(d.size, d.create, d.update)
	t.ShowHeaderRow = true
	t.CreateHeader = func() fyne.CanvasObject {
		b := widget.NewButton("", nil)
		b.Importance = widget.LowImportance
		b.Alignment = widget.ButtonAlignLeading
		return b
	}
	t.UpdateHeader = d.updateHeader
	d.table = t
	d.object = t
	return d
}

func (d *dataTable) size() (int, int) {
	if len(d.headers) == 0 {
		return 0, 0
	}
	return len(d.rows), len(d.headers)
}

func (d *dataTable) create() fyne.CanvasObject { return widget.NewLabel("") }

func (d *dataTable) update(id widget.TableCellID, o fyne.CanvasObject) {
	lbl := o.(*widget.Label)
	if id.Row >= 0 && id.Row < len(d.rows) && id.Col >= 0 && id.Col < len(d.rows[id.Row]) {
		lbl.SetText(d.rows[id.Row][id.Col])
		return
	}
	lbl.SetText("")
}

func (d *dataTable) updateHeader(id widget.TableCellID, o fyne.CanvasObject) {
	btn, ok := o.(*widget.Button)
	if !ok {
		return
	}
	col := id.Col
	if col < 0 || col >= len(d.headers) {
		btn.SetText("")
		btn.OnTapped = nil
		return
	}
	text := d.headers[col]
	if col == d.sortCol {
		if d.sortAsc {
			text += " \u25b2" // ▲
		} else {
			text += " \u25bc" // ▼
		}
	}
	btn.SetText(text)
	btn.OnTapped = func() { d.sortBy(col) }
}

func (d *dataTable) set(headers []string, rows [][]string) {
	d.headers = headers
	d.orig = rows
	if d.sortCol >= len(headers) {
		d.sortCol = -1
	}
	d.applySort()
	d.table.Refresh()
	for c := 0; c < len(headers); c++ {
		wd := len(headers[c]) + 2 // room for the sort arrow
		for r := 0; r < len(rows); r++ {
			if c < len(rows[r]) && len(rows[r][c]) > wd {
				wd = len(rows[r][c])
			}
		}
		px := float32(wd)*8 + 24
		if px < 60 {
			px = 60
		}
		if px > 280 {
			px = 280
		}
		d.table.SetColumnWidth(c, px)
	}
}

// sortBy toggles the sort direction for a column (or selects it ascending).
func (d *dataTable) sortBy(col int) {
	if col < 0 || col >= len(d.headers) {
		return
	}
	if d.sortCol == col {
		d.sortAsc = !d.sortAsc
	} else {
		d.sortCol = col
		d.sortAsc = true
	}
	d.applySort()
	d.table.Refresh()
}

// applySort rebuilds d.rows from the original order according to the current
// sort column/direction. Empty/non-numeric cells in a numeric column sort last.
func (d *dataTable) applySort() {
	d.rows = make([][]string, len(d.orig))
	copy(d.rows, d.orig)
	if d.sortCol < 0 || d.sortCol >= len(d.headers) {
		return
	}
	col := d.sortCol
	numeric := columnNumeric(d.orig, col)
	asc := d.sortAsc
	sort.SliceStable(d.rows, func(i, j int) bool {
		a, b := cellAt(d.rows[i], col), cellAt(d.rows[j], col)
		if numeric {
			fa, oka := parseNum(a)
			fb, okb := parseNum(b)
			if !oka && !okb {
				return false
			}
			if !oka {
				return false // a missing -> after b
			}
			if !okb {
				return true // b missing -> a before b
			}
			if asc {
				return fa < fb
			}
			return fa > fb
		}
		if asc {
			return a < b
		}
		return a > b
	})
}

func cellAt(row []string, col int) string {
	if col >= 0 && col < len(row) {
		return row[col]
	}
	return ""
}

// columnNumeric reports whether every non-empty cell in a column parses as a
// number (so it should be compared numerically).
func columnNumeric(rows [][]string, col int) bool {
	any := false
	for _, r := range rows {
		s := strings.TrimSpace(cellAt(r, col))
		if s == "" {
			continue
		}
		if _, ok := parseNum(s); !ok {
			return false
		}
		any = true
	}
	return any
}

func parseNum(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "%", "")
	s = strings.ReplaceAll(s, " ", "")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}
