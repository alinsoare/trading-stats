"""Application-wide Qt stylesheet and colour tokens for the dark trading theme."""

from __future__ import annotations

# ── colour tokens ──────────────────────────────────────────────────────────────
BG_BASE      = "#12121f"   # window background
BG_SURFACE   = "#1c1c2e"   # panel / group-box fill
BG_SIDEBAR   = "#161625"   # left sidebar strip
BG_INPUT     = "#23233a"   # text inputs / list boxes
BG_HOVER     = "#2d2d50"
BG_SELECTED  = "#3d5a80"

ACCENT_PRIMARY = "#4f9cf9"
ACCENT_HOVER   = "#6baeff"
ACCENT_PRESSED = "#3580e0"

TEXT_PRIMARY   = "#e8eaf6"
TEXT_SECONDARY = "#9e9ebf"
TEXT_MUTED     = "#5c5c7a"

BORDER_NORMAL = "#2e2e48"
BORDER_FOCUS  = ACCENT_PRIMARY

POSITIVE = "#4caf50"
NEGATIVE = "#ef5350"

# matplotlib companion tokens (assign on the Figure/Axes directly)
MPL_BG     = BG_SURFACE
MPL_AX_BG  = BG_BASE
MPL_GRID   = "#282844"
MPL_TEXT   = TEXT_SECONDARY
MPL_SPINE  = BORDER_NORMAL

