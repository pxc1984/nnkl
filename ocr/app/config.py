"""Configuration via pydantic-settings."""

from pathlib import Path

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """API and synchronous OCR parser settings."""

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
    database_url_override: str | None = Field(
        default=None, validation_alias="DATABASE_URL", exclude=True
    )

    @property
    def database_url(self) -> str:
        if self.database_url_override:
            return self.database_url_override
        return (
            f"postgresql+psycopg://{self.postgres_user}:{self.postgres_password}"
            f"@{self.postgres_host}:{self.postgres_port}/{self.postgres_db}"
            f"?sslmode={self.postgres_ssl_mode}"
        )

    ocr_temp_dir: Path = Path("/app/tmp")

    # Yandex Vision API settings
    yandex_vision_api_key: str = Field(default="", validation_alias="YANDEX_VISION_API_KEY")
    yandex_folder_id: str = Field(default="", validation_alias="YANDEX_FOLDER_ID")

    # OCR processing settings (keeping for backward compatibility)
    ocr_mineru_models_dir: Path | None = None  # Not used with Yandex Vision
    ocr_mineru_use_gpu: bool = False  # Not used with Yandex Vision
    ocr_mineru_backend: str = "yandex-vision"  # Updated default
    ocr_mineru_document_timeout_seconds: float = 1800.0
    ocr_mineru_preprocess_scans: bool = True
    ocr_mineru_scan_dpi: int = 220
    ocr_mineru_max_page_megapixels: float = 12.0
    ocr_native_min_characters: int = 40
    ocr_native_minimum_usable_page_ratio: float = 0.95

    ocr_api_prefix: str = "/api/v1"


def get_settings() -> Settings:
    return Settings()