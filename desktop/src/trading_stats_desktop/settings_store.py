"""Persist data-folder paths with QSettings (Linux: ~/.config, Windows: registry)."""

from __future__ import annotations

import json

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


def load_be_thresholds(settings: QSettings) -> dict[str, float]:
    """Load per-account breakeven tolerance values (account_label → ±$)."""
    raw = settings.value("be_thresholds", defaultValue="{}")
    try:
        return {str(k): float(v) for k, v in json.loads(raw).items()}
    except Exception:  # noqa: BLE001
        return {}


def save_be_thresholds(settings: QSettings, thresholds: dict[str, float]) -> None:
    settings.setValue("be_thresholds", json.dumps(thresholds))
