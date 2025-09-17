"""Request and response models"""

from typing import Dict, Any, Optional
from pydantic import BaseModel


class InferenceRequest(BaseModel):
    model: str
    prompt: str
    temperature: float = 0.1
    max_tokens: int = 512
    region_preference: Optional[str] = None
    cost_priority: bool = True


class InferenceResponse(BaseModel):
    success: bool
    response: Optional[str] = None
    error: Optional[str] = None
    provider_used: str
    inference_time: float
    cost_estimate: float
    metadata: Dict[str, Any] = {}
