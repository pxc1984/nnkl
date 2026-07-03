"""Конфигурация приложения через pydantic-settings."""

from pathlib import Path

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Настройки API и синхронного OCR-парсера."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    app_name: str = "PDF OCR Service"
    debug: bool = False
    log_level: str = "INFO"

    database_url: str = "postgresql+psycopg://postgres:postgres@postgres:5432/ocr"
    temp_dir: Path = Path("/app/tmp")

    docling_artifacts_path: Path | None = None
    docling_use_gpu: bool = False
    docling_do_formula_enrichment: bool = False

    api_prefix: str = "/api/v1"


def get_settings() -> Settings:
    return Settings()
