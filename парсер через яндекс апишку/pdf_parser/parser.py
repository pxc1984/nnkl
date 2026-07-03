"""Основной модуль парсера — оркестратор."""

import logging
from pathlib import Path

from config import Settings, get_settings
from llm_client import LLMError, YandexLLMClient
from pdf_extractor import PDFExtractor

logger = logging.getLogger(__name__)


class PDFParser:
    """
    Парсер PDF-документов с неявной структурой.

    Использует комбинацию извлечения текста/изображений из PDF
    и LLM (Qwen3 235B через Yandex AI Studio) для структуризации в markdown.
    """

    def __init__(self, settings: Settings | None = None):
        self.settings = settings or get_settings()
        self.extractor = PDFExtractor(
            dpi=self.settings.dpi_resolution,
            max_image_size=self.settings.max_image_size,
        )
        self.llm = YandexLLMClient(self.settings)

    def parse(
        self,
        pdf_path: str | Path,
        use_images: bool = True,
        context_window: int = 3,
    ) -> str:
        """
        Парсит PDF-документ в структурированный markdown.

        Args:
            pdf_path: Путь к PDF-файлу
            use_images: Использовать ли изображения страниц (дороже, точнее)
            context_window: Количество предыдущих страниц для контекста

        Returns:
            Чистый markdown-документ
        """
        pdf_path = Path(pdf_path)
        logger.info(f"Начало парсинга: {pdf_path}")

        # Извлечение контента
        if use_images:
            pages = self.extractor.extract(pdf_path)
        else:
            pages = self.extractor.extract_text_only(pdf_path)

        # Обработка страниц
        markdown_parts: list[str] = []
        previous_contexts: list[str] = []

        for i, page in enumerate(pages):
            logger.info(f"Обработка страницы {page.page_number}/{len(pages)}...")

            # Формирование контекста из предыдущих страниц
            context = "\n\n".join(previous_contexts[-context_window:])

            try:
                # Отправка в LLM
                md_page = self.llm.analyze_page(
                    text=page.text,
                    image_base64=page.image_base64 if use_images else None,
                    previous_context=context if context else "",
                )

                markdown_parts.append(md_page)

                # Сохраняем часть для контекста следующих страниц
                # Берём первые 500 символов как "описание" страницы
                context_preview = md_page[:500].strip()
                if context_preview:
                    previous_contexts.append(context_preview)

                logger.debug(f"Страница {page.page_number} обработана: "
                            f"{len(md_page)} символов markdown")

            except LLMError as e:
                logger.error(f"Ошибка обработки страницы {page.page_number}: {e}")
                # В случае ошибки добавляем raw текст с пометкой
                markdown_parts.append(
                    f"\n\n<!-- Ошибка обработки страницы {page.page_number} -->\n\n"
                    f"```\n{page.text[:2000]}\n```"
                )

        # Объединение и финальная очистка
        full_markdown = "\n\n---\n\n".join(markdown_parts)
        full_markdown = self._post_process(full_markdown)

        logger.info(f"Парсинг завершён: {len(full_markdown)} символов markdown")

        return full_markdown

    def parse_batch(
        self,
        pdf_path: str | Path,
        use_images: bool = True,
    ) -> str:
        """
        Альтернативный режим: отправляет весь документ одним запросом.
        Подходит для коротких документов (< 20 страниц).
        Быстрее, но может потерять детали на длинных документах.
        """
        pdf_path = Path(pdf_path)
        pages = self.extractor.extract_text_only(pdf_path)

        # Объединяем весь текст
        all_text = "\n\n---PAGE BREAK---\n\n".join(
            f"[Страница {p.page_number}]\n{p.text}" for p in pages
        )

        logger.info(f"Batch-режим: {len(pages)} страниц, "
                    f"{len(all_text)} символов текста")

        return self.llm.analyze_page(
            text=all_text,
            image_base64=None,
            previous_context="",
        )

    def _post_process(self, markdown: str) -> str:
        """Финальная очистка markdown."""
        lines = markdown.split("\n")
        cleaned: list[str] = []

        for line in lines:
            stripped = line.strip()

            # Пропускаем пустые строки подряд (максимум 1)
            if not stripped and cleaned and not cleaned[-1].strip():
                continue

            # Убираем артефакты
            if stripped in ("---", "***", "___"):
                if cleaned and cleaned[-1].strip() == "---":
                    continue

            cleaned.append(line)

        result = "\n".join(cleaned)

        # Удаляем множественные разделители страниц
        import re
        result = re.sub(r"\n---\n\n---\n", "\n---\n", result)
        result = re.sub(r"\n{4,}", "\n\n", result)

        return result.strip()
