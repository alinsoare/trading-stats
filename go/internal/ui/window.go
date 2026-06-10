// Package ui builds the Fyne desktop window: paths, load, filters, KPIs,
// equity chart and tables. Mirrors desktop/.../main_window.py.
package ui

import (
	"encoding/csv"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/alinsoare/trading-stats/go/internal/ingest"
	"github.com/alinsoare/trading-stats/go/internal/model"
	"github.com/alinsoare/trading-stats/go/internal/stats"
)

const dateLayout = "2006-01-02"

var kpiTitles = []string{
	"Trades", "Win rate", "Net P/L", "Profit factor",
	"Max DD", "Expectancy", "Payoff", "Breakeven",
}

type acctWidget struct {
	label   string
	check   *widget.Check
	beEntry *widget.Entry
}

// Window holds all UI state for the application.
type Window struct {
	app   fyne.App
	win   fyne.Window
	prefs fyne.Preferences

	deals      []model.Deal
	pos        []model.ClosedPosition
	thresholds map[string]float64

	paths        []string
	selectedPath int
	pathList     *widget.List
	status       *widget.Label

	accountsBox *fyne.Container
	accts       []*acctWidget
	fromEntry   *widget.Entry
	toEntry     *widget.Entry
	bucket      *widget.Select

	cards []*kpiCard

	leftScroll *container.Scroll
	panelBtn   *widget.Button

	chartW *chartWidget

	rollupTbl  *dataTable
	symbolTbl  *dataTable
	weekdayTbl *dataTable
	flowTbl    *dataTable
	acctTbl    *dataTable
}

// NewWindow constructs the main application window.
func NewWindow(app fyne.App) *Window {
	w := &Window{
		app:          app,
		prefs:        app.Preferences(),
		selectedPath: -1,
		thresholds:   map[string]float64{},
	}
	w.win = app.NewWindow("Trading Stats")
	w.thresholds = loadThresholds(w.prefs)
	w.paths = loadPaths(w.prefs)

	w.win.SetContent(w.buildUI())
	w.win.Resize(fyne.NewSize(1280, 820))
	w.win.SetOnClosed(w.persist)
	return w
}

// Show displays the window.
func (w *Window) Show() { w.win.ShowAndRun() }

func (w *Window) persist() {
	savePaths(w.prefs, w.paths)
	saveThresholds(w.prefs, w.thresholds)
}

func (w *Window) buildUI() fyne.CanvasObject {
	panel := w.buildLeftPanel()
	w.leftScroll = container.NewVScroll(panel)
	w.leftScroll.SetMinSize(fyne.NewSize(280, 100))
	right := w.buildRightContent()
	return container.NewBorder(nil, nil, w.leftScroll, nil, right)
}

// ── left panel (data folders + accounts + filters) ──────────────────────────

