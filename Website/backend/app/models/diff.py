from datetime import datetime
from typing import Optional, List
from pydantic import BaseModel, Field


class RegionOutput(BaseModel):
    region: str
    text: str


class DiffSegment(BaseModel):
    type: str  # equal|insert|delete|replace
    a: str = ""
    b: str = ""


class DiffResult(BaseModel):
    id: str
    created_at: datetime = Field(default_factory=datetime.utcnow)
    a: RegionOutput
    b: RegionOutput
    segments: List[DiffSegment] = []
    similarity: float = 0.0
    notes: Optional[str] = None


class DiffRequest(BaseModel):
    a: RegionOutput
    b: RegionOutput
    algorithm: str = "simple"
