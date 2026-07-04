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
    ocr_debug: bool = False
    ocr_log_level: str = "INFO"

    postgres_host: str = "localhost"
    postgres_port: int = 5432
    postgres_user: str = "admin"
    postgres_password: str = "admin"
    postgres_db: str = "db"
    postgres_ssl_mode: str = "disable"

    @property
    def database_url(self) -> str:
        return (
            f"postgresql+psycopg://{self.postgres_user}:{self.postgres_password}"
            f"@{self.postgres_host}:{self.postgres_port}/{self.postgres_db}"
            f"?sslmode={self.postgres_ssl_mode}"
        )

    ocr_temp_dir: Path = Path("/app/tmp")

    ocr_docling_artifacts_path: Path | None = None
    ocr_docling_use_gpu: bool = False
    ocr_docling_do_formula_enrichment: bool = False
    ocr_docling_document_timeout_seconds: float = 30.0

    ocr_api_prefix: str = "/api/v1"


def get_settings() -> Settings:
    return Settings()
