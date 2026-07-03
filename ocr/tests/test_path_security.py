"""Тесты защиты от path traversal."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest
from fastapi.testclient import TestClient

from app.core.path_security import PathSecurityError, validate_file_path


class TestPathSecurityUnit:
    def test_valid_path(self, allowed_base: Path, sample_pdf: Path) -> None:
        result = validate_file_path(
            str(sample_pdf),
            allowed_base=allowed_base,
            max_size_bytes=10 * 1024 * 1024,
        )
        assert result.resolved == sample_pdf.resolve()

    def test_path_traversal(self, allowed_base: Path) -> None:
        malicious = str(allowed_base / ".." / ".." / "etc" / "passwd")
        with pytest.raises(PathSecurityError) as exc_info:
            validate_file_path(
                malicious,
                allowed_base=allowed_base,
                max_size_bytes=1024,
            )
        assert exc_info.value.reason in ("path_traversal_sequence", "outside_allowed_base")

    def test_outside_allowed_base(self, allowed_base: Path) -> None:
        with pytest.raises(PathSecurityError) as exc_info:
            validate_file_path(
                "/etc/passwd",
                allowed_base=allowed_base,
                max_size_bytes=1024,
            )
        assert exc_info.value.reason == "outside_allowed_base"

    def test_directory_instead_of_file(self, allowed_base: Path) -> None:
        with pytest.raises(PathSecurityError) as exc_info:
            validate_file_path(
                str(allowed_base),
                allowed_base=allowed_base,
                max_size_bytes=1024,
            )
        assert exc_info.value.reason == "path_is_directory"


class TestPathSecurityAPI:
    def test_path_traversal_returns_400(self, client: TestClient, allowed_base: Path) -> None:
        response = client.post(
            "/api/v1/convert",
            json={"file_path": f"{allowed_base}/../../etc/passwd"},
        )
        assert response.status_code == 400

    def test_outside_base_returns_400(self, client: TestClient) -> None:
        response = client.post(
            "/api/v1/convert",
            json={"file_path": "/etc/passwd"},
        )
        assert response.status_code == 400

    def test_directory_returns_400(self, client: TestClient, allowed_base: Path) -> None:
        response = client.post(
            "/api/v1/convert",
            json={"file_path": str(allowed_base)},
        )
        assert response.status_code == 400

    @patch("app.api.routes.celery_app.send_task")
    def test_valid_path_returns_200(
        self,
        mock_send_task: MagicMock,
        client: TestClient,
        sample_pdf: Path,
    ) -> None:
        mock_result = MagicMock()
        mock_result.id = "test-task-id-123"
        mock_send_task.return_value = mock_result

        response = client.post(
            "/api/v1/convert",
            json={"file_path": str(sample_pdf)},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["task_id"] == "test-task-id-123"
        assert data["status"] == "pending"
        mock_send_task.assert_called_once()
