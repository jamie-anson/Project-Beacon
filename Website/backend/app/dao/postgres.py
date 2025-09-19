from __future__ import annotations
from typing import List
from datetime import datetime
import uuid

from sqlalchemy import Column, String, Text, Float, DateTime
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.ext.asyncio import create_async_engine, async_sessionmaker, AsyncAttrs
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column

from .base import DiffDAO
from ..models.diff import DiffRequest, DiffResult, DiffSegment, RegionOutput


class Base(AsyncAttrs, DeclarativeBase):
    pass


class DiffResultRow(Base):
    __tablename__ = "diff_results"

    id: Mapped[str] = mapped_column(String(64), primary_key=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False)
    region_a: Mapped[str] = mapped_column(String(64), nullable=False)
    text_a: Mapped[str] = mapped_column(Text, nullable=False)
    region_b: Mapped[str] = mapped_column(String(64), nullable=False)
    text_b: Mapped[str] = mapped_column(Text, nullable=False)
    segments: Mapped[dict] = mapped_column(JSONB, nullable=False)
    similarity: Mapped[float] = mapped_column(Float, nullable=False, default=0.0)
    notes: Mapped[str | None] = mapped_column(Text, nullable=True)


def _normalize_async_url(database_url: str) -> str:
    if database_url.startswith("postgresql://") and "+" not in database_url.split("://", 1)[0]:
        return database_url.replace("postgresql://", "postgresql+asyncpg://", 1)
    return database_url


def create_engine_and_session(database_url: str):
    # Expect asyncpg URL, e.g. postgresql+asyncpg://user:pass@host:5432/db
    database_url = _normalize_async_url(database_url)
    engine = create_async_engine(database_url, pool_pre_ping=True)
    SessionLocal = async_sessionmaker(engine, expire_on_commit=False)
    return engine, SessionLocal


async def create_tables(engine):
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)


class PostgresDiffDAO(DiffDAO):
    def __init__(self, database_url: str):
        self.database_url = database_url
        self.engine, self._session = create_engine_and_session(database_url)

    async def save_result(self, req: DiffRequest, segments: List[DiffSegment], similarity: float) -> DiffResult:
        rid = uuid.uuid4().hex[:12]
        row = DiffResultRow(
            id=rid,
            created_at=datetime.utcnow(),
            region_a=req.a.region,
            text_a=req.a.text,
            region_b=req.b.region,
            text_b=req.b.text,
            segments=[s.model_dump() for s in segments],
            similarity=float(similarity),
            notes=None,
        )
        async with self._session() as session:
            session.add(row)
            await session.commit()

        return DiffResult(
            id=rid,
            created_at=row.created_at,
            a=RegionOutput(region=row.region_a, text=row.text_a),
            b=RegionOutput(region=row.region_b, text=row.text_b),
            segments=[DiffSegment(**s) for s in row.segments],
            similarity=row.similarity,
            notes=row.notes,
        )

    async def list_recent(self, limit: int = 10) -> List[DiffResult]:
        from sqlalchemy import select, desc

        async with self._session() as session:
            res = await session.execute(
                select(DiffResultRow).order_by(desc(DiffResultRow.created_at)).limit(limit)
            )
            rows = list(res.scalars().all())
        return [
            DiffResult(
                id=r.id,
                created_at=r.created_at,
                a=RegionOutput(region=r.region_a, text=r.text_a),
                b=RegionOutput(region=r.region_b, text=r.text_b),
                segments=[DiffSegment(**s) for s in r.segments],
                similarity=r.similarity,
                notes=r.notes,
            ) for r in rows
        ]
