"""Pydantic-модели запросов и ответов API."""

from __future__ import annotations

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


class ConvertRequest(BaseModel):
    """Запрос на конвертацию PDF по локальному пути."""

    file_path: str = Field(..., min_length=1, description="Абсолютный путь к PDF на локальной ФС")
    output_format: OutputFormat = OutputFormat.LATEX
    language: Language = Language.AUTO


class TaskStatus(str, Enum):
    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"


class ConvertResponse(BaseModel):
    task_id: str
    status: Literal[TaskStatus.PENDING] = TaskStatus.PENDING


class StatusResponse(BaseModel):
    task_id: str
    status: TaskStatus
    progress: int = Field(ge=0, le=100, default=0)
    stage: str | None = Field(default=None, description="Текущий этап: docling, exporting, ...")
    result_url: str | None = None
    error: str | None = None


class HealthResponse(BaseModel):
    status: Literal["ok", "degraded", "error"]
    api: Literal["ok"] = "ok"
    redis: Literal["ok", "error"]
    celery: Literal["ok", "error", "unknown"]


class ErrorResponse(BaseModel):
    detail: str
    reason: str | None = None


class QueueTaskItem(BaseModel):
    """Задача в очереди Celery."""

    task_id: str
    status: TaskStatus
    name: str = "convert_pdf"
    file_path: str | None = None
    output_format: str | None = None
    language: str | None = None
    worker: str | None = None
    progress: int = Field(ge=0, le=100, default=0)
    eta: str | None = Field(default=None, description="Время запуска для отложенных задач")


class QueueResponse(BaseModel):
    """Снимок очереди задач конвертации."""

    total: int = Field(ge=0, description="Всего задач во всех категориях")
    pending_count: int = Field(ge=0, description="В брокере, ожидают воркера")
    processing_count: int = Field(ge=0, description="Выполняются или зарезервированы")
    scheduled_count: int = Field(ge=0, description="Отложенные (ETA)")
    pending: list[QueueTaskItem] = Field(default_factory=list)
    processing: list[QueueTaskItem] = Field(default_factory=list)
    scheduled: list[QueueTaskItem] = Field(default_factory=list)
