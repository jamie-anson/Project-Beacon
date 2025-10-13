"""Queue management endpoints"""

from fastapi import APIRouter
from typing import Dict, Any

router = APIRouter()


@router.get("/queue/status")
async def get_queue_status() -> Dict[str, Any]:
    """Get status of all region queues"""
    from ..core.region_queue import queue_manager
    
    return {
        "queues": queue_manager.get_all_statuses(),
        "timestamp": __import__("time").time(),
    }


@router.get("/queue/status/{region}")
async def get_region_queue_status(region: str) -> Dict[str, Any]:
    """Get status of a specific region queue"""
    from ..core.region_queue import queue_manager
    
    region = region.upper()
    if region not in queue_manager.queues:
        return {"error": f"Unknown region: {region}"}
    
    return queue_manager.queues[region].get_status()
