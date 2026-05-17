"""Show Polars frames in QTableWidget with correct numeric and date sorting."""

from __future__ import annotations

import datetime as _dt

import polars as pl
from PySide6.QtCore import Qt
from PySide6.QtWidgets import QTableWidget, QTableWidgetItem


def _sort_key(val: object) -> float | None:
    """
    Return a float sort key for any value that has a natural numeric order.
    - int / float          → float(val)
    - datetime.datetime    → UTC timestamp (float)
    - datetime.date        → ordinal day (float)
    - everything else      → None  (falls back to string sort)
    """
    if val is None:
        return None
    if isinstance(val, bool):
        return None  # bool is a subclass of int; treat as label
    if isinstance(val, (int, float)):
        return float(val)
    if isinstance(val, _dt.datetime):
        return val.timestamp()
    if isinstance(val, _dt.date):
        return float(val.toordinal())
    return None


class _SortableItem(QTableWidgetItem):
    """QTableWidgetItem that sorts by a pre-computed numeric key when available."""

    __slots__ = ("_num",)

    def __init__(self, text: str, num: float | None) -> None:
        super().__init__(text)
        self._num = num

    def __lt__(self, other: QTableWidgetItem) -> bool:
        if self._num is not None and isinstance(other, _SortableItem) and other._num is not None:
            return self._num < other._num
        return self.text() < other.text()


def fill_table_from_polars(
    table: QTableWidget,
    df: pl.DataFrame,
    *,
    max_rows: int | None = None,
) -> None:
    if df.is_empty():
        table.setSortingEnabled(False)
        table.setRowCount(0)
        table.setColumnCount(0)
        table.clearContents()
        return
    if max_rows is not None and len(df) > max_rows:
        df = df.head(max_rows)

    cols = list(df.columns)
    nrows, ncols = len(df), len(cols)

    table.setSortingEnabled(False)  # prevent mid-fill re-sorts
    table.clear()
    table.setColumnCount(ncols)
    table.setRowCount(nrows)
    table.setHorizontalHeaderLabels(cols)

    ro = Qt.ItemFlag.ItemIsSelectable | Qt.ItemFlag.ItemIsEnabled
    for r, row in enumerate(df.iter_rows(named=False)):
        for c, val in enumerate(row):
            s = "" if val is None else str(val)
            item = _SortableItem(s, _sort_key(val))
            item.setFlags(ro)
            table.setItem(r, c, item)

    table.setSortingEnabled(True)
    table.resizeColumnsToContents()
