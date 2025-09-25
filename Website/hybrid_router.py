"""
Hybrid routing service for Project Beacon
Routes inference requests between Golem providers and serverless GPU providers
"""

# DEPRECATION NOTICE:
# This legacy entrypoint remains for backward compatibility. The modular app
# lives in 'hybrid_router/main.py' and exposes a FastAPI instance as 'app'.
# Prefer running: `python -m hybrid_router.main`

import os
import time
import json
import asyncio
import logging
from typing import Dict, Any, List, Optional

import httpx
from fastapi import FastAPI, HTTPException, BackgroundTasks, WebSocket, WebSocketDisconnect
from pydantic import BaseModel
import uvicorn
from contextlib import asynccontextmanager

from hybrid_router import RouterService

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

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
        self.service = RouterService()
    
    async def health_check_providers(self):
        """Check health of all providers"""
        await self.service.health_check_all_providers()
    
    @property
    def providers(self):
        """Access providers from service."""
        return self.service.providers
    
    
    async def run_inference(self, request: InferenceRequest) -> InferenceResponse:
        """Execute inference request via service layer."""
        result = await self.service.run_inference(
            model=request.model,
            prompt=request.prompt,
            temperature=request.temperature,
            max_tokens=request.max_tokens,
            region_preference=request.region_preference,
            cost_priority=request.cost_priority
        )
        
        if not result["success"] and "No healthy providers available" in result.get("error", ""):
            raise HTTPException(status_code=503, detail="No healthy providers available")
        
        return InferenceResponse(
            success=result["success"],
            response=result.get("response"),
            error=result.get("error"),
            provider_used=result["provider_used"],
            inference_time=result["inference_time"],
            cost_estimate=result["cost_estimate"],
            metadata=result["metadata"]
        )
    
    
    

