"""Celery-приложение и задачи OCR."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import structlog
from celery import Celery, Task
from celery.signals import worker_process_init, worker_process_shutdown
from celery.exceptions import SoftTimeLimitExceeded

from app.config import get_settings
from app.services.ocr_service import get_ocr_service, reset_ocr_service

settings = get_settings()
logger = structlog.get_logger(__name__)

celery_app = Celery(
    "pdf_ocr_worker",
    broker=settings.celery_broker_url,
    backend=settings.celery_result_backend,
)

celery_app.conf.update(
    task_serializer="json",
    result_serializer="json",
    accept_content=["json"],
    timezone="UTC",
    enable_utc=True,
    worker_prefetch_multiplier=1,
    worker_concurrency=1,
    task_soft_time_limit=settings.task_soft_time_limit,
    task_time_limit=settings.task_time_limit,
    task_acks_late=True,
    task_reject_on_worker_lost=True,
    task_track_started=True,
    result_expires=86400,
    broker_connection_retry_on_startup=True,
)

# Флаг graceful shutdown: не принимаем новые задачи после SIGTERM
_shutdown_requested = False


@worker_process_init.connect
def init_worker(**_kwargs: Any) -> None:
    """
    Инициализация процесса воркера.

    Модели Docling НЕ грузим здесь по умолчанию — warm_up() блокирует процесс
    на несколько минут (скачивание весов), и задачи не начинают выполняться.
    Загрузка — при первой задаче (stage=loading_models) или при DOCLING_PRELOAD_MODELS=true.
    """
    logger.info("celery.worker_process_started")
    if not settings.docling_preload_models:
        logger.info("celery.worker_ready", preload=False)
        return

    try:
        logger.info("celery.worker_init", message="Предзагрузка моделей Docling...")
        service = get_ocr_service(
            artifacts_path=settings.docling_artifacts_path,
            use_gpu=settings.docling_use_gpu,
            do_formula_enrichment=settings.docling_do_formula_enrichment,
        )
        service.warm_up()
        logger.info("celery.worker_ready", preload=True)
    except Exception as exc:  # noqa: BLE001 — воркер должен стартовать даже без моделей
        logger.error("celery.worker_preload_failed", error=str(exc))
        reset_ocr_service()
        logger.info("celery.worker_ready", preload=False, note="models will load on first task")


@worker_process_shutdown.connect
def shutdown_worker(**_kwargs: Any) -> None:
    """Graceful shutdown: освобождаем ресурсы после завершения текущей задачи."""
    global _shutdown_requested  # noqa: PLW0603
    _shutdown_requested = True
    logger.info("celery.worker_shutdown", message="Воркер завершает работу")
    reset_ocr_service()


class OCRTask(Task):
    """Базовый класс задачи с автоматическим retry."""

    autoretry_for = (OSError, RuntimeError)
    retry_kwargs = {"max_retries": 2, "countdown": 30}
    retry_backoff = True


@celery_app.task(bind=True, base=OCRTask, name="convert_pdf")
def convert_pdf_task(
    self: Task,
    file_path: str,
    output_format: str,
    language: str,
    correlation_id: str | None = None,
) -> dict[str, Any]:
    """
    Асинхронная конвертация PDF.

    Сохраняет результат в results_dir и возвращает метаданные.
    """
    task_id = self.request.id or "unknown"
    results_base = settings.results_dir / task_id
    results_base.mkdir(parents=True, exist_ok=True)

    log = logger.bind(task_id=task_id, correlation_id=correlation_id)
    log.info("task.started", file_path=file_path, output_format=output_format)

    def update_progress(progress: int, stage: str = "processing") -> None:
        self.update_state(
            state="PROGRESS",
            meta={"progress": progress, "status": "processing", "stage": stage},
        )

    try:
        self.update_state(state="STARTED", meta={"progress": 0, "status": "processing"})
        update_progress(1)

        ocr_service = get_ocr_service(
            artifacts_path=settings.docling_artifacts_path,
            use_gpu=settings.docling_use_gpu,
            do_formula_enrichment=settings.docling_do_formula_enrichment,
        )

        if not ocr_service.is_ready:
            update_progress(2, "loading_models")
            ocr_service.warm_up()

        content, _image_dir = ocr_service.convert(
            Path(file_path),
            output_format=output_format,  # type: ignore[arg-type]
            language=language,  # type: ignore[arg-type]
            progress_callback=update_progress,
            correlation_id=correlation_id,
            results_dir=results_base,
        )

        extension = ".tex" if output_format == "latex" else ".md"
        result_path = results_base / f"result{extension}"
        result_path.write_text(content, encoding="utf-8")

        meta = {
            "status": "completed",
            "progress": 100,
            "result_path": str(result_path),
            "output_format": output_format,
            "file_path": file_path,
        }
        meta_path = results_base / "meta.json"
        meta_path.write_text(json.dumps(meta, ensure_ascii=False), encoding="utf-8")

        log.info("task.completed", result_path=str(result_path))
        return meta

    except SoftTimeLimitExceeded as exc:
        log.error("task.soft_timeout", error=str(exc))
        raise
    except Exception as exc:
        log.error("task.failed", error=str(exc))
        self.update_state(
            state="FAILURE",
            meta={"progress": 0, "status": "failed", "error": str(exc)},
        )
        raise


def get_task_result_path(task_id: str) -> Path | None:
    """Возвращает путь к файлу результата по task_id."""
    meta_path = settings.results_dir / task_id / "meta.json"
    if not meta_path.exists():
        return None
    meta = json.loads(meta_path.read_text(encoding="utf-8"))
    result_path = meta.get("result_path")
    if result_path and Path(result_path).exists():
        return Path(result_path)
    return None


def get_task_meta(task_id: str) -> dict[str, Any] | None:
    """Читает метаданные задачи из файловой системы."""
    meta_path = settings.results_dir / task_id / "meta.json"
    if meta_path.exists():
        return json.loads(meta_path.read_text(encoding="utf-8"))
    return None
