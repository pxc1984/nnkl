"""Обёртка над MinerU для OCR PDF."""

from __future__ import annotations

import contextlib
import shutil
import subprocess
from pathlib import Path
from typing import Literal

import fitz
import structlog

from app.core.pdf_validator import validate_pdf
from app.services.pdf_quality import analyze_pdf
from app.services.progress import ProgressCallback
from app.services.table_postprocessor import postprocess_markdown_tables

logger = structlog.get_logger(__name__)

LanguageChoice = Literal["ru", "en", "auto"]

_MINERU_LANG_MAP: dict[LanguageChoice, str] = {
    "ru": "cyrillic",
    "en": "ch",
    "auto": "cyrillic",
}


class OCRService:
    """Сервис OCR на базе MinerU CLI."""

    def __init__(
        self,
        *,
        models_dir: Path | None = None,
        use_gpu: bool = False,
        backend: str = "pipeline",
        document_timeout: float = 1800.0,
        preprocess_scans: bool = True,
        scan_dpi: int = 220,
        max_page_megapixels: float = 12.0,
    ) -> None:
        self._models_dir = models_dir
        self._use_gpu = use_gpu
        self._backend = backend if use_gpu else "pipeline"
        self._document_timeout = document_timeout
        self._preprocess_scans = preprocess_scans
        self._scan_dpi = max(72, scan_dpi)
        self._max_page_pixels = max(1.0, max_page_megapixels) * 1_000_000
        self._ready = False
        logger.info(
            "ocr_service.initializing",
            engine="mineru",
            backend=self._backend,
            use_gpu=use_gpu,
            document_timeout=document_timeout,
            preprocess_scans=preprocess_scans,
            scan_dpi=scan_dpi,
            max_page_megapixels=max_page_megapixels,
        )

    def warm_up(self) -> None:
        """Проверяет доступность MinerU CLI."""
        self._run_mineru_version_check()
        self._ready = True
        logger.info("ocr_service.warm_up_completed", engine="mineru")

    @property
    def is_ready(self) -> bool:
        return self._ready

    def convert(
        self,
        file_path: Path,
        *,
        language: LanguageChoice = "auto",
        progress_callback: ProgressCallback | None = None,
        correlation_id: str | None = None,
        results_dir: Path | None = None,
        needs_ocr: bool | None = None,
    ) -> tuple[str, Path | None]:
        """Конвертирует PDF в Markdown через MinerU."""
        validate_pdf(file_path)

        if progress_callback:
            progress_callback(5, "validating")

        if needs_ocr is None:
            needs_ocr = _detect_scan_quality(file_path)

        working_path = file_path
        if needs_ocr and self._preprocess_scans:
            working_path = self._preprocess_scan(
                file_path, correlation_id=correlation_id
            )
            if progress_callback:
                progress_callback(10, "preprocessing")

        mineru_lang = _MINERU_LANG_MAP.get(language, "ch")
        output_root = results_dir or working_path.parent / "mineru_output"
        output_root.mkdir(parents=True, exist_ok=True)

        logger.info(
            "ocr_service.conversion_started",
            path=str(working_path),
            language=language,
            mineru_lang=mineru_lang,
            backend=self._backend,
            needs_ocr=needs_ocr,
            correlation_id=correlation_id,
        )

        if progress_callback:
            progress_callback(15, "mineru")

        self._run_mineru(
            working_path,
            output_root,
            lang=mineru_lang,
            correlation_id=correlation_id,
        )

        if progress_callback:
            progress_callback(70, "reading_output")

        markdown_path = _find_markdown_output(output_root)
        # Handle potential encoding issues by trying different encodings
        markdown = _read_with_encoding_handling(markdown_path)
        markdown = postprocess_markdown_tables(markdown)

        image_dir = _collect_images(markdown_path.parent, results_dir)

        if progress_callback:
            progress_callback(100, "completed")

        logger.info(
            "ocr_service.conversion_completed",
            path=str(file_path),
            markdown_path=str(markdown_path),
            correlation_id=correlation_id,
        )

        if working_path != file_path:
            with contextlib.suppress(OSError):
                shutil.rmtree(working_path.parent, ignore_errors=True)

        return markdown, image_dir

    def _preprocess_scan(
        self, source_path: Path, *, correlation_id: str | None = None
    ) -> Path:
        logger.info(
            "ocr_service.preprocess_started",
            path=str(source_path),
            dpi=self._scan_dpi,
            correlation_id=correlation_id,
        )
        temp_dir = source_path.parent / f"mineru_preprocess_{source_path.stem}"
        temp_dir.mkdir(parents=True, exist_ok=True)
        output_path = temp_dir / source_path.name

        with fitz.open(source_path) as source, fitz.open() as rasterized:
            for page_number, page in enumerate(source, start=1):
                scale = self._scan_dpi / 72.0
                expected_pixels = page.rect.width * scale * page.rect.height * scale
                if expected_pixels > self._max_page_pixels:
                    scale *= (self._max_page_pixels / expected_pixels) ** 0.5
                pixmap = page.get_pixmap(matrix=fitz.Matrix(scale, scale), alpha=False)
                target = rasterized.new_page(
                    width=page.rect.width, height=page.rect.height
                )
                target.insert_image(target.rect, stream=pixmap.tobytes("png"))
                logger.debug(
                    "ocr_service.page_preprocessed",
                    page_number=page_number,
                    width=pixmap.width,
                    height=pixmap.height,
                    correlation_id=correlation_id,
                )
            rasterized.save(output_path, garbage=4, deflate=True)

        return output_path

    def _run_mineru(
        self,
        input_path: Path,
        output_dir: Path,
        *,
        lang: str,
        correlation_id: str | None,
    ) -> None:
        command = [
            "mineru",
            "-p",
            str(input_path),
            "-o",
            str(output_dir),
            "-b",
            self._backend,
            "-l",
            lang,
        ]
        env = None
        if self._models_dir is not None:
            import os

            env = os.environ.copy()
            env["MINERU_MODELS_DIR"] = str(self._models_dir)

        logger.info(
            "ocr_service.mineru_started",
            command=command,
            correlation_id=correlation_id,
        )
        try:
            completed = subprocess.run(
                command,
                check=True,
                capture_output=True,
                text=True,
                timeout=self._document_timeout,
                env=env,
            )
        except subprocess.TimeoutExpired as exc:
            raise TimeoutError(
                f"MinerU timed out after {self._document_timeout}s"
            ) from exc
        except subprocess.CalledProcessError as exc:
            stderr = (exc.stderr or "").strip()
            stdout = (exc.stdout or "").strip()
            details = stderr or stdout or str(exc)
            raise RuntimeError(f"MinerU failed: {details}") from exc

        logger.info(
            "ocr_service.mineru_completed",
            stdout_tail=(completed.stdout or "")[-500:],
            correlation_id=correlation_id,
        )

    def _run_mineru_version_check(self) -> None:
        try:
            subprocess.run(
                ["mineru", "--version"],
                check=True,
                capture_output=True,
                text=True,
                timeout=30,
            )
        except (subprocess.CalledProcessError, FileNotFoundError) as exc:
            raise RuntimeError("MinerU CLI is not available") from exc


