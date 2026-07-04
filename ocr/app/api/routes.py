"""HTTP-эндпоинты API."""

from __future__ import annotations

import structlog
from uuid import UUID
from fastapi import APIRouter, HTTPException, Request, status
from sqlalchemy.orm import Session

from app.api.schemas import (
    ErrorResponse,
    HealthResponse,
    ParseRequest,
    ParseResponse,
    ParseResultResponse,
    StatusResponse,
    TaskStatus,
)
from app.config import get_settings
from app.db.models import Upload
from app.db.session import check_database
from app.use_cases.parse_document import InputBlobNotFoundError, parse_document

logger = structlog.get_logger(__name__)
router = APIRouter()


def _get_correlation_id(request: Request) -> str | None:
    return getattr(request.state, "correlation_id", None)


def _get_session(request: Request) -> Session:
    return request.app.state.session_factory()


@router.post(
    "/parse",
    response_model=ParseResponse,
    status_code=status.HTTP_201_CREATED,
    responses={404: {"model": ErrorResponse}},
)
async def parse_pdf(request_body: ParseRequest, request: Request) -> ParseResponse:
    """Синхронный OCR-парсинг документа, уже загруженного gateway в shared Postgres."""
    settings = get_settings()
    correlation_id = _get_correlation_id(request)
    session = _get_session(request)

    try:
        job = parse_document(
            session,
            request_body,
            settings=settings,
            correlation_id=correlation_id,
        )
        logger.info(
            "api.parse_completed",
            upload_id=job.id,
            correlation_id=correlation_id,
        )
        if job.output_blob is None:
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail="Parse result was not persisted",
            )
        return ParseResponse(
            upload_id=job.id,
            output_blob_id=job.output_blob.id,
            status=TaskStatus.COMPLETED,
        )
    except InputBlobNotFoundError as exc:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Input blob '{exc.args[0]}' not found",
        ) from exc
    finally:
        session.close()


@router.get("/status/{upload_id}", response_model=StatusResponse)
async def get_status(upload_id: UUID, request: Request) -> StatusResponse:
    """Возвращает последний статус OCR-парсинга по upload_id."""
    session = _get_session(request)

    try:
        job = session.get(Upload, upload_id)
        if job is None:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND, detail="Upload not found"
            )

        return StatusResponse(
            upload_id=job.id,
            status=TaskStatus(job.status),
            input_blob_id=job.input_blob_id,
            output_blob_id=job.output_blob_id,
            language=job.language,
            error=job.error,
        )
    finally:
        session.close()


@router.get("/result/{upload_id}", response_model=ParseResultResponse)
async def get_result(upload_id: UUID, request: Request) -> ParseResultResponse:
    """Возвращает OCR-результат, сохранённый в shared Postgres."""
    session = _get_session(request)

    try:
        job = session.get(Upload, upload_id)
        if job is None or job.output_blob is None:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND, detail="Parse result not found"
            )

        return ParseResultResponse(
            upload_id=job.id,
            output_blob_id=job.output_blob.id,
            content_type=job.output_blob.content_type,
            content_text=job.output_blob.content.decode("utf-8"),
            has_assets_zip=False,
        )
    finally:
        session.close()


@router.get("/health", response_model=HealthResponse)
async def health_check(request: Request) -> HealthResponse:
    """Проверка живости API и shared Postgres."""
    database_status = "error"

    try:
        check_database(request.app.state.engine)
        database_status = "ok"
    except Exception:  # noqa: BLE001
        database_status = "error"

    overall = "ok" if database_status == "ok" else "error"
    return HealthResponse(status=overall, database=database_status)
