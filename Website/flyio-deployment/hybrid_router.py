"""
Hybrid routing service for Project Beacon
Routes inference requests between Golem providers and serverless GPU providers
"""

import os
import time
import json
import asyncio
import logging
from typing import Dict, Any, List, Optional
from dataclasses import dataclass
from enum import Enum

import httpx
from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel
import uvicorn

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class ProviderType(Enum):
    GOLEM = "golem"
    MODAL = "modal"
    RUNPOD = "runpod"

@dataclass
class Provider:
    name: str
    type: ProviderType
    endpoint: str
    region: str
    cost_per_second: float
    max_concurrent: int
    healthy: bool = True
    last_health_check: float = 0
    avg_latency: float = 0
    success_rate: float = 1.0

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

class HybridRouter:
    def __init__(self):
        self.providers: List[Provider] = []
        self.client = httpx.AsyncClient(timeout=30.0)
        self.setup_providers()
        
    def setup_providers(self):
        """Initialize provider configurations"""
        
        # Golem providers (baseline capacity)
        golem_endpoints = os.getenv("GOLEM_PROVIDER_ENDPOINTS", "").split(",")
        for i, endpoint in enumerate(golem_endpoints):
            if endpoint.strip():
                self.providers.append(Provider(
                    name=f"golem-{i+1}",
                    type=ProviderType.GOLEM,
                    endpoint=endpoint.strip(),
                    region=self._get_region_from_endpoint(endpoint),
                    cost_per_second=0.0001,  # Very low cost
                    max_concurrent=5
                ))
        
        # Modal serverless (burst capacity)
        modal_endpoint = os.getenv("MODAL_API_BASE")
        if modal_endpoint:
            for region in ["us-east", "eu-west", "asia-pacific"]:
                self.providers.append(Provider(
                    name=f"modal-{region}",
                    type=ProviderType.MODAL,
                    endpoint=modal_endpoint,
                    region=region,
                    cost_per_second=0.0003,  # T4 pricing
                    max_concurrent=10
                ))
        
        # RunPod serverless (cost optimization)
        runpod_endpoint = os.getenv("RUNPOD_API_BASE")
        if runpod_endpoint:
            for region in ["us-east", "eu-west", "asia-pacific"]:
                self.providers.append(Provider(
                    name=f"runpod-{region}",
                    type=ProviderType.RUNPOD,
                    endpoint=runpod_endpoint,
                    region=region,
                    cost_per_second=0.00025,  # 15% savings claimed
                    max_concurrent=8
                ))
    
    def _get_region_from_endpoint(self, endpoint: str) -> str:
        """Extract region from endpoint URL"""
        if "us-east" in endpoint or "iad" in endpoint:
            return "us-east"
        elif "eu-west" in endpoint or "ams" in endpoint:
            return "eu-west"
        elif "asia" in endpoint or "sin" in endpoint:
            return "asia-pacific"
        return "unknown"
    
    async def health_check_providers(self):
        """Check health of all providers"""
        tasks = []
        for provider in self.providers:
            tasks.append(self._check_provider_health(provider))
        
        await asyncio.gather(*tasks, return_exceptions=True)
    
    async def _check_provider_health(self, provider: Provider):
        """Check individual provider health"""
        try:
            if provider.type == ProviderType.GOLEM:
                # Simple ping for Golem providers
                response = await self.client.get(f"{provider.endpoint}/health", timeout=5.0)
                provider.healthy = response.status_code == 200
            
            elif provider.type == ProviderType.MODAL:
                # Modal health check
                health_payload = {"action": "health_check"}
                response = await self.client.post(f"{provider.endpoint}/health", json=health_payload, timeout=5.0)
                provider.healthy = response.status_code == 200
            
            elif provider.type == ProviderType.RUNPOD:
                # RunPod health check
                headers = {"Authorization": f"Bearer {os.getenv('RUNPOD_API_KEY')}"}
                response = await self.client.get(f"{provider.endpoint}/health", headers=headers, timeout=5.0)
                provider.healthy = response.status_code == 200
            
            provider.last_health_check = time.time()
            
        except Exception as e:
            logger.warning(f"Health check failed for {provider.name}: {e}")
            provider.healthy = False
    
    def select_provider(self, request: InferenceRequest) -> Optional[Provider]:
        """Select best provider based on request requirements"""
        
        # Filter healthy providers
        healthy_providers = [p for p in self.providers if p.healthy]
        
        if not healthy_providers:
            return None
        
        # Filter by region preference
        if request.region_preference:
            region_providers = [p for p in healthy_providers if p.region == request.region_preference]
            if region_providers:
                healthy_providers = region_providers
        
        # Sort by cost priority or performance priority
        if request.cost_priority:
            # Prefer Golem (lowest cost) -> RunPod -> Modal
            healthy_providers.sort(key=lambda p: (p.cost_per_second, p.avg_latency))
        else:
            # Prefer lowest latency providers
            healthy_providers.sort(key=lambda p: (p.avg_latency, p.cost_per_second))
        
        # Return first available provider with capacity
        for provider in healthy_providers:
            # Simple capacity check (in real implementation, track active requests)
            if provider.max_concurrent > 0:  # Simplified capacity check
                return provider
        
        return healthy_providers[0] if healthy_providers else None
    
    async def run_inference(self, request: InferenceRequest) -> InferenceResponse:
        """Execute inference request on selected provider"""
        
        provider = self.select_provider(request)
        if not provider:
            raise HTTPException(status_code=503, detail="No healthy providers available")
        
        start_time = time.time()
        
        try:
            if provider.type == ProviderType.GOLEM:
                result = await self._run_golem_inference(provider, request)
            elif provider.type == ProviderType.MODAL:
                result = await self._run_modal_inference(provider, request)
            elif provider.type == ProviderType.RUNPOD:
                result = await self._run_runpod_inference(provider, request)
            else:
                raise ValueError(f"Unknown provider type: {provider.type}")
            
            inference_time = time.time() - start_time
            cost_estimate = inference_time * provider.cost_per_second
            
            # Update provider metrics
            provider.avg_latency = (provider.avg_latency * 0.9) + (inference_time * 0.1)
            provider.success_rate = (provider.success_rate * 0.9) + (0.1 if result["success"] else 0.0)
            
            return InferenceResponse(
                success=result["success"],
                response=result.get("response"),
                error=result.get("error"),
                provider_used=provider.name,
                inference_time=inference_time,
                cost_estimate=cost_estimate,
                metadata={
                    "provider_type": provider.type.value,
                    "region": provider.region,
                    "model": request.model
                }
            )
            
        except Exception as e:
            inference_time = time.time() - start_time
            logger.error(f"Inference failed on {provider.name}: {e}")
            
            return InferenceResponse(
                success=False,
                error=str(e),
                provider_used=provider.name,
                inference_time=inference_time,
                cost_estimate=0.0,
                metadata={"provider_type": provider.type.value, "region": provider.region}
            )
    
    async def _run_golem_inference(self, provider: Provider, request: InferenceRequest) -> Dict[str, Any]:
        """Run inference on Golem provider"""
        payload = {
            "model": request.model,
            "prompt": request.prompt,
            "temperature": request.temperature,
            "max_tokens": request.max_tokens
        }
        
        response = await self.client.post(f"{provider.endpoint}/inference", json=payload)
        
        if response.status_code == 200:
            return response.json()
        else:
            return {"success": False, "error": f"HTTP {response.status_code}: {response.text}"}
    
    async def _run_modal_inference(self, provider: Provider, request: InferenceRequest) -> Dict[str, Any]:
        """Run inference on Modal provider"""
        payload = {
            "model": request.model,
            "prompt": request.prompt,
            "temperature": request.temperature,
            "max_tokens": request.max_tokens
        }
        
        headers = {"Authorization": f"Bearer {os.getenv('MODAL_API_TOKEN')}"}
        response = await self.client.post(provider.endpoint, json=payload, headers=headers)
        
        if response.status_code == 200:
            return response.json()
        else:
            return {"success": False, "error": f"HTTP {response.status_code}: {response.text}"}
    
    async def _run_runpod_inference(self, provider: Provider, request: InferenceRequest) -> Dict[str, Any]:
        """Run inference on RunPod provider"""
        payload = {
            "input": {
                "model": request.model,
                "prompt": request.prompt,
                "temperature": request.temperature,
                "max_tokens": request.max_tokens
            }
        }
        
        headers = {"Authorization": f"Bearer {os.getenv('RUNPOD_API_KEY')}"}
        response = await self.client.post(f"{provider.endpoint}/run", json=payload, headers=headers)
        
        if response.status_code == 200:
            result = response.json()
            return {
                "success": True,
                "response": result.get("output", {}).get("response", ""),
                "metadata": result.get("output", {})
            }
        else:
            return {"success": False, "error": f"HTTP {response.status_code}: {response.text}"}

