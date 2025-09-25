"""Main hybrid router implementation"""

import os
import time
import asyncio
import logging
from typing import Dict, Any, List, Optional

import httpx
from fastapi import HTTPException

from ..models import Provider, ProviderType, InferenceRequest, InferenceResponse

logger = logging.getLogger(__name__)


class HybridRouter:
    def __init__(self):
        self.providers: List[Provider] = []
        self.client = httpx.AsyncClient(timeout=120.0)
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
        
        # Modal serverless (burst capacity)
        modal_defaults = {
            "us-east": os.getenv("MODAL_US_INFERENCE_URL"),
            "eu-west": os.getenv("MODAL_EU_INFERENCE_URL"),
            "asia-pacific": os.getenv("MODAL_APAC_INFERENCE_URL")
        }
        modal_endpoint = os.getenv("MODAL_API_BASE")

        for region, endpoint in modal_defaults.items():
            target_endpoint = endpoint or modal_endpoint
            if not target_endpoint:
                continue

            self.providers.append(Provider(
                name=f"modal-{region}",
                type=ProviderType.MODAL,
                endpoint=target_endpoint,
                region=region,
                cost_per_second=0.0003,  # T4 pricing
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
                response = await self.client.get(f"{provider.endpoint}/health", timeout=5.0)
                provider.healthy = response.status_code == 200
            
            elif provider.type == ProviderType.MODAL:
                # Modal health check - use per-region override or fallback
                health_defaults = {
                    "us-east": os.getenv("MODAL_US_HEALTH_URL"),
                    "eu-west": os.getenv("MODAL_EU_HEALTH_URL"),
                    "asia-pacific": os.getenv("MODAL_APAC_HEALTH_URL")
                }
                fallback_health = os.getenv("MODAL_HEALTH_ENDPOINT", "https://jamie-anson--health.modal.run")
                # Use per-region health endpoint when set; otherwise fall back
                # Note: dict.get(key, default) would return None if the key exists with a None value,
                # which would skip the desired fallback. Hence the explicit "or" usage.
                health_endpoint = health_defaults.get(provider.region) or fallback_health

                response = await self.client.get(health_endpoint, timeout=5.0)
                if response.status_code == 200:
                    health_data = response.json()
                    provider.healthy = health_data.get("status") == "healthy"
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
                    "model": request.model,
                    "receipt": result.get("receipt"),
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
        # IMPORTANT: Forward the selected provider's region to Modal's unified inference API
        # This ensures the Modal web endpoint routes to the correct regional function
        # e.g., region == "asia-pacific" will invoke run_inference_apac
        payload["region"] = provider.region

        token = os.getenv("MODAL_API_TOKEN")
        headers = {"Authorization": f"Bearer {token}"} if token else {}

        response = None
        for attempt in range(3):
            response = await self.client.post(provider.endpoint, json=payload, headers=headers)

            if response.status_code == 404 and "app for invoked web endpoint is stopped" in response.text:
                logger.warning(
                    "Modal endpoint reported stopped app; retrying",
                    extra={"provider": provider.name, "attempt": attempt + 1}
                )
                await asyncio.sleep(2 * (attempt + 1))
                continue
            break

        # Normalize Modal response to router's expected schema
        try:
            data = response.json() if response is not None else None
        except Exception:
            data = None

        if response.status_code == 200 and isinstance(data, dict):
            # Modal returns { status: "success" | "error", response?: str, error?: str, ... }
            status = data.get("status")
            success = data.get("success")
            if success is None and status is not None:
                success = str(status).lower() == "success"

            # Extract response text from common fields
            resp_text = data.get("response") or data.get("output") or data.get("text")
            error_msg = data.get("error")

            return {
                "success": bool(success),
                "response": resp_text,
                "error": error_msg,
                "receipt": data.get("receipt"),
                "modal_raw": data,
            }
        else:
            return {"success": False, "error": f"HTTP {response.status_code}: {response.text}"}
    
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
