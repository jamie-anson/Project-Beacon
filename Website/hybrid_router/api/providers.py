"""Provider management endpoints"""

import os
from fastapi import APIRouter
from typing import Optional

router = APIRouter()


@router.get("/providers")
async def list_providers(region: Optional[str] = None):
    """List all providers and their status (optionally filter by region)"""
    from ..main import router_instance
    
    providers = router_instance.providers
    if region:
        providers = [p for p in providers if p.region == region]
    
    return {
        "providers": [
            {
                "name": p.name,
                "type": p.type.value,
                "region": p.region,
                "healthy": p.healthy,
                "cost_per_second": p.cost_per_second,
                "avg_latency": p.avg_latency,
                "success_rate": p.success_rate,
                "last_health_check": p.last_health_check
            }
            for p in providers
        ]
    }


@router.get("/metrics")
async def get_metrics():
    """Get routing metrics"""
    from ..main import router_instance
    
    healthy_providers = [p for p in router_instance.providers if p.healthy]
    
    return {
        "total_providers": len(router_instance.providers),
        "healthy_providers": len(healthy_providers),
        "avg_latency": sum(p.avg_latency for p in healthy_providers) / len(healthy_providers) if healthy_providers else 0,
        "avg_success_rate": sum(p.success_rate for p in healthy_providers) / len(healthy_providers) if healthy_providers else 0,
        "cost_range": {
            "min": min(p.cost_per_second for p in healthy_providers) if healthy_providers else 0,
            "max": max(p.cost_per_second for p in healthy_providers) if healthy_providers else 0
        }
    }


@router.get("/env")
async def env_dump():
    """Debug endpoint to inspect provider-related environment variables"""
    return {
        "GOLEM_US_ENDPOINT": os.getenv("GOLEM_US_ENDPOINT"),
        "GOLEM_EU_ENDPOINT": os.getenv("GOLEM_EU_ENDPOINT"),
        "GOLEM_APAC_ENDPOINT": os.getenv("GOLEM_APAC_ENDPOINT"),
        "GOLEM_PROVIDER_ENDPOINTS": os.getenv("GOLEM_PROVIDER_ENDPOINTS"),
        "MODAL_API_BASE": os.getenv("MODAL_API_BASE"),
        "MODAL_HEALTH_ENDPOINT": os.getenv("MODAL_HEALTH_ENDPOINT"),
        # "RUNPOD_API_BASE": os.getenv("RUNPOD_API_BASE"),  # REMOVED
    }
