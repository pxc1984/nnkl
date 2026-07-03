"""Synchronous OCR parsing orchestration over shared Postgres storage."""

from __future__ import annotations

import shutil
import tempfile
import uuid
import zipfile
from io import BytesIO
from pathlib import Path

from sqlalchemy.orm import Session

from app.api.schemas import ParseRequest, TaskStatus
from app.config import Settings
from app.db.models import InputBlob, ParseJob, ParseResult
from app.services.ocr_service import get_ocr_service


class InputBlobNotFoundError(Exception):
    """Raised when the requested shared input blob does not exist."""


def parse_document(
    session: Session,
    request: ParseRequest,
    *,
    settings: Settings,
    correlation_id: str | None,
) -> ParseJob:
    blob = session.get(InputBlob, request.input_blob_id)
    if blob is None:
        raise InputBlobNotFoundError(request.input_blob_id)

    job = session.query(ParseJob).filter(ParseJob.document_id == request.document_id).one_or_none()
    if job is None:
        job = ParseJob(
            id=str(uuid.uuid4()),
            document_id=request.document_id,
            input_blob_id=blob.id,
            status=TaskStatus.PROCESSING.value,
            output_format=request.output_format.value,
            language=request.language.value,
            error=None,
        )
        session.add(job)
    else:
        job.input_blob_id = blob.id
        job.status = TaskStatus.PROCESSING.value
        job.output_format = request.output_format.value
        job.language = request.language.value
        job.error = None
    session.commit()

    work_dir = Path(tempfile.mkdtemp(prefix=f"ocr_{job.id}_", dir=str(settings.temp_dir)))
    try:
        input_path = work_dir / _resolve_filename(blob.filename)
        input_path.write_bytes(blob.content)
        output_dir = work_dir / "result"
        output_dir.mkdir(parents=True, exist_ok=True)

        ocr_service = get_ocr_service(
            artifacts_path=settings.docling_artifacts_path,
            use_gpu=settings.docling_use_gpu,
            do_formula_enrichment=settings.docling_do_formula_enrichment,
        )
        content, image_dir = ocr_service.convert(
            input_path,
            output_format=request.output_format.value,
            language=request.language.value,
            correlation_id=correlation_id,
            results_dir=output_dir,
        )

        result = job.result
        if result is None:
            result = ParseResult(
                id=str(uuid.uuid4()),
                job_id=job.id,
                content_type=_resolve_content_type(request.output_format.value),
                content_text=content,
                assets_zip=_pack_assets(image_dir),
            )
            session.add(result)
        else:
            result.content_type = _resolve_content_type(request.output_format.value)
            result.content_text = content
            result.assets_zip = _pack_assets(image_dir)

        job.status = TaskStatus.COMPLETED.value
        session.commit()
        session.refresh(job)
        return job
    except Exception as exc:
        job.status = TaskStatus.FAILED.value
        job.error = str(exc)
        session.commit()
        raise
    finally:
        shutil.rmtree(work_dir, ignore_errors=True)


def _resolve_filename(filename: str) -> str:
    candidate = Path(filename).name or "document.pdf"
    if candidate.lower().endswith(".pdf"):
        return candidate
    return f"{candidate}.pdf"


def _resolve_content_type(output_format: str) -> str:
    if output_format == "latex":
        return "application/x-tex"
    return "text/markdown"


def _pack_assets(image_dir: Path | None) -> bytes | None:
    if image_dir is None or not image_dir.exists() or not any(image_dir.iterdir()):
        return None

    buffer = BytesIO()
    with zipfile.ZipFile(buffer, mode="w", compression=zipfile.ZIP_DEFLATED) as archive:
        for path in sorted(image_dir.rglob("*")):
            if path.is_file():
                archive.write(path, arcname=path.relative_to(image_dir))
    return buffer.getvalue()
