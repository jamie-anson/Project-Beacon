"""
Simplified Modal deployment for Project Beacon - Mock responses for testing
This bypasses the Ollama startup issues by providing mock inference responses
"""

import modal
import time
import json
from typing import Dict, Any

# Create Modal app
app = modal.App("project-beacon-simple")

# Simple image without Ollama
image = modal.Image.debian_slim(python_version="3.11").pip_install("fastapi", "pydantic")

@app.function(
    image=image,
    timeout=30,
    region=["us-east", "us-central", "us-west"],
)
def run_inference_us(
    model: str,
    prompt: str,
    temperature: float = 0.1,
    max_tokens: int = 512
) -> Dict[str, Any]:
    """Mock inference for US region"""
    time.sleep(0.5)  # Simulate inference time
    return {
        "status": "success",
        "response": f"Mock response from {model} in US region for: {prompt[:50]}...",
        "model": model,
        "inference_time": 0.5,
        "region": "us-east",
        "tokens_generated": 20,
        "mock": True
    }

@app.function(
    image=image,
    timeout=30,
    region=["eu-west", "eu-north"],
)
def run_inference_eu(
    model: str,
    prompt: str,
    temperature: float = 0.1,
    max_tokens: int = 512
) -> Dict[str, Any]:
    """Mock inference for EU region"""
    time.sleep(0.5)  # Simulate inference time
    return {
        "status": "success",
        "response": f"Mock response from {model} in EU region for: {prompt[:50]}...",
        "model": model,
        "inference_time": 0.5,
        "region": "eu-west",
        "tokens_generated": 20,
        "mock": True
    }

@app.function(
    image=image,
    timeout=30,
    region=["ap-southeast", "ap-northeast"],
)
def run_inference_apac(
    model: str,
    prompt: str,
    temperature: float = 0.1,
    max_tokens: int = 512
) -> Dict[str, Any]:
    """Mock inference for APAC region"""
    time.sleep(0.5)  # Simulate inference time
    return {
        "status": "success",
        "response": f"Mock response from {model} in APAC region for: {prompt[:50]}...",
        "model": model,
        "inference_time": 0.5,
        "region": "asia-pacific",
        "tokens_generated": 20,
        "mock": True
    }

@app.function(
    image=image,
    timeout=30,
)
def health_check() -> Dict[str, Any]:
    """Simple health check that always returns healthy"""
    return {
        "status": "healthy",
        "timestamp": time.time(),
        "models_loaded": 3,
        "available_models": ["llama3.2:1b", "mistral:7b", "qwen2.5:1.5b"],
        "mock": True
    }

# Web endpoint for HTTP API access
@app.function(
    image=image,
    timeout=30,
)
@modal.web_endpoint(method="POST")
def inference_api(item: Dict[str, Any]) -> Dict[str, Any]:
    """HTTP API endpoint for inference requests"""
    model = item.get("model", "llama3.2:1b")
    prompt = item.get("prompt", "")
    temperature = item.get("temperature", 0.1)
    max_tokens = item.get("max_tokens", 512)
    region = item.get("region", "us-east")
    
    if not prompt:
        return {
            "success": False,
            "error": "Prompt is required"
        }
    
    # Route to appropriate region function
    if region == "eu-west":
        return run_inference_eu.local(model, prompt, temperature, max_tokens)
    elif region == "asia-pacific":
        return run_inference_apac.local(model, prompt, temperature, max_tokens)
    else:
        return run_inference_us.local(model, prompt, temperature, max_tokens)

# Health check web endpoint
@app.function(
    image=image,
    timeout=30,
)
@modal.web_endpoint(method="GET", label="health")
def health_api() -> Dict[str, Any]:
    """HTTP health check endpoint"""
    return health_check.local()
