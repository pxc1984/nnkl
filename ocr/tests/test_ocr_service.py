"""Тесты постобработки таблиц и OCR-сервиса."""

from __future__ import annotations

from pathlib import Path

from app.services.table_postprocessor import (
    normalize_numeric_text,
    normalize_unit_text,
    postprocess_markdown_tables,
    postprocess_table_cell,
)
from app.use_cases.document_extractor import (
    has_native_pdf_text,
    sanitize_extracted_text,
)


class TestTablePostprocessor:
    def test_normalize_units(self) -> None:
        assert "МПа" in normalize_unit_text("σ = 450 Мпа")

    def test_normalize_numeric_thousands(self) -> None:
        assert normalize_numeric_text("1 234,5") == "1234.5"

    def test_postprocess_cell(self) -> None:
        result = postprocess_table_cell("  450  Мпа  ")
        assert "МПа" in result


class TestMarkdownPostprocessor:
    def test_postprocess_markdown_tables(self) -> None:
        markdown = "<table><tr><td>450 Мпа</td></tr></table>"
        result = postprocess_markdown_tables(markdown)
        assert "МПа" in result


class TestScanDetection:
    def test_detect_scan_low_text(self, sample_pdf: Path) -> None:
        from app.services.ocr_service import _detect_scan_quality

        assert _detect_scan_quality(sample_pdf) is False

    def test_has_native_pdf_text(self, sample_pdf: Path) -> None:
        assert has_native_pdf_text(sample_pdf) is True

    def test_sanitize_extracted_text_removes_nul_and_controls(self) -> None:
        assert sanitize_extracted_text("abc\x00def\x1bghi\n\tjkl") == "abcdefghi\n\tjkl"


class TestPDFQualityRouting:
    def test_mixed_pdf_requires_ocr(self, tmp_path: Path) -> None:
        import fitz

        from app.services.pdf_quality import analyze_pdf
        from app.use_cases.document_extractor import should_use_native_pdf_text

        path = tmp_path / "mixed.pdf"
        with fitz.open() as document:
            text_page = document.new_page()
            text_page.insert_text((72, 72), "Reliable native text " * 10)
            document.new_page()
            document.save(path)

        report = analyze_pdf(path)
        assert report.page_count == 2
        assert report.usable_pages == 1
        assert should_use_native_pdf_text(path) is False

    def test_native_pdf_contains_page_markers(self, sample_pdf: Path) -> None:
        from app.use_cases.document_extractor import extract_native_document_text

        content = extract_native_document_text(sample_pdf)
        assert content.startswith("<!-- page: 1 -->")

    def test_preprocess_scan_creates_image_only_pdf(
        self, sample_pdf: Path, tmp_path: Path
    ) -> None:
        import fitz

        from app.services.ocr_service import OCRService

        source = tmp_path / "source.pdf"
        source.write_bytes(sample_pdf.read_bytes())
        service = OCRService(
            preprocess_scans=True,
            scan_dpi=144,
            max_page_megapixels=2,
        )
        output = service._preprocess_scan(source)

        with fitz.open(output) as document:
            assert document.page_count > 0
            assert document[0].get_text().strip() == ""
            assert document[0].get_images(full=True)


class TestConservativeMarkdownCleanup:
    def test_plain_prose_keeps_decimal_commas(self) -> None:
        markdown = "Обычный текст, значение 12,5 и продолжение предложения."
        assert postprocess_markdown_tables(markdown) == markdown

    def test_markdown_table_normalizes_numeric_cells(self) -> None:
        markdown = "| Значение | Давление |\n| --- | --- |\n| 1 234,5 | 450 Мпа |"
        result = postprocess_markdown_tables(markdown)
        assert "1234.5" in result
        assert "450 МПа" in result


class TestMinerULanguageMapping:
    def test_language_mapping_matches_mineru_cli(self) -> None:
        from app.services.ocr_service import _MINERU_LANG_MAP

        assert _MINERU_LANG_MAP["ru"] == "cyrillic"
        assert _MINERU_LANG_MAP["auto"] == "cyrillic"
        assert _MINERU_LANG_MAP["en"] == "ch"
