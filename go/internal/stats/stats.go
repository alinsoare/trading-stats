// Package stats rebuilds closed positions from deal rows and computes
// KPIs / rollups. Ported from src/trading_stats/kpis.py.
package stats

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/alinsoare/trading-stats/go/internal/model"
)

// Bucket is a time rollup granularity.
type Bucket string

const (
	BucketDay   Bucket = "day"
	BucketWeek  Bucket = "week"
	BucketMonth Bucket = "month"
	BucketYear  Bucket = "year"
)

// KPIColumns is the ordered list of KPI field names used in grouped tables.
var KPIColumns = []string{
	"trades", "wins", "losses", "breakeven", "win_rate",
	"net_pnl", "gross_win", "gross_loss", "profit_factor",
	"expectancy", "avg_win", "avg_loss", "payoff",
	"max_dd_cum_pnl", "best_trade", "worst_trade",
}

// ClosedPositions rebuilds one row per closed round-trip (or single-ticket
// fallback when position_id == 0). P/L = sum(profit + swap + commission).
func ClosedPositions(deals []model.Deal) []model.ClosedPosition {
	if len(deals) == 0 {
		return nil
	}

	type agg struct {
		pos          model.ClosedPosition
		bestExit     time.Time
		haveBestExit bool
	}
	groups := map[string]*agg{}
	var order []string

	for _, d := range deals {
		if d.IsNonTrade {
			continue
		}
		netRow := d.Profit + d.Swap + d.Commission
		var posKey string
		if d.PositionID > 0 {
			posKey = strconv.FormatInt(d.PositionID, 10)
		} else {
			posKey = "u" + strconv.FormatInt(d.Ticket, 10)
		}
		key := d.AccountLabel + "\x00" + posKey

		a, ok := groups[key]
		if !ok {
			a = &agg{pos: model.ClosedPosition{
				AccountLabel: d.AccountLabel,
				PosKey:       posKey,
			}}
			groups[key] = a
			order = append(order, key)
		}
		a.pos.NetPnL += netRow
		a.pos.NLegs++

		// Track the row with the latest exit time for display fields and exit_time.
		if d.HasTime && (!a.haveBestExit || !d.Time.Before(a.bestExit)) {
			a.bestExit = d.Time
			a.haveBestExit = true
			a.pos.ExitTime = d.Time
			a.pos.HasExit = true
			a.pos.Symbol = d.Symbol
			a.pos.Magic = d.Magic
			a.pos.CommentSample = d.Comment
		}
		// When no exit time has been seen yet, still keep display fields current.
		if !a.haveBestExit {
			a.pos.Symbol = d.Symbol
			a.pos.Magic = d.Magic
			a.pos.CommentSample = d.Comment
		}
	}

	out := make([]model.ClosedPosition, 0, len(order))
	for _, k := range order {
		out = append(out, groups[k].pos)
	}
	sortByAccountThenExit(out)
	return out
}

func sortByAccountThenExit(pos []model.ClosedPosition) {
	sort.SliceStable(pos, func(i, j int) bool {
		if pos[i].AccountLabel != pos[j].AccountLabel {
			return pos[i].AccountLabel < pos[j].AccountLabel
		}
		return pos[i].ExitTime.Before(pos[j].ExitTime)
	})
}

// ThresholdFunc returns the per-position breakeven tolerance (±$).
type ThresholdFunc func(model.ClosedPosition) float64

// ZeroThreshold is a ThresholdFunc that always returns 0.
func ZeroThreshold(model.ClosedPosition) float64 { return 0 }

// SummarizePositions aggregates KPIs for a set of positions. Trades within
// ±threshold are counted as breakeven and excluded from wins and losses.
func SummarizePositions(pos []model.ClosedPosition, label string, beThr ThresholdFunc) model.KPI {
	if len(pos) == 0 {
		if label == "" {
			label = "—"
		}
		return model.KPI{Label: label}
	}
	if label == "" {
		label = "all"
	}
	if beThr == nil {
		beThr = ZeroThreshold
	}

	var nWin, nLoss int
	var grossWin, grossLoss float64
	var sumPnL float64
	best := math.Inf(-1)
	worst := math.Inf(1)

	for _, p := range pos {
		thr := beThr(p)
		r := round2(p.NetPnL)
		sumPnL += p.NetPnL
		if p.NetPnL > best {
			best = p.NetPnL
		}
		if p.NetPnL < worst {
			worst = p.NetPnL
		}
		if r > thr {
			nWin++
			grossWin += p.NetPnL
		} else if r < -thr {
			nLoss++
			grossLoss += p.NetPnL
		}
	}

	n := len(pos)
	be := n - nWin - nLoss

	denom := nWin + nLoss
	winRate := 0.0
	if denom > 0 {
		winRate = float64(nWin) / float64(denom)
	}

	var profitFactor float64
	switch {
	case grossLoss < 0:
		profitFactor = grossWin / math.Abs(grossLoss)
	case grossWin > 0:
		profitFactor = math.Inf(1)
	default:
		profitFactor = 0
	}

	avgWin := 0.0
	if nWin > 0 {
		avgWin = grossWin / float64(nWin)
	}
	avgLoss := 0.0
	if nLoss > 0 {
		avgLoss = grossLoss / float64(nLoss)
	}
	payoff := 0.0
	if avgLoss != 0 {
		payoff = avgWin / math.Abs(avgLoss)
	}

	expectancy := sumPnL / float64(n)
	maxDD := maxDrawdown(pos)

	return model.KPI{
		Label:        label,
		Trades:       n,
		Wins:         nWin,
		Losses:       nLoss,
		Breakeven:    be,
		WinRate:      r2(winRate),
		NetPnL:       r2(sumPnL),
		GrossWin:     r2(grossWin),
		GrossLoss:    r2(grossLoss),
		ProfitFactor: r2(profitFactor),
		Expectancy:   r2(expectancy),
		AvgWin:       r2(avgWin),
		AvgLoss:      r2(avgLoss),
		Payoff:       r2(payoff),
		MaxDDCumPnL:  r2(maxDD),
		BestTrade:    r2(best),
		WorstTrade:   r2(worst),
	}
}

