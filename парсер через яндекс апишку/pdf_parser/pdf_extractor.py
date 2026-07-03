"""Модуль извлечения контента из PDF-документов."""

import base64
import io
import logging
from pathlib import Path
from typing import NamedTuple

import fitz  # pymupdf
from PIL import Image

logger = logging.getLogger(__name__)


class PageContent(NamedTuple):
    """Контент одной страницы PDF."""

    page_number: int
    text: str
    image_base64: str | None = None
    width: int = 0
    height: int = 0


class PDFExtractor:
    """Извлекает текст и изображения из PDF-файлов."""

    def __init__(self, dpi: int = 200, max_image_size: int = 2048):
        self.dpi = dpi
        self.max_image_size = max_image_size
        self.zoom = dpi / 72  # 72 DPI — базовое разрешение PDF

    def extract(self, pdf_path: str | Path) -> list[PageContent]:
        """
        Извлекает контент из всех страниц PDF.

        Args:
            pdf_path: Путь к PDF-файлу

        Returns:
            Список PageContent для каждой страницы
        """
        pdf_path = Path(pdf_path)
        if not pdf_path.exists():
            raise FileNotFoundError(f"PDF файл не найден: {pdf_path}")

        pages: list[PageContent] = []
        doc = fitz.open(str(pdf_path))

        logger.info(f"Обработка PDF: {pdf_path.name}, страниц: {len(doc)}")

        for page_num in range(len(doc)):
            page = doc[page_num]

            # Извлечение сырого текста
            text = page.get_text()

            # Рендер страницы в изображение
            mat = fitz.Matrix(self.zoom, self.zoom)
            pix = page.get_pixmap(matrix=mat)

            img_data = pix.tobytes("png")
            img = Image.open(io.BytesIO(img_data))

            # Масштабирование при необходимости
            img = self._resize_if_needed(img)

            # Конвертация в base64
            buffered = io.BytesIO()
            img.save(buffered, format="PNG")
            img_base64 = base64.b64encode(buffered.getvalue()).decode("utf-8")

            pages.append(
                PageContent(
                    page_number=page_num + 1,
                    text=text.strip(),
                    image_base64=img_base64,
                    width=img.width,
                    height=img.height,
                )
            )

            logger.debug(f"Страница {page_num + 1}: текст={len(text)} симв, "
                        f"изображение={img.width}x{img.height}")

        doc.close()
        logger.info(f"Извлечено {len(pages)} страниц")

        return pages

    def extract_text_only(self, pdf_path: str | Path) -> list[PageContent]:
        """Извлекает только текст без изображений (быстрее, дешевле)."""
        pdf_path = Path(pdf_path)
        if not pdf_path.exists():
            raise FileNotFoundError(f"PDF файл не найден: {pdf_path}")

        pages: list[PageContent] = []
        doc = fitz.open(str(pdf_path))

        for page_num in range(len(doc)):
            page = doc[page_num]
            text = page.get_text()
            pages.append(
                PageContent(
                    page_number=page_num + 1,
                    text=text.strip(),
                    image_base64=None,
                    width=int(page.rect.width),
                    height=int(page.rect.height),
                )
            )

        doc.close()
        return pages

    def _resize_if_needed(self, img: Image.Image) -> Image.Image:
        """Уменьшает изображение если оно превышает max_image_size."""
        if max(img.width, img.height) > self.max_image_size:
            ratio = self.max_image_size / max(img.width, img.height)
            new_size = (int(img.width * ratio), int(img.height * ratio))
            img = img.resize(new_size, Image.LANCZOS)
            logger.debug(f"Изображение масштабировано до {new_size}")
        return img