# FastAPI app
app = FastAPI(title="Project Beacon Hybrid Router", version="1.0.0")
router = HybridRouter()

@app.on_event("startup")
async def startup_event():
    """Initialize router on startup"""
    logger.info("Starting Project Beacon Hybrid Router...")
    await router.health_check_providers()
    logger.info(f"Initialized with {len(router.providers)} providers")

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    healthy_providers = [p for p in router.providers if p.healthy]
    return {
        "status": "healthy",
        "timestamp": time.time(),
        "providers_total": len(router.providers),
        "providers_healthy": len(healthy_providers),
        "regions": list(set(p.region for p in healthy_providers))
    }

@app.post("/inference", response_model=InferenceResponse)
async def inference_endpoint(request: InferenceRequest, background_tasks: BackgroundTasks):
    """Main inference endpoint"""
    
    # Run health checks in background
    background_tasks.add_task(router.health_check_providers)
    
    return await router.run_inference(request)

@app.get("/providers")
async def list_providers():
    """List all providers and their status"""
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
            for p in router.providers
        ]
    }

@app.get("/metrics")
async def get_metrics():
    """Get routing metrics"""
    healthy_providers = [p for p in router.providers if p.healthy]
    
    return {
        "total_providers": len(router.providers),
        "healthy_providers": len(healthy_providers),
        "avg_latency": sum(p.avg_latency for p in healthy_providers) / len(healthy_providers) if healthy_providers else 0,
        "avg_success_rate": sum(p.success_rate for p in healthy_providers) / len(healthy_providers) if healthy_providers else 0,
        "cost_range": {
            "min": min(p.cost_per_second for p in healthy_providers) if healthy_providers else 0,
            "max": max(p.cost_per_second for p in healthy_providers) if healthy_providers else 0
        }
    }

if __name__ == "__main__":
    port = int(os.getenv("PORT", 8080))
    uvicorn.run(app, host="0.0.0.0", port=port)
