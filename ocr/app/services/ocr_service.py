"""Обёртка над Docling для OCR материаловедческих PDF."""

from __future__ import annotations

import contextlib
import tempfile
from pathlib import Path
from typing import Literal

import fitz
import structlog
from docling.datamodel.base_models import InputFormat
from docling.datamodel.pipeline_options import (
    AcceleratorOptions,
    PdfPipelineOptions,
    TableFormerMode,
    TableStructureOptions,
)
from docling.document_converter import DocumentConverter, PdfFormatOption

from app.core.pdf_validator import validate_pdf
from app.services.latex_exporter import export_to_latex
from app.services.progress import ProgressCallback, run_with_progress
from app.services.table_postprocessor import postprocess_document_tables

logger = structlog.get_logger(__name__)

LanguageChoice = Literal["ru", "en", "auto"]
OutputFormatChoice = Literal["latex", "markdown"]

_OCR_LANG_MAP: dict[LanguageChoice, list[str]] = {
    "ru": ["ru"],
    "en": ["en"],
    "auto": ["ru", "en"],
}


class OCRService:
    """Сервис OCR с единоразовой загрузкой модели Docling."""

    def __init__(
        self,
        *,
        artifacts_path: Path | None = None,
        use_gpu: bool = True,
        do_formula_enrichment: bool = True,
        document_timeout: float = 30.0,
    ) -> None:
        self._artifacts_path = artifacts_path
        self._use_gpu = use_gpu
        self._do_formula_enrichment = do_formula_enrichment
        self._document_timeout = document_timeout
        self._converter: DocumentConverter | None = None
        logger.info(
            "ocr_service.initializing",
            use_gpu=use_gpu,
            do_formula_enrichment=do_formula_enrichment,
            document_timeout=document_timeout,
        )

    @property
    def converter(self) -> DocumentConverter:
        if self._converter is None:
            self._converter = self._build_converter()
        return self._converter

    def warm_up(self) -> None:
        """Предзагрузка pipeline Docling."""
        _ = self.converter
        logger.info("ocr_service.warm_up_completed")

    @property
    def is_ready(self) -> bool:
        return self._converter is not None

    def _build_converter(self) -> DocumentConverter:
        """Создаёт DocumentConverter по официальному API Docling."""
        pipeline_options = PdfPipelineOptions(
            do_ocr=True,
            do_table_structure=True,
            do_formula_enrichment=False,  # жёстко выкл. — VLM для формул крайне медленный на CPU
            do_code_enrichment=False,
            document_timeout=float(self._document_timeout),
        )
        pipeline_options.table_structure_options = TableStructureOptions(
            do_cell_matching=True,
            mode=TableFormerMode.ACCURATE,
        )
        pipeline_options.ocr_options.lang = _OCR_LANG_MAP["auto"]
        pipeline_options.accelerator_options = AcceleratorOptions(
            device="auto" if self._use_gpu else "cpu",
        )

        if self._artifacts_path is not None:
            pipeline_options.artifacts_path = str(self._artifacts_path)

        converter = DocumentConverter(
            format_options={
                InputFormat.PDF: PdfFormatOption(pipeline_options=pipeline_options),
            },
        )
        logger.info(
            "ocr_service.pipeline_initializing",
            do_formula_enrichment=False,
            do_code_enrichment=False,
        )
        converter.initialize_pipeline(InputFormat.PDF)
        logger.info("ocr_service.converter_ready", engine="docling")
        return converter

    def preprocess_pdf(
        self, source_path: Path, *, correlation_id: str | None = None
    ) -> Path:
        """Предобработка сканов через PyMuPDF."""
        return self.preprocess_pdf_for_ocr(
            source_path, needs_enhancement=True, correlation_id=correlation_id
        )

    def preprocess_pdf_for_ocr(
        self,
        source_path: Path,
        *,
        needs_enhancement: bool,
        correlation_id: str | None = None,
    ) -> Path:
        """Предобработка PDF только когда OCR действительно нужен."""
        if not needs_enhancement:
            logger.info("ocr_service.preprocess_skipped", path=str(source_path))
            return source_path

        logger.info(
            "ocr_service.preprocess_started",
            path=str(source_path),
            correlation_id=correlation_id,
        )

        temp_dir = Path(tempfile.mkdtemp(prefix="pdf_preprocess_"))
        output_path = temp_dir / f"enhanced_{source_path.name}"

        with fitz.open(source_path) as doc:
            for page in doc:
                page.get_pixmap(dpi=200, alpha=False)
                page.clean_contents()
            doc.save(output_path, garbage=4, deflate=True)

        return output_path

    def convert(
        self,
        file_path: Path,
        *,
        output_format: OutputFormatChoice = "latex",
        language: LanguageChoice = "auto",
        progress_callback: ProgressCallback | None = None,
        correlation_id: str | None = None,
        results_dir: Path | None = None,
        needs_ocr: bool | None = None,
    ) -> tuple[str, Path | None]:
        """Конвертирует PDF в LaTeX или Markdown через Docling."""
        validate_pdf(file_path)

        if progress_callback:
            progress_callback(5, "validating")

        self._apply_language(language)
        if needs_ocr is None:
            needs_ocr = _detect_scan_quality(file_path)
        # Pipeline уже может быть загружен в warm_up; применяем OCR до convert
        _ = self.converter
        self._apply_ocr(needs_ocr)

        preprocessed_path: Path | None = None

        try:
            preprocessed_path = self.preprocess_pdf_for_ocr(
                file_path,
                needs_enhancement=needs_ocr,
                correlation_id=correlation_id,
            )
            working_path = preprocessed_path

            if progress_callback:
                progress_callback(10, "preprocessing")

            logger.info(
                "ocr_service.conversion_started",
                path=str(working_path),
                output_format=output_format,
                language=language,
                needs_ocr=needs_ocr,
                correlation_id=correlation_id,
            )

            if progress_callback:
                progress_callback(15, "docling")

            # Docling не отдаёт прогресс внутри convert() — heartbeat 15→65
            result = run_with_progress(
                progress_callback,
                lambda: self.converter.convert(str(working_path)),
                start=15,
                end=65,
                stage="docling",
            )
            document = result.document

            if progress_callback:
                progress_callback(70, "postprocessing_tables")

            postprocess_document_tables(document)

            if progress_callback:
                progress_callback(85, "exporting")

            image_dir: Path | None = None
            if output_format == "latex":
                if results_dir is not None:
                    image_dir = results_dir / "images"
                content = export_to_latex(document, image_output_dir=image_dir)
            else:
                content = document.export_to_markdown()

            if progress_callback:
                progress_callback(100, "completed")

            logger.info(
                "ocr_service.conversion_completed",
                path=str(file_path),
                output_format=output_format,
                correlation_id=correlation_id,
            )
            return content, image_dir

        finally:
            if preprocessed_path is not None and preprocessed_path != file_path:
                with contextlib.suppress(OSError):
                    preprocessed_path.unlink(missing_ok=True)
                    if preprocessed_path.parent.exists():
                        preprocessed_path.parent.rmdir()

    def _apply_language(self, language: LanguageChoice) -> None:
        """Обновляет языки OCR в pipeline Docling перед конвертацией."""
        langs = _OCR_LANG_MAP.get(language, ["ru", "en"])
        if self._converter is not None:
            pdf_option = self._converter.format_to_options.get(InputFormat.PDF)
            if pdf_option and hasattr(pdf_option, "pipeline_options"):
                pdf_option.pipeline_options.ocr_options.lang = langs

    def _apply_ocr(self, do_ocr: bool) -> None:
        """Включает OCR только для сканов — ускоряет PDF с текстовым слоем."""
        if self._converter is not None:
            pdf_option = self._converter.format_to_options.get(InputFormat.PDF)
            if pdf_option and hasattr(pdf_option, "pipeline_options"):
                pdf_option.pipeline_options.do_ocr = do_ocr
                logger.info("ocr_service.ocr_toggled", do_ocr=do_ocr)


def _detect_scan_quality(pdf_path: Path) -> bool:
    """Эвристика: мало текста на первой странице — вероятно скан."""
    try:
        with fitz.open(pdf_path) as doc:
            if doc.page_count == 0:
                return True
            text = doc[0].get_text().strip()
            return len(text) < 50
    except Exception:  # noqa: BLE001
        return True


_ocr_service_instance: OCRService | None = None


def get_ocr_service(
    *,
    artifacts_path: Path | None = None,
    use_gpu: bool = True,
    do_formula_enrichment: bool = False,
    document_timeout: float = 30.0,
) -> OCRService:
    """Возвращает синглтон OCRService для воркера."""
    global _ocr_service_instance  # noqa: PLW0603
    if _ocr_service_instance is None:
        _ocr_service_instance = OCRService(
            artifacts_path=artifacts_path,
            use_gpu=use_gpu,
            do_formula_enrichment=do_formula_enrichment,
            document_timeout=document_timeout,
        )
    return _ocr_service_instance


def reset_ocr_service() -> None:
    """Сброс синглтона (для тестов)."""
    global _ocr_service_instance  # noqa: PLW0603
    _ocr_service_instance = None
