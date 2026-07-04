"""Database engine and session helpers."""

from __future__ import annotations

from sqlalchemy import create_engine, text
from sqlalchemy.engine import Engine
from sqlalchemy.orm import Session, sessionmaker

from app.config import Settings
from app.db.models import Base


def create_session_factory(settings: Settings) -> sessionmaker[Session]:
    connect_args = (
        {"check_same_thread": False}
        if settings.database_url.startswith("sqlite")
        else {}
    )
    engine = create_engine(
        settings.database_url,
        future=True,
        pool_pre_ping=True,
        connect_args=connect_args,
    )
    Base.metadata.create_all(engine)
    migrate_legacy_blob_tables(engine)
    factory = sessionmaker(bind=engine, autoflush=False, autocommit=False, future=True)
    factory.engine = engine  # type: ignore[attr-defined]
    return factory


def check_database(engine: Engine) -> None:
    with engine.connect() as connection:
        connection.execute(text("SELECT 1"))


def migrate_legacy_blob_tables(engine: Engine) -> None:
    if engine.dialect.name != "postgresql":
        return

    with engine.begin() as connection:
        exists = connection.execute(
            text("SELECT to_regclass(current_schema() || '.input_blobs') IS NOT NULL")
        ).scalar()
        if not exists:
            return

        connection.execute(
            text(
                """
                INSERT INTO blobs (id, filename, file_type, content_type, size_bytes, sha256, content, created_at, updated_at)
                SELECT id, filename, file_type, content_type, size_bytes, sha256, content, created_at, updated_at
                FROM input_blobs
                ON CONFLICT (id) DO NOTHING
                """
            )
        )
        connection.execute(
            text(
                """
                INSERT INTO blobs (id, filename, file_type, content_type, size_bytes, sha256, content, created_at, updated_at)
                SELECT pr.id,
                       CONCAT(COALESCE(NULLIF(split_part(ib.filename, '.', 1), ''), 'upload'), '.md'),
                       'markdown',
                       COALESCE(pr.content_type, 'text/markdown'),
                       octet_length(convert_to(pr.content_text, 'UTF8')),
                       NULL,
                       convert_to(pr.content_text, 'UTF8'),
                       pr.created_at,
                       pr.updated_at
                FROM parse_results pr
                JOIN parse_jobs pj ON pj.id = pr.job_id
                LEFT JOIN input_blobs ib ON ib.id = pj.input_blob_id
                ON CONFLICT (id) DO NOTHING
                """
            )
        )
        connection.execute(
            text(
                """
                INSERT INTO uploads (id, input_blob, output_blob, status, language, error, created_at, updated_at)
                SELECT pj.input_blob_id,
                       pj.input_blob_id,
                       pr.id,
                       pj.status,
                       pj.language,
                       pj.error,
                       pj.created_at,
                       pj.updated_at
                FROM parse_jobs pj
                LEFT JOIN parse_results pr ON pr.job_id = pj.id
                ON CONFLICT (id) DO UPDATE SET
                  input_blob = EXCLUDED.input_blob,
                  output_blob = EXCLUDED.output_blob,
                  status = EXCLUDED.status,
                  language = EXCLUDED.language,
                  error = EXCLUDED.error,
                  updated_at = EXCLUDED.updated_at
                """
            )
        )
