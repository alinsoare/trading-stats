package ui

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/alinsoare/trading-stats/go/internal/stats"
)

// crossColor is the hover crosshair line colour (semi-transparent accent).
var crossColor = color.NRGBA{R: 0x4f, G: 0x9c, B: 0xf9, A: 0xcc}

// Equity-curve palette (dark theme, ported from the desktop app).
var chartPalette = []color.NRGBA{
	hexColor("#4f9cf9"),
	hexColor("#4caf50"),
	hexColor("#f9a84f"),
	hexColor("#c67bf9"),
	hexColor("#f94f7c"),
	hexColor("#4ff9e2"),
}

const (
	chartPadTop    = 22
	chartPadBottom = 20
	chartPadLeft   = 56
	chartPadRight  = 14
	chartYTicks    = 4 // intervals -> chartYTicks+1 gridlines
	chartLabelSize = 10
	// Max points drawn per series. An equity curve is visually identical at
	// this density, and keeping the segment count low is what makes resizing
	// cheap (Fyne re-strokes one texture per line whose bounds change).
	chartMaxSeg = 220
	// After the size stops changing, settle once at the exact final size.
	chartSettleDelay = 90 * time.Millisecond
)

// chartWidget draws the per-account cumulative P/L curve with native Fyne vector
// primitives. Data-dependent work (downsampling, text measurement, colours) is
// done once per data change in prepare(); resizing only runs position(), which
// repositions the pooled objects with plain arithmetic and no allocation, so
// dragging the splitter is smooth.
type chartWidget struct {
	widget.BaseWidget
	series []stats.EquitySeries

	r     *chartRenderer
	hover bool
	curY  float32
}

func newChartWidget() *chartWidget {
	c := &chartWidget{}
	c.ExtendBaseWidget(c)
	return c
}

// SetSeries swaps the data and re-prepares (the only place heavy work happens).
func (c *chartWidget) SetSeries(s []stats.EquitySeries) {
	c.series = s
	c.Refresh()
}

var _ desktop.Hoverable = (*chartWidget)(nil)

func (c *chartWidget) MouseIn(e *desktop.MouseEvent) { c.MouseMoved(e) }

func (c *chartWidget) MouseMoved(e *desktop.MouseEvent) {
	c.hover = true
	c.curY = e.Position.Y
	if c.r != nil {
		c.r.updateCrosshair()
	}
}

func (c *chartWidget) MouseOut() {
	c.hover = false
	if c.r != nil {
		c.r.updateCrosshair()
	}
}

func (c *chartWidget) CreateRenderer() fyne.WidgetRenderer {
	r := &chartRenderer{c: c}
	r.bg = canvas.NewRectangle(colBGBase)
	r.bg.StrokeColor = colBorder
	r.bg.StrokeWidth = 1
	r.empty = canvas.NewText("No data", colTextMuted)
	r.empty.TextSize = 12
	r.empty.Alignment = fyne.TextAlignCenter
	r.zeroLine = canvas.NewLine(color.Transparent)
	r.crossLine = canvas.NewLine(crossColor)
	r.crossLine.StrokeWidth = 1
	r.crossLine.Hidden = true
	r.crossBg = canvas.NewRectangle(colBGInput)
	r.crossBg.Hidden = true
	r.crossLabel = canvas.NewText("", colTextPrimary)
	r.crossLabel.TextSize = chartLabelSize
	r.crossLabel.Hidden = true
	r.c.r = r
	r.prepare()
	return r
}

// prepSeries is a downsampled curve ready to be mapped to pixels.
type prepSeries struct {
	xs, ys   []float64 // unix-nano (as float64) and cumulative P/L
	lineFrom int       // index of this series' first segment in r.lines
	segs     int       // number of line segments
}

type chartRenderer struct {
	c    *chartWidget
	size fyne.Size

	bg         *canvas.Rectangle
	empty      *canvas.Text
	zeroLine   *canvas.Line
	crossLine  *canvas.Line
	crossBg    *canvas.Rectangle
	crossLabel *canvas.Text

	gridLines  []*canvas.Line
	yLabels    []*canvas.Text
	xLabels    []*canvas.Text
	lines      []*canvas.Line
	legendDots []*canvas.Rectangle
	legendTxts []*canvas.Text

	objects []fyne.CanvasObject

	// Prepared, size-independent state (rebuilt only on data change):
	has                      bool
	tMin, tSpan, yMin, ySpan float64
	zero                     bool
	prep                     []prepSeries
	tickVals                 []float64
	yLblW                    []float32
	xRightW                  float32

	// Plot box in pixels, cached by position() for the hover crosshair.
	px0, py0, px1, py1 float32

	// Resize step state (all touched on the UI thread only). The curve is only
	// re-laid-out when the size changes by at least one table row height.
	laidW, laidH float32
	timer        *time.Timer
}

