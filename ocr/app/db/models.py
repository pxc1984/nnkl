"""SQLAlchemy models for shared OCR storage."""

from __future__ import annotations

import uuid
from datetime import datetime

from sqlalchemy import DateTime, ForeignKey, LargeBinary, String, Text, UniqueConstraint, Uuid, func
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship


class Base(DeclarativeBase):
    """Base class for ORM models."""


class InputBlob(Base):
    __tablename__ = "input_blobs"

    id: Mapped[uuid.UUID] = mapped_column(Uuid, primary_key=True)
    filename: Mapped[str] = mapped_column(String(512))
    content_type: Mapped[str] = mapped_column(String(255), default="application/pdf")
    sha256: Mapped[str | None] = mapped_column(String(64), nullable=True)
    content: Mapped[bytes] = mapped_column(LargeBinary)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())

    jobs: Mapped[list[ParseJob]] = relationship(back_populates="input_blob")


class ParseJob(Base):
    __tablename__ = "parse_jobs"
    __table_args__ = (UniqueConstraint("document_id", name="uq_parse_jobs_document_id"),)

    id: Mapped[uuid.UUID] = mapped_column(Uuid, primary_key=True)
    document_id: Mapped[str] = mapped_column(String(255), index=True)
    input_blob_id: Mapped[uuid.UUID] = mapped_column(ForeignKey("input_blobs.id"), index=True)
    status: Mapped[str] = mapped_column(String(32), default="pending")
    output_format: Mapped[str] = mapped_column(String(32))
    language: Mapped[str] = mapped_column(String(32))
    error: Mapped[str | None] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    input_blob: Mapped[InputBlob] = relationship(back_populates="jobs")
    result: Mapped[ParseResult | None] = relationship(back_populates="job", uselist=False)


class ParseResult(Base):
    __tablename__ = "parse_results"

    id: Mapped[uuid.UUID] = mapped_column(Uuid, primary_key=True)
    job_id: Mapped[uuid.UUID] = mapped_column(ForeignKey("parse_jobs.id"), unique=True, index=True)
    content_type: Mapped[str] = mapped_column(String(255))
    content_text: Mapped[str] = mapped_column(Text)
    assets_zip: Mapped[bytes | None] = mapped_column(LargeBinary, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    job: Mapped[ParseJob] = relationship(back_populates="result")
