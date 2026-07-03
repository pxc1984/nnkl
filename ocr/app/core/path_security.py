"""Валидация путей к PDF с защитой от path traversal."""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

import structlog

logger = structlog.get_logger(__name__)


class PathSecurityError(Exception):
    """Ошибка валидации пути к файлу."""

    def __init__(self, message: str, *, reason: str, requested_path: str) -> None:
        super().__init__(message)
        self.reason = reason
        self.requested_path = requested_path


@dataclass(frozen=True)
class ValidatedPath:
    """Результат успешной валидации пути."""

    original: str
    resolved: Path


def _log_suspicious(requested_path: str, reason: str, correlation_id: str | None = None) -> None:
    """Аудит подозрительных попыток доступа к файлам."""
    logger.warning(
        "security.path_access_denied",
        requested_path=requested_path,
        reason=reason,
        correlation_id=correlation_id,
    )


def validate_file_path(
    file_path: str,
    *,
    allowed_base: Path,
    max_size_bytes: int,
    correlation_id: str | None = None,
) -> ValidatedPath:
    """
    Проверяет путь к PDF-файлу.

    Защита:
    - нормализация через resolve()
    - проверка is_relative_to(allowed_base)
    - запрет symlink вне разрешённой директории
    - проверка что это файл, а не директория
    - проверка прав на чтение и размера
    """
    if not file_path or not file_path.strip():
        _log_suspicious(file_path, "empty_path", correlation_id)
        raise PathSecurityError(
            "Путь к файлу не может быть пустым",
            reason="empty_path",
            requested_path=file_path,
        )

    requested = file_path.strip()

    # Явная проверка на .. до resolve — дополнительный слой защиты
    if ".." in Path(requested).parts:
        _log_suspicious(requested, "path_traversal_sequence", correlation_id)
        raise PathSecurityError(
            "Обнаружена попытка path traversal",
            reason="path_traversal_sequence",
            requested_path=requested,
        )

    try:
        candidate = Path(requested)
    except (ValueError, OSError) as exc:
        _log_suspicious(requested, "invalid_path", correlation_id)
        raise PathSecurityError(
            f"Некорректный путь: {exc}",
            reason="invalid_path",
            requested_path=requested,
        ) from exc

    if not _is_absolute_api_path(candidate, requested):
        _log_suspicious(requested, "relative_path_not_allowed", correlation_id)
        raise PathSecurityError(
            "Разрешены только абсолютные пути",
            reason="relative_path_not_allowed",
            requested_path=requested,
        )

    allowed_resolved = allowed_base.resolve(strict=False)
    if not allowed_resolved.exists():
        allowed_resolved.mkdir(parents=True, exist_ok=True)

    # strict=False: не падаем, если файл ещё не существует — проверим отдельно
    resolved = candidate.resolve(strict=False)

    # Запрет symlink, указывающего за пределы allowed_base
    if candidate.is_symlink():
        link_target = candidate.resolve(strict=False)
        if not _is_relative_to(link_target, allowed_resolved):
            _log_suspicious(requested, "symlink_outside_base", correlation_id)
            raise PathSecurityError(
                "Символические ссылки за пределы разрешённой директории запрещены",
                reason="symlink_outside_base",
                requested_path=requested,
            )

    if not _is_relative_to(resolved, allowed_resolved):
        _log_suspicious(requested, "outside_allowed_base", correlation_id)
        raise PathSecurityError(
            f"Путь должен находиться внутри {allowed_resolved}",
            reason="outside_allowed_base",
            requested_path=requested,
        )

    if not resolved.exists():
        raise PathSecurityError(
            "Файл не найден",
            reason="file_not_found",
            requested_path=requested,
        )

    if resolved.is_dir():
        _log_suspicious(requested, "path_is_directory", correlation_id)
        raise PathSecurityError(
            "Указанный путь является директорией, ожидается файл",
            reason="path_is_directory",
            requested_path=requested,
        )

    if not resolved.is_file():
        _log_suspicious(requested, "not_a_regular_file", correlation_id)
        raise PathSecurityError(
            "Путь не указывает на обычный файл",
            reason="not_a_regular_file",
            requested_path=requested,
        )

    if not os.access(resolved, os.R_OK):
        _log_suspicious(requested, "not_readable", correlation_id)
        raise PathSecurityError(
            "Нет прав на чтение файла",
            reason="not_readable",
            requested_path=requested,
        )

    suffix = resolved.suffix.lower()
    if suffix != ".pdf":
        _log_suspicious(requested, "invalid_extension", correlation_id)
        raise PathSecurityError(
            "Разрешены только PDF-файлы (.pdf)",
            reason="invalid_extension",
            requested_path=requested,
        )

    file_size = resolved.stat().st_size
    if file_size > max_size_bytes:
        _log_suspicious(requested, "file_too_large", correlation_id)
        raise PathSecurityError(
            f"Размер файла превышает лимит {max_size_bytes // (1024 * 1024)} МБ",
            reason="file_too_large",
            requested_path=requested,
        )

    if file_size == 0:
        raise PathSecurityError(
            "Файл пустой",
            reason="empty_file",
            requested_path=requested,
        )

    logger.info(
        "security.path_validated",
        requested_path=requested,
        resolved_path=str(resolved),
        correlation_id=correlation_id,
    )
    return ValidatedPath(original=requested, resolved=resolved)


def _is_relative_to(path: Path, base: Path) -> bool:
    """Совместимость с Python 3.9+ is_relative_to."""
    try:
        path.relative_to(base)
        return True
    except ValueError:
        return False


def _is_absolute_api_path(path: Path, raw: str) -> bool:
    """
    Проверка абсолютности пути.

    API принимает Linux-пути вида /data/pdfs/file.pdf; на Windows
    pathlib считает их относительными, поэтому дополнительно проверяем '/'.
    """
    if path.is_absolute():
        return True
    return raw.startswith("/")
