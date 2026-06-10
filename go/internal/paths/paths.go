// Package paths resolves user-supplied folders to MT5 deal CSV export locations.
// Ported from src/trading_stats/paths.py.
package paths

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var loginRe = regexp.MustCompile(`(?i)^deals_(\d+)_`)

// ResolveExportCSVDir returns the directory that should contain deals_*.csv, or "".
//
// Accepts:
//   - The export folder itself (contains deals_*.csv).
//   - A portable MT5 terminal root: uses <root>/MQL5/Files/trading_stats when it exists.
func ResolveExportCSVDir(userPath string) string {
	p := expand(userPath)
	abs, err := filepath.Abs(p)
	if err == nil {
		p = abs
	}
	info, err := os.Stat(p)
	if err != nil || !info.IsDir() {
		return ""
	}
	if matches, _ := filepath.Glob(filepath.Join(p, "deals_*.csv")); len(matches) > 0 {
		return p
	}
	nested := filepath.Join(p, "MQL5", "Files", "trading_stats")
	if info, err := os.Stat(nested); err == nil && info.IsDir() {
		return nested
	}
	return ""
}

// CSVSource pairs a CSV file path with the account hint derived from the
// folder the user entered (used when the CSV lacks a login column).
type CSVSource struct {
	Path string
	Hint string
}

// IterDealCSVFromDataFolders collects (csv_path, account_hint) from one or more
// user-configured folders, de-duplicated and sorted by path.
func IterDealCSVFromDataFolders(folders []string) []CSVSource {
	var out []CSVSource
	seen := map[string]bool{}
	for _, raw := range folders {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		stats := ResolveExportCSVDir(s)
		if stats == "" {
			continue
		}
		abs, err := filepath.Abs(expand(s))
		hint := s
		if err == nil {
			hint = filepath.Base(abs)
		}
		matches, _ := filepath.Glob(filepath.Join(stats, "deals_*.csv"))
		sort.Strings(matches)
		for _, csv := range matches {
			info, err := os.Stat(csv)
			if err != nil || info.IsDir() {
				continue
			}
			if seen[csv] {
				continue
			}
			seen[csv] = true
			out = append(out, CSVSource{Path: csv, Hint: hint})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// ParseLoginFromFilename extracts the login from "deals_<LOGIN>_...csv".
// Returns (login, true) on success.
func ParseLoginFromFilename(path string) (int64, bool) {
	m := loginRe.FindStringSubmatch(filepath.Base(path))
	if m == nil {
		return 0, false
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func expand(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