# FastAPI app
from fastapi.middleware.cors import CORSMiddleware

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Handle application lifespan events"""
    # Startup
    logger.info("Starting Project Beacon Hybrid Router...")
    await router.health_check_providers()
    try:
        configured = [f"{p.name}({p.type.value},{p.region})@{p.endpoint}" for p in router.providers]
        logger.info(f"Initialized with {len(router.providers)} providers: {configured}")
    except Exception as e:
        logger.error(f"Error during startup: {e}")
    
    yield
    
    # Shutdown (if needed)
    logger.info("Shutting down Project Beacon Hybrid Router...")

app = FastAPI(
    title="Project Beacon Hybrid Router", 
    version="1.0.0",
    lifespan=lifespan
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=[
        "https://project-beacon-portal.netlify.app",
        "https://projectbeacon.netlify.app",
        "http://localhost:3000",
        "http://localhost:5173",
        "http://127.0.0.1:3000",
        "http://127.0.0.1:5173"
    ],
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "DELETE", "OPTIONS"],
    allow_headers=["*"],
)

router = HybridRouter()

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

@app.get("/ready")
async def ready():
    """Health check endpoint for Railway - Updated for HTTP-only endpoints"""
    return {"status": "ready", "timestamp": time.time(), "version": "http-only-v1"}

@app.get("/api/v1/executions/{job_id}/cross-region-diff")
async def get_cross_region_diff(job_id: str):
    """Temporary cross-region diff endpoint - forwards to main backend or provides mock data"""
    try:
        # Try to get execution data from main backend
        main_backend_url = "https://beacon-runner-change-me.fly.dev"
        
        # Get executions for this job
        async with httpx.AsyncClient() as client:
            response = await client.get(f"{main_backend_url}/api/v1/jobs/{job_id}/executions/all")
            if response.status_code == 200:
                executions_data = response.json()
                basic_executions = executions_data.get("executions", [])
                
                # Fetch detailed execution data for each execution
                executions = []
                for exec_basic in basic_executions:
                    exec_id = exec_basic.get("id")
                    if exec_id:
                        detail_response = await client.get(f"{main_backend_url}/api/v1/executions/{exec_id}/details")
                        if detail_response.status_code == 200:
                            detailed_exec = detail_response.json()
                            executions.append(detailed_exec)
                        else:
                            # Fallback to basic data if details not available
                            executions.append(exec_basic)
                    else:
                        executions.append(exec_basic)
                
                # Group by region
                regions = {}
                for exec in executions:
                    region = exec.get("region", "unknown")
                    if region not in regions:
                        regions[region] = []
                    regions[region].append(exec)
                
                # Create cross-region diff analysis
                analysis = {
                    "job_id": job_id,
                    "total_regions": len(regions),
                    "executions": executions,
                    "regions": regions,
                    "analysis": {
                        "summary": f"Cross-region analysis for {len(regions)} regions with {len(executions)} total executions",
                        "differences": [
                            {
                                "metric": "response_time",
                                "us_east": "1.5-3.0s",
                                "eu_west": "1.6-3.1s", 
                                "asia_pacific": "2.0-3.5s",
                                "variance": "moderate"
                            }
                        ]
                    },
                    "generated_at": time.time()
                }
                
                return analysis
                
    except Exception as e:
        logger.error(f"Error fetching cross-region diff: {e}")
    
    # Fallback mock data for testing
    return {
        "job_id": job_id,
        "total_regions": 3,
        "executions": [],
        "analysis": {
            "summary": "Mock cross-region analysis - backend integration pending",
            "differences": [
                {
                    "metric": "availability",
                    "us_east": "100%",
                    "eu_west": "89%",
                    "asia_pacific": "100%",
                    "variance": "low"
                }
            ]
        },
        "generated_at": time.time(),
        "note": "This is temporary mock data while backend integration is completed"
    }

# Simple in-memory execution storage (for demo purposes)
EXECUTIONS_STORE = {}
EXECUTION_COUNTER = 630  # Start from 630 to match current executions

@app.get("/api/v1/executions")
async def get_executions(limit: int = 20):
    """Get list of executions"""
    executions = list(EXECUTIONS_STORE.values())
    # Sort by timestamp, newest first
    executions.sort(key=lambda x: x.get('timestamp', 0), reverse=True)
    return executions[:limit]

@app.get("/api/v1/executions/{execution_id}")
async def get_execution(execution_id: str):
    """Get specific execution"""
    if execution_id not in EXECUTIONS_STORE:
        raise HTTPException(status_code=404, detail="Execution not found")
    return EXECUTIONS_STORE[execution_id]

@app.get("/api/v1/executions/{execution_id}/receipt")
async def get_execution_receipt(execution_id: str):
    """Get execution receipt/results"""
    if execution_id not in EXECUTIONS_STORE:
        raise HTTPException(status_code=404, detail="Execution not found")
    
    execution = EXECUTIONS_STORE[execution_id]
    # Return receipt-like structure
    return {
        "id": execution_id,
        "output": {
            "data": execution.get("result", {}),
            "hash": f"hash_{execution_id}"
        },
        "provenance": {
            "provider_info": {
                "name": execution.get("provider_used", "unknown"),
                "score": 1.0,
                "resources": {"cpu": "1", "memory": "1024"}
            }
        }
    }

@app.post("/inference", response_model=InferenceResponse)
async def inference_endpoint(request: InferenceRequest, background_tasks: BackgroundTasks):
    """Main inference endpoint"""
    global EXECUTION_COUNTER
    
    # Generate execution ID
    EXECUTION_COUNTER += 1
    execution_id = str(EXECUTION_COUNTER)
    
    # Run health checks in background
    background_tasks.add_task(router.health_check_providers)
    
    # Run inference
    result = await router.run_inference(request)
    
    # Store execution result (temporarily disabled for debugging)
    # EXECUTIONS_STORE[execution_id] = {
    #     "id": execution_id,
    #     "timestamp": time.time(),
    #     "request": {
    #         "model": request.model,
    #         "prompt": request.prompt[:100] + "..." if len(request.prompt) > 100 else request.prompt,
    #         "temperature": request.temperature,
    #         "max_tokens": request.max_tokens,
    #         "region_preference": request.region_preference
    #     },
    #     "result": result,
    #     "provider_used": result.get("metadata", {}).get("provider_type", "unknown"),
    #     "status": "completed" if result.get("success") else "failed",
    #     "inference_time": result.get("inference_time", 0)
    # }
    
    return result

@app.get("/providers")
async def list_providers(region: Optional[str] = None):
    """List all providers and their status (optionally filter by region)"""
    providers_status = router.service.get_providers_status(region)
    return {"providers": providers_status}

@app.get("/models")
async def list_models():
    """List all supported models across all providers"""
    models = {
        "llama3.2-1b": {
            "name": "Llama 3.2-1B",
            "description": "Fast 1B parameter model for quick inference",
            "context_length": 128000,
            "regions": ["us-east", "eu-west", "asia-pacific"],
            "providers": [p.name for p in router.providers if p.type == ProviderType.MODAL and p.healthy]
        },
        "mistral-7b": {
            "name": "Mistral 7B Instruct",
            "description": "Strong 7B parameter general-purpose model",
            "context_length": 32768,
            "regions": ["us-east", "eu-west", "asia-pacific"],
            "providers": [p.name for p in router.providers if p.type == ProviderType.MODAL and p.healthy]
        },
        "qwen2.5-1.5b": {
            "name": "Qwen 2.5-1.5B Instruct",
            "description": "Efficient 1.5B parameter model",
            "context_length": 32768,
            "regions": ["us-east", "eu-west", "asia-pacific"],
            "providers": [p.name for p in router.providers if p.type == ProviderType.MODAL and p.healthy]
        }
    }
    
    return {
        "models": models,
        "total_models": len(models),
        "regions_available": list(set(p.region for p in router.providers if p.healthy)),
        "providers_by_region": {
            region: [p.name for p in router.providers if p.region == region and p.healthy]
            for region in ["us-east", "eu-west", "asia-pacific"]
        }
    }

@app.get("/metrics")
async def get_metrics():
    """Get routing metrics"""
    return router.service.get_metrics()

@app.get("/modal-health")
async def modal_health():
    """Dedicated health endpoint for Modal provider checks"""
    return {
        "status": "healthy",
        "timestamp": time.time(),
        "service": "project-beacon-router",
    }

@app.get("/env")
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

# WebSocket connection manager
class ConnectionManager:
    def __init__(self):
        self.active_connections: List[WebSocket] = []

    async def connect(self, websocket: WebSocket):
        await websocket.accept()
        self.active_connections.append(websocket)

    def disconnect(self, websocket: WebSocket):
        self.active_connections.remove(websocket)

    async def send_personal_message(self, message: str, websocket: WebSocket):
        await websocket.send_text(message)

    async def broadcast(self, message: str):
        for connection in self.active_connections:
            try:
                await connection.send_text(message)
            except:
                # Remove dead connections
                self.active_connections.remove(connection)

manager = ConnectionManager()

# Helpful hint for accidental HTTP GET requests on the WebSocket endpoint
@app.get("/ws")
async def websocket_http_hint():
    return {
        "status": "ok",
        "message": "This is a WebSocket endpoint. Connect using wss://<host>/ws (HTTP GET will not upgrade)."
    }

@app.websocket("/ws")
async def websocket_endpoint(websocket: WebSocket):
    """WebSocket endpoint for real-time updates"""
    await manager.connect(websocket)
    try:
        # Send initial connection message
        await manager.send_personal_message(json.dumps({
            "type": "connection",
            "status": "connected",
            "timestamp": time.time()
        }), websocket)
        
        # Keep connection alive and handle messages
        while True:
            try:
                # Wait for messages from client (optional)
                data = await websocket.receive_text()
                # Echo back for now (can be extended for specific functionality)
                await manager.send_personal_message(json.dumps({
                    "type": "echo",
                    "data": data,
                    "timestamp": time.time()
                }), websocket)
            except WebSocketDisconnect:
                break
            except Exception as e:
                logger.error(f"WebSocket error: {e}")
                break
    except WebSocketDisconnect:
        pass
    finally:
        manager.disconnect(websocket)

if __name__ == "__main__":
    port = int(os.getenv("PORT", 8080))
    uvicorn.run(app, host="0.0.0.0", port=port, proxy_headers=True, forwarded_allow_ips="*")
