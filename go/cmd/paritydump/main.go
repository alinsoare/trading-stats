// Command paritydump loads a deals folder and prints computed KPIs as JSON,
// used to verify parity with the Python reference implementation.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alinsoare/trading-stats/go/internal/ingest"
	"github.com/alinsoare/trading-stats/go/internal/model"
	"github.com/alinsoare/trading-stats/go/internal/stats"
)

type groupOut struct {
	Keys []string `json:"keys"`
	Vals []string `json:"vals"`
}

func groups(rows []stats.GroupRow) []groupOut {
	out := make([]groupOut, 0, len(rows))
	for _, r := range rows {
		out = append(out, groupOut{Keys: r.Keys, Vals: stats.KPIValues(r.KPI)})
	}
	return out
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: paritydump <folder>")
		os.Exit(2)
	}
	deals, err := ingest.LoadDeals([]string{os.Args[1]})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	pos := stats.ClosedPositions(deals)
	thr := stats.ZeroThreshold

	acctKey := func(p model.ClosedPosition) []string { return []string{p.AccountLabel} }
	symKey := func(p model.ClosedPosition) []string { return []string{p.Symbol} }
	rollKey := func(p model.ClosedPosition) []string {
		return []string{p.AccountLabel, stats.PeriodOf(p.ExitTime, stats.BucketMonth)}
	}

	out := map[string]any{
		"overall":      stats.KPIValues(stats.SummarizePositions(pos, "all", thr)),
		"by_account":   groups(stats.SummarizeGroups(pos, acctKey, thr)),
		"by_symbol":    groups(stats.SummarizeGroups(pos, symKey, thr)),
		"rollup_month": groups(stats.SummarizeGroups(pos, rollKey, thr)),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
