"""Валидация PDF через pikepdf и PyMuPDF."""

from __future__ import annotations

from pathlib import Path

import fitz
import pikepdf
import structlog

logger = structlog.get_logger(__name__)


class PDFValidationError(Exception):
    """PDF не прошёл валидацию."""


def validate_pdf(file_path: Path) -> None:
    """
    Проверяет, что файл является корректным PDF.

    Использует pikepdf для структурной валидации и PyMuPDF для проверки
    возможности открытия документа.
    """
    try:
        with pikepdf.open(file_path) as pdf:
            if len(pdf.pages) == 0:
                raise PDFValidationError("PDF не содержит страниц")
    except pikepdf.PdfError as exc:
        logger.warning(
            "pdf.pikepdf_validation_failed", path=str(file_path), error=str(exc)
        )
        raise PDFValidationError(f"Некорректный PDF (pikepdf): {exc}") from exc

    try:
        with fitz.open(file_path) as doc:
            if doc.page_count == 0:
                raise PDFValidationError("PDF не содержит страниц (PyMuPDF)")
    except Exception as exc:  # noqa: BLE001 — fitz может бросать разные типы
        logger.warning(
            "pdf.pymupdf_validation_failed", path=str(file_path), error=str(exc)
        )
        raise PDFValidationError(f"Некорректный PDF (PyMuPDF): {exc}") from exc

    logger.info("pdf.validated", path=str(file_path))
