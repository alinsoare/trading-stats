// Package ingest loads and normalizes exported deal CSVs.
// Ported from src/trading_stats/ingest.py.
package ingest

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alinsoare/trading-stats/go/internal/model"
	"github.com/alinsoare/trading-stats/go/internal/paths"
)

// timeLayouts mirror kpis._parse_time_col formats.
var timeLayouts = []string{
	"2006.01.02 15:04:05",
	"2006-01-02 15:04:05",
}

// LoadDeals loads deal rows from the given data folders (each resolved to a
// deals_*.csv directory). Adds AccountLabel and SourceFile to every row.
func LoadDeals(dataFolders []string) ([]model.Deal, error) {
	sources := paths.IterDealCSVFromDataFolders(dataFolders)
	if len(sources) == 0 {
		return nil, nil
	}

	var all []model.Deal
	for _, src := range sources {
		rows, err := loadFile(src.Path, src.Hint)
		if err != nil {
			return nil, err
		}
		all = append(all, rows...)
	}
	return all, nil
}

func loadFile(path, hint string) ([]model.Deal, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := sanitizeUTF8(string(raw))

	r := csv.NewReader(strings.NewReader(text))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, nil
	}

	idx := map[string]int{}
	for i, h := range records[0] {
		idx[strings.TrimSpace(strings.ToLower(h))] = i
	}
	get := func(row []string, name string) string {
		if i, ok := idx[name]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}

	deals := make([]model.Deal, 0, len(records)-1)
	var maxLogin int64
	var haveLogin bool
	var server string
	_, hasLoginCol := idx["login"]

	for _, row := range records[1:] {
		if len(row) == 0 {
			continue
		}
		d := model.Deal{
			Ticket:     parseInt(get(row, "ticket")),
			Type:       int(parseInt(get(row, "type"))),
			Entry:      int(parseInt(get(row, "entry"))),
			Symbol:     get(row, "symbol"),
			Volume:     parseFloat(get(row, "volume")),
			Price:      parseFloat(get(row, "price")),
			Commission: parseFloat(get(row, "commission")),
			Swap:       parseFloat(get(row, "swap")),
			Profit:     parseFloat(get(row, "profit")),
			Magic:      parseInt(get(row, "magic")),
			Comment:    get(row, "comment"),
			PositionID: parseInt(get(row, "position_id")),
			Reason:     int(parseInt(get(row, "reason"))),
			Login:      parseInt(get(row, "login")),
			Server:     get(row, "server"),
			IsNonTrade: parseInt(get(row, "is_non_trade")) == 1,
			SourceFile: path,
		}
		if t, ok := parseTime(get(row, "time")); ok {
			d.Time = t
			d.HasTime = true
		}
		if hasLoginCol {
			if v := d.Login; v != 0 {
				if !haveLogin || v > maxLogin {
					maxLogin = v
					haveLogin = true
				}
			}
		}
		if server == "" && d.Server != "" {
			server = d.Server
		}
		deals = append(deals, d)
	}

	label := accountLabel(hasLoginCol, haveLogin, maxLogin, server, path, hint)
	for i := range deals {
		deals[i].AccountLabel = label
	}
	return deals, nil
}

func accountLabel(hasLoginCol, haveLogin bool, maxLogin int64, server, path, hint string) string {
	if hasLoginCol {
		login := maxLogin
		if !haveLogin {
			if v, ok := paths.ParseLoginFromFilename(path); ok {
				login = v
			} else {
				login = 0
			}
		}
		if server != "" {
			return strconv.FormatInt(login, 10) + "_" + server
		}
		return strconv.FormatInt(login, 10)
	}
	if v, ok := paths.ParseLoginFromFilename(path); ok {
		return strconv.FormatInt(v, 10)
	}
	return hint
}

// AccountFlows returns non-trade rows (deposits, credits, etc.).
func AccountFlows(deals []model.Deal) []model.Deal {
	var out []model.Deal
	for _, d := range deals {
		if d.IsNonTrade {
			out = append(out, d)
		}
	}
	return out
}

func parseTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseInt(s string) int64 {
	if s == "" {
		return 0
	}
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v
	}
	// Tolerate values like "123.0".
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(f)
	}
	return 0
}

// sanitizeUTF8 replaces invalid UTF-8 bytes (ANSI export remnants) so the CSV
// reader does not choke, mirroring Polars' utf8-lossy read.
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "\uFFFD")
}
