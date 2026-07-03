"""Конфигурация парсера PDF."""

import os
from pathlib import Path

from dotenv import load_dotenv
from pydantic import Field
from pydantic_settings import BaseSettings

load_dotenv()


class Settings(BaseSettings):
    """Настройки приложения из переменных окружения."""

    # Yandex AI Studio API
    yandex_api_key: str = Field(default="", alias="YANDEX_API_KEY")
    yandex_folder_id: str = Field(default="", alias="YANDEX_FOLDER_ID")

    # LLM параметры
    llm_model: str = Field(default="qwen3:235b", alias="LLM_MODEL")
    max_tokens: int = Field(default=32000, alias="MAX_TOKENS")
    temperature: float = Field(default=0.1, alias="TEMPERATURE")

    # API endpoints
    yandex_base_url: str = "https://llm.api.cloud.yandex.net/foundationModels/v1/completion"

    # PDF обработка
    dpi_resolution: int = 200
    max_image_size: int = 2048

    # Пути
    output_dir: Path = Path("./output")
    temp_dir: Path = Path("./temp")

    class Config:
        env_prefix = ""
        case_sensitive = False


def get_settings() -> Settings:
    """Возвращает настройки приложения."""
    return Settings()
