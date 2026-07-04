"""Native extraction for documents with a reliable text layer."""

from __future__ import annotations

import re
from pathlib import Path

import fitz
from docx import Document as DocxDocument
from pptx import Presentation

from app.services.pdf_quality import analyze_pdf, sanitize_text

_PAGE_MARKER = "<!-- page: {page_number} -->"
_WHITESPACE_LINES_RE = re.compile(r"[ \t]+\n")
_EXCESS_NEWLINES_RE = re.compile(r"\n{3,}")


class UnsupportedDocumentTypeError(Exception):
    """Raised when the document type is not supported."""


def extract_native_document_text(file_path: Path) -> str:
    suffix = file_path.suffix.lower()
    if suffix == ".pdf":
        return _extract_pdf_text(file_path)
    if suffix == ".docx":
        return _extract_docx_text(file_path)
    if suffix == ".pptx":
        return _extract_pptx_text(file_path)
    raise UnsupportedDocumentTypeError(
        f"Unsupported document type: {suffix or 'unknown'}"
    )


def has_native_pdf_text(file_path: Path, *, min_chars: int = 40) -> bool:
    """Returns true when at least one page has a usable native text layer."""
    try:
        report = analyze_pdf(file_path, minimum_characters=min_chars)
        return report.usable_pages > 0
    except Exception:  # noqa: BLE001
        return False


def should_use_native_pdf_text(
    file_path: Path,
    *,
    min_chars: int = 40,
    minimum_usable_ratio: float = 0.95,
) -> bool:
    """Uses native extraction only when nearly every page has reliable text."""
    try:
        report = analyze_pdf(file_path, minimum_characters=min_chars)
        return report.supports_native_extraction(
            minimum_usable_ratio=minimum_usable_ratio
        )
    except Exception:  # noqa: BLE001
        return False


def _extract_pdf_text(file_path: Path) -> str:
    pages: list[str] = []
    with fitz.open(file_path) as document:
        for page_number, page in enumerate(document, start=1):
            text = _clean_extracted_text(page.get_text())
            if text:
                pages.append(
                    f"{_PAGE_MARKER.format(page_number=page_number)}\n\n{text}"
                )
    return "\n\n".join(pages).strip()


def sanitize_extracted_text(text: str) -> str:
    """Compatibility wrapper used by tests and callers."""
    return sanitize_text(text)


def _is_reasonable_pdf_text(raw_text: str, sanitized_text: str) -> bool:
    """Compatibility quality check for a single page."""
    from app.services.pdf_quality import analyze_text

    quality = analyze_text(raw_text, page_number=1)
    return quality.usable and bool(sanitized_text)


def _clean_extracted_text(text: str) -> str:
    cleaned = sanitize_text(text).replace("\u00ad", "")
    cleaned = _WHITESPACE_LINES_RE.sub("\n", cleaned)
    cleaned = _EXCESS_NEWLINES_RE.sub("\n\n", cleaned)
    return cleaned.strip()


def _extract_docx_text(file_path: Path) -> str:
    document = DocxDocument(file_path)
    blocks: list[str] = []
    for paragraph in document.paragraphs:
        text = _clean_extracted_text(paragraph.text)
        if text:
            blocks.append(text)

    for table in document.tables:
        rows: list[str] = []
        for row in table.rows:
            cells = [_clean_extracted_text(cell.text) for cell in row.cells]
            if any(cells):
                rows.append(" | ".join(cells))
        if rows:
            blocks.append("\n".join(rows))

    return "\n\n".join(blocks).strip()


def _extract_pptx_text(file_path: Path) -> str:
    presentation = Presentation(file_path)
    slides: list[str] = []
    for index, slide in enumerate(presentation.slides, start=1):
        parts = [f"<!-- slide: {index} -->", f"## Slide {index}"]
        for shape in slide.shapes:
            text = _clean_extracted_text(getattr(shape, "text", ""))
            if text:
                parts.append(text)
        if len(parts) > 2:
            slides.append("\n\n".join(parts))
    return "\n\n".join(slides).strip()