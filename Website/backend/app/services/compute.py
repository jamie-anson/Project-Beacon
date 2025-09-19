from __future__ import annotations
from typing import List
from ..models.diff import DiffRequest, DiffResult, DiffSegment
from ..dao.base import DiffDAO


class ComputeService:
    def __init__(self, dao: DiffDAO):
        self.dao = dao

    async def compare_texts(self, req: DiffRequest) -> DiffResult:
        # naive diff: line-by-line compare
        a_lines = req.a.text.splitlines()
        b_lines = req.b.text.splitlines()

        segments: List[DiffSegment] = []
        max_len = max(len(a_lines), len(b_lines))
        equal = 0
        total = 0
        for i in range(max_len):
            a = a_lines[i] if i < len(a_lines) else ""
            b = b_lines[i] if i < len(b_lines) else ""
            total += 1
            if a == b:
                segments.append(DiffSegment(type="equal", a=a, b=b))
                equal += 1
            else:
                if a and b:
                    segments.append(DiffSegment(type="replace", a=a, b=b))
                elif a and not b:
                    segments.append(DiffSegment(type="delete", a=a, b=""))
                elif b and not a:
                    segments.append(DiffSegment(type="insert", a="", b=b))

        similarity = equal / total if total else 1.0
        result = await self.dao.save_result(req, segments, similarity)
        return result

    async def recent_diffs(self, limit: int = 10):
        return await self.dao.list_recent(limit)