func (w *Window) buildLeftPanel() fyne.CanvasObject {
	// Data folders
	foldersTitle := widget.NewLabelWithStyle("DATA FOLDERS", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	w.pathList = widget.NewList(
		func() int { return len(w.paths) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(w.paths[i])
		},
	)
	w.pathList.OnSelected = func(id widget.ListItemID) { w.selectedPath = id }
	w.pathList.OnUnselected = func(widget.ListItemID) { w.selectedPath = -1 }
	pathScroll := container.NewVScroll(w.pathList)
	pathScroll.SetMinSize(fyne.NewSize(0, 90))

	browseBtn := widget.NewButton("Browse…", w.onBrowse)
	pasteBtn := widget.NewButton("Paste path…", w.onPaste)
	removeBtn := widget.NewButton("Remove", w.onRemovePath)
	btnRow := container.NewGridWithColumns(3, browseBtn, pasteBtn, removeBtn)

	loadBtn := widget.NewButton("Load data", w.onLoad)
	loadBtn.Importance = widget.HighImportance

	w.status = widget.NewLabel("Configure paths, then Load data.")
	w.status.Wrapping = fyne.TextWrapWord

	dataSection := container.NewVBox(
		foldersTitle, pathScroll, btnRow, loadBtn, w.status,
	)

	// Accounts (rebuilt on load)
	w.accountsBox = container.NewVBox()
	acctScroll := container.NewVScroll(w.accountsBox)
	acctScroll.SetMinSize(fyne.NewSize(240, 140))
	acctSection := container.NewBorder(
		widget.NewLabelWithStyle("ACCOUNTS", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil, acctScroll,
	)

	// Date / rollup / export
	w.fromEntry = widget.NewEntry()
	w.fromEntry.SetPlaceHolder("YYYY-MM-DD")
	w.fromEntry.OnChanged = func(string) { w.refresh() }
	w.toEntry = widget.NewEntry()
	w.toEntry.SetPlaceHolder("YYYY-MM-DD")
	w.toEntry.OnChanged = func(string) { w.refresh() }

	w.bucket = widget.NewSelect([]string{"day", "week", "month", "year"}, func(string) { w.refresh() })
	w.bucket.SetSelected("month")

	exportBtn := widget.NewButton("Export CSV…", w.onExport)

	filterForm := container.New(layout.NewFormLayout(),
		widget.NewLabel("From"), w.fromEntry,
		widget.NewLabel("To"), w.toEntry,
		widget.NewLabel("Rollup"), w.bucket,
		widget.NewLabel(""), exportBtn,
	)

	return container.NewVBox(
		dataSection,
		widget.NewSeparator(),
		acctSection,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("FILTERS", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		filterForm,
	)
}

func (w *Window) onBrowse() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		w.addPath(uri.Path())
	}, w.win)
}

func (w *Window) onPaste() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Absolute path to folder")
	form := []*widget.FormItem{widget.NewFormItem("Path", entry)}
	dialog.ShowForm("Add path", "Add", "Cancel", form, func(ok bool) {
		if ok && strings.TrimSpace(entry.Text) != "" {
			w.addPath(strings.TrimSpace(entry.Text))
		}
	}, w.win)
}

func (w *Window) addPath(p string) {
	w.paths = append(w.paths, p)
	w.pathList.Refresh()
	w.persist()
}

func (w *Window) onRemovePath() {
	if w.selectedPath < 0 || w.selectedPath >= len(w.paths) {
		return
	}
	w.paths = append(w.paths[:w.selectedPath], w.paths[w.selectedPath+1:]...)
	w.selectedPath = -1
	w.pathList.UnselectAll()
	w.pathList.Refresh()
	w.persist()
}

// ── right content ────────────────────────────────────────────────────────────

func (w *Window) buildRightContent() fyne.CanvasObject {
	w.panelBtn = widget.NewButton("‹", w.onTogglePanel)
	w.panelBtn.Importance = widget.LowImportance
	kpis := w.buildKPIs()
	topBar := container.NewBorder(nil, nil, w.panelBtn, nil, kpis)

	w.chartW = newChartWidget()
	chartCard := widget.NewCard("Cumulative Net P/L", "", w.chartW)

	tabs := w.buildTables()
	lower := container.NewVSplit(chartCard, tabs)
	lower.SetOffset(0.42)

	return container.NewBorder(topBar, nil, nil, nil, lower)
}

func (w *Window) onTogglePanel() {
	if w.leftScroll == nil {
		return
	}
	if w.leftScroll.Visible() {
		w.leftScroll.Hide()
		w.panelBtn.SetText("›")
	} else {
		w.leftScroll.Show()
		w.panelBtn.SetText("‹")
	}
}

func (w *Window) buildKPIs() fyne.CanvasObject {
	w.cards = make([]*kpiCard, len(kpiTitles))
	objs := make([]fyne.CanvasObject, len(kpiTitles))
	for i, t := range kpiTitles {
		c := newKpiCard(t)
		w.cards[i] = c
		objs[i] = c.object
	}
	return container.NewGridWithColumns(8, objs...)
}

