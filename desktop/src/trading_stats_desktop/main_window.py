"""Main window: paths, load, filters, KPIs, matplotlib chart, Polars tables."""

from __future__ import annotations

from datetime import datetime
from pathlib import Path
from typing import cast

import polars as pl
from matplotlib.backends.backend_qtagg import FigureCanvasQTAgg as FigureCanvas
from matplotlib.figure import Figure
from PySide6.QtCore import QDate, QSettings, Qt
from PySide6.QtGui import QCloseEvent
from PySide6.QtWidgets import (
    QAbstractItemView,
    QApplication,
    QComboBox,
    QDateEdit,
    QFileDialog,
    QGridLayout,
    QGroupBox,
    QHBoxLayout,
    QInputDialog,
    QLabel,
    QListWidget,
    QListWidgetItem,
    QMainWindow,
    QMessageBox,
    QPushButton,
    QSplitter,
    QTabWidget,
    QTableWidget,
    QVBoxLayout,
    QWidget,
)

from trading_stats.ingest import account_flows, load_deals
from trading_stats.kpis import (
    Bucket,
    add_period_col,
    aggregate_equity_curve,
    closed_positions,
    summarize_groups,
    summarize_positions,
)
from trading_stats_desktop.kpi_card import KpiCard
from trading_stats_desktop.settings_store import load_paths, save_paths
from trading_stats_desktop.table_util import fill_table_from_polars
from trading_stats_desktop.theme import (
    ACCENT_PRIMARY,
    MPL_AX_BG,
    MPL_BG,
    MPL_GRID,
    MPL_SPINE,
    MPL_TEXT,
    POSITIVE,
)


def _fmt_pf(v: float) -> str:
    if v == float("inf"):
        return "∞"
    if v != v:
        return "—"
    return f"{v:.2f}"


def _section_label(text: str) -> QLabel:
    lbl = QLabel(text)
    lbl.setProperty("role", "section")
    return lbl


