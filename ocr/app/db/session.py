"""Database engine and session helpers."""

from __future__ import annotations

from sqlalchemy import create_engine, text
from sqlalchemy.engine import Engine
from sqlalchemy.orm import Session, sessionmaker

from app.config import Settings
from app.db.models import Base


def create_session_factory(settings: Settings) -> sessionmaker[Session]:
    connect_args = {"check_same_thread": False} if settings.database_url.startswith("sqlite") else {}
    engine = create_engine(
        settings.database_url,
        future=True,
        pool_pre_ping=True,
        connect_args=connect_args,
    )
    Base.metadata.create_all(engine)
    factory = sessionmaker(bind=engine, autoflush=False, autocommit=False, future=True)
    factory.engine = engine  # type: ignore[attr-defined]
    return factory


def check_database(engine: Engine) -> None:
    with engine.connect() as connection:
        connection.execute(text("SELECT 1"))
