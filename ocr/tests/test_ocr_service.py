"""Тесты постобработки таблиц и OCR-сервиса."""

from __future__ import annotations

from pathlib import Path

from app.services.table_postprocessor import (
    normalize_numeric_text,
    normalize_unit_text,
    postprocess_table_cell,
)
from app.use_cases.document_extractor import has_native_pdf_text, sanitize_extracted_text


class TestTablePostprocessor:
    def test_normalize_units(self) -> None:
        assert "МПа" in normalize_unit_text("σ = 450 Мпа")

    def test_normalize_numeric_thousands(self) -> None:
        assert normalize_numeric_text("1 234,5") == "1234.5"

    def test_postprocess_cell(self) -> None:
        result = postprocess_table_cell("  450  Мпа  ")
        assert "МПа" in result


class TestScanDetection:
    def test_detect_scan_low_text(self, sample_pdf: Path) -> None:
        from app.services.ocr_service import _detect_scan_quality

        assert _detect_scan_quality(sample_pdf) is False

    def test_has_native_pdf_text(self, sample_pdf: Path) -> None:
        assert has_native_pdf_text(sample_pdf) is True

    def test_sanitize_extracted_text_removes_nul_and_controls(self) -> None:
        assert sanitize_extracted_text("abc\x00def\x1bghi\n\tjkl") == "abcdefghi\n\tjkl"
