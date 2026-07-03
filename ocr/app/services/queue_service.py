"""Получение состояния очереди Celery."""

from __future__ import annotations

import json
from typing import Any

import redis
import structlog
from celery import Celery
from celery.result import AsyncResult

from app.api.schemas import QueueResponse, QueueTaskItem, TaskStatus

logger = structlog.get_logger(__name__)

CONVERT_TASK_NAME = "convert_pdf"
DEFAULT_QUEUE_NAME = "celery"
INSPECT_TIMEOUT = 2.0


def get_queue_snapshot(
    celery_app: Celery,
    *,
    broker_url: str,
    status_filter: TaskStatus | None = None,
) -> QueueResponse:
    """
    Собирает снимок очереди: pending (в брокере), processing (active/reserved), scheduled.

    pending — задачи в Redis, ещё не взятые воркером.
    processing — выполняются или зарезервированы воркером.
    scheduled — отложенные (ETA).
    """
    pending = _get_broker_pending_tasks(broker_url)
    processing = _get_inspect_tasks(celery_app, kind="processing")
    scheduled = _get_inspect_tasks(celery_app, kind="scheduled")

    # Дедупликация: reserved/active не должны дублировать broker pending
    processing_ids = {t.task_id for t in processing}
    pending = [t for t in pending if t.task_id not in processing_ids]

    if status_filter is not None:
        if status_filter == TaskStatus.PENDING:
            tasks = pending + scheduled
            return QueueResponse(
                total=len(tasks),
                pending_count=len(pending),
                processing_count=0,
                scheduled_count=len(scheduled),
                pending=pending,
                processing=[],
                scheduled=scheduled,
            )
        if status_filter == TaskStatus.PROCESSING:
            return QueueResponse(
                total=len(processing),
                pending_count=0,
                processing_count=len(processing),
                scheduled_count=0,
                pending=[],
                processing=processing,
                scheduled=[],
            )
        if status_filter == TaskStatus.COMPLETED:
            return QueueResponse(
                total=0,
                pending_count=0,
                processing_count=0,
                scheduled_count=0,
                pending=[],
                processing=[],
                scheduled=[],
            )

    total = len(pending) + len(processing) + len(scheduled)
    return QueueResponse(
        total=total,
        pending_count=len(pending),
        processing_count=len(processing),
        scheduled_count=len(scheduled),
        pending=pending,
        processing=processing,
        scheduled=scheduled,
    )


def _get_inspect_tasks(celery_app: Celery, *, kind: str) -> list[QueueTaskItem]:
    """Читает active/reserved или scheduled через Celery inspect API."""
    try:
        inspect = celery_app.control.inspect(timeout=INSPECT_TIMEOUT)
        if kind == "processing":
            active = inspect.active() or {}
            reserved = inspect.reserved() or {}
            raw_tasks: list[tuple[str, dict[str, Any]]] = []
            for worker, tasks in active.items():
                for task in tasks:
                    raw_tasks.append((worker, task))
            for worker, tasks in reserved.items():
                for task in tasks:
                    raw_tasks.append((worker, task))
        elif kind == "scheduled":
            scheduled = inspect.scheduled() or {}
            raw_tasks = []
            for worker, tasks in scheduled.items():
                for task in tasks:
                    raw_tasks.append((worker, task.get("request", task)))
        else:
            return []

        items: list[QueueTaskItem] = []
        seen: set[str] = set()
        for worker, task in raw_tasks:
            item = _task_dict_to_item(task, worker=worker, status=TaskStatus.PROCESSING if kind == "processing" else TaskStatus.PENDING)
            if item is None or item.task_id in seen:
                continue
            seen.add(item.task_id)
            if kind == "processing":
                item = _enrich_with_progress(celery_app, item)
            if kind == "scheduled" and "eta" in task:
                item.eta = str(task.get("eta"))
            items.append(item)
        return items
    except Exception as exc:  # noqa: BLE001
        logger.warning("queue.inspect_failed", kind=kind, error=str(exc))
        return []


def _get_broker_pending_tasks(broker_url: str) -> list[QueueTaskItem]:
    """Читает задачи из Redis-очереди брокера Celery."""
    try:
        client = redis.from_url(broker_url)
        raw_messages = client.lrange(DEFAULT_QUEUE_NAME, 0, -1)
    except Exception as exc:  # noqa: BLE001
        logger.warning("queue.broker_read_failed", error=str(exc))
        return []

    items: list[QueueTaskItem] = []
    seen: set[str] = set()

    for raw in raw_messages:
        item = _parse_broker_message(raw)
        if item is None or item.task_id in seen:
            continue
        seen.add(item.task_id)
        items.append(item)

    return items


def _parse_broker_message(raw: bytes | str) -> QueueTaskItem | None:
    """Разбирает сообщение Celery из Redis (JSON-сериализация)."""
    try:
        if isinstance(raw, bytes):
            raw = raw.decode("utf-8")
        envelope = json.loads(raw)
        headers = envelope.get("headers") or {}
        task_name = headers.get("task", "")
        if task_name and task_name != CONVERT_TASK_NAME:
            return None

        task_id = headers.get("id")
        if not task_id:
            return None

        kwargs: dict[str, Any] = {}
        body = envelope.get("body")
        if body:
            if isinstance(body, str):
                body_data = json.loads(body)
            else:
                body_data = body
            if isinstance(body_data, (list, tuple)) and len(body_data) >= 2:
                kwargs = body_data[1] if isinstance(body_data[1], dict) else {}

        return QueueTaskItem(
            task_id=task_id,
            status=TaskStatus.PENDING,
            name=task_name or CONVERT_TASK_NAME,
            file_path=kwargs.get("file_path"),
            output_format=kwargs.get("output_format"),
            language=kwargs.get("language"),
        )
    except (json.JSONDecodeError, TypeError, KeyError) as exc:
        logger.ocr_debug("queue.broker_message_parse_failed", error=str(exc))
        return None


def _task_dict_to_item(
    task: dict[str, Any],
    *,
    worker: str,
    status: TaskStatus,
) -> QueueTaskItem | None:
    task_id = task.get("id")
    if not task_id:
        return None

    name = task.get("name", "")
    if name and name != CONVERT_TASK_NAME:
        return None

    kwargs = task.get("kwargs") or {}
    if not isinstance(kwargs, dict):
        kwargs = {}

    return QueueTaskItem(
        task_id=task_id,
        status=status,
        name=name or CONVERT_TASK_NAME,
        file_path=kwargs.get("file_path"),
        output_format=kwargs.get("output_format"),
        language=kwargs.get("language"),
        worker=worker,
    )


def _enrich_with_progress(celery_app: Celery, item: QueueTaskItem) -> QueueTaskItem:
    """Добавляет progress из result backend для выполняющихся задач."""
    result = AsyncResult(item.task_id, app=celery_app)
    if result.state in ("STARTED", "PROGRESS") and isinstance(result.info, dict):
        item.progress = int(result.info.get("progress", 0))
        item.status = TaskStatus.PROCESSING
    elif result.state == "SUCCESS":
        item.progress = 100
    return item