func (w *Window) buildTables() fyne.CanvasObject {
	w.rollupTbl = newDataTable()
	w.symbolTbl = newDataTable()
	w.weekdayTbl = newDataTable()
	w.flowTbl = newDataTable()
	w.acctTbl = newDataTable()
	return container.NewAppTabs(
		container.NewTabItem("Rollup", w.rollupTbl.object),
		container.NewTabItem("By symbol", w.symbolTbl.object),
		container.NewTabItem("By weekday", w.weekdayTbl.object),
		container.NewTabItem("Non-trade sample", w.flowTbl.object),
		container.NewTabItem("Per-account", w.acctTbl.object),
	)
}

// ── load ─────────────────────────────────────────────────────────────────────

func (w *Window) onLoad() {
	w.persist()
	if len(w.paths) == 0 {
		dialog.ShowInformation("Load data", "Add at least one folder path.", w.win)
		return
	}
	deals, err := ingest.LoadDeals(w.paths)
	if err != nil {
		dialog.ShowError(err, w.win)
		return
	}
	w.deals = deals
	if len(deals) == 0 {
		w.pos = nil
		w.status.SetText("No deals_*.csv found under the given paths.")
		w.clearViews()
		return
	}

	w.pos = stats.ClosedPositions(deals)
	if len(w.pos) == 0 {
		w.status.SetText("No trade rows after filtering non-trade deals.")
		w.clearViews()
		return
	}

	var tmin, tmax time.Time
	have := false
	for _, p := range w.pos {
		if !p.HasExit {
			continue
		}
		if !have || p.ExitTime.Before(tmin) {
			tmin = p.ExitTime
		}
		if !have || p.ExitTime.After(tmax) {
			tmax = p.ExitTime
		}
		have = true
	}
	if !have {
		w.status.SetText("No valid exit times in positions.")
		w.clearViews()
		return
	}
	w.fromEntry.SetText(tmin.Format(dateLayout))
	w.toEntry.SetText(tmax.Format(dateLayout))

	w.buildAccountRows()

	w.status.SetText(fmt.Sprintf("Loaded %d deal rows → %d positions.", len(w.deals), len(w.pos)))
	w.refresh()
}

func (w *Window) buildAccountRows() {
	set := map[string]bool{}
	var accounts []string
	for _, p := range w.pos {
		if !set[p.AccountLabel] {
			set[p.AccountLabel] = true
			accounts = append(accounts, p.AccountLabel)
		}
	}
	sort.Strings(accounts)

	w.accts = nil
	w.accountsBox.Objects = nil
	for _, acc := range accounts {
		acc := acc
		if _, ok := w.thresholds[acc]; !ok {
			w.thresholds[acc] = 0.0
		}
		chk := widget.NewCheck(acc, func(bool) { w.refresh() })
		chk.SetChecked(true)
		beEntry := widget.NewEntry()
		beEntry.SetText(strconv.FormatFloat(w.thresholds[acc], 'f', -1, 64))
		beEntry.OnChanged = func(s string) {
			if v, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
				w.thresholds[acc] = v
				saveThresholds(w.prefs, w.thresholds)
				w.refresh()
			}
		}
		aw := &acctWidget{label: acc, check: chk, beEntry: beEntry}
		w.accts = append(w.accts, aw)
		row := container.NewBorder(nil, nil, chk,
			container.NewHBox(widget.NewLabel("BE ±$"),
				container.NewGridWrap(fyne.NewSize(80, 36), beEntry)),
		)
		w.accountsBox.Add(row)
	}
	w.accountsBox.Refresh()
}

// ── filters ─────────────────────────────────────────────────────────────────

func (w *Window) selectedAccounts() map[string]bool {
	out := map[string]bool{}
	for _, a := range w.accts {
		if a.check.Checked {
			out[a.label] = true
		}
	}
	return out
}

