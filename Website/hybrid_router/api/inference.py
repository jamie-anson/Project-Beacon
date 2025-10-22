"""Inference endpoints"""

from fastapi import APIRouter, BackgroundTasks, Query, Request
from typing import Optional
import uuid
import logging

# Sentry SDK (optional)
try:
    import sentry_sdk
    SENTRY_AVAILABLE = True
except ImportError:
    SENTRY_AVAILABLE = False
    sentry_sdk = None

from ..models import InferenceRequest, InferenceResponse
from ..core.region_queue import QueuedJob, queue_manager

router = APIRouter()
logger = logging.getLogger(__name__)


@router.post("/inference", response_model=InferenceResponse)
async def inference_endpoint(inference_request: InferenceRequest, background_tasks: BackgroundTasks, request: Request):
    """Main inference endpoint"""
    # Get router_instance and tracer from app state
    router_instance = request.app.state.router_instance
    db_tracer = getattr(request.app.state, 'db_tracer', None)
    
    # 🔍 SENTRY: Start transaction (if available)
    transaction = None
    if SENTRY_AVAILABLE:
        transaction = sentry_sdk.start_transaction(op="inference", name="router.inference")
        transaction.__enter__()
        transaction.set_tag("model", inference_request.model)
        transaction.set_tag("region", inference_request.region_preference or "auto")
        
        # 🔍 SENTRY: Add breadcrumb
        sentry_sdk.add_breadcrumb(
            category="inference",
            message="Inference request received",
            level="info",
            data={
                "model": inference_request.model,
                "region": inference_request.region_preference,
                "has_prompt": bool(inference_request.prompt),
            }
        )
    
    # 🔍 TRACING: Start span
    trace_id = str(uuid.uuid4())
    span_id = None
    if db_tracer:
        span_id = await db_tracer.start_span(
            trace_id=trace_id,
            parent_span_id=None,
            service="router",
            operation="inference_request",
            metadata={
                "model": inference_request.model,
                "region": inference_request.region_preference,
            }
        )
    
    # Run health checks in background
    background_tasks.add_task(router_instance.health_check_providers)
    
    try:
        result = await router_instance.run_inference(inference_request)
        
        # 🔍 TRACING: Complete span
        if db_tracer and span_id:
            await db_tracer.complete_span(span_id, "completed")
        
        # 🔍 SENTRY: Mark success (if available)
        if SENTRY_AVAILABLE and transaction:
            transaction.set_status("ok")
        
        return result
        
    except Exception as e:
        # 🔍 TRACING: Complete span with error
        if db_tracer and span_id:
            await db_tracer.complete_span(
                span_id, 
                "failed",
                error_message=str(e),
                error_type=type(e).__name__
            )
        
        # 🔍 SENTRY: Capture error with context (if available)
        if SENTRY_AVAILABLE:
            sentry_sdk.capture_exception(e)
            if transaction:
                transaction.set_status("internal_error")
        raise
    finally:
        # Close sentry transaction if it was started
        if SENTRY_AVAILABLE and transaction:
            transaction.__exit__(None, None, None)


@router.post("/v1/inference", response_model=InferenceResponse)
async def inference_endpoint_v1(inference_request: InferenceRequest, background_tasks: BackgroundTasks, request: Request):
    """Legacy v1 inference endpoint - alias for /inference"""
    # Get router_instance from app state instead of circular import
    router_instance = request.app.state.router_instance
    
    # Run health checks in background
    background_tasks.add_task(router_instance.health_check_providers)
    
    return await router_instance.run_inference(inference_request)


@router.post("/inference/queued")
async def inference_queued(inference_request: InferenceRequest, fastapi_request: Request, use_queue: bool = Query(default=True)):
    """Queue-based inference endpoint for GPU resource management
    
    This endpoint adds the inference request to a region queue instead of
    executing immediately. Useful for managing GPU limits and fair resource allocation.
    
    Returns:
        job_id: Unique identifier to check job status
        queue_position: Position in queue
        estimated_wait: Estimated wait time in seconds
    """
    # Get router_instance from app state instead of circular import
    router_instance = fastapi_request.app.state.router_instance
    
    if not use_queue:
        # Fallback to direct execution
        return await router_instance.run_inference(inference_request)
    
    # Create queued job
    job_id = f"inference-{uuid.uuid4().hex[:12]}"
    region = inference_request.region_preference or "US"
    
    queued_job = QueuedJob(
        job_id=job_id,
        model=inference_request.model,
        prompt=inference_request.prompt,
        temperature=inference_request.temperature,
        max_tokens=inference_request.max_tokens,
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