class MainWindow(QMainWindow):
    _KPI_TITLES = (
        "Trades",
        "Win rate",
        "Net P/L",
        "Profit factor",
        "Max DD",
        "Expectancy",
        "Payoff",
        "Breakeven",
    )

    def __init__(self) -> None:
        super().__init__()
        self.setWindowTitle("Trading Stats")
        self.resize(1280, 820)
        self.setMinimumSize(900, 600)

        self._settings = QSettings()
        self._deals: pl.DataFrame = pl.DataFrame()
        self._pos: pl.DataFrame = pl.DataFrame()

        central = QWidget()
        self.setCentralWidget(central)
        root = QHBoxLayout(central)
        root.setContentsMargins(0, 0, 0, 0)
        root.setSpacing(0)

        splitter = QSplitter(Qt.Orientation.Horizontal)
        root.addWidget(splitter)

        # ── left sidebar ───────────────────────────────────────────────────
        left = QWidget()
        left.setObjectName("sidebar")
        left_l = QVBoxLayout(left)
        left_l.setContentsMargins(12, 16, 12, 16)
        left_l.setSpacing(8)

        left_l.addWidget(_section_label("Data Folders"))

        self._list_paths = QListWidget()
        self._list_paths.setSelectionMode(QAbstractItemView.SelectionMode.ExtendedSelection)
        self._list_paths.setToolTip(
            "Absolute path to a folder containing deals_*.csv "
            "or an MT5 terminal root."
        )
        for p in load_paths(self._settings):
            self._list_paths.addItem(p)
        left_l.addWidget(self._list_paths, stretch=1)

        btn_row = QHBoxLayout()
        btn_row.setSpacing(4)
        self._btn_add_dir = QPushButton("Browse…")
        self._btn_add_dir.clicked.connect(self._on_add_folder)
        btn_row.addWidget(self._btn_add_dir)
        self._btn_add_text = QPushButton("Paste path…")
        self._btn_add_text.clicked.connect(self._on_add_typed)
        btn_row.addWidget(self._btn_add_text)
        self._btn_remove = QPushButton("✕")
        self._btn_remove.setFixedWidth(32)
        self._btn_remove.setToolTip("Remove selected")
        self._btn_remove.clicked.connect(self._on_remove_paths)
        btn_row.addWidget(self._btn_remove)
        left_l.addLayout(btn_row)

        self._btn_load = QPushButton("  Load data  ")
        self._btn_load.setProperty("accent", True)
        self._btn_load.setDefault(True)
        self._btn_load.setCursor(Qt.CursorShape.PointingHandCursor)
        self._btn_load.clicked.connect(self._on_load)
        left_l.addWidget(self._btn_load)

        self._status = QLabel("Configure paths, then Load data.")
        self._status.setProperty("role", "status")
        self._status.setWordWrap(True)
        left_l.addWidget(self._status)

        # ── right content area ─────────────────────────────────────────────
        right = QWidget()
        rl = QVBoxLayout(right)
        rl.setContentsMargins(12, 12, 12, 12)
        rl.setSpacing(8)

        # filters row
        filt = QGroupBox("Filters")
        fl = QGridLayout(filt)
        fl.setHorizontalSpacing(8)
        fl.setVerticalSpacing(6)
        fl.setContentsMargins(10, 14, 10, 10)

        fl.addWidget(QLabel("Exit date from"), 0, 0)
        self._d0 = QDateEdit()
        self._d0.setCalendarPopup(True)
        self._d0.dateChanged.connect(lambda *_: self._refresh_filtered())
        fl.addWidget(self._d0, 0, 1)
        fl.addWidget(QLabel("to"), 0, 2)
        self._d1 = QDateEdit()
        self._d1.setCalendarPopup(True)
        self._d1.dateChanged.connect(lambda *_: self._refresh_filtered())
        fl.addWidget(self._d1, 0, 3)

        fl.addWidget(QLabel("Accounts"), 1, 0)
        self._list_accounts = QListWidget()
        self._list_accounts.setMaximumHeight(100)
        self._list_accounts.itemChanged.connect(lambda *_: self._refresh_filtered())
        fl.addWidget(self._list_accounts, 1, 1, 1, 3)

        fl.addWidget(QLabel("Rollup"), 2, 0)
        self._bucket = QComboBox()
        for b in ("day", "week", "month", "year"):
            self._bucket.addItem(b)
        self._bucket.setCurrentIndex(2)
        self._bucket.currentIndexChanged.connect(lambda *_: self._refresh_filtered())
        fl.addWidget(self._bucket, 2, 1)

        self._btn_export = QPushButton("Export CSV…")
        self._btn_export.clicked.connect(self._on_export_csv)
        fl.addWidget(self._btn_export, 2, 2, 1, 2)

        rl.addWidget(filt)

        # KPI cards row
        kpi_box = QGroupBox("Summary")
        kpi_row = QHBoxLayout(kpi_box)
        kpi_row.setContentsMargins(10, 14, 10, 10)
        kpi_row.setSpacing(8)
        self._kpi_cards: list[KpiCard] = []
        for title in self._KPI_TITLES:
            card = KpiCard(title)
            self._kpi_cards.append(card)
            kpi_row.addWidget(card)
        rl.addWidget(kpi_box)

        # matplotlib equity chart
        self._fig = Figure(figsize=(8, 3), layout="tight")
        self._fig.patch.set_facecolor(MPL_BG)
        self._canvas = FigureCanvas(self._fig)
        self._canvas.setMinimumHeight(180)
        rl.addWidget(self._canvas, stretch=2)

        # data tabs
        tabs = QTabWidget()
        self._tab_roll  = QTableWidget()
        self._tab_sym   = QTableWidget()
        self._tab_flow  = QTableWidget()
        self._tab_acct  = QTableWidget()
        for t in (self._tab_roll, self._tab_sym, self._tab_flow, self._tab_acct):
            t.setAlternatingRowColors(True)
            t.setSelectionBehavior(QAbstractItemView.SelectionBehavior.SelectRows)
            t.setSortingEnabled(True)
        tabs.addTab(self._tab_roll,  "Rollup")
        tabs.addTab(self._tab_sym,   "By symbol")
        tabs.addTab(self._tab_flow,  "Non-trade sample")
        tabs.addTab(self._tab_acct,  "Per-account")
        rl.addWidget(tabs, stretch=3)

        splitter.addWidget(left)
        splitter.addWidget(right)
        splitter.setStretchFactor(0, 0)
        splitter.setStretchFactor(1, 1)
        splitter.setSizes([240, 1040])

    # ── persistence ────────────────────────────────────────────────────────
    def closeEvent(self, event: QCloseEvent) -> None:
        self._persist_paths()
        super().closeEvent(event)

    def _persist_paths(self) -> None:
        paths = [self._list_paths.item(i).text().strip()
                 for i in range(self._list_paths.count())]
        save_paths(self._settings, paths)

    def _path_tuple(self) -> tuple[str, ...]:
        return tuple(
            self._list_paths.item(i).text().strip()
            for i in range(self._list_paths.count())
            if self._list_paths.item(i).text().strip()
        )

    # ── folder management ──────────────────────────────────────────────────
    def _on_add_folder(self) -> None:
        d = QFileDialog.getExistingDirectory(self, "Select folder containing deals CSV export")
        if d:
            self._list_paths.addItem(str(Path(d)))
            self._persist_paths()

    def _on_add_typed(self) -> None:
        text, ok = QInputDialog.getText(self, "Add path", "Absolute path to folder:")
        if ok and text.strip():
            self._list_paths.addItem(text.strip())
            self._persist_paths()

    def _on_remove_paths(self) -> None:
        for item in self._list_paths.selectedItems():
            self._list_paths.takeItem(self._list_paths.row(item))
        self._persist_paths()

    # ── load ───────────────────────────────────────────────────────────────
    def _on_load(self) -> None:
        self._persist_paths()
        paths = self._path_tuple()
        if not paths:
            QMessageBox.information(self, "Load data", "Add at least one folder path.")
            return
        QApplication.setOverrideCursor(Qt.CursorShape.WaitCursor)
        try:
            self._deals = load_deals(list(paths))
        except Exception as e:  # noqa: BLE001
            QApplication.restoreOverrideCursor()
            QMessageBox.critical(self, "Load failed", str(e))
            return
        QApplication.restoreOverrideCursor()

        if self._deals.is_empty():
            self._pos = pl.DataFrame()
            self._status.setText("No deals_*.csv found under the given paths.")
            self._clear_views()
            return

        self._pos = closed_positions(self._deals)
        if self._pos.is_empty():
            self._status.setText("No trade rows after filtering non-trade deals.")
            self._clear_views()
            return

        pos_t = self._pos.drop_nulls("exit_time")
        if pos_t.is_empty():
            self._status.setText("No valid exit times in positions.")
            self._clear_views()
            return

        tmin = pos_t.select(pl.col("exit_time").min()).item()
        tmax = pos_t.select(pl.col("exit_time").max()).item()
        d_lo = tmin.date() if isinstance(tmin, datetime) else tmin
        d_hi = tmax.date() if isinstance(tmax, datetime) else tmax

        for w in (self._d0, self._d1):
            w.blockSignals(True)
        self._d0.setMinimumDate(QDate(d_lo.year, d_lo.month, d_lo.day))
        self._d0.setMaximumDate(QDate(d_hi.year, d_hi.month, d_hi.day))
        self._d1.setMinimumDate(QDate(d_lo.year, d_lo.month, d_lo.day))
        self._d1.setMaximumDate(QDate(d_hi.year, d_hi.month, d_hi.day))
        self._d0.setDate(QDate(d_lo.year, d_lo.month, d_lo.day))
        self._d1.setDate(QDate(d_hi.year, d_hi.month, d_hi.day))
        for w in (self._d0, self._d1):
            w.blockSignals(False)

        accounts = sorted(self._pos["account_label"].unique().to_list())
        self._list_accounts.blockSignals(True)
        self._list_accounts.clear()
        for acc in accounts:
            it = QListWidgetItem(acc)
            it.setFlags(it.flags() | Qt.ItemFlag.ItemIsUserCheckable)
            it.setCheckState(Qt.CheckState.Checked)
            self._list_accounts.addItem(it)
        self._list_accounts.blockSignals(False)

        self._status.setText(f"Loaded {len(self._deals)} deal rows → {len(self._pos)} positions.")
        self._refresh_filtered()

    # ── filters ────────────────────────────────────────────────────────────
    def _selected_accounts(self) -> list[str]:
        return [
            self._list_accounts.item(i).text()
            for i in range(self._list_accounts.count())
            if self._list_accounts.item(i).checkState() == Qt.CheckState.Checked
        ]

    def _filtered_positions(self) -> pl.DataFrame | None:
        if self._pos.is_empty():
            return None
        d0 = self._d0.date().toPython()
        d1 = self._d1.date().toPython()
        if d0 > d1:
            d0, d1 = d1, d0
        pos_f = self._pos.filter(
            (pl.col("exit_time").dt.date() >= pl.lit(d0))
            & (pl.col("exit_time").dt.date() <= pl.lit(d1))
        )
        if pos_f.is_empty():
            return None
        sel = self._selected_accounts()
        if sel:
            pos_f = pos_f.filter(pl.col("account_label").is_in(sel))
        return pos_f if not pos_f.is_empty() else None

    # ── refresh ────────────────────────────────────────────────────────────
    def _refresh_filtered(self) -> None:
        if self._pos.is_empty():
            return
        pos_f = self._filtered_positions()
        if pos_f is None:
            self._status.setText("No positions for current filters.")
            self._clear_kpi_tables_chart()
            return
        agg = summarize_positions(pos_f, label="filtered")
        if agg.get("trades", 0) == 0:
            self._status.setText("No trades in current filters.")
            self._clear_kpi_tables_chart()
            return

        net_pnl  = float(agg["net_pnl"])
        max_dd   = float(agg["max_dd_cum_pnl"])
        exp_val  = float(agg["expectancy"])
        pf_val   = float(agg.get("profit_factor") or 0)
        payoff   = float(agg["payoff"])
        wr       = float(agg["win_rate"])

        self._kpi_cards[0].set_value(f"{agg['trades']}")
        self._kpi_cards[1].set_value(f"{wr*100:.1f}%", positive=wr >= 0.5)
        self._kpi_cards[2].set_value(f"{net_pnl:.2f}", positive=None if net_pnl == 0 else net_pnl > 0)
        self._kpi_cards[3].set_value(_fmt_pf(pf_val), positive=pf_val > 1 if pf_val not in (0, float("inf")) else None)
        self._kpi_cards[4].set_value(f"{max_dd:.2f}", positive=False if max_dd < 0 else None)
        self._kpi_cards[5].set_value(f"{exp_val:.4f}", positive=exp_val > 0 if exp_val != 0 else None)
        self._kpi_cards[6].set_value(f"{payoff:.2f}", positive=payoff > 1 if payoff != 0 else None)
        self._kpi_cards[7].set_value(f"{agg['breakeven']}")

        self._plot_equity(pos_f)

        bucket = cast(Bucket, self._bucket.currentText())
        fill_table_from_polars(
            self._tab_roll,
            summarize_groups(add_period_col(pos_f, bucket), ["account_label", "period"]),
        )

        fill_table_from_polars(
            self._tab_sym,
            summarize_groups(pos_f, ["symbol"]),
        )

        flows = account_flows(self._deals)
        fill_table_from_polars(
            self._tab_flow,
            flows.head(200) if not flows.is_empty() else flows,
            max_rows=200,
        )

        fill_table_from_polars(
            self._tab_acct,
            summarize_groups(pos_f, ["account_label"]),
        )

    # ── chart ──────────────────────────────────────────────────────────────
    def _plot_equity(self, pos_f: pl.DataFrame) -> None:
        self._fig.clear()
        ax = self._fig.add_subplot(111)

        # dark axes
        ax.set_facecolor(MPL_AX_BG)
        self._fig.patch.set_facecolor(MPL_BG)
        for spine in ax.spines.values():
            spine.set_edgecolor(MPL_SPINE)
        ax.tick_params(colors=MPL_TEXT, labelsize=7)
        ax.xaxis.label.set_color(MPL_TEXT)
        ax.yaxis.label.set_color(MPL_TEXT)
        ax.title.set_color(MPL_TEXT)
        ax.grid(True, color=MPL_GRID, linewidth=0.6)

        eq = aggregate_equity_curve(pos_f)
        if eq.is_empty():
            self._canvas.draw_idle()
            return

        palette = [ACCENT_PRIMARY, POSITIVE, "#f9a84f", "#c67bf9", "#f94f7c", "#4ff9e2"]
        accounts = eq["account_label"].unique().to_list()
        for idx, acc in enumerate(accounts):
            sub = eq.filter(pl.col("account_label") == acc).sort("exit_time")
            colour = palette[idx % len(palette)]
            ax.plot(
                sub["exit_time"].to_list(),
                sub["cum_pnl"].to_list(),
                label=str(acc),
                color=colour,
                linewidth=1.5,
            )
            # shade area under curve
            ax.fill_between(
                sub["exit_time"].to_list(),
                sub["cum_pnl"].to_list(),
                alpha=0.08,
                color=colour,
            )

        ax.set_title("Cumulative Net P/L", fontsize=9, color=MPL_TEXT, pad=6)
        ax.set_ylabel("Cum. net P/L", fontsize=8)
        if len(accounts) > 1:
            leg = ax.legend(fontsize=7, loc="best")
            leg.get_frame().set_facecolor(MPL_BG)
            leg.get_frame().set_edgecolor(MPL_SPINE)
            for text in leg.get_texts():
                text.set_color(MPL_TEXT)
        self._fig.autofmt_xdate(rotation=30, ha="right")
        self._canvas.draw_idle()

    # ── clear helpers ──────────────────────────────────────────────────────
    def _clear_kpi_tables_chart(self) -> None:
        for card in self._kpi_cards:
            card.reset()
        self._fig.clear()
        self._fig.patch.set_facecolor(MPL_BG)
        self._canvas.draw_idle()
        for t in (self._tab_roll, self._tab_sym, self._tab_flow, self._tab_acct):
            fill_table_from_polars(t, pl.DataFrame())

    def _clear_views(self) -> None:
        self._list_accounts.clear()
        self._clear_kpi_tables_chart()

    # ── export ─────────────────────────────────────────────────────────────
    def _on_export_csv(self) -> None:
        pos_f = self._filtered_positions()
        if pos_f is None or pos_f.is_empty():
            QMessageBox.information(self, "Export", "Nothing to export for current filters.")
            return
        path, _ = QFileDialog.getSaveFileName(
            self,
            "Save positions CSV",
            "trading_stats_positions_filtered.csv",
            "CSV (*.csv)",
        )
        if not path:
            return
        pos_f.write_csv(path)
        QMessageBox.information(self, "Export", f"Wrote:\n{path}")
