"""Pytest fixtures."""

from __future__ import annotations

import os
import sys
from collections.abc import Generator
from pathlib import Path
from unittest.mock import MagicMock

import pytest
from fastapi.testclient import TestClient

PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(PROJECT_ROOT))

# Лёгкие моки docling для unit-тестов без тяжёлых зависимостей
if "docling" not in sys.modules:
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

TEST_ROOT = Path(__file__).resolve().parent / "fixtures"
TEST_DATABASE = TEST_ROOT / "test.sqlite3"

os.environ.setdefault("DATABASE_URL", f"sqlite:///{TEST_DATABASE}")
os.environ.setdefault("TEMP_DIR", str(TEST_ROOT / "tmp"))
os.environ.setdefault("DOCLING_USE_GPU", "false")
os.environ.setdefault("DOCLING_DO_FORMULA_ENRICHMENT", "false")


@pytest.fixture(autouse=True)
def setup_dirs() -> Generator[None, None, None]:
    TEST_ROOT.mkdir(parents=True, exist_ok=True)
    if TEST_DATABASE.exists():
        TEST_DATABASE.unlink()
    yield
    if TEST_DATABASE.exists():
        TEST_DATABASE.unlink()


@pytest.fixture
def client() -> Generator[TestClient, None, None]:
    from app.main import app

    with TestClient(app) as test_client:
        yield test_client


@pytest.fixture
def sample_pdf() -> Path:
    """Минимальный валидный PDF для тестов."""
    pdf_path = TEST_ROOT / "sample.pdf"
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


@pytest.fixture
def db_session(client: TestClient):
    session = client.app.state.session_factory()
    try:
        yield session
    finally:
        session.close()
