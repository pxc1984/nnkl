"""Тесты HTTP API."""

from __future__ import annotations

import uuid
from pathlib import Path
from unittest.mock import patch

from fastapi.testclient import TestClient

from app.db.models import InputBlob, ParseJob


def _insert_blob(
    db_session, sample_pdf: Path, blob_id: str = "00000000-0000-0000-0000-000000000001"
) -> InputBlob:
    blob = InputBlob(
        id=uuid.UUID(blob_id),
        filename=sample_pdf.name,
        content_type="application/pdf",
        content=sample_pdf.read_bytes(),
    )
    db_session.add(blob)
    db_session.commit()
    return blob


def _insert_blob_bytes(
    db_session,
    *,
    blob_id: str,
    filename: str,
    content_type: str,
    content: bytes,
) -> InputBlob:
    blob = InputBlob(
        id=uuid.UUID(blob_id),
        filename=filename,
        content_type=content_type,
        content=content,
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
    def test_parse_native_pdf_bypasses_mineru(
        self, mock_get_ocr_service, client: TestClient, db_session, sample_pdf: Path
    ) -> None:
        blob_id = str(uuid.uuid4())
        _insert_blob(db_session, sample_pdf, blob_id=blob_id)

        response = client.post(
            "/api/v1/parse",
            json={
                "document_id": "doc-native-pdf",
                "input_blob_id": blob_id,
                "language": "auto",
            },
        )

        assert response.status_code == 201
        mock_get_ocr_service.assert_not_called()

        result_response = client.get("/api/v1/result/doc-native-pdf")
        assert result_response.status_code == 200
        content = result_response.json()["content_text"]
        assert "Test material" in content
        assert "GOST 19281" in content

    @patch("app.use_cases.parse_document.extract_native_document_text")
    @patch("app.use_cases.parse_document.get_ocr_service")
    def test_parse_corrupted_native_pdf_falls_back_to_mineru(
        self,
        mock_get_ocr_service,
        mock_extract_native_document_text,
        client: TestClient,
        db_session,
        sample_pdf: Path,
    ) -> None:
        blob_id = str(uuid.uuid4())
        _insert_blob(db_session, sample_pdf, blob_id=blob_id)
        ocr_service = mock_get_ocr_service.return_value
        ocr_service.convert.return_value = ("mineru fallback", None)

        with patch(
            "app.use_cases.parse_document.should_use_native_pdf_text",
            return_value=False,
        ):
            response = client.post(
                "/api/v1/parse",
                json={
                    "document_id": "doc-fallback-pdf",
                    "input_blob_id": blob_id,
                    "language": "auto",
                },
            )

        assert response.status_code == 201
        mock_extract_native_document_text.assert_not_called()
        ocr_service.convert.assert_called_once()

    @patch("app.use_cases.parse_document.get_ocr_service")
    def test_parse_persists_result(
        self, mock_get_ocr_service, client: TestClient, db_session, sample_pdf: Path
    ) -> None:
        blob_id = str(uuid.uuid4())
        _insert_blob(db_session, sample_pdf, blob_id=blob_id)
        ocr_service = mock_get_ocr_service.return_value
        ocr_service.convert.return_value = ("parsed content", None)

        with patch("app.use_cases.parse_document.should_use_native_pdf_text", return_value=False):
            response = client.post(
            "/api/v1/parse",
            json={
                "document_id": "doc-1",
                "input_blob_id": blob_id,
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
        assert job.result.content_text.endswith("parsed content")

    def test_parse_missing_blob_returns_404(self, client: TestClient) -> None:
        missing_blob_id = str(uuid.uuid4())
        response = client.post(
            "/api/v1/parse",
            json={
                "document_id": "doc-missing",
                "input_blob_id": missing_blob_id,
                "language": "auto",
            },
        )

        assert response.status_code == 404


class TestStatusAndResultEndpoints:
    @patch("app.use_cases.parse_document.get_ocr_service")
    def test_status_and_result_return_db_content(
        self, mock_get_ocr_service, client: TestClient, db_session, sample_pdf: Path
    ) -> None:
        document_id = f"doc-{uuid.uuid4()}"
        blob_id = str(uuid.uuid4())
        _insert_blob(db_session, sample_pdf, blob_id=blob_id)
        ocr_service = mock_get_ocr_service.return_value
        ocr_service.convert.return_value = ("# markdown", None)

        with patch("app.use_cases.parse_document.should_use_native_pdf_text", return_value=False):
            parse_response = client.post(
            "/api/v1/parse",
            json={
                "document_id": document_id,
                "input_blob_id": blob_id,
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
        assert data["content_text"].endswith("# markdown")
        assert data["has_assets_zip"] is False


class TestNativeOfficeExtraction:
    def test_parse_docx_uses_native_extraction(
        self, client: TestClient, db_session, sample_docx_bytes: bytes
    ) -> None:
        blob_id = str(uuid.uuid4())
        _insert_blob_bytes(
            db_session,
            blob_id=blob_id,
            filename="sample.docx",
            content_type="application/vnd.openxmlformats-officedocument.wordprocessingml.document",
            content=sample_docx_bytes,
        )

        response = client.post(
            "/api/v1/parse",
            json={
                "document_id": "doc-docx",
                "input_blob_id": blob_id,
                "language": "auto",
            },
        )

        assert response.status_code == 201
        result_response = client.get("/api/v1/result/doc-docx")
        assert result_response.status_code == 200
        content = result_response.json()["content_text"]
        assert "DOCX title" in content
        assert "DOCX body text" in content
        assert "Cell A | Cell B" in content

    def test_parse_pptx_uses_native_extraction(
        self, client: TestClient, db_session, sample_pptx_bytes: bytes
    ) -> None:
        blob_id = str(uuid.uuid4())
        _insert_blob_bytes(
            db_session,
            blob_id=blob_id,
            filename="slides.pptx",
            content_type="application/vnd.openxmlformats-officedocument.presentationml.presentation",
            content=sample_pptx_bytes,
        )

        response = client.post(
            "/api/v1/parse",
            json={
                "document_id": "doc-pptx",
                "input_blob_id": blob_id,
                "language": "auto",
            },
        )

        assert response.status_code == 201
        result_response = client.get("/api/v1/result/doc-pptx")
        assert result_response.status_code == 200
        content = result_response.json()["content_text"]
        assert "Slide 1" in content
        assert "PPTX title" in content
        assert "PPTX bullet" in content
