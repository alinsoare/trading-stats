"""Rebuild closed positions from deal rows and compute KPIs / rollups."""

from __future__ import annotations

from typing import Literal

import polars as pl

Bucket = Literal["day", "week", "month", "year"]


def _parse_time_col(df: pl.DataFrame) -> pl.DataFrame:
    if df.is_empty() or "time" not in df.columns:
        return df
    t = pl.col("time")
    if df["time"].dtype == pl.Datetime:
        return df.with_columns(pl.col("time").alias("exit_time"))
    parsed = pl.coalesce(
        t.str.strptime(pl.Datetime, format="%Y.%m.%d %H:%M:%S", strict=False),
        t.str.strptime(pl.Datetime, format="%Y-%m-%d %H:%M:%S", strict=False),
        t.str.strptime(pl.Datetime, strict=False),
    )
    return df.with_columns(parsed.alias("exit_time"))


def closed_positions(deals: pl.DataFrame) -> pl.DataFrame:
    """
    One row per closed round-trip (or single-ticket fallback when position_id == 0).
    P/L = sum(profit + swap + commission) over grouped deals.
    """
    if deals.is_empty():
        return deals

    df = _parse_time_col(deals)

    if "is_non_trade" in df.columns:
        df = df.filter(pl.col("is_non_trade").fill_null(0) == 0)

    df = df.with_columns(
        (
            pl.col("profit").fill_null(0.0)
            + pl.col("swap").fill_null(0.0)
            + pl.col("commission").fill_null(0.0)
        ).alias("net_row"),
    )

    df = df.with_columns(
        pl.when(pl.col("position_id").fill_null(0) > 0)
        .then(pl.col("position_id").cast(pl.Utf8))
        .otherwise(pl.lit("u") + pl.col("ticket").cast(pl.Utf8))
        .alias("pos_key"),
    )

    df = df.sort(["account_label", "pos_key", "exit_time"])
    agg = df.group_by(["account_label", "pos_key"], maintain_order=True).agg(
        pl.sum("net_row").alias("net_pnl"),
        pl.col("exit_time").max().alias("exit_time"),
        pl.col("symbol").last().alias("symbol"),
        pl.col("magic").last().alias("magic"),
        pl.col("comment").last().alias("comment_sample"),
        pl.len().alias("n_legs"),
    )
    return agg.sort(["account_label", "exit_time"])


def _bucket_expr(bucket: Bucket) -> pl.Expr:
    t = pl.col("exit_time")
    if bucket == "day":
        return t.dt.date().alias("period")
    if bucket == "week":
        return t.dt.strftime("%G-W%V").alias("period")
    if bucket == "month":
        return t.dt.strftime("%Y-%m").alias("period")
    if bucket == "year":
        return t.dt.strftime("%Y").alias("period")
    raise ValueError(bucket)


_KPI_COLS = [
    "trades", "wins", "losses", "breakeven", "win_rate",
    "net_pnl", "gross_win", "gross_loss", "profit_factor",
    "expectancy", "avg_win", "avg_loss", "payoff",
    "max_dd_cum_pnl", "best_trade", "worst_trade",
]


def add_period_col(pos: pl.DataFrame, bucket: Bucket) -> pl.DataFrame:
    """Add a ``period`` string column for the chosen time bucket."""
    if pos.is_empty():
        return pos
    return pos.with_columns(_bucket_expr(bucket))


def summarize_groups(pos: pl.DataFrame, group_by: list[str]) -> pl.DataFrame:
    """
    Run summarize_positions for every unique combination of group_by columns.
    Returns group columns followed by all KPI columns, sorted by the group keys.
    """
    if pos.is_empty():
        return pl.DataFrame()
    rows: list[dict] = []
    for keys, sub in pos.group_by(group_by, maintain_order=False):
        if not isinstance(keys, tuple):
            keys = (keys,)
        agg = summarize_positions(sub)
        row: dict = {col: val for col, val in zip(group_by, keys)}
        row.update({k: agg[k] for k in _KPI_COLS})
        rows.append(row)
    if not rows:
        return pl.DataFrame()
    return pl.DataFrame(rows).select(group_by + _KPI_COLS).sort(group_by)


