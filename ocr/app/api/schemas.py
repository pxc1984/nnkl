"""Pydantic-модели запросов и ответов API."""

from __future__ import annotations

import uuid
from enum import Enum
from typing import Literal

from pydantic import BaseModel, Field


class OutputFormat(str, Enum):
    LATEX = "latex"
    MARKDOWN = "markdown"


class Language(str, Enum):
    RU = "ru"
    EN = "en"
    AUTO = "auto"


class ParseRequest(BaseModel):
    """Запрос OCR-парсинга документа, загруженного gateway в shared Postgres."""

    document_id: str = Field(..., min_length=1, max_length=255)
    input_blob_id: uuid.UUID
    output_format: OutputFormat = OutputFormat.LATEX
    language: Language = Language.AUTO


class TaskStatus(str, Enum):
    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"


class ParseResponse(BaseModel):
    document_id: str
    job_id: uuid.UUID
    result_id: uuid.UUID
    status: Literal[TaskStatus.COMPLETED] = TaskStatus.COMPLETED


class StatusResponse(BaseModel):
    document_id: str
    job_id: uuid.UUID
    status: TaskStatus
    input_blob_id: uuid.UUID
    output_format: OutputFormat
    language: Language
    result_id: uuid.UUID | None = None
    error: str | None = None


class HealthResponse(BaseModel):
    status: Literal["ok", "error"]
    api: Literal["ok"] = "ok"
    database: Literal["ok", "error"]


class ErrorResponse(BaseModel):
    detail: str
    reason: str | None = None


class ParseResultResponse(BaseModel):
    document_id: str
    job_id: uuid.UUID
    result_id: uuid.UUID
    content_type: str
    content_text: str
    has_assets_zip: bool
