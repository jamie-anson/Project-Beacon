"""Request and response models"""

from typing import Dict, Any, Optional
from pydantic import BaseModel


class InferenceRequest(BaseModel):
    model: str
    prompt: str
    temperature: float = 0.1
    max_tokens: int = 500
    region_preference: Optional[str] = None
    cost_priority: bool = True
    trace_id: Optional[str] = None


class InferenceResponse(BaseModel):
    success: bool
    response: Optional[str] = None
    error: Optional[str] = None
    error_code: Optional[str] = None
    failure: Optional[Dict[str, Any]] = None
    provider_used: str
    inference_time: float
    cost_estimate: float
    metadata: Dict[str, Any] = {}
