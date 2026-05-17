"""Application entry: set Qt matplotlib backend before any GUI imports."""

from __future__ import annotations

import sys


def main() -> None:
    import matplotlib

    matplotlib.use("QtAgg")

    from PySide6.QtWidgets import QApplication

    from trading_stats_desktop.main_window import MainWindow
    from trading_stats_desktop.theme import apply_theme

    app = QApplication(sys.argv)
    app.setOrganizationName("TradingStats")
    app.setApplicationName("TradingStatsDesktop")
    apply_theme(app)
    win = MainWindow()
    win.show()
    raise SystemExit(app.exec())


if __name__ == "__main__":
    main()