func (w *Window) filteredPositions() []model.ClosedPosition {
	if len(w.pos) == 0 {
		return nil
	}
	from, okFrom := parseDate(w.fromEntry.Text)
	to, okTo := parseDate(w.toEntry.Text)
	if !okFrom || !okTo {
		return nil
	}
	if from.After(to) {
		from, to = to, from
	}
	sel := w.selectedAccounts()
	if len(sel) == 0 {
		return nil
	}
	var out []model.ClosedPosition
	for _, p := range w.pos {
		if !p.HasExit {
			continue
		}
		d := dateOnly(p.ExitTime)
		if d.Before(from) || d.After(to) {
			continue
		}
		if !sel[p.AccountLabel] {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (w *Window) beThr() stats.ThresholdFunc {
	return func(p model.ClosedPosition) float64 { return w.thresholds[p.AccountLabel] }
}

// ── refresh ──────────────────────────────────────────────────────────────────

func (w *Window) refresh() {
	if len(w.pos) == 0 {
		return
	}
	pos := w.filteredPositions()
	if len(pos) == 0 {
		w.status.SetText("No positions for current filters.")
		w.clearKPIsTablesChart()
		return
	}
	thr := w.beThr()
	agg := stats.SummarizePositions(pos, "filtered", thr)
	if agg.Trades == 0 {
		w.status.SetText("No trades in current filters.")
		w.clearKPIsTablesChart()
		return
	}

	w.setKPICards(agg)
	w.renderChart(pos)

	bucket := stats.Bucket(w.bucket.Selected)
	rollupKey := func(p model.ClosedPosition) []string {
		return []string{p.AccountLabel, stats.PeriodOf(p.ExitTime, bucket)}
	}
	w.rollupTbl.set(groupTable(stats.SummarizeGroups(pos, rollupKey, thr),
		[]string{"account_label", "period"}))

	symbolKey := func(p model.ClosedPosition) []string { return []string{p.Symbol} }
	w.symbolTbl.set(groupTable(stats.SummarizeGroups(pos, symbolKey, thr),
		[]string{"symbol"}))

	weekdayKey := func(p model.ClosedPosition) []string { return []string{weekdayLabel(p)} }
	weekdayRows := stats.SummarizeGroups(pos, weekdayKey, thr)
	sort.SliceStable(weekdayRows, func(i, j int) bool {
		return weekdayOrder(weekdayRows[i].Keys[0]) < weekdayOrder(weekdayRows[j].Keys[0])
	})
	w.weekdayTbl.set(groupTable(weekdayRows, []string{"weekday"}))

	acctKey := func(p model.ClosedPosition) []string { return []string{p.AccountLabel} }
	w.acctTbl.set(groupTable(stats.SummarizeGroups(pos, acctKey, thr),
		[]string{"account_label"}))

	w.setFlowsTable()
}

func (w *Window) setKPICards(k model.KPI) {
	w.cards[0].setValue(strconv.Itoa(k.Trades), nil)

	wr := k.WinRate
	w.cards[1].setValue(fmt.Sprintf("%.1f%%", wr*100), boolPtr(wr >= 0.5))

	net := k.NetPnL
	var netPos *bool
	if net != 0 {
		netPos = boolPtr(net > 0)
	}
	w.cards[2].setValue(fmt.Sprintf("%.2f", net), netPos)

	pf := k.ProfitFactor
	var pfPos *bool
	if pf != 0 && !math.IsInf(pf, 0) {
		pfPos = boolPtr(pf > 1)
	}
	w.cards[3].setValue(fmtPF(pf), pfPos)

	dd := k.MaxDDCumPnL
	var ddPos *bool
	if dd < 0 {
		ddPos = boolPtr(false)
	}
	w.cards[4].setValue(fmt.Sprintf("%.2f", dd), ddPos)

	exp := k.Expectancy
	var expPos *bool
	if exp != 0 {
		expPos = boolPtr(exp > 0)
	}
	w.cards[5].setValue(fmt.Sprintf("%.4f", exp), expPos)

	po := k.Payoff
	var poPos *bool
	if po != 0 {
		poPos = boolPtr(po > 1)
	}
	w.cards[6].setValue(fmt.Sprintf("%.2f", po), poPos)

	w.cards[7].setValue(strconv.Itoa(k.Breakeven), nil)
}

func (w *Window) renderChart(pos []model.ClosedPosition) {
	w.chartW.SetSeries(stats.EquityCurves(pos))
}

// weekdayLabel groups a position by the weekday of its exit time.
func weekdayLabel(p model.ClosedPosition) string {
	if !p.HasExit {
		return "(no exit)"
	}
	return p.ExitTime.Weekday().String()
}

// weekdayOrder maps a weekday label to its Monday-first position for sorting.
func weekdayOrder(s string) int {
	switch s {
	case time.Monday.String():
		return 0
	case time.Tuesday.String():
		return 1
	case time.Wednesday.String():
		return 2
	case time.Thursday.String():
		return 3
	case time.Friday.String():
		return 4
	case time.Saturday.String():
		return 5
	case time.Sunday.String():
		return 6
	default:
		return 7
	}
}

func (w *Window) setFlowsTable() {
	flows := ingest.AccountFlows(w.deals)
	sel := w.selectedAccounts()
	headers := []string{"account_label", "time", "type", "symbol", "profit", "comment"}
	var rows [][]string
	for _, d := range flows {
		if len(sel) > 0 && !sel[d.AccountLabel] {
			continue
		}
		t := ""
		if d.HasTime {
			t = d.Time.Format("2006-01-02 15:04:05")
		}
		rows = append(rows, []string{
			d.AccountLabel, t, strconv.Itoa(d.Type), d.Symbol,
			strconv.FormatFloat(d.Profit, 'f', -1, 64), d.Comment,
		})
		if len(rows) >= 200 {
			break
		}
	}
	w.flowTbl.set(headers, rows)
}

// ── clear helpers ────────────────────────────────────────────────────────────

func (w *Window) clearKPIsTablesChart() {
	for _, c := range w.cards {
		c.reset()
	}
	w.chartW.SetSeries(nil)
	for _, t := range []*dataTable{w.rollupTbl, w.symbolTbl, w.weekdayTbl, w.flowTbl, w.acctTbl} {
		t.set(nil, nil)
	}
}

func (w *Window) clearViews() {
	w.accts = nil
	if w.accountsBox != nil {
		w.accountsBox.Objects = nil
		w.accountsBox.Refresh()
	}
	w.clearKPIsTablesChart()
}

// ── export ───────────────────────────────────────────────────────────────────

func (w *Window) onExport() {
	pos := w.filteredPositions()
	if len(pos) == 0 {
		dialog.ShowInformation("Export", "Nothing to export for current filters.", w.win)
		return
	}
	dialog.ShowFileSave(func(wc fyne.URIWriteCloser, err error) {
		if err != nil || wc == nil {
			return
		}
		defer wc.Close()
		cw := csv.NewWriter(wc)
		_ = cw.Write([]string{"account_label", "pos_key", "net_pnl", "exit_time", "symbol", "magic", "comment_sample", "n_legs"})
		for _, p := range pos {
			exit := ""
			if p.HasExit {
				exit = p.ExitTime.Format("2006-01-02 15:04:05")
			}
			_ = cw.Write([]string{
				p.AccountLabel, p.PosKey,
				strconv.FormatFloat(p.NetPnL, 'f', -1, 64),
				exit, p.Symbol, strconv.FormatInt(p.Magic, 10),
				p.CommentSample, strconv.Itoa(p.NLegs),
			})
		}
		cw.Flush()
		dialog.ShowInformation("Export", "Wrote:\n"+wc.URI().Path(), w.win)
	}, w.win)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func groupTable(groups []stats.GroupRow, keyHeaders []string) ([]string, [][]string) {
	headers := append(append([]string{}, keyHeaders...), stats.KPIColumns...)
	rows := make([][]string, 0, len(groups))
	for _, g := range groups {
		row := append([]string{}, g.Keys...)
		row = append(row, stats.KPIValues(g.KPI)...)
		rows = append(rows, row)
	}
	return headers, rows
}

func fmtPF(v float64) string {
	if math.IsInf(v, 1) {
		return "∞"
	}
	if math.IsNaN(v) {
		return "—"
	}
	return fmt.Sprintf("%.2f", v)
}

func parseDate(s string) (time.Time, bool) {
	t, err := time.Parse(dateLayout, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