def rollup_positions(pos: pl.DataFrame, bucket: Bucket, *, be_threshold: float = 0.0) -> pl.DataFrame:
    if pos.is_empty():
        return pos
    b = _bucket_expr(bucket)
    thr = be_threshold
    return (
        pos.with_columns(b)
        .group_by(["account_label", "period"])
        .agg(
            pl.len().alias("trades"),
            pl.col("net_pnl").sum().round(2).alias("net_pnl"),
            (pl.col("net_pnl").round(2) > thr).sum().alias("wins"),
            (pl.col("net_pnl").round(2) < -thr).sum().alias("losses"),
            ((pl.col("net_pnl").round(2) >= -thr) & (pl.col("net_pnl").round(2) <= thr)).sum().alias("breakeven"),
        )
        .sort(["account_label", "period"])
    )


def _max_drawdown(cum: pl.Series) -> float:
    if cum.len() == 0:
        return 0.0
    peak = cum.cum_max()
    dd = cum - peak
    return float(dd.min())


def _r2(v: float) -> float:
    """Round to 2 decimal places; preserve inf/nan as-is."""
    if v != v or v == float("inf") or v == float("-inf"):
        return v
    return round(v, 2)


def summarize_positions(
    pos: pl.DataFrame,
    *,
    be_threshold: float = 0.0,
    label: str | None = None,
) -> dict:
    """Aggregate KPIs for a position table (single account or all).

    If the DataFrame contains a ``_be_thr`` column (per-row tolerance injected
    by the caller), it is used for win/loss/breakeven classification.  Otherwise
    ``be_threshold`` (a single scalar) is applied uniformly.  Breakeven trades
    are excluded from both winners and losers before any further KPI maths.
    """
    if pos.is_empty() or len(pos) == 0:
        return {
            "label": label or "—",
            "trades": 0,
            "wins": 0,
            "losses": 0,
            "breakeven": 0,
            "win_rate": 0.0,
            "net_pnl": 0.0,
            "gross_win": 0.0,
            "gross_loss": 0.0,
            "profit_factor": 0.0,
            "expectancy": 0.0,
            "avg_win": 0.0,
            "avg_loss": 0.0,
            "payoff": 0.0,
            "max_dd_cum_pnl": 0.0,
            "best_trade": 0.0,
            "worst_trade": 0.0,
        }

    pnl = pos["net_pnl"]

    if "_be_thr" in pos.columns:
        win_expr  = pl.col("net_pnl").round(2) > pl.col("_be_thr")
        loss_expr = pl.col("net_pnl").round(2) < pl.col("_be_thr") * -1
    else:
        win_expr  = pl.col("net_pnl").round(2) > pl.lit(be_threshold)
        loss_expr = pl.col("net_pnl").round(2) < pl.lit(-be_threshold)

    n_win  = int(pos.select(win_expr.sum()).item())
    n_loss = int(pos.select(loss_expr.sum()).item())
    be     = len(pos) - n_win - n_loss

    wins   = pos.filter(win_expr)
    losses = pos.filter(loss_expr)

    n = len(pos)
    denom = n_win + n_loss
    win_rate = float(n_win / denom) if denom else 0.0

    gross_win = float(wins["net_pnl"].sum()) if n_win else 0.0
    gross_loss = float(losses["net_pnl"].sum()) if n_loss else 0.0

    if gross_loss < 0:
        profit_factor = gross_win / abs(gross_loss)
    elif gross_win > 0:
        profit_factor = float("inf")
    else:
        profit_factor = 0.0

    avg_win = float(wins["net_pnl"].mean()) if n_win else 0.0
    avg_loss = float(losses["net_pnl"].mean()) if n_loss else 0.0
    payoff = (avg_win / abs(avg_loss)) if avg_loss != 0 else 0.0

    ordered = pos.sort("exit_time")
    cum = ordered["net_pnl"].cum_sum()

    return {
        "label": label or "all",
        "trades": n,
        "wins": n_win,
        "losses": n_loss,
        "breakeven": be,
        "win_rate": _r2(win_rate),
        "net_pnl": _r2(float(pnl.sum())),
        "gross_win": _r2(gross_win),
        "gross_loss": _r2(gross_loss),
        "profit_factor": _r2(profit_factor),
        "expectancy": _r2(float(pnl.mean())),
        "avg_win": _r2(avg_win),
        "avg_loss": _r2(avg_loss),
        "payoff": _r2(payoff),
        "max_dd_cum_pnl": _r2(_max_drawdown(cum)),
        "best_trade": _r2(float(pnl.max())),
        "worst_trade": _r2(float(pnl.min())),
    }


def aggregate_equity_curve(pos: pl.DataFrame) -> pl.DataFrame:
    """Per-account cumulative P/L (sorted by exit time)."""
    if pos.is_empty():
        return pos
    return pos.sort(["account_label", "exit_time"]).with_columns(
        pl.col("net_pnl").cum_sum().over("account_label").alias("cum_pnl"),
    )
