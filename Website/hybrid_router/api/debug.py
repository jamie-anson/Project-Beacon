"""Debug endpoints for troubleshooting provider health and routing issues"""

import asyncio
import time
from fastapi import APIRouter, Request
from typing import Dict, Any

router = APIRouter()


@router.get("/debug/providers")
async def debug_providers(request: Request):
    """Detailed provider status including internal state"""
    router_instance = request.app.state.router_instance
    
    providers_detail = []
    for p in router_instance.providers:
        providers_detail.append({
            "name": p.name,
            "type": p.type.value,
            "endpoint": p.endpoint,
            "region": p.region,
            "healthy": p.healthy,
            "last_health_check": p.last_health_check,
            "last_health_check_ago_seconds": time.time() - p.last_health_check if p.last_health_check > 0 else None,
            "avg_latency": p.avg_latency,
            "success_rate": p.success_rate,
            "cost_per_second": p.cost_per_second,
            "max_concurrent": p.max_concurrent,
        })
    
    return {
        "total_providers": len(router_instance.providers),
        "healthy_count": len([p for p in router_instance.providers if p.healthy]),
        "providers": providers_detail,
        "timestamp": time.time(),
    }


@router.post("/debug/force-health-check")
async def force_health_check(request: Request):
    """Manually trigger health check for all providers"""
    router_instance = request.app.state.router_instance
    
    start_time = time.time()
    
    # Capture provider state before
    before_state = {p.name: p.healthy for p in router_instance.providers}
    
    # Run health checks
    await router_instance.health_check_providers()
    
    # Capture provider state after
    after_state = {p.name: p.healthy for p in router_instance.providers}
    
    duration = time.time() - start_time
    
    return {
        "success": True,
        "duration_seconds": duration,
        "before": before_state,
        "after": after_state,
        "changes": {
            name: {"from": before_state[name], "to": after_state[name]}
            for name in before_state
            if before_state[name] != after_state[name]
        },
        "timestamp": time.time(),
    }


@router.post("/debug/test-provider/{provider_name}")
async def test_provider(provider_name: str, request: Request):
    """Test a specific provider's health check logic"""
    router_instance = request.app.state.router_instance
    
    # Find the provider
    provider = next((p for p in router_instance.providers if p.name == provider_name), None)
    if not provider:
        return {
            "success": False,
            "error": f"Provider '{provider_name}' not found",
            "available_providers": [p.name for p in router_instance.providers]
        }
    
    # Capture state before
    before_healthy = provider.healthy
    before_last_check = provider.last_health_check
    
    start_time = time.time()
    error = None
    
    try:
        # Run health check for this specific provider
        await router_instance._check_provider_health(provider)
        success = True
    except Exception as e:
        success = False
        error = str(e)
    
    duration = time.time() - start_time
    
    return {
        "success": success,
        "error": error,
        "provider": provider_name,
        "duration_seconds": duration,
        "before": {
            "healthy": before_healthy,
            "last_health_check": before_last_check,
        },
        "after": {
            "healthy": provider.healthy,
            "last_health_check": provider.last_health_check,
        },
        "changed": before_healthy != provider.healthy,
        "timestamp": time.time(),
    }


@router.post("/debug/test-inference")
async def test_inference(request: Request):
    """Test inference with detailed diagnostics"""
    from ..models import InferenceRequest
    
    router_instance = request.app.state.router_instance
    
    # Create a minimal test request
    test_request = InferenceRequest(
        model="llama3.2-1b",
        prompt="test",
        temperature=0.1,
        max_tokens=5,
        region_preference="us-east",
        cost_priority=False
    )
    
    # Capture provider state
    provider_states = {
        p.name: {"healthy": p.healthy, "region": p.region}
        for p in router_instance.providers
    }
    
    # Try to select a provider
    selected_provider = router_instance.select_provider(test_request)
    
    if not selected_provider:
        return {
            "success": False,
            "error": "No provider selected",
            "provider_states": provider_states,
            "healthy_count": len([p for p in router_instance.providers if p.healthy]),
            "requested_region": test_request.region_preference,
        }
    
    # Try to run inference
    start_time = time.time()
    try:
        result = await router_instance.run_inference(test_request)
        duration = time.time() - start_time
        
        return {
            "success": result.success,
            "provider_selected": selected_provider.name,
            "provider_used": result.provider_used,
            "duration_seconds": duration,
            "error": result.error,
            "provider_states": provider_states,
        }
    except Exception as e:
        duration = time.time() - start_time
        return {
            "success": False,
            "provider_selected": selected_provider.name,
            "duration_seconds": duration,
            "error": str(e),
            "provider_states": provider_states,
        }


@router.get("/debug/health-check-history")
async def health_check_history(request: Request):
    """Show when health checks last ran for each provider"""
    router_instance = request.app.state.router_instance
    
    current_time = time.time()
    
    history = []
    for p in router_instance.providers:
        if p.last_health_check > 0:
            ago_seconds = current_time - p.last_health_check
            history.append({
                "provider": p.name,
                "healthy": p.healthy,
                "last_check_timestamp": p.last_health_check,
                "last_check_ago_seconds": ago_seconds,
                "last_check_ago_human": f"{int(ago_seconds)}s ago" if ago_seconds < 60 else f"{int(ago_seconds/60)}m ago",
            })
        else:
            history.append({
                "provider": p.name,
                "healthy": p.healthy,
                "last_check_timestamp": 0,
                "last_check_ago_seconds": None,
                "last_check_ago_human": "Never",
            })
    
    return {
        "providers": history,
        "current_timestamp": current_time,
    }


@router.get("/debug/startup-status")
async def startup_status(request: Request):
    """Check if startup health checks have completed"""
    router_instance = request.app.state.router_instance
    
    all_checked = all(p.last_health_check > 0 for p in router_instance.providers)
    any_checked = any(p.last_health_check > 0 for p in router_instance.providers)
    
    return {
        "startup_health_checks_completed": all_checked,
        "some_health_checks_completed": any_checked,
        "providers_checked": len([p for p in router_instance.providers if p.last_health_check > 0]),
        "providers_total": len(router_instance.providers),
        "providers_healthy": len([p for p in router_instance.providers if p.healthy]),
        "providers_never_checked": [
            p.name for p in router_instance.providers if p.last_health_check == 0
        ],
    }
