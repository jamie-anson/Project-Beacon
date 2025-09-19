from fastapi import APIRouter, Depends
from typing import List
from ....models.diff import DiffRequest, DiffResult
from ....services.compute import ComputeService
from ....deps import get_compute_service

router = APIRouter(prefix="/diffs")


@router.post("/compare", response_model=DiffResult)
async def compare(req: DiffRequest, compute: ComputeService = Depends(get_compute_service)):
    return await compute.compare_texts(req)


@router.get("/recent", response_model=List[DiffResult])
async def recent(compute: ComputeService = Depends(get_compute_service)):
    return await compute.recent_diffs(limit=10)
