"""Lightweight native text extraction for office documents."""

from __future__ import annotations

from pathlib import Path

from docx import Document as DocxDocument
from pptx import Presentation


class UnsupportedDocumentTypeError(Exception):
    """Raised when the document type is not supported."""


def extract_native_document_text(file_path: Path, *, output_format: str) -> str:
    suffix = file_path.suffix.lower()
    if suffix == ".docx":
        text = _extract_docx_text(file_path)
    elif suffix == ".pptx":
        text = _extract_pptx_text(file_path)
    else:
        raise UnsupportedDocumentTypeError(f"Unsupported document type: {suffix or 'unknown'}")

    if output_format == "latex":
        return _to_latex(text)
    return _to_markdown(text)


def _extract_docx_text(file_path: Path) -> str:
    document = DocxDocument(file_path)
    lines = [paragraph.text.strip() for paragraph in document.paragraphs if paragraph.text.strip()]

    for table in document.tables:
        for row in table.rows:
            cells = [cell.text.strip() for cell in row.cells if cell.text.strip()]
            if cells:
                lines.append(" | ".join(cells))

    return "\n\n".join(lines).strip()


def _extract_pptx_text(file_path: Path) -> str:
    presentation = Presentation(file_path)
    slides: list[str] = []

    for index, slide in enumerate(presentation.slides, start=1):
        parts: list[str] = [f"Slide {index}"]
        for shape in slide.shapes:
            text = getattr(shape, "text", "")
            text = text.strip()
            if text:
                parts.append(text)
        if len(parts) > 1:
            slides.append("\n\n".join(parts))

    return "\n\n".join(slides).strip()


def _to_markdown(text: str) -> str:
    return text


def _to_latex(text: str) -> str:
    replacements = {
        "\\": r"\textbackslash{}",
        "&": r"\&",
        "%": r"\%",
        "$": r"\$",
        "#": r"\#",
        "_": r"\_",
        "{": r"\{",
        "}": r"\}",
        "~": r"\textasciitilde{}",
        "^": r"\textasciicircum{}",
    }
    escaped = "".join(replacements.get(char, char) for char in text)
    paragraphs = [part.strip() for part in escaped.split("\n\n") if part.strip()]
    return "\n\n".join(paragraphs)
