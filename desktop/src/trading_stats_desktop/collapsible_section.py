"""Collapsible section widget for use inside a vertical QSplitter."""

from __future__ import annotations

from PySide6.QtCore import Qt, QTimer
from PySide6.QtWidgets import QFrame, QPushButton, QSplitter, QVBoxLayout, QWidget

_QWIDGETSIZE_MAX = 16_777_215


class CollapsibleSection(QFrame):
    """A titled panel whose content can be hidden/shown with a header button.

    When used inside a ``QSplitter`` the toggle redistributes the freed (or
    reclaimed) space to the other expanded siblings so no gap is left behind.
    """

    HEADER_H = 30

    def __init__(
        self,
        title: str,
        content: QWidget,
        parent: QWidget | None = None,
        *,
        min_content_h: int = 80,
    ) -> None:
        super().__init__(parent)
        self.setObjectName("collapsibleSection")
        self._title = title
        self._expanded = True
        self._min_content_h = min_content_h

        vl = QVBoxLayout(self)
        vl.setContentsMargins(0, 0, 0, 0)
        vl.setSpacing(0)

        self._btn = QPushButton(f"▼  {title}")
        self._btn.setObjectName("sectionHeader")
        self._btn.setFixedHeight(self.HEADER_H)
        self._btn.setCursor(Qt.CursorShape.PointingHandCursor)
        self._btn.clicked.connect(self._toggle)
        vl.addWidget(self._btn)

        self._content = content
        vl.addWidget(self._content, stretch=1)

        # When expanded the splitter cannot drag below header + min_content_h.
        # When collapsed it snaps to exactly HEADER_H.
        self.setMinimumHeight(self.HEADER_H + min_content_h)

    # ── public API ─────────────────────────────────────────────────────────

    @property
    def is_expanded(self) -> bool:
        return self._expanded

    def set_expanded(self, expanded: bool) -> None:
        if expanded == self._expanded:
            return
        self._toggle()

    # ── internal ───────────────────────────────────────────────────────────

    def _toggle(self) -> None:
        self._expanded = not self._expanded
        self._content.setVisible(self._expanded)
        self._btn.setText(f"{'▼' if self._expanded else '▶'}  {self._title}")
        if self._expanded:
            self.setMinimumHeight(self.HEADER_H + self._min_content_h)
            self.setMaximumHeight(_QWIDGETSIZE_MAX)
        else:
            self.setMinimumHeight(self.HEADER_H)
            self.setMaximumHeight(self.HEADER_H)
        # Defer redistribution by one tick so Qt commits the constraint first.
        QTimer.singleShot(0, self._redistribute)

    def _redistribute(self) -> None:
        """Push freed / reclaim needed space from/to expanded siblings."""
        parent = self.parentWidget()
        if not isinstance(parent, QSplitter):
            return

        n = parent.count()
        current_sizes = parent.sizes()
        total = sum(current_sizes)

        # Pin every collapsed section to HEADER_H; collect expanded indices.
        new_sizes: list[int] = []
        expanded_idx: list[int] = []
        collapsed_total = 0
        for i in range(n):
            w = parent.widget(i)
            if isinstance(w, CollapsibleSection) and not w.is_expanded:
                new_sizes.append(self.HEADER_H)
                collapsed_total += self.HEADER_H
            else:
                new_sizes.append(current_sizes[i])  # placeholder
                expanded_idx.append(i)

        available = total - collapsed_total
        if not expanded_idx:
            parent.setSizes(new_sizes)
            return

        # Distribute available space proportionally to current expanded sizes.
        expanded_total = sum(current_sizes[i] for i in expanded_idx)
        if expanded_total > 0:
            for i in expanded_idx:
                new_sizes[i] = max(self.HEADER_H, round(available * current_sizes[i] / expanded_total))
        else:
            per = available // len(expanded_idx)
            for i in expanded_idx:
                new_sizes[i] = max(self.HEADER_H, per)

        # Correct rounding drift so the total stays exact.
        drift = total - sum(new_sizes)
        new_sizes[expanded_idx[-1]] = max(self.HEADER_H, new_sizes[expanded_idx[-1]] + drift)

        parent.setSizes(new_sizes)
