"""Health check endpoints"""

import time
from fastapi import APIRouter

router = APIRouter()


@router.get("/health")
async def health_check():
    """Health check endpoint"""
    from ..main import router_instance
    
    healthy_providers = [p for p in router_instance.providers if p.healthy]
    return {
        "status": "healthy",
        "timestamp": time.time(),
        "providers_total": len(router_instance.providers),
        "providers_healthy": len(healthy_providers),
        "regions": list(set(p.region for p in healthy_providers))
    }


@router.get("/modal-health")
async def modal_health():
    """Dedicated health endpoint for Modal provider checks"""
    return {
        "status": "healthy",
        "timestamp": time.time(),
        "service": "project-beacon-router",
    }
