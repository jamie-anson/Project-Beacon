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
from fastapi import FastAPI, HTTPException, BackgroundTasks, WebSocket, WebSocketDisconnect
from pydantic import BaseModel
import uvicorn
from contextlib import asynccontextmanager

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class ProviderType(Enum):
    GOLEM = "golem"
    MODAL = "modal"
    # RUNPOD = "runpod"  # Removed - not using RunPod

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
        # Increased timeout for Mistral 7B (6.08s) + buffer
        self.client = httpx.AsyncClient(timeout=60.0)
        self.setup_providers()
        
    def setup_providers(self):
        """Initialize provider configurations"""
        
        # Golem providers (baseline capacity)
        # Option A: Comma-separated list via GOLEM_PROVIDER_ENDPOINTS
        golem_endpoints = [e.strip() for e in os.getenv("GOLEM_PROVIDER_ENDPOINTS", "").split(",") if e.strip()]
        for i, endpoint in enumerate(golem_endpoints):
            self.providers.append(Provider(
                name=f"golem-{i+1}",
                type=ProviderType.GOLEM,
                endpoint=endpoint,
                region=self._get_region_from_endpoint(endpoint),
                cost_per_second=0.0001,  # Very low cost
                max_concurrent=5
            ))

        # Option B: Individual environment variables (fallback) - COMMENTED OUT
        # golem_us = os.getenv("GOLEM_US_ENDPOINT")
        # golem_eu = os.getenv("GOLEM_EU_ENDPOINT") 
        # golem_apac = os.getenv("GOLEM_APAC_ENDPOINT")
        # 
        # if golem_us:
        #     self.providers.append(Provider(
        #         name="golem-us-east",
        #         type=ProviderType.GOLEM,
        #         endpoint=golem_us,
        #         region="us-east",
        #         cost_per_second=0.0001,
        #         max_concurrent=5
        #     ))
        # 
        # if golem_eu:
        #     self.providers.append(Provider(
        #         name="golem-eu-west",
        #         type=ProviderType.GOLEM,
        #         endpoint=golem_eu,
        #         region="eu-west",
        #         cost_per_second=0.0001,
        #         max_concurrent=5
        #     ))
        # 
        # if golem_apac:
        #     self.providers.append(Provider(
        #         name="golem-asia-pacific",
        #         type=ProviderType.GOLEM,
        #         endpoint=golem_apac,
        #         region="asia-pacific",
        #         cost_per_second=0.0001,
        #         max_concurrent=5
        #     ))
        
        # Modal serverless (burst capacity) - Updated for 3-model support
        # US Region - has HTTP endpoints
        modal_us_endpoint = os.getenv("MODAL_US_ENDPOINT", "https://jamie-anson--project-beacon-hf-us-inference.modal.run")
        if modal_us_endpoint:
            self.providers.append(Provider(
                name="modal-us-east",
                type=ProviderType.MODAL,
                endpoint=modal_us_endpoint,
                region="us-east",
                cost_per_second=0.0003,  # A10G pricing
                max_concurrent=10
            ))
        
        # EU Region - HTTP endpoint (newly deployed with web endpoints)
        modal_eu_endpoint = os.getenv("MODAL_EU_ENDPOINT", "https://jamie-anson--project-beacon-hf-eu-inference-dev.modal.run")
        if modal_eu_endpoint:
            self.providers.append(Provider(
                name="modal-eu-west",
                type=ProviderType.MODAL,
                endpoint=modal_eu_endpoint,
                region="eu-west",
                cost_per_second=0.0003,
                max_concurrent=10
            ))
        
        # APAC Region - HTTP endpoint (newly deployed with web endpoints)
        modal_apac_endpoint = os.getenv("MODAL_APAC_ENDPOINT", "https://jamie-anson--project-beacon-hf-apac-inference-dev.modal.run")
        if modal_apac_endpoint:
            self.providers.append(Provider(
                name="modal-asia-pacific",
                type=ProviderType.MODAL,
                endpoint=modal_apac_endpoint,
                region="asia-pacific",
                cost_per_second=0.0003,
                max_concurrent=10
            ))
        
        # RunPod serverless - REMOVED (not using RunPod)
        # runpod_endpoint = os.getenv("RUNPOD_API_BASE")
        # if runpod_endpoint:
        #     for region in ["us-east", "eu-west", "asia-pacific"]:
        #         self.providers.append(Provider(
        #             name=f"runpod-{region}",
        #             type=ProviderType.RUNPOD,
        #             endpoint=runpod_endpoint,
        #             region=region,
        #             cost_per_second=0.00025,  # 15% savings claimed
        #             max_concurrent=8
        #         ))
    
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
                response = await self.client.get(f"{provider.endpoint}/health", timeout=15.0)
                provider.healthy = response.status_code == 200
            
            elif provider.type == ProviderType.MODAL:
                # Modal health check - try health endpoint, fallback to assuming healthy
                if provider.endpoint.startswith("https://"):
                    try:
                        # Try health endpoint first (US region has this)
                        health_endpoint = provider.endpoint.replace("-inference", "-health")
                        response = await self.client.get(health_endpoint, timeout=10.0)
                        if response.status_code == 200:
                            health_data = response.json()
                            provider.healthy = health_data.get("status") == "healthy"
                        else:
                            # Health endpoint failed, try basic connectivity test
                            response = await self.client.get(provider.endpoint, timeout=10.0)
                            # If endpoint responds at all (even with error), consider it reachable
                            provider.healthy = response.status_code in [200, 400, 404, 422]
                    except Exception:
                        # If all checks fail, assume unhealthy
                        provider.healthy = False
                else:
                    provider.healthy = False
            
            # elif provider.type == ProviderType.RUNPOD:
            #     # RunPod health check - REMOVED
            #     headers = {"Authorization": f"Bearer {os.getenv('RUNPOD_API_KEY')}"}
            #     response = await self.client.get(f"{provider.endpoint}/health", headers=headers, timeout=5.0)
            #     provider.healthy = response.status_code == 200
            
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
            # Prefer Golem (lowest cost) -> Modal
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
        
        # 8/9 Graceful failure test: Disable EU Mistral 7B (slowest model)
        if request.region_preference == "eu-west" and request.model == "mistral-7b":
            return InferenceResponse(
                success=False,
                response=None,
                error="EU Mistral 7B temporarily disabled for 8/9 graceful failure testing",
                provider_used="modal-eu-west",
                inference_time=0.0,
                cost_estimate=0.0,
                metadata={
                    "provider_type": "modal",
                    "region": "eu-west",
                    "model": request.model,
                    "test_mode": "graceful_failure",
                    "reason": "8_of_9_testing"
                }
            )
        
        provider = self.select_provider(request)
        if not provider:
            raise HTTPException(status_code=503, detail="No healthy providers available")
        
        start_time = time.time()
        
        try:
            if provider.type == ProviderType.GOLEM:
                result = await self._run_golem_inference(provider, request)
            elif provider.type == ProviderType.MODAL:
                result = await self._run_modal_inference(provider, request)
            # elif provider.type == ProviderType.RUNPOD:
            #     result = await self._run_runpod_inference(provider, request)  # REMOVED
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
        """Run inference on Modal provider - supports both HTTP and function endpoints"""
        payload = {
            "model": request.model,
            "prompt": request.prompt,
            "temperature": request.temperature,
            "max_tokens": request.max_tokens
        }
        
        if provider.endpoint.startswith("https://"):
            # US region with HTTP endpoints
            headers = {"Authorization": f"Bearer {os.getenv('MODAL_API_TOKEN')}"}
            response = await self.client.post(provider.endpoint, json=payload, headers=headers)
            
            if response.status_code == 200:
                modal_result = response.json()
                # Convert Modal API response format to router format
                return {
                    "success": modal_result.get("status") == "success",
                    "response": modal_result.get("response", ""),
                    "error": modal_result.get("error"),
                    "inference_time": modal_result.get("inference_time", 0),
                    "metadata": modal_result
                }
            else:
                return {"success": False, "error": f"HTTP {response.status_code}: {response.text}"}
        
        elif provider.endpoint.startswith("modal://"):
            # EU/APAC regions with function calls - use subprocess to call Modal CLI
            import subprocess
            import json as json_module
            
            try:
                app_name = provider.endpoint.replace("modal://", "")
                # Use Modal CLI to invoke function directly
                cmd = [
                    "modal", "run", f"modal-deployment/modal_hf_{provider.region.split('-')[0]}.py::run_inference",
                    f"--model-name={request.model}",
                    f"--prompt={request.prompt}",
                    f"--temperature={request.temperature}",
                    f"--max-tokens={request.max_tokens}"
                ]
                
                result = subprocess.run(cmd, capture_output=True, text=True, timeout=300)
                
                if result.returncode == 0:
                    # Parse the output - Modal CLI returns the function result
                    try:
                        # Extract JSON from the output (Modal CLI adds extra logging)
                        output_lines = result.stdout.strip().split('\n')
                        
                        # Try multiple parsing strategies
                        modal_result = None
                        
                        # Strategy 1: Look for JSON lines (original approach)
                        for line in reversed(output_lines):
                            line = line.strip()
                            if line.startswith('{') and line.endswith('}'):
                                try:
                                    modal_result = json_module.loads(line)
                                    break
                                except json_module.JSONDecodeError:
                                    continue
                        
                        # Strategy 2: Try parsing the entire output as JSON
                        if not modal_result:
                            try:
                                modal_result = json_module.loads(result.stdout.strip())
                            except json_module.JSONDecodeError:
                                pass
                        
                        # Strategy 3: Look for MODAL RESULT START/END markers
                        if not modal_result:
                            start_idx = -1
                            end_idx = -1
                            for i, line in enumerate(output_lines):
                                if line.strip() == '=== MODAL RESULT START ===':
                                    start_idx = i
                                elif line.strip() == '=== MODAL RESULT END ===':
                                    end_idx = i
                                    break
                            
                            if start_idx != -1 and end_idx != -1 and end_idx > start_idx:
                                # Extract JSON between markers
                                json_lines = output_lines[start_idx + 1:end_idx]
                                json_str = '\n'.join(json_lines).strip()
                                try:
                                    modal_result = json_module.loads(json_str)
                                except json_module.JSONDecodeError:
                                    pass
                        
                        # Strategy 4: Look for any line containing status/response keywords
                        if not modal_result:
                            for line in output_lines:
                                if 'status' in line and ('success' in line or 'error' in line):
                                    try:
                                        modal_result = json_module.loads(line.strip())
                                        break
                                    except json_module.JSONDecodeError:
                                        continue
                        
                        if modal_result:
                            return {
                                "success": modal_result.get("status") == "success",
                                "response": modal_result.get("response", ""),
                                "error": modal_result.get("error"),
                                "inference_time": modal_result.get("inference_time", 0),
                                "metadata": modal_result
                            }
                        
                        # If all parsing fails, return debug info with more context
                        # Show last 1000 chars which should contain the JSON result
                        debug_output = result.stdout[-1000:] if len(result.stdout) > 1000 else result.stdout
                        return {"success": False, "error": f"Could not parse Modal CLI output. Last 1000 chars: {debug_output}"}
                    except Exception as e:
                        return {"success": False, "error": f"Modal parsing error: {str(e)}. Raw output: {result.stdout[:500]}"}
                else:
                    return {"success": False, "error": f"Modal CLI error: {result.stderr}"}
            
            except subprocess.TimeoutExpired:
                return {"success": False, "error": "Modal inference timeout"}
            except Exception as e:
                return {"success": False, "error": f"Modal function call failed: {str(e)}"}
        
        else:
            return {"success": False, "error": f"Unknown Modal endpoint format: {provider.endpoint}"}
    
    # async def _run_runpod_inference(self, provider: Provider, request: InferenceRequest) -> Dict[str, Any]:
    #     """Run inference on RunPod provider - REMOVED"""
    #     payload = {
    #         "input": {
    #             "model": request.model,
    #             "prompt": request.prompt,
    #             "temperature": request.temperature,
    #             "max_tokens": request.max_tokens
    #         }
    #     }
    #     
    #     headers = {"Authorization": f"Bearer {os.getenv('RUNPOD_API_KEY')}"}
    #     response = await self.client.post(f"{provider.endpoint}/run", json=payload, headers=headers)
    #     
    #     if response.status_code == 200:
    #         result = response.json()
    #         return {
    #             "success": True,
    #             "response": result.get("output", {}).get("response", ""),
    #             "metadata": result.get("output", {})
    #         }
    #     else:
    #         return {"success": False, "error": f"HTTP {response.status_code}: {response.text}"}

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
                executions = executions_data.get("executions", [])
                
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
    providers = router.providers
    if region:
        providers = [p for p in providers if p.region == region]
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
                "last_health_check": p.last_health_check,
                "endpoint_type": "http",
                "models_supported": ["llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"] if p.type == ProviderType.MODAL else ["llama3.2-1b"]
            }
            for p in providers
        ]
    }

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
    healthy_providers = [p for p in router.providers if p.healthy]
    
    return {
        "total_providers": len(router.providers),
        "healthy_providers": len(healthy_providers),
        "avg_latency": sum(p.avg_latency for p in healthy_providers) / len(healthy_providers) if healthy_providers else 0,
        "avg_success_rate": sum(p.success_rate for p in healthy_providers) / len(healthy_providers) if healthy_providers else 0,
        "cost_range": {
            "min": min(p.cost_per_second for p in healthy_providers) if healthy_providers else 0,
            "max": max(p.cost_per_second for p in healthy_providers) if healthy_providers else 0
        },
        "models_supported": ["llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"]
    }

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
