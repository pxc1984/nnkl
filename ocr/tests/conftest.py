"""Pytest fixtures."""

from __future__ import annotations

import os
import sys
from collections.abc import Generator
from pathlib import Path

import pytest
from fastapi.testclient import TestClient

PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(PROJECT_ROOT))

# Лёгкие моки docling для unit-тестов без тяжёлых зависимостей
if "docling" not in sys.modules:
    from unittest.mock import MagicMock

    _mock_format = MagicMock()
    _mock_format.PDF = "pdf"
    for _name in [
        "docling",
        "docling.datamodel",
        "docling.datamodel.base_models",
        "docling.datamodel.pipeline_options",
        "docling.document_converter",
        "docling_core",
        "docling_core.transforms",
        "docling_core.transforms.serializer",
        "docling_core.transforms.serializer.base",
        "docling_core.transforms.serializer.common",
        "docling_core.transforms.serializer.latex",
        "docling_core.types",
        "docling_core.types.doc",
        "docling_core.types.doc.base",
        "docling_core.types.doc.document",
    ]:
        sys.modules.setdefault(_name, MagicMock())
    sys.modules["docling.datamodel.base_models"].InputFormat = _mock_format
    sys.modules["docling.datamodel.pipeline_options"].AcceleratorOptions = MagicMock

# Тестовые настройки до импорта приложения
TEST_BASE = Path(__file__).resolve().parent / "fixtures" / "pdfs"
TEST_RESULTS = Path(__file__).resolve().parent / "fixtures" / "results"

os.environ.setdefault("ALLOWED_BASE_PATH", str(TEST_BASE))
os.environ.setdefault("RESULTS_DIR", str(TEST_RESULTS))
os.environ.setdefault("REDIS_URL", "redis://localhost:6379/0")
os.environ.setdefault("CELERY_BROKER_URL", "redis://localhost:6379/0")
os.environ.setdefault("CELERY_RESULT_BACKEND", "redis://localhost:6379/1")
os.environ.setdefault("DOCLING_USE_GPU", "false")


@pytest.fixture(autouse=True)
def setup_dirs() -> Generator[None, None, None]:
    TEST_BASE.mkdir(parents=True, exist_ok=True)
    TEST_RESULTS.mkdir(parents=True, exist_ok=True)
    yield


@pytest.fixture
def allowed_base() -> Path:
    return TEST_BASE


@pytest.fixture
def client() -> Generator[TestClient, None, None]:
    from app.config import get_settings
    from app.main import app

    get_settings.cache_clear()
    with TestClient(app) as test_client:
        yield test_client
    get_settings.cache_clear()


@pytest.fixture
def sample_pdf(allowed_base: Path) -> Path:
    """Минимальный валидный PDF для тестов."""
    pdf_path = allowed_base / "sample.pdf"
    import fitz

    doc = fitz.open()
    page = doc.new_page()
    page.insert_text(
        (72, 72),
        "Тестовый материал: сталь 09Г2С, предел прочности σ = 450 МПа при 20°C. "
        "Марка стали соответствует ГОСТ 19281.",
    )
    doc.save(pdf_path)
    doc.close()
    return pdf_path
