"""Fast PDF text-layer quality analysis used to select native extraction or OCR."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

import fitz

_ALLOWED_CONTROL_CHARS = {"\n", "\r", "\t"}


@dataclass(frozen=True, slots=True)
class PageQuality:
    page_number: int
    text_length: int
    alpha_ratio: float
    replacement_ratio: float
    usable: bool


@dataclass(frozen=True, slots=True)
class PDFQualityReport:
    page_count: int
    pages: tuple[PageQuality, ...]

    @property
    def usable_pages(self) -> int:
        return sum(page.usable for page in self.pages)

    @property
    def usable_ratio(self) -> float:
        return self.usable_pages / len(self.pages) if self.pages else 0.0

    def supports_native_extraction(self, *, minimum_usable_ratio: float = 0.95) -> bool:
        return bool(self.pages) and self.usable_ratio >= minimum_usable_ratio


def sanitize_text(text: str) -> str:
    if not text:
        return ""
    return "".join(
        character
        for character in text
        if character >= " " or character in _ALLOWED_CONTROL_CHARS
    )


def analyze_text(
    text: str,
    *,
    page_number: int,
    minimum_characters: int = 40,
    minimum_alpha_ratio: float = 0.18,
    maximum_replacement_ratio: float = 0.02,
) -> PageQuality:
    sanitized = sanitize_text(text).strip()
    visible = [character for character in sanitized if not character.isspace()]
    alpha_ratio = (
        sum(character.isalpha() for character in visible) / len(visible)
        if visible
        else 0.0
    )
    replacement_ratio = (
        sum(character in {"\ufffd", "\u00b7"} for character in visible) / len(visible)
        if visible
        else 1.0
    )
    usable = (
        len(sanitized) >= minimum_characters
        and alpha_ratio >= minimum_alpha_ratio
        and replacement_ratio <= maximum_replacement_ratio
    )
    return PageQuality(
        page_number=page_number,
        text_length=len(sanitized),
        alpha_ratio=alpha_ratio,
        replacement_ratio=replacement_ratio,
        usable=usable,
    )


def analyze_pdf(
    pdf_path: Path,
    *,
    minimum_characters: int = 40,
    minimum_alpha_ratio: float = 0.18,
    maximum_replacement_ratio: float = 0.02,
) -> PDFQualityReport:
    pages: list[PageQuality] = []
    with fitz.open(pdf_path) as document:
        for index, page in enumerate(document, start=1):
            pages.append(
                analyze_text(
                    page.get_text(),
                    page_number=index,
                    minimum_characters=minimum_characters,
                    minimum_alpha_ratio=minimum_alpha_ratio,
                    maximum_replacement_ratio=maximum_replacement_ratio,
                )
            )
        return PDFQualityReport(page_count=document.page_count, pages=tuple(pages))