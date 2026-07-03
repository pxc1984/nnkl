"""Тесты сервиса очереди Celery."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

from app.api.schemas import TaskStatus
from app.services.queue_service import get_queue_snapshot


class TestQueueService:
    @patch("app.services.queue_service._get_broker_pending_tasks")
    @patch("app.services.queue_service._get_inspect_tasks")
    def test_get_queue_snapshot_dedup(
        self,
        mock_inspect: MagicMock,
        mock_broker: MagicMock,
    ) -> None:
        from app.api.schemas import QueueTaskItem

        mock_broker.return_value = [
            QueueTaskItem(task_id="same-id", status=TaskStatus.PENDING, file_path="/a.pdf"),
        ]
        mock_inspect.side_effect = [
            [QueueTaskItem(task_id="same-id", status=TaskStatus.PROCESSING, progress=10)],
            [],
        ]

        celery_app = MagicMock()
        snapshot = get_queue_snapshot(celery_app, broker_url="redis://localhost:6379/0")

        assert snapshot.total == 1
        assert snapshot.processing_count == 1
        assert snapshot.pending_count == 0
        assert len(snapshot.pending) == 0

    def test_parse_broker_message(self) -> None:
        import json

        from app.services.queue_service import _parse_broker_message

        envelope = {
            "body": json.dumps([
                [],
                {
                    "file_path": "/data/pdfs/test.pdf",
                    "output_format": "latex",
                    "language": "auto",
                },
                {"callbacks": None, "errbacks": None, "chain": None, "chord": None},
            ]),
            "headers": {
                "id": "broker-task-1",
                "task": "convert_pdf",
            },
        }
        item = _parse_broker_message(json.dumps(envelope))

        assert item is not None
        assert item.task_id == "broker-task-1"
        assert item.file_path == "/data/pdfs/test.pdf"
        assert item.status == TaskStatus.PENDING
