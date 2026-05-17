"""KPI card: titled frame with a large coloured value label."""

from __future__ import annotations

from PySide6.QtCore import Qt
from PySide6.QtWidgets import QFrame, QLabel, QVBoxLayout

from trading_stats_desktop.theme import NEGATIVE, POSITIVE, TEXT_PRIMARY, TEXT_SECONDARY


class KpiCard(QFrame):
    def __init__(self, title: str, parent: object = None) -> None:
        super().__init__(parent)  # type: ignore[arg-type]
        self.setProperty("role", "kpi_card")
        self.setMinimumWidth(110)

        layout = QVBoxLayout(self)
        layout.setContentsMargins(12, 8, 12, 12)
        layout.setSpacing(2)

        self._title_lbl = QLabel(title.upper())
        self._title_lbl.setAlignment(Qt.AlignmentFlag.AlignLeft)
        self._title_lbl.setStyleSheet(
            f"color: {TEXT_SECONDARY}; font-size: 7.5pt; font-weight: 700;"
            " letter-spacing: 0.5px; background: transparent; border: none;"
        )
        layout.addWidget(self._title_lbl)

        self._value_lbl = QLabel("—")
        self._value_lbl.setAlignment(Qt.AlignmentFlag.AlignLeft)
        self._value_lbl.setTextInteractionFlags(Qt.TextInteractionFlag.TextSelectableByMouse)
        self._value_lbl.setStyleSheet(
            f"color: {TEXT_PRIMARY}; font-size: 15pt; font-weight: 700;"
            " background: transparent; border: none;"
        )
        layout.addWidget(self._value_lbl)

    def set_value(self, text: str, *, positive: bool | None = None) -> None:
        if positive is True:
            colour = POSITIVE
        elif positive is False:
            colour = NEGATIVE
        else:
            colour = TEXT_PRIMARY
        self._value_lbl.setStyleSheet(
            f"color: {colour}; font-size: 15pt; font-weight: 700;"
            " background: transparent; border: none;"
        )
        self._value_lbl.setText(text)

    def reset(self) -> None:
        self.set_value("—")
