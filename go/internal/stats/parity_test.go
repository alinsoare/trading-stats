package stats_test

import (
	"reflect"
	"testing"

	"github.com/alinsoare/trading-stats/go/internal/ingest"
	"github.com/alinsoare/trading-stats/go/internal/model"
	"github.com/alinsoare/trading-stats/go/internal/stats"
)

// Expected values were verified to match the Python trading_stats reference
// (polars) exactly on testdata/parity.
var (
	wantOverall = []string{
		"6", "3", "3", "0", "0.5", "28.2", "71.8", "-43.6",
		"1.65", "4.7", "23.93", "-14.53", "1.65", "-31", "48.8", "-31",
	}
	wantBySymbol = map[string][]string{
		"EURUSD": {"2", "1", "1", "0", "0.5", "48.2", "48.8", "-0.6", "81.33", "24.1", "48.8", "-0.6", "81.33", "-0.6", "48.8", "-0.6"},
		"GBPUSD": {"1", "0", "1", "0", "0", "-31", "0", "-31", "0", "-31", "0", "-31", "0", "0", "-31", "-31"},
		"USDJPY": {"2", "1", "1", "0", "0.5", "6", "18", "-12", "1.5", "3", "18", "-12", "1.5", "-12", "18", "-12"},
		"XAUUSD": {"1", "1", "0", "0", "1", "5", "5", "0", "∞", "5", "5", "0", "0", "0", "5", "5"},
	}
	wantByAccount = map[string][]string{
		"1001_Demo-1": {"4", "2", "2", "0", "0.5", "22.2", "53.8", "-31.6", "1.7", "5.55", "26.9", "-15.8", "1.7", "-31.6", "48.8", "-31"},
		"2002_Live-2": {"2", "1", "1", "0", "0.5", "6", "18", "-12", "1.5", "3", "18", "-12", "1.5", "-12", "18", "-12"},
	}
)

func loadParityPos(t *testing.T) []model.ClosedPosition {
	t.Helper()
	deals, err := ingest.LoadDeals([]string{"../../testdata/parity"})
	if err != nil {
		t.Fatalf("LoadDeals: %v", err)
	}
	if len(deals) == 0 {
		t.Fatal("no deals loaded from testdata/parity")
	}
	return stats.ClosedPositions(deals)
}

func TestParityOverall(t *testing.T) {
	pos := loadParityPos(t)
	got := stats.KPIValues(stats.SummarizePositions(pos, "all", stats.ZeroThreshold))
	if !reflect.DeepEqual(got, wantOverall) {
		t.Errorf("overall KPIs mismatch\n got: %v\nwant: %v", got, wantOverall)
	}
}

func TestParityBySymbol(t *testing.T) {
	pos := loadParityPos(t)
	rows := stats.SummarizeGroups(pos, func(p model.ClosedPosition) []string {
		return []string{p.Symbol}
	}, stats.ZeroThreshold)
	checkGroups(t, "by_symbol", rows, wantBySymbol)
}

func TestParityByAccount(t *testing.T) {
	pos := loadParityPos(t)
	rows := stats.SummarizeGroups(pos, func(p model.ClosedPosition) []string {
		return []string{p.AccountLabel}
	}, stats.ZeroThreshold)
	checkGroups(t, "by_account", rows, wantByAccount)
}

func TestParityRollupMonthCounts(t *testing.T) {
	pos := loadParityPos(t)
	rows := stats.SummarizeGroups(pos, func(p model.ClosedPosition) []string {
		return []string{p.AccountLabel, stats.PeriodOf(p.ExitTime, stats.BucketMonth)}
	}, stats.ZeroThreshold)
	if len(rows) != 4 {
		t.Fatalf("rollup month: want 4 groups, got %d", len(rows))
	}
	// Groups must be sorted by (account_label, period).
	wantKeys := [][]string{
		{"1001_Demo-1", "2024-01"},
		{"1001_Demo-1", "2024-02"},
		{"2002_Live-2", "2024-01"},
		{"2002_Live-2", "2024-03"},
	}
	for i, w := range wantKeys {
		if !reflect.DeepEqual(rows[i].Keys, w) {
			t.Errorf("rollup group %d keys = %v, want %v", i, rows[i].Keys, w)
		}
	}
}

func checkGroups(t *testing.T, name string, rows []stats.GroupRow, want map[string][]string) {
	t.Helper()
	got := map[string][]string{}
	for _, r := range rows {
		got[r.Keys[0]] = stats.KPIValues(r.KPI)
	}
	if len(got) != len(want) {
		t.Errorf("%s: got %d groups, want %d", name, len(got), len(want))
	}
	for k, w := range want {
		if !reflect.DeepEqual(got[k], w) {
			t.Errorf("%s[%s] mismatch\n got: %v\nwant: %v", name, k, got[k], w)
		}
	}
}
