"""Конфигурация приложения через pydantic-settings."""

from functools import lru_cache
from pathlib import Path

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Настройки API, воркера и безопасности путей."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    # Общие
    app_name: str = "PDF OCR Service"
    debug: bool = False
    log_level: str = "INFO"

    # Безопасность путей
    allowed_base_path: Path = Field(
        default=Path("/data/pdfs"),
        description="Корневая директория, внутри которой разрешены PDF-файлы",
    )
    max_file_size_mb: int = Field(default=200, ge=1, le=2000)

    # Хранение результатов
    results_dir: Path = Field(default=Path("/app/results"))
    temp_dir: Path = Field(default=Path("/app/tmp"))

    # Redis / Celery
    redis_url: str = "redis://redis:6379/0"
    celery_broker_url: str = "redis://redis:6379/0"
    celery_result_backend: str = "redis://redis:6379/1"

    # Celery task limits (секунды) — большие документы на CPU требуют больше времени
    task_soft_time_limit: int = 1800
    task_time_limit: int = 2100

    # Docling
    docling_artifacts_path: Path | None = None
    docling_use_gpu: bool = True
    # formula_enrichment тянет VLM и на CPU часто «висит» — по умолчанию выкл.
    docling_do_formula_enrichment: bool = True
    # Предзагрузка моделей при старте воркера (блокирует приём задач на несколько минут)
    docling_preload_models: bool = False

    # API
    api_prefix: str = "/api/v1"

    @property
    def max_file_size_bytes(self) -> int:
        return self.max_file_size_mb * 1024 * 1024


@lru_cache
def get_settings() -> Settings:
    """Кэшированный синглтон настроек."""
    return Settings()
