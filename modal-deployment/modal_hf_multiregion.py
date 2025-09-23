"""
Project Beacon - HF Transformers Multi-Region Emergency Fix
Replaces broken Ollama with direct Hugging Face transformers
"""
import modal
import os
import time
from typing import Dict, Any

# Create Modal app
app = modal.App("project-beacon-hf")

# Optional Hugging Face secret for gated models (Llama/Mistral)
# Default to 'custom-secret' as per user's setup; allow override via HF_SECRET_NAME
HF_SECRET_NAME = os.getenv("HF_SECRET_NAME", "custom-secret")
try:
    HF_SECRET = modal.Secret.from_name(HF_SECRET_NAME)
except Exception:
    HF_SECRET = None

SECRETS = [HF_SECRET] if HF_SECRET else []

# Optimized image with HF transformers
image = (
    modal.Image.debian_slim(python_version="3.11")
    .pip_install(
        "torch>=2.0.0",
        "transformers>=4.35.0", 
        "accelerate>=0.24.0",
        "bitsandbytes>=0.41.0",  # For 8-bit quantization
        "sentencepiece>=0.1.99",  # Needed for Mistral tokenizer
        "safetensors>=0.4.5",
        "huggingface_hub>=0.21.4",
        "fastapi>=0.104.0",
        "pydantic>=2.0.0"
    )
    .apt_install("git")  # For model downloads
)

# Persistent volume for model caching
models_volume = modal.Volume.from_name("beacon-hf-models", create_if_missing=True)

# Model registry with configurations
MODEL_REGISTRY = {
    "llama3.2-1b": {
        "hf_model": "meta-llama/Llama-3.2-1B-Instruct",
        "gpu": "T4",
        "memory_gb": 8,
        "context_length": 128000,
        "description": "Fast 1B parameter model for quick inference"
    },
    "mistral-7b": {
        "hf_model": "mistralai/Mistral-7B-Instruct-v0.3", 
        "gpu": "A10G",
        "memory_gb": 16,
        "context_length": 32768,
        "description": "Strong 7B parameter general-purpose model"
    },
    "qwen2.5-1.5b": {
        "hf_model": "Qwen/Qwen2.5-1.5B-Instruct",
        "gpu": "T4", 
        "memory_gb": 8,
        "context_length": 32768,
        "description": "Efficient 1.5B parameter model"
    }
}

# Legacy MODELS dict for backward compatibility
MODELS = {k: v["hf_model"] for k, v in MODEL_REGISTRY.items()}

# Global model cache - populated on container start
MODEL_CACHE = {}

def preload_all_models():
    """Preload all models on container start to eliminate cold starts"""
    import torch
    from transformers import AutoTokenizer, AutoModelForCausalLM
    
    print(f"[PRELOAD] Starting preload of {len(MODEL_REGISTRY)} models...")
    
    for model_name, config in MODEL_REGISTRY.items():
        try:
            print(f"[PRELOAD] Loading {model_name} ({config['hf_model']})...")
            model_path = f"/models/{model_name}"
            tokenizer, model = load_model_and_tokenizer(model_name, model_path)
            
            MODEL_CACHE[model_name] = {
                "tokenizer": tokenizer,
                "model": model,
                "config": config,
                "loaded_at": time.time(),
                "status": "ready"
            }
            print(f"[PRELOAD] ✓ {model_name} loaded successfully")
            
        except Exception as e:
            print(f"[PRELOAD] ✗ Failed to load {model_name}: {e}")
            MODEL_CACHE[model_name] = {
                "status": "error",
                "error": str(e),
                "loaded_at": time.time()
            }
    
    print(f"[PRELOAD] Completed. {len([k for k, v in MODEL_CACHE.items() if v.get('status') == 'ready'])} models ready")
    return MODEL_CACHE

def load_model_and_tokenizer(model_name: str, model_path: str):
    """Load or download model and tokenizer"""
    import torch
    from transformers import AutoTokenizer, AutoModelForCausalLM
    
    # Explicitly forward HF token to handle gated models
    hf_token = os.getenv("HUGGINGFACE_HUB_TOKEN") or os.getenv("HF_TOKEN")
    # Ensure token is available via environment for huggingface_hub auto-discovery
    if hf_token:
        try:
            os.environ.setdefault("HF_TOKEN", hf_token)
            os.environ.setdefault("HUGGINGFACE_HUB_TOKEN", hf_token)
        except Exception:
            pass
    token_kwargs = {"token": hf_token} if hf_token else {}
    try:
        # Avoid leaking the token; just log presence
        print(f"[HF] Token present: {bool(hf_token)}")
    except Exception:
        pass

    if os.path.exists(model_path):
        print(f"Loading cached model from {model_path}")
        tokenizer = AutoTokenizer.from_pretrained(model_path)
        model = AutoModelForCausalLM.from_pretrained(
            model_path,
            torch_dtype=torch.float16,
            device_map="auto",
            load_in_8bit=True  # Memory optimization
        )
    else:
        print(f"Downloading model {model_name}")
        hf_model_name = MODELS[model_name]
        # Try 'token' parameter first, then fallback to 'use_auth_token'
        try:
            tokenizer = AutoTokenizer.from_pretrained(hf_model_name, **token_kwargs)
        except TypeError:
            # transformers older versions expect 'use_auth_token'
            fallback_kwargs = {"use_auth_token": hf_token} if hf_token else {}
            tokenizer = AutoTokenizer.from_pretrained(hf_model_name, **fallback_kwargs)

        try:
            model = AutoModelForCausalLM.from_pretrained(
                hf_model_name,
                torch_dtype=torch.float16,
                device_map="auto",
                load_in_8bit=True,
                **token_kwargs,
            )
        except TypeError:
            fallback_kwargs = {"use_auth_token": hf_token} if hf_token else {}
            model = AutoModelForCausalLM.from_pretrained(
                hf_model_name,
                torch_dtype=torch.float16,
                device_map="auto",
                load_in_8bit=True,
                **fallback_kwargs,
            )
        # Cache for future use
        tokenizer.save_pretrained(model_path)
        model.save_pretrained(model_path)
        print(f"Model cached to {model_path}")
    
    return tokenizer, model

