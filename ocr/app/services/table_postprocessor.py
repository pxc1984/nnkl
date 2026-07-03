"""Постобработка таблиц Docling для материаловедческих документов."""

from __future__ import annotations

import re
from typing import TYPE_CHECKING

import structlog

if TYPE_CHECKING:
    from docling_core.types.doc.document import DoclingDocument, TableItem

logger = structlog.get_logger(__name__)

# Нормализация единиц измерения (частые OCR-ошибки в сканах)
_UNIT_REPLACEMENTS: dict[str, str] = {
    "МПа": "МПа",
    "Мпа": "МПа",
    "ГПа": "ГПа",
    "Гпа": "ГПа",
    "кг/м3": "кг/м³",
    "кг/м^3": "кг/м³",
    "Вт/(м*К)": "Вт/(м·К)",
    "Вт/(м·K)": "Вт/(м·К)",
    "°С": "°C",
    "oC": "°C",
    "deg C": "°C",
}

# Разделители тысяч: "1 234,5" → "1234.5" для числовых ячеек
_THOUSANDS_PATTERN = re.compile(r"(\d{1,3}(?:\s\d{3})+)([.,]\d+)?")
_DECIMAL_COMMA = re.compile(r"^[\d\s]+,\d+$")


def normalize_unit_text(text: str) -> str:
    """Нормализует единицы измерения и пробелы в тексте ячейки."""
    normalized = text.strip()
    for wrong, correct in _UNIT_REPLACEMENTS.items():
        normalized = normalized.replace(wrong, correct)
    # Схлопываем множественные пробелы, сохраняя переносы строк внутри ячейки
    normalized = re.sub(r"[ \t]+", " ", normalized)
    return normalized


def normalize_numeric_text(text: str) -> str:
    """Нормализует числовые значения с разделителями тысяч."""
    stripped = text.strip()
    if not stripped or not re.search(r"\d", stripped):
        return text

    def _replace_thousands(match: re.Match[str]) -> str:
        integer_part = match.group(1).replace(" ", "").replace("\u00a0", "")
        fraction = match.group(2) or ""
        if fraction:
            fraction = fraction.replace(",", ".")
        return integer_part + fraction

    result = _THOUSANDS_PATTERN.sub(_replace_thousands, stripped)
    if _DECIMAL_COMMA.match(result.replace(" ", "")):
        result = result.replace(" ", "").replace(",", ".")
    return result


def postprocess_table_cell(text: str) -> str:
    """Полная постобработка текста ячейки таблицы."""
    text = normalize_unit_text(text)
    text = normalize_numeric_text(text)
    return text


def postprocess_document_tables(document: "DoclingDocument") -> "DoclingDocument":
    """
    Проходит по всем таблицам документа и нормализует текст ячеек.

    Модифицирует документ in-place — DoclingDocument мутабелен на уровне ячеек.
    """
    from docling_core.types.doc.document import RichTableCell, TableCell

    tables_processed = 0
    cells_processed = 0

    for table in document.tables:
        if table.data is None or not table.data.grid:
            continue

        for row in table.data.grid:
            for idx, cell in enumerate(row):
                if cell is None:
                    continue
                raw_text = cell.text or ""
                processed = postprocess_table_cell(raw_text)
                if processed != raw_text:
                    if isinstance(cell, RichTableCell):
                        cell.text = processed
                    elif isinstance(cell, TableCell):
                        cell.text = processed
                    cells_processed += 1
        tables_processed += 1

    logger.info(
        "table_postprocessor.completed",
        tables_processed=tables_processed,
        cells_processed=cells_processed,
    )
    return document


def estimate_column_widths(table: "TableItem") -> list[float]:
    """
    Оценивает относительную ширину колонок по максимальной длине текста.

    Используется для tabularx: сумма должна быть около \\textwidth.
    """
    if not table.data or not table.data.grid:
        return [1.0]

    num_cols = table.data.num_cols or max(len(row) for row in table.data.grid)
    max_lengths = [1] * num_cols

    for row in table.data.grid:
        for col_idx, cell in enumerate(row):
            if cell is None:
                continue
            text_len = len((cell.text or "").strip())
            if col_idx < num_cols:
                max_lengths[col_idx] = max(max_lengths[col_idx], text_len)

    total = sum(max_lengths) or 1
    return [length / total for length in max_lengths]
