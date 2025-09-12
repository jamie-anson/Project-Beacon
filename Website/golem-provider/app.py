from fastapi import FastAPI
from pydantic import BaseModel
import os
import time

app = FastAPI(title="Beacon Golem Provider Health/Inference", version="0.1.0")

REGION = os.getenv("BEACON_REGION", "eu-west")

class InferenceRequest(BaseModel):
    model: str
    prompt: str
    temperature: float = 0.1
    max_tokens: int = 256

class InferenceResponse(BaseModel):
    success: bool
    response: str | None = None
    error: str | None = None
    metadata: dict = {}

@app.get("/health")
def health():
    return {
        "status": "healthy",
        "region": REGION,
        "timestamp": time.time(),
        "service": "beacon-golem-provider"
    }

@app.post("/inference", response_model=InferenceResponse)
def inference(req: InferenceRequest):
    # Minimal placeholder implementation to make this provider router-ready.
    # Replace with real delegation (e.g., local Ollama or external GPU backend) if needed.
    try:
        reply = f"[provider:{REGION}] echo: {req.prompt[:200]}"
        return InferenceResponse(success=True, response=reply, metadata={
            "model": req.model,
            "temperature": req.temperature,
            "max_tokens": req.max_tokens,
            "region": REGION
        })
    except Exception as e:
        return InferenceResponse(success=False, error=str(e), metadata={"region": REGION})