func (r *chartRenderer) MinSize() fyne.Size { return fyne.NewSize(160, 100) }

// Layout runs on every resize. The expensive curve repositioning happens in
// pixel steps of one table row height, so dragging the splitter steps row by row
// and the CPU is released in between. The background is resized every frame so
// the area never shows a gap, and a trailing timer settles exactly at the final
// size when the drag stops mid-step.
func (r *chartRenderer) Layout(size fyne.Size) {
	r.size = size
	r.bg.Resize(size)

	step := tableRowHeight()
	if step < 1 {
		step = 1
	}
	if absf(size.Width-r.laidW) >= step || absf(size.Height-r.laidH) >= step {
		if r.timer != nil {
			r.timer.Stop()
			r.timer = nil
		}
		r.laidW, r.laidH = size.Width, size.Height
		r.position()
		return
	}

	if r.timer != nil {
		r.timer.Stop()
	}
	r.timer = time.AfterFunc(chartSettleDelay, func() {
		fyne.Do(func() {
			r.laidW, r.laidH = r.size.Width, r.size.Height
			r.position()
			canvas.Refresh(r.c)
		})
	})
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// Refresh runs only on data change: re-prepare, then position.
func (r *chartRenderer) Refresh() {
	if r.size.IsZero() {
		r.size = r.c.Size()
	}
	r.prepare()
	r.position()
	canvas.Refresh(r.c)
}

func (r *chartRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *chartRenderer) Destroy() {}

// prepare does all the data-dependent work and assembles the draw list. It runs
// only when the data changes, never on resize.
func (r *chartRenderer) prepare() {
	tMin, tMax, yMin, yMax, ok := dataBounds(r.c.series)
	r.has = ok
	if !ok {
		r.objects = []fyne.CanvasObject{r.bg, r.empty}
		return
	}

	span := yMax - yMin
	if span == 0 {
		yMin, span = yMin-1, 2
	} else {
		yMin -= span * 0.05
		yMax += span * 0.05
		span = yMax - yMin
	}
	r.tMin, r.tSpan = tMin, tMax-tMin
	r.yMin, r.ySpan = yMin, span
	r.zero = yMin < 0 && (yMin+span) > 0

	objs := []fyne.CanvasObject{r.bg}

	// Gridlines + Y labels (text + width cached so resize never re-measures).
	nGrid := chartYTicks + 1
	r.gridLines = ensureLines(r.gridLines, nGrid)
	r.yLabels = ensureTexts(r.yLabels, nGrid)
	r.tickVals = r.tickVals[:0]
	if cap(r.yLblW) < nGrid {
		r.yLblW = make([]float32, nGrid)
	}
	r.yLblW = r.yLblW[:nGrid]
	for i := 0; i < nGrid; i++ {
		v := yMin + span*float64(i)/float64(chartYTicks)
		r.tickVals = append(r.tickVals, v)
		ln := r.gridLines[i]
		ln.StrokeColor = colBorder
		ln.StrokeWidth = 0.6
		objs = append(objs, ln)

		lbl := r.yLabels[i]
		lbl.Text = fmtAxis(v)
		lbl.Color = colTextSecond
		lbl.TextSize = chartLabelSize
		r.yLblW[i] = fyne.MeasureText(lbl.Text, chartLabelSize, fyne.TextStyle{}).Width
		objs = append(objs, lbl)
	}
	if r.zero {
		r.zeroLine.StrokeColor = colTextMuted
		r.zeroLine.StrokeWidth = 1
		r.zeroLine.Hidden = false
		objs = append(objs, r.zeroLine)
	} else {
		r.zeroLine.Hidden = true
	}

	// X labels (first / last timestamp).
	r.xLabels = ensureTexts(r.xLabels, 2)
	layout := xTimeLayout(time.Duration(int64(r.tSpan)))
	left := time.Unix(0, int64(tMin)).Local().Format(layout)
	right := time.Unix(0, int64(tMax)).Local().Format(layout)
	for i, s := range []string{left, right} {
		lbl := r.xLabels[i]
		lbl.Text = s
		lbl.Color = colTextSecond
		lbl.TextSize = chartLabelSize
		objs = append(objs, lbl)
	}
	r.xRightW = fyne.MeasureText(right, chartLabelSize, fyne.TextStyle{}).Width

	// Series polylines (downsampled once; pooled line objects coloured here).
	r.prep = r.prep[:0]
	seg := 0
	for si, s := range r.c.series {
		if len(s.Times) == 0 {
			continue
		}
		col := chartPalette[si%len(chartPalette)]
		xs, ys := sampleSeries(s.Times, s.Cum, chartMaxSeg)
		ps := prepSeries{xs: xs, ys: ys, lineFrom: seg, segs: len(xs) - 1}
		for k := 0; k < ps.segs; k++ {
			r.lines = ensureLines(r.lines, seg+1)
			ln := r.lines[seg]
			ln.StrokeColor = col
			ln.StrokeWidth = 1.6
			objs = append(objs, ln)
			seg++
		}
		r.prep = append(r.prep, ps)
	}

	// Legend (only when more than one account). Positions are size-independent,
	// so they're set here.
	if len(r.prep) > 1 {
		drawn := len(r.prep)
		r.legendDots = ensureRects(r.legendDots, drawn)
		r.legendTxts = ensureTexts(r.legendTxts, drawn)
		lx, ly := float32(chartPadLeft+8), float32(chartPadTop+4)
		j := 0
		for si, s := range r.c.series {
			if len(s.Times) == 0 {
				continue
			}
			col := chartPalette[si%len(chartPalette)]
			dot := r.legendDots[j]
			dot.FillColor = col
			dot.Resize(fyne.NewSize(9, 9))
			dot.Move(fyne.NewPos(lx, ly+2))
			objs = append(objs, dot)

			lbl := r.legendTxts[j]
			lbl.Text = s.Account
			lbl.Color = colTextSecond
			lbl.TextSize = chartLabelSize
			lbl.Move(fyne.NewPos(lx+13, ly))
			objs = append(objs, lbl)

			lx += 13 + fyne.MeasureText(s.Account, chartLabelSize, fyne.TextStyle{}).Width + 14
			j++
		}
	}

	// Hover crosshair on top (toggled visible by the mouse handlers).
	objs = append(objs, r.crossLine, r.crossBg, r.crossLabel)

	r.objects = objs
}

// position maps the prepared data to the current size. Hot path on resize:
// arithmetic + field assignments only, no allocation and no text measuring.
func (r *chartRenderer) position() {
	w, h := r.size.Width, r.size.Height
	r.bg.Resize(fyne.NewSize(w, h))
	r.bg.Move(fyne.NewPos(0, 0))

	if !r.has {
		r.empty.Move(fyne.NewPos(0, h/2-8))
		r.empty.Resize(fyne.NewSize(w, 16))
		return
	}

	x0, y0 := float32(chartPadLeft), float32(chartPadTop)
	x1, y1 := w-chartPadRight, h-chartPadBottom
	plotW, plotH := x1-x0, y1-y0
	r.px0, r.py0, r.px1, r.py1 = x0, y0, x1, y1
	if plotW < 1 || plotH < 1 {
		r.placeCrosshair()
		return
	}

	mapX := func(t float64) float32 {
		if r.tSpan == 0 {
			return x0 + plotW/2
		}
		return x0 + float32((t-r.tMin)/r.tSpan)*plotW
	}
	mapY := func(v float64) float32 {
		return y1 - float32((v-r.yMin)/r.ySpan)*plotH
	}

	for i, v := range r.tickVals {
		y := mapY(v)
		ln := r.gridLines[i]
		ln.Position1 = fyne.NewPos(x0, y)
		ln.Position2 = fyne.NewPos(x1, y)
		r.yLabels[i].Move(fyne.NewPos(x0-6-r.yLblW[i], y-float32(chartLabelSize)/2-2))
	}
	if r.zero {
		y := mapY(0)
		r.zeroLine.Position1 = fyne.NewPos(x0, y)
		r.zeroLine.Position2 = fyne.NewPos(x1, y)
	}

	r.xLabels[0].Move(fyne.NewPos(x0, y1+3))
	r.xLabels[1].Move(fyne.NewPos(x1-r.xRightW, y1+3))

	for _, ps := range r.prep {
		for k := 1; k < len(ps.xs); k++ {
			ln := r.lines[ps.lineFrom+k-1]
			ln.Position1 = fyne.NewPos(mapX(ps.xs[k-1]), mapY(ps.ys[k-1]))
			ln.Position2 = fyne.NewPos(mapX(ps.xs[k]), mapY(ps.ys[k]))
		}
	}

	r.placeCrosshair()
}

// updateCrosshair repositions the hover line/label and repaints. Called from the
// mouse handlers; cheap because the curve textures are already cached.
func (r *chartRenderer) updateCrosshair() {
	r.placeCrosshair()
	canvas.Refresh(r.c)
}

// placeCrosshair positions the horizontal hover line and its P/L label at the
// cursor height, or hides them when not hovering.
func (r *chartRenderer) placeCrosshair() {
	show := r.has && r.c.hover && r.py1 > r.py0
	r.crossLine.Hidden = !show
	r.crossBg.Hidden = !show
	r.crossLabel.Hidden = !show
	if !show {
		return
	}

	y := r.c.curY
	if y < r.py0 {
		y = r.py0
	}
	if y > r.py1 {
		y = r.py1
	}
	v := r.yMin + float64((r.py1-y)/(r.py1-r.py0))*r.ySpan

	r.crossLine.Position1 = fyne.NewPos(r.px0, y)
	r.crossLine.Position2 = fyne.NewPos(r.px1, y)

	r.crossLabel.Text = fmt.Sprintf("%.2f", v)
	tw := fyne.MeasureText(r.crossLabel.Text, chartLabelSize, fyne.TextStyle{}).Width
	th := float32(chartLabelSize) + 4
	bx := r.px0 + 2
	by := y - th/2
	if by < r.py0 {
		by = r.py0
	}
	if by+th > r.py1 {
		by = r.py1 - th
	}
	r.crossBg.Move(fyne.NewPos(bx, by))
	r.crossBg.Resize(fyne.NewSize(tw+6, th))
	r.crossLabel.Move(fyne.NewPos(bx+3, by+2))
}

// dataBounds returns the global min/max time (unix-nano as float64) and Y value
// across all non-empty series.
func dataBounds(series []stats.EquitySeries) (tMin, tMax, yMin, yMax float64, ok bool) {
	tMin, yMin = math.Inf(1), math.Inf(1)
	tMax, yMax = math.Inf(-1), math.Inf(-1)
	for _, s := range series {
		for i := range s.Times {
			t := float64(s.Times[i].UnixNano())
			v := s.Cum[i]
			if t < tMin {
				tMin = t
			}
			if t > tMax {
				tMax = t
			}
			if v < yMin {
				yMin = v
			}
			if v > yMax {
				yMax = v
			}
			ok = true
		}
	}
	return
}

// sampleSeries downsamples a curve to at most max points, always keeping the
// first and last sample.
func sampleSeries(times []time.Time, cum []float64, max int) (xs, ys []float64) {
	n := len(times)
	if max < 2 {
		max = 2
	}
	if n <= max {
		xs = make([]float64, n)
		ys = make([]float64, n)
		for i := range times {
			xs[i] = float64(times[i].UnixNano())
			ys[i] = cum[i]
		}
		return
	}
	xs = make([]float64, max)
	ys = make([]float64, max)
	stride := float64(n-1) / float64(max-1)
	for i := 0; i < max; i++ {
		idx := int(math.Round(float64(i) * stride))
		if idx > n-1 {
			idx = n - 1
		}
		xs[i] = float64(times[idx].UnixNano())
		ys[i] = cum[idx]
	}
	return
}

func fmtAxis(v float64) string {
	if math.Abs(v) >= 1000 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

func xTimeLayout(span time.Duration) string {
	switch {
	case span > 48*time.Hour:
		return "Jan 02"
	case span > 2*time.Hour:
		return "01-02 15:04"
	default:
		return "15:04"
	}
}

// --- pooled-object helpers (avoid per-frame allocation while dragging) ---

func ensureLines(pool []*canvas.Line, n int) []*canvas.Line {
	for len(pool) < n {
		pool = append(pool, canvas.NewLine(color.Transparent))
	}
	return pool
}

func ensureTexts(pool []*canvas.Text, n int) []*canvas.Text {
	for len(pool) < n {
		t := canvas.NewText("", colTextSecond)
		t.TextSize = chartLabelSize
		pool = append(pool, t)
	}
	return pool
}

func ensureRects(pool []*canvas.Rectangle, n int) []*canvas.Rectangle {
	for len(pool) < n {
		pool = append(pool, canvas.NewRectangle(color.Transparent))
	}
	return pool
}
