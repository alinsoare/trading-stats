"""Persist data-folder paths with QSettings (Linux: ~/.config, Windows: registry)."""

from __future__ import annotations

from PySide6.QtCore import QSettings


def load_paths(settings: QSettings) -> list[str]:
    raw = settings.value("data_folders", defaultValue=[])
    if raw is None:
        return []
    if isinstance(raw, str):
        return [raw] if raw.strip() else []
    return [str(x).strip() for x in raw if str(x).strip()]


def save_paths(settings: QSettings, paths: list[str]) -> None:
    settings.setValue("data_folders", paths)
