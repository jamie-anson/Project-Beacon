"""Inference endpoints"""

from fastapi import APIRouter, BackgroundTasks

from ..models import InferenceRequest, InferenceResponse

router = APIRouter()


@router.post("/inference", response_model=InferenceResponse)
async def inference_endpoint(request: InferenceRequest, background_tasks: BackgroundTasks):
    """Main inference endpoint"""
    from ..main import router_instance
    
    # Run health checks in background
    background_tasks.add_task(router_instance.health_check_providers)
    
    return await router_instance.run_inference(request)
