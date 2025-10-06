from fastapi import APIRouter, HTTPException, Depends
from pydantic import BaseModel
from typing import Optional
import httpx
import os

router = APIRouter(prefix="/executions")

# Get runner URL from environment
RUNNER_URL = os.getenv("RUNNER_URL", "https://beacon-runner-change-me.fly.dev")


class RetryQuestionRequest(BaseModel):
    region: str
    question_index: int


class RetryQuestionResponse(BaseModel):
    execution_id: str
    region: str
    question_index: int
    status: str
    retry_attempt: int
    updated_at: str
    result: Optional[dict] = None
    error: Optional[str] = None


@router.post("/{execution_id}/retry-question", response_model=RetryQuestionResponse)
async def retry_question(execution_id: str, req: RetryQuestionRequest):
    """
    Retry a single failed question for a specific region.
    
    This endpoint proxies the request to the runner app which handles:
    - Validation that execution exists
    - Checking question actually failed
    - Re-running inference for the specific question/region
    - Tracking retry attempts (max 3)
    - Updating execution record
    """
    
    # Validate retry attempt limit (client-side check before proxying)
    # The runner will also enforce this server-side
    
    async with httpx.AsyncClient(timeout=60.0) as client:
        try:
            # Proxy request to runner app
            runner_url = f"{RUNNER_URL}/api/v1/executions/{execution_id}/retry-question"
            
            response = await client.post(
                runner_url,
                json={
                    "region": req.region,
                    "question_index": req.question_index
                },
                headers={
                    "Content-Type": "application/json",
                    "Accept": "application/json"
                }
            )
            
            if response.status_code == 404:
                raise HTTPException(
                    status_code=404,
                    detail=f"Execution {execution_id} not found"
                )
            
            if response.status_code == 400:
                error_data = response.json()
                raise HTTPException(
                    status_code=400,
                    detail=error_data.get("error", "Invalid retry request")
                )
            
            if response.status_code == 429:
                raise HTTPException(
                    status_code=429,
                    detail="Max retry attempts reached for this question"
                )
            
            if not response.is_success:
                raise HTTPException(
                    status_code=response.status_code,
                    detail=f"Runner API error: {response.text}"
                )
            
            return response.json()
            
        except httpx.TimeoutException:
            raise HTTPException(
                status_code=504,
                detail="Retry request timed out. The question may still be processing."
            )
        except httpx.RequestError as e:
            raise HTTPException(
                status_code=503,
                detail=f"Failed to connect to runner service: {str(e)}"
            )


@router.post("/{execution_id}/retry-all-failed")
async def retry_all_failed(execution_id: str):
    """
    Retry all failed questions for a given execution across all regions.
    
    This is a convenience endpoint that batches multiple retry requests.
    """
    
    async with httpx.AsyncClient(timeout=120.0) as client:
        try:
            # Get execution details to find failed questions
            runner_url = f"{RUNNER_URL}/api/v1/executions/{execution_id}/details"
            
            response = await client.get(runner_url)
            
            if response.status_code == 404:
                raise HTTPException(
                    status_code=404,
                    detail=f"Execution {execution_id} not found"
                )
            
            if not response.is_success:
                raise HTTPException(
                    status_code=response.status_code,
                    detail=f"Failed to fetch execution details: {response.text}"
                )
            
            execution_data = response.json()
            
            # Find all failed questions
            failed_questions = []
            # TODO: Parse execution_data to find failed questions
            # This depends on the actual structure of execution data from runner
            
            # For now, proxy to runner's batch retry endpoint if it exists
            retry_url = f"{RUNNER_URL}/api/v1/executions/{execution_id}/retry-all-failed"
            retry_response = await client.post(retry_url)
            
            if retry_response.is_success:
                return retry_response.json()
            else:
                raise HTTPException(
                    status_code=retry_response.status_code,
                    detail=f"Batch retry failed: {retry_response.text}"
                )
                
        except httpx.TimeoutException:
            raise HTTPException(
                status_code=504,
                detail="Batch retry request timed out"
            )
        except httpx.RequestError as e:
            raise HTTPException(
                status_code=503,
                detail=f"Failed to connect to runner service: {str(e)}"
            )
