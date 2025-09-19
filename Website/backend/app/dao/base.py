from __future__ import annotations
from abc import ABC, abstractmethod
from typing import List
from ..models.diff import DiffRequest, DiffResult, DiffSegment, RegionOutput
from datetime import datetime
import uuid


class DiffDAO(ABC):
    @abstractmethod
    async def save_result(self, req: DiffRequest, segments: List[DiffSegment], similarity: float) -> DiffResult:
        ...

    @abstractmethod
    async def list_recent(self, limit: int = 10) -> List[DiffResult]:
        ...


class InMemoryDiffDAO(DiffDAO):
    def __init__(self):
        self._items: List[DiffResult] = []

    async def save_result(self, req: DiffRequest, segments: List[DiffSegment], similarity: float) -> DiffResult:
        rid = uuid.uuid4().hex[:12]
        item = DiffResult(
            id=rid,
            created_at=datetime.utcnow(),
            a=req.a,
            b=req.b,
            segments=segments,
            similarity=similarity,
        )
        self._items.insert(0, item)
        self._items = self._items[:100]
        return item

    async def list_recent(self, limit: int = 10) -> List[DiffResult]:
        return self._items[:limit]
