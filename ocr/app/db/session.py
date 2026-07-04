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
        # Check if the legacy input_blobs table exists
        input_blobs_exists = connection.execute(
            text("""
                SELECT EXISTS (
                    SELECT FROM information_schema.tables 
                    WHERE table_schema = current_schema() 
                    AND table_name = 'input_blobs'
                )
            """)
        ).scalar()

        # Check if the legacy parse_jobs table exists
        parse_jobs_exists = connection.execute(
            text("""
                SELECT EXISTS (
                    SELECT FROM information_schema.tables 
                    WHERE table_schema = current_schema() 
                    AND table_name = 'parse_jobs'
                )
            """)
        ).scalar()

        # Check if the legacy parse_results table exists
        parse_results_exists = connection.execute(
            text("""
                SELECT EXISTS (
                    SELECT FROM information_schema.tables 
                    WHERE table_schema = current_schema() 
                    AND table_name = 'parse_results'
                )
            """)
        ).scalar()

        # Check if file_type column exists in input_blobs table
        file_type_col_exists = False
        if input_blobs_exists:
            file_type_col_result = connection.execute(
                text("""
                    SELECT EXISTS (
                        SELECT FROM information_schema.columns
                        WHERE table_schema = current_schema()
                        AND table_name = 'input_blobs'
                        AND column_name = 'file_type'
                    )
                """)
            ).scalar()
            file_type_col_exists = file_type_col_result

        # Migrate from input_blobs if it exists
        if input_blobs_exists:
            if file_type_col_exists:
                # input_blobs has file_type column
                connection.execute(
                    text(
                        """
                        INSERT INTO blobs (id, filename, file_type, content_type, size_bytes, sha256, content, created_at, updated_at)
                        SELECT id, filename, COALESCE(file_type, 'unknown'), content_type, size_bytes, sha256, content, created_at, updated_at
                        FROM input_blobs
                        ON CONFLICT (id) DO NOTHING
                        """
                    )
                )
            else:
                # input_blobs doesn't have file_type column, use a default value
                connection.execute(
                    text(
                        """
                        INSERT INTO blobs (id, filename, file_type, content_type, size_bytes, sha256, content, created_at, updated_at)
                        SELECT id, filename, 'unknown', content_type, size_bytes, sha256, content, created_at, updated_at
                        FROM input_blobs
                        ON CONFLICT (id) DO NOTHING
                        """
                    )
                )

        # Check if parse_results has content_text column
        content_text_col_exists = False
        if parse_results_exists:
            content_text_col_result = connection.execute(
                text("""
                    SELECT EXISTS (
                        SELECT FROM information_schema.columns
                        WHERE table_schema = current_schema()
                        AND table_name = 'parse_results'
                        AND column_name = 'content_text'
                    )
                """)
            ).scalar()
            content_text_col_exists = content_text_col_result

        # Migrate from parse_results to blobs if both tables exist
        if parse_results_exists and parse_jobs_exists:
            if content_text_col_exists:
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
                        WHERE NOT EXISTS (SELECT 1 FROM blobs b WHERE b.id = pr.id)
                        """
                    )
                )

        # Migrate from parse_jobs to uploads if the tables exist
        if parse_jobs_exists:
            connection.execute(
                text(
                    """
                    INSERT INTO uploads (id, input_blob, output_blob, status, language, error, created_at, updated_at)
                    SELECT pj.id,
                           pj.input_blob_id,
                           pr.id,
                           pj.status,
                           pj.language,
                           pj.error,
                           pj.created_at,
                           pj.updated_at
                    FROM parse_jobs pj
                    LEFT JOIN parse_results pr ON pr.job_id = pj.id
                    WHERE NOT EXISTS (SELECT 1 FROM uploads u WHERE u.id = pj.id)
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