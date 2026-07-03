"""HTTP-эндпоинты API."""

from __future__ import annotations

import structlog
from fastapi import APIRouter, HTTPException, Request, status
from sqlalchemy.orm import Session

from app.api.schemas import ErrorResponse, HealthResponse, ParseRequest, ParseResponse, ParseResultResponse, StatusResponse, TaskStatus
from app.config import get_settings
from app.db.models import ParseJob
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
            document_id=job.document_id,
            job_id=job.id,
            correlation_id=correlation_id,
        )
        if job.result is None:
            raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail="Parse result was not persisted")
        return ParseResponse(
            document_id=job.document_id,
            job_id=job.id,
            result_id=job.result.id,
            status=TaskStatus.COMPLETED,
        )
    except InputBlobNotFoundError as exc:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Input blob '{exc.args[0]}' not found",
        ) from exc
    finally:
        session.close()


@router.get("/status/{document_id}", response_model=StatusResponse)
async def get_status(document_id: str, request: Request) -> StatusResponse:
    """Возвращает последний статус OCR-парсинга по document_id."""
    session = _get_session(request)

    try:
        job = session.query(ParseJob).filter(ParseJob.document_id == document_id).one_or_none()
        if job is None:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Parse job not found")

        return StatusResponse(
            document_id=job.document_id,
            job_id=job.id,
            status=TaskStatus(job.status),
            input_blob_id=job.input_blob_id,
            output_format=job.output_format,
            language=job.language,
            result_id=job.result.id if job.result is not None else None,
            error=job.error,
        )
    finally:
        session.close()


@router.get("/result/{document_id}", response_model=ParseResultResponse)
async def get_result(document_id: str, request: Request) -> ParseResultResponse:
    """Возвращает OCR-результат, сохранённый в shared Postgres."""
    session = _get_session(request)

    try:
        job = session.query(ParseJob).filter(ParseJob.document_id == document_id).one_or_none()
        if job is None or job.result is None:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Parse result not found")

        return ParseResultResponse(
            document_id=job.document_id,
            job_id=job.id,
            result_id=job.result.id,
            content_type=job.result.content_type,
            content_text=job.result.content_text,
            has_assets_zip=job.result.assets_zip is not None,
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
