"""Точка входа FastAPI-приложения."""

from __future__ import annotations

import logging
import uuid
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

import structlog
from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware

from app import __version__
from app.api.routes import router
from app.config import get_settings
from app.db.session import create_session_factory

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
        cache_logger_on_first_use=False,
    )


@asynccontextmanager
async def lifespan(_app: FastAPI) -> AsyncIterator[None]:
    """Lifecycle: настройка логирования, временной директории и БД."""
    _configure_logging()
    settings.temp_dir.mkdir(parents=True, exist_ok=True)
    _app.state.session_factory = create_session_factory(settings)
    _app.state.engine = _app.state.session_factory.engine
    structlog.get_logger(__name__).info(
        "app.started",
        version=__version__,
        database_url=settings.database_url,
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


app.include_router(router, prefix=settings.api_prefix)