STYLESHEET = f"""
/* ── global ──────────────────────────────────────────────────────────────── */
QMainWindow, QWidget {{
    background-color: {BG_BASE};
    color: {TEXT_PRIMARY};
    font-family: "Segoe UI", "SF Pro Text", sans-serif;
    font-size: 9pt;
}}

/* ── left sidebar ────────────────────────────────────────────────────────── */
QWidget#sidebar {{
    background-color: {BG_SIDEBAR};
    border-right: 1px solid {BORDER_NORMAL};
}}

/* ── group boxes ─────────────────────────────────────────────────────────── */
QGroupBox {{
    background-color: {BG_SURFACE};
    border: 1px solid {BORDER_NORMAL};
    border-radius: 6px;
    margin-top: 12px;
    padding-top: 8px;
}}
QGroupBox::title {{
    subcontrol-origin: margin;
    subcontrol-position: top left;
    left: 10px;
    top: 2px;
    padding: 0 4px;
    color: {TEXT_SECONDARY};
    font-size: 8pt;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}}

/* ── inputs ──────────────────────────────────────────────────────────────── */
QLineEdit, QDateEdit, QComboBox {{
    background: {BG_INPUT};
    border: 1px solid {BORDER_NORMAL};
    border-radius: 4px;
    padding: 4px 8px;
    color: {TEXT_PRIMARY};
    selection-background-color: {BG_SELECTED};
}}
QLineEdit:focus, QDateEdit:focus, QComboBox:focus {{
    border: 1px solid {BORDER_FOCUS};
}}
QComboBox::drop-down {{
    border: none;
    width: 20px;
}}
QComboBox QAbstractItemView {{
    background: {BG_INPUT};
    selection-background-color: {BG_SELECTED};
    border: 1px solid {BORDER_NORMAL};
    color: {TEXT_PRIMARY};
}}
QDateEdit::drop-down {{
    border: none;
    width: 20px;
}}
QDateEdit::up-button, QDateEdit::down-button {{ height: 0; width: 0; }}

/* ── buttons ─────────────────────────────────────────────────────────────── */
QPushButton {{
    background: {BG_INPUT};
    color: {TEXT_PRIMARY};
    border: 1px solid {BORDER_NORMAL};
    border-radius: 4px;
    padding: 5px 12px;
}}
QPushButton:hover {{
    background: {BG_HOVER};
    border-color: {ACCENT_PRIMARY};
}}
QPushButton:pressed {{ background: {BG_SELECTED}; }}
QPushButton:disabled {{ color: {TEXT_MUTED}; border-color: {BG_INPUT}; }}

QPushButton[accent="true"] {{
    background: {ACCENT_PRIMARY};
    color: {BG_BASE};
    border: none;
    font-weight: 700;
}}
QPushButton[accent="true"]:hover  {{ background: {ACCENT_HOVER};   }}
QPushButton[accent="true"]:pressed {{ background: {ACCENT_PRESSED}; }}

/* ── checkboxes ──────────────────────────────────────────────────────────── */
QCheckBox {{
    color: {TEXT_SECONDARY};
    spacing: 6px;
}}
QCheckBox::indicator {{
    width: 14px; height: 14px;
    border: 1px solid {BORDER_NORMAL};
    border-radius: 3px;
    background: {BG_INPUT};
}}
QCheckBox::indicator:checked {{
    background: {ACCENT_PRIMARY};
    border-color: {ACCENT_PRIMARY};
    image: none;
}}

/* ── list widgets ────────────────────────────────────────────────────────── */
QListWidget {{
    background: {BG_INPUT};
    border: 1px solid {BORDER_NORMAL};
    border-radius: 4px;
    color: {TEXT_PRIMARY};
    outline: 0;
}}
QListWidget::item {{ padding: 3px 4px; }}
QListWidget::item:selected {{
    background: {BG_SELECTED};
    color: {TEXT_PRIMARY};
}}
QListWidget::item:hover:!selected {{ background: {BG_HOVER}; }}

/* ── tables ──────────────────────────────────────────────────────────────── */
QTableWidget {{
    background: {BG_BASE};
    gridline-color: {BORDER_NORMAL};
    border: 1px solid {BORDER_NORMAL};
    border-radius: 4px;
    alternate-background-color: {BG_SURFACE};
    color: {TEXT_PRIMARY};
    selection-background-color: {BG_SELECTED};
    outline: 0;
}}
QTableWidget::item {{ padding: 3px 6px; }}
QTableWidget::item:selected {{ background: {BG_SELECTED}; }}
QHeaderView::section {{
    background: {BG_SIDEBAR};
    color: {TEXT_SECONDARY};
    border: none;
    border-right: 1px solid {BORDER_NORMAL};
    border-bottom: 1px solid {BORDER_NORMAL};
    padding: 5px 8px;
    font-weight: 700;
    font-size: 8pt;
    text-transform: uppercase;
    letter-spacing: 0.3px;
}}
QHeaderView {{ background: {BG_SIDEBAR}; }}

/* ── tab widget ──────────────────────────────────────────────────────────── */
QTabWidget::pane {{
    border: 1px solid {BORDER_NORMAL};
    border-radius: 0 4px 4px 4px;
    background: {BG_BASE};
}}
QTabBar::tab {{
    background: {BG_SURFACE};
    color: {TEXT_SECONDARY};
    border: 1px solid {BORDER_NORMAL};
    border-bottom: none;
    border-radius: 4px 4px 0 0;
    padding: 6px 16px;
    margin-right: 2px;
    font-size: 8.5pt;
}}
QTabBar::tab:selected {{
    background: {BG_BASE};
    color: {TEXT_PRIMARY};
    border-bottom: 2px solid {ACCENT_PRIMARY};
    font-weight: 700;
}}
QTabBar::tab:hover:!selected {{ background: {BG_HOVER}; color: {TEXT_PRIMARY}; }}

/* ── splitter ────────────────────────────────────────────────────────────── */
QSplitter::handle:horizontal {{
    background: {BORDER_NORMAL};
    width: 1px;
}}
QSplitter::handle:vertical {{
    background: {BORDER_NORMAL};
    height: 1px;
}}

/* ── collapsible section ─────────────────────────────────────────────────── */
QFrame#collapsibleSection {{
    background: {BG_SURFACE};
    border: 1px solid {BORDER_NORMAL};
    border-radius: 6px;
}}
QPushButton#sectionHeader {{
    background: {BG_SURFACE};
    color: {TEXT_SECONDARY};
    border: none;
    border-bottom: 1px solid {BORDER_NORMAL};
    border-radius: 6px 6px 0 0;
    padding: 0 12px;
    text-align: left;
    font-weight: 700;
    font-size: 8pt;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}}
QPushButton#sectionHeader:hover {{
    background: {BG_HOVER};
    color: {TEXT_PRIMARY};
}}

/* ── scroll bars ─────────────────────────────────────────────────────────── */
QScrollBar:vertical {{
    background: {BG_BASE};
    width: 8px;
    margin: 0;
}}
QScrollBar::handle:vertical {{
    background: {BG_HOVER};
    border-radius: 4px;
    min-height: 24px;
}}
QScrollBar::add-line:vertical, QScrollBar::sub-line:vertical {{ height: 0; }}
QScrollBar:horizontal {{
    background: {BG_BASE};
    height: 8px;
    margin: 0;
}}
QScrollBar::handle:horizontal {{
    background: {BG_HOVER};
    border-radius: 4px;
    min-width: 24px;
}}
QScrollBar::add-line:horizontal, QScrollBar::sub-line:horizontal {{ width: 0; }}

/* ── KPI cards ───────────────────────────────────────────────────────────── */
QFrame[role="kpi_card"] {{
    background: {BG_SURFACE};
    border: 1px solid {BORDER_NORMAL};
    border-radius: 6px;
}}

/* ── labels ──────────────────────────────────────────────────────────────── */
QLabel[role="section"] {{
    color: {TEXT_SECONDARY};
    font-size: 8pt;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}}
QLabel[role="status"] {{
    color: {TEXT_MUTED};
    font-size: 8pt;
    padding: 4px 0;
}}
"""


def apply_theme(app: object) -> None:
    from PySide6.QtWidgets import QApplication
    assert isinstance(app, QApplication)
    app.setStyle("Fusion")
    app.setStyleSheet(STYLESHEET)
