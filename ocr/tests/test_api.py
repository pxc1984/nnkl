"""Тесты HTTP API."""

from __future__ import annotations

import uuid
from pathlib import Path
from unittest.mock import patch

from fastapi.testclient import TestClient

from app.db.models import InputBlob, ParseJob


def _insert_blob(db_session, sample_pdf: Path, blob_id: str = "blob-1") -> InputBlob:
    blob = InputBlob(
        id=blob_id,
        filename=sample_pdf.name,
        content_type="application/pdf",
        content=sample_pdf.read_bytes(),
    )
    db_session.add(blob)
    db_session.commit()
    return blob


class TestHealthEndpoint:
    def test_health_returns_response(self, client: TestClient) -> None:
        response = client.get("/api/v1/health")

        assert response.status_code == 200
        data = response.json()
        assert data == {"status": "ok", "api": "ok", "database": "ok"}


class TestParseEndpoint:
    @patch("app.use_cases.parse_document.get_ocr_service")
    def test_parse_persists_result(self, mock_get_ocr_service, client: TestClient, db_session, sample_pdf: Path) -> None:
        _insert_blob(db_session, sample_pdf)
        ocr_service = mock_get_ocr_service.return_value
        ocr_service.convert.return_value = ("parsed content", None)

        response = client.post(
            "/api/v1/parse",
            json={
                "document_id": "doc-1",
                "input_blob_id": "blob-1",
                "output_format": "latex",
                "language": "auto",
            },
        )

        assert response.status_code == 201
        data = response.json()
        assert data["document_id"] == "doc-1"
        assert data["status"] == "completed"
        assert data["result_id"]

        job = db_session.query(ParseJob).filter(ParseJob.document_id == "doc-1").one()
        assert job.status == "completed"
        assert job.result is not None
        assert job.result.content_text == "parsed content"

    def test_parse_missing_blob_returns_404(self, client: TestClient) -> None:
        response = client.post(
            "/api/v1/parse",
            json={
                "document_id": "doc-missing",
                "input_blob_id": "blob-missing",
                "output_format": "latex",
                "language": "auto",
            },
        )

        assert response.status_code == 404


class TestStatusAndResultEndpoints:
    @patch("app.use_cases.parse_document.get_ocr_service")
    def test_status_and_result_return_db_content(self, mock_get_ocr_service, client: TestClient, db_session, sample_pdf: Path) -> None:
        document_id = f"doc-{uuid.uuid4()}"
        _insert_blob(db_session, sample_pdf, blob_id="blob-2")
        ocr_service = mock_get_ocr_service.return_value
        ocr_service.convert.return_value = ("# markdown", None)

        parse_response = client.post(
            "/api/v1/parse",
            json={
                "document_id": document_id,
                "input_blob_id": "blob-2",
                "output_format": "markdown",
                "language": "en",
            },
        )
        assert parse_response.status_code == 201

        status_response = client.get(f"/api/v1/status/{document_id}")
        assert status_response.status_code == 200
        assert status_response.json()["status"] == "completed"

        result_response = client.get(f"/api/v1/result/{document_id}")
        assert result_response.status_code == 200
        data = result_response.json()
        assert data["content_type"] == "text/markdown"
        assert data["content_text"] == "# markdown"
        assert data["has_assets_zip"] is False
