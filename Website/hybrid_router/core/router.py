"""Main hybrid router implementation"""

import os
import time
import asyncio
import logging
import uuid
from typing import Dict, Any, List, Optional

import httpx
from fastapi import HTTPException
from datetime import datetime

from ..models import Provider, ProviderType, InferenceRequest, InferenceResponse

logger = logging.getLogger(__name__)


class HybridRouter:
    def __init__(self):
        self.providers: List[Provider] = []
        # Configure granular timeouts for different stages
        # connect: time to establish connection
        # read: time to read response (includes Modal execution time)
        # write: time to write request
        # pool: time to acquire connection from pool
        timeout = httpx.Timeout(
            connect=10.0,   # 10s to connect
            read=600.0,     # 10min for Modal GPU queue + cold starts + execution
            write=10.0,     # 10s to send request
            pool=10.0       # 10s to get connection from pool
        )
        # CRITICAL: Disable automatic redirect following
        # Modal sometimes returns 303 redirects when containers fail
        # httpx auto-following these redirects cancels the original request
        # causing Modal to receive cancellation signals and terminate containers
        self.client = httpx.AsyncClient(
            timeout=timeout,
            follow_redirects=False  # Handle redirects explicitly
        )
        self.setup_providers()

    def _build_failure(
        self,
        *,
        code: str,
        stage: str,
        message: str,
        provider: Optional[str] = None,
        provider_type: Optional[str] = None,
        region: Optional[str] = None,
        model: Optional[str] = None,
        transient: bool = False,
        http_status: Optional[int] = None,
        url: Optional[str] = None,
        extra: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        failure: Dict[str, Any] = {
            "code": code,
            "stage": stage,
            "component": "hybrid_router",
            "subcomponent": "inference",
            "message": message,
            "provider": provider,
            "provider_type": provider_type,
            "region": region,
            "model": model,
            "transient": transient,
            "timestamp": datetime.utcnow().replace(microsecond=0).isoformat() + "Z",
        }
        if http_status is not None:
            failure["http_status"] = http_status
        if url:
            failure["url"] = url
        if extra:
            failure.update(extra)
        # Drop keys with None values for cleaner payloads
        return {k: v for k, v in failure.items() if v is not None}
        
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
        
        # Modal serverless (burst capacity) - HARDCODED FOR DEBUGGING
        modal_endpoints = {
            "us-east": "https://jamie-anson--project-beacon-hf-us-inference.modal.run",
            "eu-west": "https://jamie-anson--project-beacon-hf-eu-inference.modal.run",
            # TEMPORARILY DISABLED: APAC has 6+ minute cold starts causing Railway 150s timeouts
            # Re-enable when keep-warm strategy implemented (~$87/month) or platform timeout increased
            # "asia-pacific": "https://jamie-anson--project-beacon-hf-apac-inference.modal.run"
        }

        for region, endpoint in modal_endpoints.items():
            self.providers.append(Provider(
                name=f"modal-{region}",
                type=ProviderType.MODAL,
                endpoint=endpoint,
                region=region,
                cost_per_second=0.00005,  # Lower than Golem to prefer Modal
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
        logger.info(f"🔍 [HEALTH_CHECK] Starting health check for {provider.name}")
        try:
            if provider.type == ProviderType.GOLEM:
                # Simple ping for Golem providers
                response = await self.client.get(f"{provider.endpoint}/health", timeout=5.0)
                provider.healthy = response.status_code == 200
            
            elif provider.type == ProviderType.MODAL:
                # Modal health check - use actual inference endpoint
                # Send a minimal test request to verify the endpoint is responsive
                try:
                    test_payload = {
                        "model": "llama3.2-1b",
                        "prompt": "test",
                        "temperature": 0.1,
                        "max_tokens": 5
                    }
                    # Use provider endpoint directly (Modal endpoints are at root path)
                    # Use client's default timeout (600s) to handle Modal cold starts (can be 2-3 minutes)
                    response = await self.client.post(
                        provider.endpoint,
                        json=test_payload
                    )
                    if response.status_code == 200:
                        data = response.json()
                        # Check if response has success field
                        has_success = data.get("success", False)
                        provider.healthy = has_success
                        logger.info(f"✅ [HEALTH_CHECK] {provider.name} response: status={response.status_code}, success={has_success}, data_keys={list(data.keys())}")
                    else:
                        provider.healthy = False
                        logger.warning(f"❌ [HEALTH_CHECK] {provider.name} returned status {response.status_code}")
                except Exception as health_err:
                    logger.error(
                        f"❌ [HEALTH_CHECK] {provider.name} failed: {type(health_err).__name__}: {str(health_err)}",
                        exc_info=True
                    )
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
        
        # 🔍 DEBUG: Log provider health status
        logger.error(
            f"[SELECT_PROVIDER] Total providers: {len(self.providers)}, "
            f"Healthy: {len(healthy_providers)}, "
            f"Provider health: {[(p.name, p.healthy, p.last_health_check) for p in self.providers]}"
        )
        
        if not healthy_providers:
            logger.error(
                f"[SELECT_PROVIDER] NO HEALTHY PROVIDERS! "
                f"All providers: {[(p.name, p.healthy) for p in self.providers]}"
            )
            return None
        
        # STRICT region matching when region is specified
        if request.region_preference:
            region_providers = [p for p in healthy_providers if p.region == request.region_preference]
            
            if not region_providers:
                # NO FALLBACK - return None to trigger error
                available_regions = list(set(p.region for p in healthy_providers))
                logger.error(
                    f"No healthy providers available for region {request.region_preference}. "
                    f"Available regions: {available_regions}"
                )
                return None
            
            healthy_providers = region_providers
            logger.info(
                f"Region-locked provider selection: {request.region_preference} "
                f"(provider_count={len(region_providers)})"
            )
        
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
                logger.info(
                    f"Provider selected: {provider.name} (region={provider.region}, "
                    f"type={provider.type.value}, region_locked={bool(request.region_preference)})"
                )
                return provider
        
        # Fallback to first provider if none have capacity
        selected = healthy_providers[0] if healthy_providers else None
        if selected:
            logger.info(
                f"Provider selected (fallback): {selected.name} (region={selected.region}, "
                f"type={selected.type.value}, region_locked={bool(request.region_preference)})"
            )
        return selected
    
    async def run_inference(self, request: InferenceRequest) -> InferenceResponse:
        """Execute inference request on selected provider"""
        
        provider = self.select_provider(request)
        if not provider:
            failure = self._build_failure(
                code="PROVIDER_UNAVAILABLE",
                stage="provider_selection",
                message="No healthy providers available",
                region=request.region_preference,
                model=request.model,
                transient=True,
            )
            return InferenceResponse(
                success=False,
                error=failure["message"],
                error_code=failure["code"],
                failure=failure,
                provider_used="",
                inference_time=0.0,
                cost_estimate=0.0,
                metadata={"requested_region": request.region_preference or "unknown"},
            )
        
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
                error_code=result.get("error_code"),
                failure=result.get("failure"),
                provider_used=provider.name,
                inference_time=inference_time,
                cost_estimate=cost_estimate,
                metadata={
                    "provider_type": provider.type.value,
                    "region": provider.region,
                    "model": request.model,
                    "receipt": result.get("receipt"),
                    "failure": result.get("failure"),
                }
            )
            
        except Exception as e:
            inference_time = time.time() - start_time
            logger.error(f"Inference failed on {provider.name}: {e}")
            
            failure = self._build_failure(
                code="ROUTER_INTERNAL_ERROR",
                stage="router_inference",
                message=str(e),
                provider=provider.name,
                provider_type=provider.type.value,
                region=provider.region,
                model=request.model,
                transient=True,
            )
            return InferenceResponse(
                success=False,
                error=str(e),
                error_code=failure["code"],
                failure=failure,
                provider_used=provider.name,
                inference_time=inference_time,
                cost_estimate=0.0,
                metadata={"provider_type": provider.type.value, "region": provider.region, "failure": failure}
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

        failure = self._build_failure(
            code=f"PROVIDER_HTTP_{response.status_code}",
            stage="provider_execution",
            message=f"Golem HTTP {response.status_code}",
            provider=provider.name,
            provider_type=provider.type.value,
            region=provider.region,
            model=request.model,
            http_status=response.status_code,
            url=f"{provider.endpoint}/inference",
            transient=response.status_code >= 500,
        )
        return {
            "success": False,
            "error": f"HTTP {response.status_code}: {response.text}",
            "error_code": failure["code"],
            "failure": failure,
        }
    
    async def _run_modal_inference(self, provider: Provider, request: InferenceRequest) -> Dict[str, Any]:
        """Run inference on Modal provider with detailed timing and error tracking"""
        
        # Generate correlation ID for request tracking
        request_id = str(uuid.uuid4())[:8]
        
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

        # Log request initiation with timing
        request_start = time.time()
        logger.info(
            f"[{request_id}] Sending request to Modal",
            extra={
                "request_id": request_id,
                "provider": provider.name,
                "region": provider.region,
                "model": request.model,
                "prompt_len": len(request.prompt or ""),
                "endpoint": provider.endpoint
            }
        )

        response = None
        last_error = None
        
        for attempt in range(3):
            try:
                attempt_start = time.time()
                logger.debug(f"[{request_id}] Attempt {attempt + 1}/3 starting")
                
                # Make the POST request with explicit timeout handling
                # Use provider endpoint directly (Modal endpoints are at root path)
                response = await self.client.post(
                    provider.endpoint, 
                    json=payload, 
                    headers=headers
                )
                
                attempt_duration = time.time() - attempt_start
                logger.info(
                    f"[{request_id}] Modal responded in {attempt_duration:.2f}s",
                    extra={
                        "request_id": request_id,
                        "attempt": attempt + 1,
                        "status_code": response.status_code,
                        "duration_seconds": attempt_duration
                    }
                )

                # Check for 303 redirect (indicates Modal container failure)
                if response.status_code == 303:
                    redirect_url = response.headers.get("Location", "unknown")
                    logger.error(
                        f"[{request_id}] Modal returned 303 redirect (container failure)",
                        extra={
                            "request_id": request_id,
                            "attempt": attempt + 1,
                            "redirect_url": redirect_url,
                            "duration_seconds": attempt_duration
                        }
                    )
                    # Don't retry 303s - they indicate infrastructure issues
                    last_error = Exception(f"Modal container failure (303 redirect to {redirect_url})")
                    break
                
                # Check for stopped app and retry
                if response.status_code == 404 and "app for invoked web endpoint is stopped" in response.text:
                    logger.warning(
                        f"[{request_id}] Modal app stopped, retrying",
                        extra={"request_id": request_id, "attempt": attempt + 1}
                    )
                    await asyncio.sleep(2 * (attempt + 1))
                    continue
                    
                # Success or non-retryable error
                break
                
            except httpx.TimeoutException as e:
                attempt_duration = time.time() - attempt_start
                last_error = e
                logger.error(
                    f"[{request_id}] Modal request timed out after {attempt_duration:.2f}s",
                    extra={
                        "request_id": request_id,
                        "attempt": attempt + 1,
                        "timeout_type": type(e).__name__,
                        "duration_seconds": attempt_duration
                    }
                )
                if attempt < 2:
                    await asyncio.sleep(2 * (attempt + 1))
                    continue
                # Final attempt failed, will handle below
                
            except httpx.RequestError as e:
                attempt_duration = time.time() - attempt_start
                last_error = e
                logger.error(
                    f"[{request_id}] Modal request failed: {e}",
                    extra={
                        "request_id": request_id,
                        "attempt": attempt + 1,
                        "error_type": type(e).__name__,
                        "error_message": str(e),
                        "duration_seconds": attempt_duration
                    }
                )
                if attempt < 2:
                    await asyncio.sleep(2 * (attempt + 1))
                    continue
                # Final attempt failed, will handle below
                
            except Exception as e:
                attempt_duration = time.time() - attempt_start
                last_error = e
                logger.error(
                    f"[{request_id}] Unexpected error calling Modal: {e}",
                    extra={
                        "request_id": request_id,
                        "attempt": attempt + 1,
                        "error_type": type(e).__name__,
                        "error_message": str(e),
                        "duration_seconds": attempt_duration
                    },
                    exc_info=True
                )
                if attempt < 2:
                    await asyncio.sleep(2 * (attempt + 1))
                    continue
                # Final attempt failed, will handle below

        # If all attempts failed, return error
        if response is None:
            total_duration = time.time() - request_start
            logger.error(
                f"[{request_id}] All Modal attempts failed after {total_duration:.2f}s",
                extra={"request_id": request_id, "total_duration_seconds": total_duration}
            )
            failure = self._build_failure(
                code="MODAL_REQUEST_FAILED",
                stage="modal_request",
                message=f"Failed to get response from Modal after 3 attempts: {last_error}",
                provider=provider.name,
                provider_type=provider.type.value,
                region=provider.region,
                model=request.model,
                transient=True,
            )
            return {
                "success": False,
                "error": failure["message"],
                "error_code": failure["code"],
                "failure": failure,
            }

        # Parse response with proper error handling
        data = None
        try:
            parse_start = time.time()
            data = response.json()
            parse_duration = time.time() - parse_start
            logger.debug(
                f"[{request_id}] Response parsed in {parse_duration:.3f}s",
                extra={"request_id": request_id, "parse_duration_seconds": parse_duration}
            )
        except Exception as e:
            logger.error(
                f"[{request_id}] Failed to parse Modal response as JSON: {e}",
                extra={
                    "request_id": request_id,
                    "status_code": response.status_code,
                    "content_type": response.headers.get("content-type"),
                    "body_preview": response.text[:500] if response.text else None
                },
                exc_info=True
            )
            # Return parse error instead of silently continuing
            failure = self._build_failure(
                code="MODAL_RESPONSE_PARSE_ERROR",
                stage="response_parsing",
                message=f"Could not parse Modal response as JSON: {e}",
                provider=provider.name,
                provider_type=provider.type.value,
                region=provider.region,
                model=request.model,
                http_status=response.status_code,
                transient=False,
            )
            return {
                "success": False,
                "error": failure["message"],
                "error_code": failure["code"],
                "failure": failure,
            }

        if response is not None and response.status_code == 200 and isinstance(data, dict):
            # Modal returns { status: "success" | "error", response?: str, error?: str, ... }
            status = data.get("status")
            success = data.get("success")
            if success is None and status is not None:
                success = str(status).lower() == "success"

            # Extract response text from common fields
            resp_text = data.get("response") or data.get("output") or data.get("text")
            error_msg = data.get("error")

            # Allow empty responses - some models may legitimately return empty strings
            # for certain prompts (e.g., refusals, content filtering)
            # Convert None to empty string for consistency
            if resp_text is None:
                resp_text = ""

            if not success:
                total_duration = time.time() - request_start
                logger.warning(
                    f"[{request_id}] Modal reported execution failure after {total_duration:.2f}s",
                    extra={
                        "request_id": request_id,
                        "total_duration_seconds": total_duration,
                        "error_message": error_msg
                    }
                )
                failure_payload = data.get("failure") or {}
                failure = self._build_failure(
                    code=failure_payload.get("code", "PROVIDER_EXECUTION_FAILED"),
                    stage=failure_payload.get("stage", "provider_execution"),
                    message=error_msg or failure_payload.get("message", "Modal execution failed"),
                    provider=provider.name,
                    provider_type=provider.type.value,
                    region=provider.region,
                    model=request.model,
                    transient=True,
                    extra={k: v for k, v in failure_payload.items() if k not in {"code", "stage", "message"}},
                )
                return {
                    "success": False,
                    "error": error_msg,
                    "error_code": failure.get("code"),
                    "failure": failure,
                    "modal_raw": data,
                }

            # Success path
            total_duration = time.time() - request_start
            logger.info(
                f"[{request_id}] Modal inference completed successfully in {total_duration:.2f}s",
                extra={
                    "request_id": request_id,
                    "total_duration_seconds": total_duration,
                    "response_length": len(resp_text) if resp_text else 0,
                    "model": request.model,
                    "region": provider.region
                }
            )
            return {
                "success": bool(success),
                "response": resp_text,
                "error": error_msg,
                "receipt": data.get("receipt"),
                "modal_raw": data,
            }

        # Non-200 or invalid responses fall back to HTTP failure mapping
        status_code = response.status_code if response is not None else 500
        body_text = response.text if response is not None else ""
        total_duration = time.time() - request_start

        if isinstance(data, dict) and status_code == 200:
            # Handle unexpected JSON without explicit HTTP status
            logger.error(
                f"[{request_id}] Modal returned 200 but unexpected JSON structure after {total_duration:.2f}s",
                extra={
                    "request_id": request_id,
                    "total_duration_seconds": total_duration,
                    "data_keys": list(data.keys()) if isinstance(data, dict) else None
                }
            )
            failure = self._build_failure(
                code="PROVIDER_EXECUTION_FAILED",
                stage="provider_execution",
                message=data.get("error", "Modal execution failed"),
                provider=provider.name,
                provider_type=provider.type.value,
                region=provider.region,
                model=request.model,
                transient=True,
                extra={k: v for k, v in data.items() if k not in {"code", "stage", "message"}},
            )
            return {
                "success": False,
                "error": failure.get("message"),
                "error_code": failure["code"],
                "failure": failure,
                "modal_raw": data,
            }

        logger.error(
            f"[{request_id}] Modal returned HTTP {status_code} after {total_duration:.2f}s",
            extra={
                "request_id": request_id,
                "status_code": status_code,
                "total_duration_seconds": total_duration,
                "body_preview": body_text[:200] if body_text else None
            }
        )
        failure = self._build_failure(
            code=f"PROVIDER_HTTP_{status_code}",
            stage="provider_execution",
            message=f"Modal HTTP {status_code}",
            provider=provider.name,
            provider_type=provider.type.value,
            region=provider.region,
            model=request.model,
            http_status=status_code,
            url=provider.endpoint,
            transient=status_code >= 500,
        )
        return {
            "success": False,
            "error": f"HTTP {status_code}: {body_text}",
            "error_code": failure["code"],
            "failure": failure,
        }
    
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
