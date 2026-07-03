"""Вспомогательные функции для отчёта о прогрессе долгих операций."""

from __future__ import annotations

import threading
import time
from collections.abc import Callable, Generator
from contextlib import contextmanager
from typing import TypeVar

T = TypeVar("T")

ProgressCallback = Callable[[int, str], None]


@contextmanager
def progress_heartbeat(
    callback: ProgressCallback | None,
    *,
    start: int,
    end: int,
    stage: str,
    interval_sec: float = 15.0,
    step: int = 3,
) -> Generator[None, None, None]:
    """
    Пока выполняется долгая операция, периодически обновляет progress.

    Нужен потому что Docling не отдаёт промежуточный прогресс во время convert().
    """
    if callback is None:
        yield
        return

    stop = threading.Event()
    current = start

    def _tick() -> None:
        nonlocal current
        while not stop.wait(interval_sec):
            if current < end - 1:
                current = min(current + step, end - 1)
                callback(current, stage)

    thread = threading.Thread(target=_tick, name=f"progress-{stage}", daemon=True)
    thread.start()
    try:
        yield
    finally:
        stop.set()
        thread.join(timeout=1.0)


def run_with_progress(
    callback: ProgressCallback | None,
    fn: Callable[[], T],
    *,
    start: int,
    end: int,
    stage: str,
) -> T:
    """Оборачивает блокирующий вызов heartbeat-ом."""
    with progress_heartbeat(callback, start=start, end=end, stage=stage):
        return fn()