def _find_markdown_output(output_dir: Path) -> Path:
    candidates = sorted(output_dir.rglob("*.md"))
    if not candidates:
        raise FileNotFoundError(f"MinerU markdown output not found in {output_dir}")

    preferred = [path for path in candidates if path.name.endswith(".md")]
    if len(preferred) == 1:
        return preferred[0]

    return max(candidates, key=lambda path: path.stat().st_size)


def _read_with_encoding_handling(path: Path) -> str:
    """Read file with proper encoding handling to fix character encoding issues."""
    # Try UTF-8 first (most common)
    try:
        return path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        # If UTF-8 fails, try other common encodings
        try:
            return path.read_text(encoding="cp1251")  # Windows Cyrillic
        except UnicodeDecodeError:
            try:
                return path.read_text(encoding="latin-1")  # Fallback
            except UnicodeDecodeError:
                # Last resort: read as binary and decode with error handling
                raw_bytes = path.read_bytes()
                # Try to detect encoding and handle errors gracefully
                try:
                    return raw_bytes.decode("utf-8", errors="replace")
                except UnicodeDecodeError:
                    return raw_bytes.decode("cp1251", errors="replace")


def _collect_images(source_dir: Path, results_dir: Path | None) -> Path | None:
    image_dirs = [path for path in source_dir.rglob("images") if path.is_dir()]
    if not image_dirs or results_dir is None:
        return None

    target_dir = results_dir / "images"
    if target_dir.exists():
        shutil.rmtree(target_dir)
    target_dir.mkdir(parents=True, exist_ok=True)

    copied = False
    for image_dir in image_dirs:
        for image_path in image_dir.iterdir():
            if image_path.is_file():
                shutil.copy2(image_path, target_dir / image_path.name)
                copied = True

    return target_dir if copied else None


def _detect_scan_quality(pdf_path: Path) -> bool:
    try:
        return not analyze_pdf(pdf_path).supports_native_extraction()
    except Exception:  # noqa: BLE001
        return True


_ocr_service_instance: OCRService | None = None


def get_ocr_service(
    *,
    artifacts_path: Path | None = None,
    use_gpu: bool = False,
    do_formula_enrichment: bool = False,
    document_timeout: float = 1800.0,
    backend: str = "hybrid-auto-engine",
    preprocess_scans: bool = True,
    scan_dpi: int = 220,
    max_page_megapixels: float = 12.0,
) -> OCRService:
    """Возвращает синглтон OCRService для воркера."""
    del do_formula_enrichment
    global _ocr_service_instance  # noqa: PLW0603
    if _ocr_service_instance is None:
        _ocr_service_instance = OCRService(
            models_dir=artifacts_path,
            use_gpu=use_gpu,
            backend=backend,
            document_timeout=document_timeout,
            preprocess_scans=preprocess_scans,
            scan_dpi=scan_dpi,
            max_page_megapixels=max_page_megapixels,
        )
    return _ocr_service_instance


def reset_ocr_service() -> None:
    """Сброс синглтона (для тестов)."""
    global _ocr_service_instance  # noqa: PLW0603
    _ocr_service_instance = None
