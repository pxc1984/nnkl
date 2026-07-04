"""Постобработка таблиц и текста после MinerU."""

from __future__ import annotations

import re

_UNIT_REPLACEMENTS = (
    (r"\bМпа\b", "МПа"),
    (r"\bмпа\b", "МПа"),
    (r"\bМПа\b", "МПа"),
    (r"\bкгс/мм\^?2\b", "кгс/мм²"),
    (r"\bкгс/см\^?2\b", "кгс/см²"),
)

_NUMERIC_THOUSANDS_RE = re.compile(r"(\d)\s+(\d{3})(?=\D|$)")


def normalize_unit_text(text: str) -> str:
    result = text
    for pattern, replacement in _UNIT_REPLACEMENTS:
        result = re.sub(pattern, replacement, result)
    return result


def normalize_numeric_text(text: str) -> str:
    result = text.replace(",", ".")
    while True:
        updated = _NUMERIC_THOUSANDS_RE.sub(r"\1\2", result)
        if updated == result:
            break
        result = updated
    return result


def postprocess_table_cell(text: str) -> str:
    cleaned = " ".join(text.split())
    cleaned = normalize_numeric_text(cleaned)
    return normalize_unit_text(cleaned)


def postprocess_markdown_tables(markdown: str) -> str:
    """Нормализует единицы и числа в ячейках HTML-таблиц MinerU."""
    if "<table" not in markdown:
        return postprocess_table_cell(markdown) if markdown else markdown

    def replace_cell(match: re.Match[str]) -> str:
        cell_content = match.group(1)
        return f">{postprocess_table_cell(cell_content)}<"

    return re.sub(r">([^<>]+)<", replace_cell, markdown)
