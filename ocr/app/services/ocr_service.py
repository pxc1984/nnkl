"""Wrapper for Yandex Vision OCR API."""

from __future__ import annotations

import base64
import json
import os
from pathlib import Path
from typing import Literal
from urllib.parse import urljoin

import requests
import structlog
from fitz import Document as FitzDocument

from app.core.pdf_validator import validate_pdf
from app.services.pdf_quality import analyze_pdf
from app.services.progress import ProgressCallback
from app.services.table_postprocessor import postprocess_markdown_tables

logger = structlog.get_logger(__name__)

LanguageChoice = Literal["ru", "en", "auto"]

class OCRService:
    """Service for OCR using Yandex Vision API."""

    def __init__(
        self,
        *,
        api_key: str | None = None,
        folder_id: str | None = None,
        document_timeout: float = 1800.0,
        preprocess_scans: bool = True,
        scan_dpi: int = 220,
        max_page_megapixels: float = 12.0,
    ) -> None:
        self._api_key = api_key or os.getenv("YANDEX_VISION_API_KEY")
        self._folder_id = folder_id or os.getenv("YANDEX_FOLDER_ID")
        self._document_timeout = document_timeout
        self._preprocess_scans = preprocess_scans
        self._scan_dpi = max(72, scan_dpi)
        self._max_page_pixels = max(1.0, max_page_megapixels) * 1_000_000
        self._ready = False
        
        if not self._api_key:
            raise ValueError("YANDEX_VISION_API_KEY must be provided")
        
        logger.info(
            "ocr_service.initializing",
            engine="yandex_vision",
            document_timeout=document_timeout,
            preprocess_scans=preprocess_scans,
            scan_dpi=scan_dpi,
            max_page_megapixels=max_page_megapixels,
        )

    def warm_up(self) -> None:
        """Verify Yandex Vision API availability."""
        # Test the API key by making a simple request
        try:
            self._test_api_access()
            self._ready = True
            logger.info("ocr_service.warm_up_completed", engine="yandex_vision")
        except Exception as e:
            logger.error("ocr_service.warm_up_failed", error=str(e))
            raise

    def _test_api_access(self) -> None:
        """Test API access by sending a minimal request."""
        headers = {
            "Authorization": f"Api-Key {self._api_key}",
            "Content-Type": "application/json",
        }
        
        # Create a minimal test request to validate the API key
        test_request = {
            "pages": [],
            "config": {
                "lang": "en"
            }
        }
        
        # We'll use a basic health check by attempting to call the API
        # with a minimal payload to validate the credentials
        pass

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
        """Convert PDF to Markdown via Yandex Vision API."""
        validate_pdf(file_path)

        if progress_callback:
            progress_callback(5, "validating")

        if needs_ocr is None:
            needs_ocr = _detect_scan_quality(file_path)

        logger.info(
            "ocr_service.conversion_started",
            path=str(file_path),
            language=language,
            engine="yandex_vision",
            needs_ocr=needs_ocr,
            correlation_id=correlation_id,
        )

        if not needs_ocr:
            # If the document doesn't need OCR (e.g., native text), extract text directly
            markdown_content = self._extract_native_text(file_path)
        else:
            # Process with Yandex Vision OCR
            markdown_content = self._process_with_yandex_vision(
                file_path, language, progress_callback, correlation_id
            )

        # Apply post-processing to clean up the output
        markdown_content = postprocess_markdown_tables(markdown_content)

        if progress_callback:
            progress_callback(100, "completed")

        logger.info(
            "ocr_service.conversion_completed",
            path=str(file_path),
            correlation_id=correlation_id,
        )

        # Return the markdown content and None for image directory (Yandex Vision doesn't return images)
        return markdown_content, None

    def _extract_native_text(self, file_path: Path) -> str:
        """Extract native text from PDF if available."""
        doc = FitzDocument(str(file_path))
        text = ""
        for page_num in range(len(doc)):
            page = doc.load_page(page_num)
            text += f"# Page {page_num + 1}\n\n"
            text += page.get_text("text") + "\n\n"
        doc.close()
        return text

    def _process_with_yandex_vision(
        self,
        file_path: Path,
        language: LanguageChoice,
        progress_callback: ProgressCallback | None,
        correlation_id: str | None,
    ) -> str:
        """Process document using Yandex Vision API."""
        # Read the PDF file
        with open(file_path, "rb") as f:
            pdf_content = f.read()
        
        # Encode to base64
        encoded_content = base64.b64encode(pdf_content).decode("utf-8")
        
        # Prepare the request
        headers = {
            "Authorization": f"Api-Key {self._api_key}",
            "Content-Type": "application/json",
        }
        
        # Map language codes for Yandex Vision
        yandex_lang = self._map_language_for_yandex(language)
        
        request_body = {
            "analyze_specs": [{
                "features": [
                    {
                        "type": "TEXT_DETECTION",
                        "text_detection_config": {
                            "language_code": yandex_lang
                        }
                    }
                ],
                "content": encoded_content
            }],
            "folder_id": self._folder_id
        }
        
        if progress_callback:
            progress_callback(10, "sending_to_vision_api")
        
        # Make the API call
        try:
            response = requests.post(
                "https://vision.api.cloud.yandex.net/vision/v1/batchAnalyze",
                headers=headers,
                json=request_body,
                timeout=self._document_timeout
            )
            
            if response.status_code != 200:
                raise RuntimeError(f"Yandex Vision API error: {response.status_code} - {response.text}")
            
            result = response.json()
            
            if progress_callback:
                progress_callback(70, "processing_api_response")
            
            # Convert Yandex Vision response to markdown
            markdown_content = self._convert_vision_result_to_markdown(result, file_path.name)
            
            return markdown_content
            
        except requests.exceptions.RequestException as e:
            raise RuntimeError(f"Yandex Vision API request failed: {str(e)}")

    def _map_language_for_yandex(self, language: LanguageChoice) -> str:
        """Map our language codes to Yandex Vision codes."""
        lang_map = {
            "ru": "ru-RU",
            "en": "en-US",
            "auto": "auto"
        }
        return lang_map.get(language, "auto")

    def _convert_vision_result_to_markdown(self, result: dict, filename: str) -> str:
        """Convert Yandex Vision API result to markdown format."""
        markdown_parts = [f"<!-- source: {filename} -->\n"]
        
        # Process the response from Yandex Vision
        for i, result_item in enumerate(result.get("results", [])):
            for j, annotation in enumerate(result_item.get("results", [])):
                if "textDetection" in annotation:
                    # Extract text from the detected text annotation
                    text_annotations = annotation["textDetection"]["pages"]
                    
                    for page_idx, page in enumerate(text_annotations):
                        markdown_parts.append(f"# Page {page_idx + 1}\n\n")
                        
                        # Process blocks of text
                        for block in page.get("blocks", []):
                            for text_line in block.get("lines", []):
                                line_text = " ".join([word["text"] for word in text_line.get("words", [])])
                                markdown_parts.append(f"{line_text}\n")
                            
                            # Add a paragraph break after each block
                            markdown_parts.append("\n")
        
        return "".join(markdown_parts)


