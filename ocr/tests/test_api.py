"""Тесты HTTP API."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

from fastapi.testclient import TestClient


class TestHealthEndpoint:
    def test_health_returns_response(self, client: TestClient) -> None:
        with patch("redis.from_url") as mock_from_url:
            mock_client = MagicMock()
            mock_client.ping.return_value = True
            mock_from_url.return_value = mock_client

            with patch("app.api.routes.celery_app.control.inspect") as mock_inspect_fn:
                mock_inspect = MagicMock()
                mock_inspect.ping.return_value = {"worker1": "pong"}
                mock_inspect_fn.return_value = mock_inspect

                response = client.get("/api/v1/health")

        assert response.status_code == 200
        data = response.json()
        assert "status" in data
        assert "redis" in data
        assert "celery" in data


class TestConvertEndpoint:
    @patch("app.api.routes.celery_app.send_task")
    def test_convert_missing_file_returns_404(
        self,
        _mock_send: MagicMock,
        client: TestClient,
        allowed_base,
    ) -> None:
        response = client.post(
            "/api/v1/convert",
            json={"file_path": str(allowed_base / "nonexistent.pdf")},
        )
        assert response.status_code == 404

    @patch("app.api.routes.celery_app.send_task")
    def test_convert_validation_error(self, _mock_send: MagicMock, client: TestClient) -> None:
        response = client.post("/api/v1/convert", json={})
        assert response.status_code == 422


class TestQueueEndpoint:
    @patch("app.api.routes.get_queue_snapshot")
    def test_queue_returns_snapshot(self, mock_snapshot: MagicMock, client: TestClient) -> None:
        from app.api.schemas import QueueResponse, QueueTaskItem, TaskStatus

        mock_snapshot.return_value = QueueResponse(
            total=2,
            pending_count=1,
            processing_count=1,
            scheduled_count=0,
            pending=[
                QueueTaskItem(
                    task_id="task-pending-1",
                    status=TaskStatus.PENDING,
                    file_path="/data/pdfs/a.pdf",
                ),
            ],
            processing=[
                QueueTaskItem(
                    task_id="task-active-1",
                    status=TaskStatus.PROCESSING,
                    file_path="/data/pdfs/b.pdf",
                    progress=30,
                    worker="celery@worker1",
                ),
            ],
        )

        response = client.get("/api/v1/queue")

        assert response.status_code == 200
        data = response.json()
        assert data["total"] == 2
        assert data["pending_count"] == 1
        assert data["processing_count"] == 1
        assert len(data["pending"]) == 1
        assert data["pending"][0]["task_id"] == "task-pending-1"

    @patch("app.api.routes.get_queue_snapshot")
    def test_queue_status_filter(self, mock_snapshot: MagicMock, client: TestClient) -> None:
        from app.api.schemas import QueueResponse

        mock_snapshot.return_value = QueueResponse(
            total=0,
            pending_count=0,
            processing_count=0,
            scheduled_count=0,
        )

        response = client.get("/api/v1/queue", params={"status": "processing"})

        assert response.status_code == 200
        mock_snapshot.assert_called_once()
        assert mock_snapshot.call_args.kwargs["status_filter"].value == "processing"
