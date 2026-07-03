"""Клиент для Yandex AI Studio API."""

import base64
import json
import logging
import time
from typing import Any

import httpx
from tenacity import (
    retry,
    retry_if_exception_type,
    stop_after_attempt,
    wait_exponential,
)

from config import Settings

logger = logging.getLogger(__name__)


class LLMError(Exception):
    """Ошибка при обращении к LLM API."""


class YandexLLMClient:
    """Клиент для работы с Yandex AI Studio API."""

    def __init__(self, settings: Settings):
        self.settings = settings
        self.api_key = settings.yandex_api_key
        self.folder_id = settings.yandex_folder_id
        self.base_url = settings.yandex_base_url

        if not self.api_key:
            raise LLMError("YANDEX_API_KEY не задан в .env")

        self.headers = {
            "Authorization": f"Api-Key {self.api_key}",
            "x-folder-id": self.folder_id,
            "Content-Type": "application/json",
        }

    @retry(
        retry=retry_if_exception_type((httpx.HTTPError, httpx.TimeoutException)),
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=2, max=30),
        reraise=True,
    )
    def analyze_page(
        self,
        text: str,
        image_base64: str | None = None,
        system_prompt: str | None = None,
        previous_context: str = "",
    ) -> str:
        """
        Отправляет страницу на анализ в LLM.

        Args:
            text: Извлеченный текст страницы
            image_base64: Изображение страницы в base64 (опционально)
            system_prompt: Системный промпт
            previous_context: Контекст предыдущих страниц

        Returns:
            Markdown-структурированный текст
        """
        messages = self._build_messages(
            text=text,
            image_base64=image_base64,
            previous_context=previous_context,
        )

        payload: dict[str, Any] = {
            "modelUri": f"gpt://{self.folder_id}/{self.settings.llm_model}",
            "completionOptions": {
                "stream": False,
                "temperature": self.settings.temperature,
                "maxTokens": str(self.settings.max_tokens),
            },
            "messages": messages,
        }

        start_time = time.time()

        try:
            response = httpx.post(
                self.base_url,
                headers=self.headers,
                json=payload,
                timeout=300.0,
            )
            response.raise_for_status()
        except httpx.HTTPStatusError as e:
            logger.error(f"HTTP ошибка {e.response.status_code}: {e.response.text}")
            raise LLMError(f"API вернул ошибку {e.response.status_code}: {e.response.text}") from e

        elapsed = time.time() - start_time
        logger.debug(f"Ответ получен за {elapsed:.1f}s")

        result = response.json()
        return self._extract_response(result)

    def _build_messages(
        self,
        text: str,
        image_base64: str | None = None,
        previous_context: str = "",
    ) -> list[dict[str, Any]]:
        """Формирует сообщения для API Yandex."""
        messages = []

        # Системное сообщение
        messages.append({
            "role": "system",
            "text": self._get_system_prompt(),
        })

        # Контекст предыдущих страниц (если есть)
        if previous_context:
            messages.append({
                "role": "user",
                "text": (
                    f"<previous_context>\n"
                    f"Это структурированный markdown предыдущих страниц документа. "
                    f"Используй его как контекст для понимания текущей страницы:\n\n"
                    f"{previous_context}\n"
                    f"</previous_context>"
                ),
            })
            messages.append({"role": "assistant", "text": "Понял, учту контекст."})

        # Основной запрос с текстом и изображением
        parts: list[dict[str, Any]] = []

        # Текстовое описание
        text_content = (
            f"Проанализируй эту страницу PDF-документа и верни структурированный markdown.\n\n"
            f"<raw_text>\n{text}\n</raw_text>"
        )
        parts.append({"type": "text", "text": text_content})

        # Изображение страницы (если доступно)
        if image_base64:
            parts.append({
                "type": "image",
                "source": {
                    "type": "base64",
                    "data": image_base64,
                    "media_type": "image/png",
                },
            })

        messages.append({"role": "user", "content": parts})

        return messages

    def _get_system_prompt(self) -> str:
        """Возвращает системный промпт для структуризации."""
        return (
            "Ты — эксперт по структуризации документов с неявной структурой. "
            "Твоя задача — преобразовать сырой текст из PDF в чистый, "
            "хорошо структурированный Markdown.\n\n"
            "## Правила:\n"
            "1. Верни ТОЛЬКО markdown без каких-либо пояснений, комментариев "
            "или обрамляющего текста\n"
            "2. Определи иерархию заголовков (H1, H2, H3) по смыслу\n"
            "3. Преобразуй таблицы в markdown-таблицы с правильным выравниванием\n"
            "4. Списки оформи как markdown-списки (- или 1.)\n"
            "5. Удаля артефакты вёрстки (повторяющиеся символы, разделители страниц, "
            "некорректные переносы строк внутри слов)\n"
            "6. Сохрани все фактические данные: имена, даты, числа, термины\n"
            "7. Если текст разбит на колонки — восстанови логический порядок чтения\n"
            "8. Разрывы строк внутри абзацев удали, объединив в连贯ные предложения\n"
            "9. Не добавляй содержимого, которого нет в исходном тексте\n"
            "10. Не используй HTML-теги в выводе\n\n"
            "## Формат вывода:\n"
            "- Чистый markdown без блоков ```markdown\n"
            "- Корректные markdown-таблицы с | разделителями\n"
            "- Иерархия заголовков через #\n"
            "- Списки через - или 1. 2. 3.\n"
            "- **жирный** для выделения важных терминов\n"
        )

    def _extract_response(self, result: dict[str, Any]) -> str:
        """Извлекает текст ответа из результата API."""
        try:
            alternatives = result["result"]["alternatives"]
            if not alternatives:
                raise LLMError("Пустой ответ от API")

            text = alternatives[0]["message"]["text"]

            # Очистка от возможного markdown-обрамления
            text = text.strip()
            if text.startswith("```markdown"):
                text = text[11:]
            if text.startswith("```"):
                text = text[3:]
            if text.endswith("```"):
                text = text[:-3]

            return text.strip()

        except (KeyError, IndexError) as e:
            logger.error(f"Неожиданная структура ответа: {result}")
            raise LLMError(f"Не удалось извлечь текст ответа: {e}") from e
