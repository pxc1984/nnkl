"""HTTP-эндпоинты API."""

from __future__ import annotations

import structlog
from celery.result import AsyncResult
from fastapi import APIRouter, HTTPException, Query, Request, status
from fastapi.responses import FileResponse

from app.api.schemas import (
    ConvertRequest,
    ConvertResponse,
    ErrorResponse,
    HealthResponse,
    QueueResponse,
    StatusResponse,
    TaskStatus,
)
from app.config import get_settings
from app.core.path_security import PathSecurityError, validate_file_path
from app.core.pdf_validator import PDFValidationError, validate_pdf
from app.services.queue_service import get_queue_snapshot
from app.workers.celery_app import celery_app, get_task_meta, get_task_result_path

logger = structlog.get_logger(__name__)
router = APIRouter()


def _get_correlation_id(request: Request) -> str | None:
    return getattr(request.state, "correlation_id", None)


@router.post(
    "/convert",
    response_model=ConvertResponse,
    responses={
        400: {"model": ErrorResponse},
        404: {"model": ErrorResponse},
    },
)
async def convert_pdf(request_body: ConvertRequest, request: Request) -> ConvertResponse:
    """Постановка задачи конвертации PDF по локальному пути."""
    settings = get_settings()
    correlation_id = _get_correlation_id(request)

    try:
        validated = validate_file_path(
            request_body.file_path,
            allowed_base=settings.allowed_base_path,
            max_size_bytes=settings.max_file_size_bytes,
            correlation_id=correlation_id,
        )
    except PathSecurityError as exc:
        if exc.reason == "file_not_found":
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=exc.args[0],
            ) from exc
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=exc.args[0],
        ) from exc

    try:
        validate_pdf(validated.resolved)
    except PDFValidationError as exc:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(exc),
        ) from exc

    task = celery_app.send_task(
        "convert_pdf",
        kwargs={
            "file_path": str(validated.resolved),
            "output_format": request_body.output_format.value,
            "language": request_body.language.value,
            "correlation_id": correlation_id,
        },
    )

    logger.info(
        "api.convert_queued",
        task_id=task.id,
        file_path=str(validated.resolved),
        correlation_id=correlation_id,
    )

    return ConvertResponse(task_id=task.id, status=TaskStatus.PENDING)


@router.get("/status/{task_id}", response_model=StatusResponse)
async def get_status(task_id: str, request: Request) -> StatusResponse:
    """Статус задачи конвертации."""
    settings = get_settings()
    correlation_id = _get_correlation_id(request)
    result = AsyncResult(task_id, app=celery_app)

    task_status = TaskStatus.PENDING
    progress = 0
    stage: str | None = None
    error: str | None = None
    result_url: str | None = None

    if result.state == "PENDING":
        task_status = TaskStatus.PENDING
    elif result.state in ("STARTED", "PROGRESS"):
        task_status = TaskStatus.PROCESSING
        meta = result.info if isinstance(result.info, dict) else {}
        progress = int(meta.get("progress", 0))
        stage = meta.get("stage")
    elif result.state == "SUCCESS":
        task_status = TaskStatus.COMPLETED
        progress = 100
        result_url = f"{settings.api_prefix}/download/{task_id}"
    elif result.state == "FAILURE":
        task_status = TaskStatus.FAILED
        error = str(result.info) if result.info else "Неизвестная ошибка"
    else:
        # Fallback: проверяем файловую систему
        fs_meta = get_task_meta(task_id)
        if fs_meta:
            task_status = TaskStatus(fs_meta.get("status", "completed"))
            progress = int(fs_meta.get("progress", 100))
            if task_status == TaskStatus.COMPLETED:
                result_url = f"{settings.api_prefix}/download/{task_id}"

    logger.debug(
        "api.status_checked",
        task_id=task_id,
        status=task_status,
        correlation_id=correlation_id,
    )

    return StatusResponse(
        task_id=task_id,
        status=task_status,
        progress=progress,
        stage=stage,
        result_url=result_url,
        error=error,
    )


@router.get("/queue", response_model=QueueResponse)
async def get_queue(
    request: Request,
    status: TaskStatus | None = Query(
        default=None,
        description="Фильтр: pending, processing (completed в очереди не хранится)",
    ),
) -> QueueResponse:
    """
    Список задач в очереди Celery.

    - **pending** — в Redis-брокере, ещё не взяты воркером
    - **processing** — active/reserved у воркера
    - **scheduled** — отложенные задачи с ETA
    """
    settings = get_settings()
    correlation_id = _get_correlation_id(request)

    snapshot = get_queue_snapshot(
        celery_app,
        broker_url=settings.celery_broker_url,
        status_filter=status,
    )

    logger.info(
        "api.queue_fetched",
        total=snapshot.total,
        pending=snapshot.pending_count,
        processing=snapshot.processing_count,
        scheduled=snapshot.scheduled_count,
        correlation_id=correlation_id,
    )

    return snapshot


@router.get("/download/{task_id}")
async def download_result(task_id: str, request: Request) -> FileResponse:
    """Скачивание результата конвертации (.tex или .md)."""
    correlation_id = _get_correlation_id(request)
    result_path = get_task_result_path(task_id)

    if result_path is None:
        async_result = AsyncResult(task_id, app=celery_app)
        if async_result.state != "SUCCESS":
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Результат ещё не готов или задача не найдена",
            )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Файл результата не найден",
        )

    media_type = "application/x-tex" if result_path.suffix == ".tex" else "text/markdown"
    logger.info(
        "api.download",
        task_id=task_id,
        path=str(result_path),
        correlation_id=correlation_id,
    )

    return FileResponse(
        path=result_path,
        media_type=media_type,
        filename=result_path.name,
    )


@router.get("/health", response_model=HealthResponse)
async def health_check() -> HealthResponse:
    """Проверка живости API, Redis и Celery."""
    settings = get_settings()
    redis_status: str = "error"
    celery_status: str = "unknown"

    try:
        import redis

        client = redis.from_url(settings.redis_url)
        client.ping()
        redis_status = "ok"
    except Exception:  # noqa: BLE001
        redis_status = "error"

    try:
        inspect = celery_app.control.inspect(timeout=2.0)
        ping = inspect.ping()
        celery_status = "ok" if ping else "error"
    except Exception:  # noqa: BLE001
        celery_status = "error"

    if redis_status == "ok" and celery_status == "ok":
        overall = "ok"
    elif redis_status == "error" and celery_status == "error":
        overall = "error"
    else:
        overall = "degraded"

    return HealthResponse(status=overall, redis=redis_status, celery=celery_status)  # type: ignore[arg-type]
