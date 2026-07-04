"""SQLAlchemy models for shared OCR storage."""

from __future__ import annotations

import uuid
from datetime import datetime

from sqlalchemy import (
    DateTime,
    ForeignKey,
    Index,
    LargeBinary,
    String,
    Uuid,
    func,
)
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship


class Base(DeclarativeBase):
    """Base class for ORM models."""


class Blob(Base):
    __tablename__ = "blobs"
    __table_args__ = (Index("ix_blobs_sha256", "sha256"),)

    id: Mapped[uuid.UUID] = mapped_column(Uuid, primary_key=True)
    filename: Mapped[str] = mapped_column(String(512))
    file_type: Mapped[str] = mapped_column(String(32), index=True)
    content_type: Mapped[str] = mapped_column(
        String(255), default="application/pdf", index=True
    )
    size_bytes: Mapped[int] = mapped_column()
    sha256: Mapped[str | None] = mapped_column(String(64), nullable=True)
    content: Mapped[bytes] = mapped_column(LargeBinary)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), onupdate=func.now()
    )

    input_uploads: Mapped[list[Upload]] = relationship(
        back_populates="input_blob", foreign_keys="Upload.input_blob_id"
    )
    output_uploads: Mapped[list[Upload]] = relationship(
        back_populates="output_blob", foreign_keys="Upload.output_blob_id"
    )


class Upload(Base):
    __tablename__ = "uploads"

    id: Mapped[uuid.UUID] = mapped_column(Uuid, primary_key=True)
    input_blob_id: Mapped[uuid.UUID] = mapped_column(
        "input_blob", ForeignKey("blobs.id"), index=True
    )
    output_blob_id: Mapped[uuid.UUID | None] = mapped_column(
        "output_blob", ForeignKey("blobs.id"), nullable=True, index=True
    )
    status: Mapped[str] = mapped_column(String(32), default="pending")
    language: Mapped[str] = mapped_column(String(32))
    error: Mapped[str | None] = mapped_column(String, nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), onupdate=func.now()
    )

    input_blob: Mapped[Blob] = relationship(
        back_populates="input_uploads", foreign_keys=[input_blob_id]
    )
    output_blob: Mapped[Blob | None] = relationship(
        back_populates="output_uploads", foreign_keys=[output_blob_id]
    )