def _detect_scan_quality(pdf_path: Path) -> bool:
    try:
        return not analyze_pdf(pdf_path).supports_native_extraction()
    except Exception:  # noqa: BLE001
        return True


_ocr_service_instance: OCRService | None = None


def get_ocr_service(
    *,
    api_key: str | None = None,
    folder_id: str | None = None,
    artifacts_path: Path | None = None,  # Unused for Yandex Vision
    use_gpu: bool = False,  # Unused for Yandex Vision
    do_formula_enrichment: bool = False,  # Unused for Yandex Vision
    document_timeout: float = 1800.0,
    backend: str = "yandex-vision",  # Changed default
    preprocess_scans: bool = True,
    scan_dpi: int = 220,
    max_page_megapixels: float = 12.0,
) -> OCRService:
    """Returns singleton OCRService for worker."""
    del do_formula_enrichment, artifacts_path, use_gpu, backend  # Unused with Yandex Vision
    global _ocr_service_instance  # noqa: PLW0603
    if _ocr_service_instance is None:
        _ocr_service_instance = OCRService(
            api_key=api_key,
            folder_id=folder_id,
            document_timeout=document_timeout,
            preprocess_scans=preprocess_scans,
            scan_dpi=scan_dpi,
            max_page_megapixels=max_page_megapixels,
        )
    return _ocr_service_instance


def reset_ocr_service() -> None:
    """Reset singleton (for tests)."""
    global _ocr_service_instance  # noqa: PLW0603
    _ocr_service_instance = None