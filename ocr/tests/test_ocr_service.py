"""Тесты постобработки таблиц и OCR-сервиса."""

from __future__ import annotations

from pathlib import Path

from app.services.table_postprocessor import (
    normalize_numeric_text,
    normalize_unit_text,
    postprocess_table_cell,
)


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
