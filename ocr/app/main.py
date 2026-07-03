"""Точка входа FastAPI-приложения."""

from __future__ import annotations

import logging
import uuid
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

import structlog
from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from app import __version__
from app.api.routes import router
from app.config import get_settings
from app.core.path_security import PathSecurityError

settings = get_settings()


def _configure_logging() -> None:
    """Настройка structlog для JSON-логов в продакшене."""
    log_level = getattr(logging, settings.log_level.upper(), logging.INFO)
    logging.basicConfig(level=log_level, format="%(message)s")

    structlog.configure(
        processors=[
            structlog.contextvars.merge_contextvars,
            structlog.processors.add_log_level,
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.JSONRenderer(),
        ],
        wrapper_class=structlog.make_filtering_bound_logger(log_level),
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
        cache_logger_on_first_use=True,
    )


@asynccontextmanager
async def lifespan(_app: FastAPI) -> AsyncIterator[None]:
    """Lifecycle: создание директорий при старте."""
    _configure_logging()
    settings.results_dir.mkdir(parents=True, exist_ok=True)
    settings.temp_dir.mkdir(parents=True, exist_ok=True)
    settings.allowed_base_path.mkdir(parents=True, exist_ok=True)
    structlog.get_logger(__name__).info(
        "app.started",
        version=__version__,
        allowed_base=str(settings.allowed_base_path),
    )
    yield
    structlog.get_logger(__name__).info("app.stopped")


app = FastAPI(
    title=settings.app_name,
    version=__version__,
    lifespan=lifespan,
    docs_url="/docs",
    redoc_url="/redoc",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.middleware("http")
async def correlation_id_middleware(request: Request, call_next):
    """Добавляет correlation_id для трейсинга запросов."""
    correlation_id = request.headers.get("X-Correlation-ID") or str(uuid.uuid4())
    request.state.correlation_id = correlation_id
    structlog.contextvars.clear_contextvars()
    structlog.contextvars.bind_contextvars(correlation_id=correlation_id)

    response = await call_next(request)
    response.headers["X-Correlation-ID"] = correlation_id
    return response


@app.exception_handler(PathSecurityError)
async def path_security_handler(_request: Request, exc: PathSecurityError) -> JSONResponse:
    status_code = 404 if exc.reason == "file_not_found" else 400
    return JSONResponse(
        status_code=status_code,
        content={"detail": str(exc), "reason": exc.reason},
    )


app.include_router(router, prefix=settings.api_prefix)