def run_inference_logic(model_name: str, prompt: str, region: str, temperature: float = 0.1, max_tokens: int = 128):
    """Shared inference logic"""
    import torch
    
    start_time = time.time()
    
    try:
        # Validate model
        if model_name not in MODEL_REGISTRY:
            return {"status": "error", "error": f"Unknown model: {model_name}", "region": region}
        
        # Use preloaded model from cache
        if model_name in MODEL_CACHE and MODEL_CACHE[model_name].get("status") == "ready":
            cached = MODEL_CACHE[model_name]
            tokenizer = cached["tokenizer"]
            model = cached["model"]
            print(f"[INFERENCE] Using preloaded {model_name}")
        else:
            # Fallback to on-demand loading if preload failed
            print(f"[INFERENCE] Fallback loading {model_name}")
            model_path = f"/models/{model_name}"
            tokenizer, model = load_model_and_tokenizer(model_name, model_path)
        
        # Prepare input and move to model device to avoid CPU/GPU mismatch
        inputs = tokenizer(prompt, return_tensors="pt")
        input_ids = inputs["input_ids"].to(model.device)
        
        # Generate response
        with torch.no_grad():
            outputs = model.generate(
                input_ids,
                max_new_tokens=max_tokens,
                temperature=temperature,
                do_sample=True if temperature > 0 else False,
                pad_token_id=tokenizer.eos_token_id,
                eos_token_id=tokenizer.eos_token_id
            )
        
        # Decode response
        full_response = tokenizer.decode(outputs[0], skip_special_tokens=True)
        # Remove the original prompt from response
        response = full_response[len(prompt):].strip()
        
        inference_time = time.time() - start_time
        
        return {
            "status": "success",
            "response": response,
            "model": model_name,
            "inference_time": inference_time,
            "region": region,
            "tokens_generated": len(tokenizer.encode(response)),
            "gpu_memory_used": torch.cuda.memory_allocated() if torch.cuda.is_available() else 0
        }
        
    except Exception as e:
        return {
            "status": "error",
            "error": str(e),
            "region": region,
            "inference_time": time.time() - start_time
        }

# Container startup function to preload models
@app.function(
    image=image,
    gpu="A10G",  # Use A10G to support Mistral 7B
    volumes={"/models": models_volume},
    timeout=1800,  # 30 minutes for initial model loading
    memory=16384,  # 16GB RAM for multiple models
    secrets=SECRETS,
)
def initialize_models():
    """Initialize and preload all models on container start"""
    return preload_all_models()

# US Region Function
@app.function(
    image=image,
    gpu="A10G",  # Use A10G to support all models including Mistral 7B
    volumes={"/models": models_volume},
    timeout=900,
    scaledown_window=600,  # Keep warm for 10 minutes
    region=["us-east", "us-central", "us-west"],
    memory=16384,  # 16GB RAM for multiple models
    secrets=SECRETS,
    startup_timeout=1800,  # Allow time for model preloading
)
def run_inference_us(
    model_name: str,
    prompt: str, 
    temperature: float = 0.1,
    max_tokens: int = 128
) -> Dict[str, Any]:
    """HF Transformers inference - US region"""
    # Ensure models are preloaded
    if not MODEL_CACHE:
        preload_all_models()
    return run_inference_logic(model_name, prompt, "us-east", temperature, max_tokens)

# EU Region Function
@app.function(
    image=image,
    gpu="A10G",  # Use A10G to support all models including Mistral 7B
    volumes={"/models": models_volume},
    timeout=900,
    scaledown_window=600,
    region=["eu-west", "eu-north"],
    memory=16384,  # 16GB RAM for multiple models
    secrets=SECRETS,
    startup_timeout=1800,
)
def run_inference_eu(
    model_name: str,
    prompt: str,
    temperature: float = 0.1,
    max_tokens: int = 128
) -> Dict[str, Any]:
    """HF Transformers inference - EU region"""
    # Ensure models are preloaded
    if not MODEL_CACHE:
        preload_all_models()
    return run_inference_logic(model_name, prompt, "eu-west", temperature, max_tokens)

