"""Предзагрузка моделей Docling при старте контейнера воркера."""

from __future__ import annotations

import sys

import structlog

from app.config import get_settings
from app.services.ocr_service import get_ocr_service

logger = structlog.get_logger(__name__)


def main() -> int:
    settings = get_settings()
    logger.info("preload.started", artifacts=str(settings.docling_artifacts_path))
    try:
        service = get_ocr_service(
            artifacts_path=settings.docling_artifacts_path,
            use_gpu=settings.docling_use_gpu,
            do_formula_enrichment=settings.docling_do_formula_enrichment,
        )
        service.warm_up()
        logger.info("preload.completed")
        return 0
    except Exception as exc:  # noqa: BLE001
        logger.error("preload.failed", error=str(exc))
        return 1


if __name__ == "__main__":
    sys.exit(main())
