"""Load and normalize exported deal CSVs with Polars."""

from __future__ import annotations

from pathlib import Path

import polars as pl

from trading_stats.paths import iter_deal_csv_from_data_folders, parse_login_from_filename


def load_deals(
    data_folders: list[Path | str] | None = None,
) -> pl.DataFrame:
    """
    Load deal rows from explicit ``data_folders`` (each resolved to a ``deals_*.csv`` directory).

    Adds ``account_label`` (login_server or folder hint) and ``source_file``.
    One CSV file per account is expected (the MQL5 script always overwrites the same file).
    """
    if not data_folders:
        return pl.DataFrame()

    dirs = [Path(d).expanduser() for d in data_folders if str(d).strip()]
    rows = iter_deal_csv_from_data_folders(dirs)
    if not rows:
        return pl.DataFrame()

    frames: list[pl.DataFrame] = []
    for path, hint in rows:
        df = pl.read_csv(
            path,
            try_parse_dates=True,
            infer_schema_length=5000,
            encoding="utf8-lossy",
        )
        if "login" in df.columns:
            mx = df.select(pl.col("login").cast(pl.Int64, strict=False).max()).to_series()
            login_v = (
                int(mx[0])
                if mx.len() and mx[0] is not None and str(mx[0]) != "nan"
                else (parse_login_from_filename(path) or 0)
            )
            srv = ""
            if "server" in df.columns:
                s = df.select(pl.col("server").drop_nulls().head(1)).to_series()
                if s.len():
                    srv = str(s[0])
            label = f"{login_v}_{srv}" if srv else str(login_v)
        else:
            login = parse_login_from_filename(path)
            label = str(login) if login is not None else str(hint)

        df = df.with_columns(
            pl.lit(label).alias("account_label"),
            pl.lit(str(path)).alias("source_file"),
        )
        frames.append(df)

    if not frames:
        return pl.DataFrame()

    out = pl.concat(frames, how="diagonal_relaxed")
    num_cols = ["volume", "price", "commission", "swap", "profit"]
    for c in num_cols:
        if c in out.columns:
            out = out.with_columns(pl.col(c).cast(pl.Float64, strict=False))
    if "position_id" in out.columns:
        out = out.with_columns(pl.col("position_id").cast(pl.Int64, strict=False))
    if "ticket" in out.columns:
        out = out.with_columns(pl.col("ticket").cast(pl.Int64, strict=False))
    if "time_msc" in out.columns:
        out = out.with_columns(pl.col("time_msc").cast(pl.Int64, strict=False))
    if "is_non_trade" in out.columns:
        out = out.with_columns(pl.col("is_non_trade").cast(pl.Int8, strict=False))
    return out


def account_flows(deals: pl.DataFrame) -> pl.DataFrame:
    """Non-trade rows (deposits, credits, etc.) for optional UI section."""
    if deals.is_empty() or "is_non_trade" not in deals.columns:
        return deals.head(0)
    return deals.filter(pl.col("is_non_trade") == 1)