# APAC Region Function
@app.function(
    image=image,
    gpu="A10G",  # Use A10G to support all models including Mistral 7B
    volumes={"/models": models_volume},
    timeout=900,
    scaledown_window=600,
    region=["ap-southeast", "ap-northeast"],
    memory=16384,  # 16GB RAM for multiple models
    secrets=SECRETS,
    startup_timeout=1800,
)
def run_inference_apac(
    model_name: str,
    prompt: str,
    temperature: float = 0.1,
    max_tokens: int = 128
) -> Dict[str, Any]:
    """HF Transformers inference - APAC region"""
    # Ensure models are preloaded
    if not MODEL_CACHE:
        preload_all_models()
    return run_inference_logic(model_name, prompt, "asia-pacific", temperature, max_tokens)

# Models inventory endpoint
@app.function(
    image=image,
    timeout=30,
    secrets=SECRETS,
)
def get_models() -> Dict[str, Any]:
    """Return available models and their status"""
    return {
        "models_available": list(MODEL_REGISTRY.keys()),
        "model_registry": MODEL_REGISTRY,
        "model_cache_status": {
            name: {
                "status": cache.get("status", "unknown"),
                "loaded_at": cache.get("loaded_at"),
                "error": cache.get("error")
            } for name, cache in MODEL_CACHE.items()
        } if MODEL_CACHE else {},
        "cache_initialized": bool(MODEL_CACHE),
        "ready_models": [name for name, cache in MODEL_CACHE.items() if cache.get("status") == "ready"] if MODEL_CACHE else []
    }

# Health check function
@app.function(
    image=image,
    timeout=30,
    secrets=SECRETS,
)
def health_check() -> Dict[str, Any]:
    """Simple health check that always returns healthy"""
    ready_models = [name for name, cache in MODEL_CACHE.items() if cache.get("status") == "ready"] if MODEL_CACHE else []
    return {
        "status": "healthy",
        "timestamp": time.time(),
        "models_available": list(MODEL_REGISTRY.keys()),
        "models_ready": ready_models,
        "models_ready_count": len(ready_models),
        "regions": ["us-east", "eu-west", "asia-pacific"],
        "architecture": "hf-transformers",
        # Diagnostics (no token value exposed)
        "hf_token_present": bool(os.getenv("HUGGINGFACE_HUB_TOKEN") or os.getenv("HF_TOKEN")),
        "secret_name": HF_SECRET_NAME,
        "secret_attached": bool(SECRETS),
        "cache_initialized": bool(MODEL_CACHE)
    }

# Models web endpoint
@app.function(
    image=image,
    timeout=30,
    secrets=SECRETS,
)
@modal.fastapi_endpoint(method="GET", label="hf-models")
def models_api() -> Dict[str, Any]:
    """HTTP endpoint for model inventory"""
    return get_models.remote()

# Web endpoints for HTTP access
@app.function(
    image=image,
    timeout=600,
    secrets=SECRETS,
)
@modal.fastapi_endpoint(method="POST")
def inference_api(item: Dict[str, Any]) -> Dict[str, Any]:
    """HTTP API endpoint for inference requests"""
    model_name = item.get("model", "llama3.2-1b")
    prompt = item.get("prompt", "")
    temperature = item.get("temperature", 0.1)
    max_tokens = item.get("max_tokens", 128)
    region = item.get("region", "us-east")
    
    if not prompt:
        return {
            "status": "error",
            "error": "Prompt is required"
        }
    
    # Route to appropriate region function
    if region == "eu-west":
        return run_inference_eu.remote(model_name, prompt, temperature, max_tokens)
    elif region == "asia-pacific":
        return run_inference_apac.remote(model_name, prompt, temperature, max_tokens)
    else:
        return run_inference_us.remote(model_name, prompt, temperature, max_tokens)

# Health check web endpoint
@app.function(
    image=image,
    timeout=30,
    secrets=SECRETS,
)
@modal.fastapi_endpoint(method="GET", label="hf-health")
def health_api() -> Dict[str, Any]:
    """HTTP health check endpoint"""
    return health_check.remote()

if __name__ == "__main__":
    # Local testing
    with app.run():
        print("Testing HF Transformers deployment...")
        
        # Test health check
        health_result = health_check.remote()
        print(f"Health check: {health_result}")
        
        # Test model preloading
        init_result = initialize_models.remote()
        print(f"Model initialization: {init_result}")
        
        # Test models endpoint
        models_result = get_models.remote()
        print(f"Models available: {models_result}")
        
        # Test US inference
        us_result = run_inference_us.remote(
            model_name="llama3.2-1b",
            prompt="What is artificial intelligence?",
            max_tokens=50
        )
        print(f"US inference: {us_result}")
