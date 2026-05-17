"""Resolve user-supplied folders to MT5 deal CSV export locations."""

from __future__ import annotations

import re
from pathlib import Path


def resolve_export_csv_dir(user_path: Path) -> Path | None:
    """
    Return the directory that should contain ``deals_*.csv``, or None.

    Accepts:
    - The export folder itself (contains ``deals_*.csv``, usually ``.../MQL5/Files/trading_stats``).
    - A portable MT5 terminal root: uses ``<root>/MQL5/Files/trading_stats`` when that directory exists.
    """
    p = user_path.expanduser().resolve()
    if not p.is_dir():
        return None
    if any(p.glob("deals_*.csv")):
        return p
    nested = p / "MQL5" / "Files" / "trading_stats"
    if nested.is_dir():
        return nested
    return None


def iter_deal_csv_from_data_folders(folders: list[Path]) -> list[tuple[Path, str]]:
    """
    Collect ``(csv_path, account_hint)`` from one or more user-configured folders.

    ``account_hint`` disambiguates rows when CSV has no login column (basename of the
    folder the user entered).
    """
    out: list[tuple[Path, str]] = []
    seen: set[Path] = set()
    for raw in folders:
        s = str(raw).strip()
        if not s:
            continue
        base = Path(s)
        stats = resolve_export_csv_dir(base)
        if stats is None:
            continue
        hint = base.resolve().name
        for csv in sorted(stats.glob("deals_*.csv")):
            if csv.is_file() and csv not in seen:
                seen.add(csv)
                out.append((csv, hint))
    return sorted(out, key=lambda x: str(x[0]))


def parse_login_from_filename(path: Path) -> int | None:
    """``deals_<LOGIN>_...csv`` → login or None."""
    m = re.match(r"deals_(\d+)_", path.name, re.I)
    if not m:
        return None
    try:
        return int(m.group(1))
    except ValueError:
        return None
