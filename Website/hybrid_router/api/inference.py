"""Inference endpoints"""

from fastapi import APIRouter, BackgroundTasks, Query
from typing import Optional
import uuid

from ..models import InferenceRequest, InferenceResponse
from ..core.region_queue import QueuedJob, queue_manager

router = APIRouter()


@router.post("/inference", response_model=InferenceResponse)
async def inference_endpoint(request: InferenceRequest, background_tasks: BackgroundTasks):
    """Main inference endpoint"""
    from ..main import router_instance
    
    # Run health checks in background
    background_tasks.add_task(router_instance.health_check_providers)
    
    return await router_instance.run_inference(request)


@router.post("/v1/inference", response_model=InferenceResponse)
async def inference_endpoint_v1(request: InferenceRequest, background_tasks: BackgroundTasks):
    """Legacy v1 inference endpoint - alias for /inference"""
    from ..main import router_instance
    
    # Run health checks in background
    background_tasks.add_task(router_instance.health_check_providers)
    
    return await router_instance.run_inference(request)


@router.post("/inference/queued")
async def inference_queued(request: InferenceRequest, use_queue: bool = Query(default=True)):
    """Queue-based inference endpoint for GPU resource management
    
    This endpoint adds the inference request to a region queue instead of
    executing immediately. Useful for managing GPU limits and fair resource allocation.
    
    Returns:
        job_id: Unique identifier to check job status
        queue_position: Position in queue
        estimated_wait: Estimated wait time in seconds
    """
    from ..main import router_instance
    
    if not use_queue:
        # Fallback to direct execution
        return await router_instance.run_inference(request)
    
    # Create queued job
    job_id = f"inference-{uuid.uuid4().hex[:12]}"
    region = request.region_preference or "US"
    
    queued_job = QueuedJob(
        job_id=job_id,
        model=request.model,
        prompt=request.prompt,
        temperature=request.temperature,
        max_tokens=request.max_tokens,
        region=region,
    )
    
    # Add to queue
    await queue_manager.enqueue_job(queued_job)
    
    # Get queue position
    queue_size = queue_manager.get_queue_size(region)
    estimated_wait = queue_size * 30  # Rough estimate: 30s per job
    
    return {
        "success": True,
        "job_id": job_id,
        "status": "queued",
        "queue_position": queue_size,
        "estimated_wait_seconds": estimated_wait,
        "region": region,
        "message": f"Job queued in {region} region. Check status at /inference/status/{job_id}"
    }


@router.get("/inference/status/{job_id}")
async def get_inference_status(job_id: str):
    """Get status of a queued inference job
    
    Returns:
        status: queued, processing, completed, failed
        result: Inference result if completed
        error: Error message if failed
    """
    from ..core.region_queue import job_results
    
    # Check if job result exists
    if job_id in job_results:
        return {
            "job_id": job_id,
            **job_results[job_id]
        }
    
    # Check if job is currently processing
    statuses = queue_manager.get_all_statuses()
    for region_data in statuses.get("regions", {}).values():
        current_job = region_data.get("current_job")
        if current_job and current_job.get("job_id") == job_id:
            return {
                "job_id": job_id,
                "status": "processing",
                "region": region_data.get("region"),
                "started_at": current_job.get("started_at")
            }
    
    # Job not found - might be queued or doesn't exist
    return {
        "job_id": job_id,
        "status": "queued_or_not_found",
        "message": "Job is either queued, not found, or result has expired. Check queue status for details.",
        "queue_statuses": statuses
    }