// maxDrawdown computes min(cum - running max) over positions ordered by exit time.
func maxDrawdown(pos []model.ClosedPosition) float64 {
	if len(pos) == 0 {
		return 0
	}
	ordered := make([]model.ClosedPosition, len(pos))
	copy(ordered, pos)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].ExitTime.Before(ordered[j].ExitTime)
	})
	var cum, peak, minDD float64
	for i, p := range ordered {
		cum += p.NetPnL
		if i == 0 || cum > peak {
			peak = cum
		}
		dd := cum - peak
		if dd < minDD {
			minDD = dd
		}
	}
	return minDD
}

// GroupRow is one grouped KPI result: group key values plus aggregated KPIs.
type GroupRow struct {
	Keys []string
	KPI  model.KPI
}

// KeyFunc extracts the group-key values from a position.
type KeyFunc func(model.ClosedPosition) []string

// SummarizeGroups runs SummarizePositions for every unique key combination,
// sorted by the group keys.
func SummarizeGroups(pos []model.ClosedPosition, keyFn KeyFunc, beThr ThresholdFunc) []GroupRow {
	if len(pos) == 0 {
		return nil
	}
	groups := map[string][]model.ClosedPosition{}
	keyVals := map[string][]string{}
	var order []string
	for _, p := range pos {
		k := keyFn(p)
		jk := joinKey(k)
		if _, ok := groups[jk]; !ok {
			keyVals[jk] = k
			order = append(order, jk)
		}
		groups[jk] = append(groups[jk], p)
	}
	rows := make([]GroupRow, 0, len(order))
	for _, jk := range order {
		rows = append(rows, GroupRow{
			Keys: keyVals[jk],
			KPI:  SummarizePositions(groups[jk], "", beThr),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return lessKeys(rows[i].Keys, rows[j].Keys)
	})
	return rows
}

// PeriodOf returns the bucket label for a time, matching kpis._bucket_expr.
func PeriodOf(t time.Time, bucket Bucket) string {
	switch bucket {
	case BucketDay:
		return t.Format("2006-01-02")
	case BucketWeek:
		y, w := t.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", y, w)
	case BucketMonth:
		return t.Format("2006-01")
	case BucketYear:
		return t.Format("2006")
	default:
		return t.Format("2006-01")
	}
}

// EquitySeries is a per-account cumulative P/L curve.
type EquitySeries struct {
	Account string
	Times   []time.Time
	Cum     []float64
}

// EquityCurves builds per-account cumulative net P/L, ordered by exit time.
func EquityCurves(pos []model.ClosedPosition) []EquitySeries {
	if len(pos) == 0 {
		return nil
	}
	byAcct := map[string][]model.ClosedPosition{}
	var accounts []string
	for _, p := range pos {
		if _, ok := byAcct[p.AccountLabel]; !ok {
			accounts = append(accounts, p.AccountLabel)
		}
		byAcct[p.AccountLabel] = append(byAcct[p.AccountLabel], p)
	}
	sort.Strings(accounts)

	out := make([]EquitySeries, 0, len(accounts))
	for _, acc := range accounts {
		rows := byAcct[acc]
		sort.SliceStable(rows, func(i, j int) bool {
			return rows[i].ExitTime.Before(rows[j].ExitTime)
		})
		s := EquitySeries{Account: acc}
		var cum float64
		for _, p := range rows {
			cum += p.NetPnL
			s.Times = append(s.Times, p.ExitTime)
			s.Cum = append(s.Cum, cum)
		}
		out = append(out, s)
	}
	return out
}

// KPIValues returns the KPI fields as display strings in KPIColumns order.
func KPIValues(k model.KPI) []string {
	return []string{
		strconv.Itoa(k.Trades),
		strconv.Itoa(k.Wins),
		strconv.Itoa(k.Losses),
		strconv.Itoa(k.Breakeven),
		fmtFloat(k.WinRate),
		fmtFloat(k.NetPnL),
		fmtFloat(k.GrossWin),
		fmtFloat(k.GrossLoss),
		fmtFloat(k.ProfitFactor),
		fmtFloat(k.Expectancy),
		fmtFloat(k.AvgWin),
		fmtFloat(k.AvgLoss),
		fmtFloat(k.Payoff),
		fmtFloat(k.MaxDDCumPnL),
		fmtFloat(k.BestTrade),
		fmtFloat(k.WorstTrade),
	}
}

func fmtFloat(v float64) string {
	if math.IsInf(v, 1) {
		return "∞"
	}
	if math.IsInf(v, -1) {
		return "-∞"
	}
	if math.IsNaN(v) {
		return "—"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func joinKey(k []string) string {
	s := ""
	for i, v := range k {
		if i > 0 {
			s += "\x00"
		}
		s += v
	}
	return s
}

func lessKeys(a, b []string) bool {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

// round2 rounds to 2 decimals for win/loss classification.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// r2 rounds to 2 decimals but preserves inf/nan, matching kpis._r2.
func r2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return v
	}
	return math.Round(v*100) / 100
}
