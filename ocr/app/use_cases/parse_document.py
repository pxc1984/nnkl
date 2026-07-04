"""Synchronous OCR parsing orchestration over shared Postgres storage."""

from __future__ import annotations

import shutil
import tempfile
import uuid
from hashlib import sha256
from pathlib import Path

import structlog
from sqlalchemy.orm import Session

from app.api.schemas import ParseRequest, TaskStatus
from app.config import Settings
from app.db.models import Blob, Upload
from app.services.ocr_service import get_ocr_service
from app.use_cases.document_extractor import (
    extract_native_document_text,
    should_use_native_pdf_text,
)


logger = structlog.get_logger(__name__)


class InputBlobNotFoundError(Exception):
    """Raised when the requested shared input blob does not exist."""


def parse_document(
    session: Session,
    request: ParseRequest,
    *,
    settings: Settings,
    correlation_id: str | None,
) -> Upload:
    blob = session.get(Blob, request.input_blob_id)
    if blob is None:
        raise InputBlobNotFoundError(request.input_blob_id)

    job = session.get(Upload, request.upload_id)
    if job is None:
        job = Upload(
            id=request.upload_id,
            input_blob_id=blob.id,
            status=TaskStatus.PROCESSING.value,
            language=request.language.value,
            error=None,
        )
        session.add(job)
    else:
        job.input_blob_id = blob.id
        job.status = TaskStatus.PROCESSING.value
        job.language = request.language.value
        job.error = None
    session.commit()

    work_dir = Path(
        tempfile.mkdtemp(prefix=f"ocr_{job.id}_", dir=str(settings.ocr_temp_dir))
    )
    try:
        input_path = work_dir / _resolve_filename(blob.filename)
        input_path.write_bytes(blob.content)
        output_dir = work_dir / "result"
        output_dir.mkdir(parents=True, exist_ok=True)

        content, image_dir = _parse_input_document(
            input_path,
            language=request.language.value,
            correlation_id=correlation_id,
            results_dir=output_dir,
            settings=settings,
        )

        content = f"<!-- source: {blob.filename} -->\n\n{content.strip()}"

        output_blob = _get_or_create_blob(
            session,
            filename=f"{Path(blob.filename).stem}.md",
            file_type="markdown",
            content_type="text/markdown",
            content=content.encode("utf-8"),
        )
        job.output_blob_id = output_blob.id

        job.status = TaskStatus.COMPLETED.value
        session.commit()
        session.refresh(job)
        return job
    except Exception as exc:
        session.rollback()
        job.status = TaskStatus.FAILED.value
        job.error = str(exc)
        session.commit()
        raise
    finally:
        shutil.rmtree(work_dir, ignore_errors=True)


def _resolve_filename(filename: str) -> str:
    candidate = Path(filename).name or "document.pdf"
    if Path(candidate).suffix.lower() in {".pdf", ".docx", ".pptx"}:
        return candidate
    return f"{candidate}.pdf"


def _parse_input_document(
    input_path: Path,
    *,
    language: str,
    correlation_id: str | None,
    results_dir: Path,
    settings: Settings,
) -> tuple[str, Path | None]:
    if input_path.suffix.lower() == ".pdf":
        use_native = should_use_native_pdf_text(
            input_path,
            min_chars=settings.ocr_native_min_characters,
            minimum_usable_ratio=settings.ocr_native_minimum_usable_page_ratio,
        )
        logger.info(
            "document_extraction.route_selected",
            path=str(input_path),
            route="native" if use_native else "yandex_vision",
            correlation_id=correlation_id,
        )
        if use_native:
            return extract_native_document_text(input_path), None

        ocr_service = get_ocr_service(
            api_key=settings.yandex_vision_api_key,
            folder_id=settings.yandex_folder_id,
            artifacts_path=settings.ocr_mineru_models_dir,
            use_gpu=settings.ocr_mineru_use_gpu,
            backend=settings.ocr_mineru_backend,
            document_timeout=settings.ocr_mineru_document_timeout_seconds,
            preprocess_scans=settings.ocr_mineru_preprocess_scans,
            scan_dpi=settings.ocr_mineru_scan_dpi,
            max_page_megapixels=settings.ocr_mineru_max_page_megapixels,
        )

        def report_progress(percent: int, stage: str) -> None:
            logger.info(
                "document_extraction.progress",
                percent=percent,
                stage=stage,
                correlation_id=correlation_id,
            )

        return ocr_service.convert(
            input_path,
            language=language,
            correlation_id=correlation_id,
            results_dir=results_dir,
            needs_ocr=True,
            progress_callback=report_progress,
        )

    content = extract_native_document_text(input_path)
    return content, None


def _get_or_create_blob(
    session: Session,
    *,
    filename: str,
    file_type: str,
    content_type: str,
    content: bytes,
) -> Blob:
    content_hash = sha256(content).hexdigest()

    existing_blob = (
        session.query(Blob)
        .filter(
            Blob.sha256 == content_hash,
            Blob.filename == filename,
            Blob.file_type == file_type,
            Blob.content_type == content_type,
            Blob.size_bytes == len(content),
        )
        .one_or_none()
    )
    if existing_blob is not None:
        return existing_blob

    blob = Blob(
        id=uuid.uuid4(),
        filename=filename,
        file_type=file_type,
        content_type=content_type,
        size_bytes=len(content),
        sha256=content_hash,
        content=content,
    )
    session.add(blob)
    session.flush()
    return blob
