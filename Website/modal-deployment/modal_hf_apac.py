"""
Project Beacon - HF Transformers APAC Region
Models served: Llama 3.2-1B, Qwen 2.5-1.5B
"""
import modal
import os
import time
from typing import Dict, Any

# Create Modal app
app = modal.App("project-beacon-hf-apac")

# Optional Hugging Face secret for gated models
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
        "bitsandbytes>=0.41.0",
        "sentencepiece>=0.1.99",
        "safetensors>=0.4.5",
        "huggingface_hub>=0.21.4",
        "fastapi>=0.104.0",
        "pydantic>=2.0.0"
    )
    .apt_install("git")
)

# Persistent volume for model caching
models_volume = modal.Volume.from_name("beacon-hf-models-apac", create_if_missing=True)

# Model registry with configurations
MODEL_REGISTRY = {
    "llama3.2-1b": {
        "hf_model": "meta-llama/Llama-3.2-1B-Instruct",
        "gpu": "T4",
        "memory_gb": 8,
        "context_length": 128000,
        "description": "Fast 1B parameter model for quick inference"
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
    
    hf_token = os.getenv("HUGGINGFACE_HUB_TOKEN") or os.getenv("HF_TOKEN")
    if hf_token:
        try:
            os.environ.setdefault("HF_TOKEN", hf_token)
            os.environ.setdefault("HUGGINGFACE_HUB_TOKEN", hf_token)
        except Exception:
            pass
    token_kwargs = {"token": hf_token} if hf_token else {}

    if os.path.exists(model_path):
        print(f"Loading cached model from {model_path}")
        tokenizer = AutoTokenizer.from_pretrained(model_path)
        model = AutoModelForCausalLM.from_pretrained(
            model_path,
            torch_dtype=torch.float16,
            device_map="auto",
            load_in_8bit=True
        )
    else:
        print(f"Downloading model {model_name}")
        hf_model_name = MODELS[model_name]
        try:
            tokenizer = AutoTokenizer.from_pretrained(hf_model_name, **token_kwargs)
        except TypeError:
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
        tokenizer.save_pretrained(model_path)
        model.save_pretrained(model_path)
        print(f"Model cached to {model_path}")
    
    return tokenizer, model

def run_inference_logic(model_name: str, prompt: str, region: str, temperature: float = 0.1, max_tokens: int = 128):
    """Shared inference logic"""
    import torch
    
    start_time = time.time()
    
    try:
        if model_name not in MODEL_REGISTRY:
            return {"status": "error", "error": f"Unknown model: {model_name}", "region": region}
        
        # Use preloaded model from cache
        if model_name in MODEL_CACHE and MODEL_CACHE[model_name].get("status") == "ready":
            cached = MODEL_CACHE[model_name]
            tokenizer = cached["tokenizer"]
            model = cached["model"]
            print(f"[INFERENCE] Using preloaded {model_name}")
        else:
            print(f"[INFERENCE] Fallback loading {model_name}")
            model_path = f"/models/{model_name}"
            tokenizer, model = load_model_and_tokenizer(model_name, model_path)
        
        # FIXED: Use proper chat template for instruction following
        system_prompt = "You are a helpful, honest, and harmless AI assistant. Answer questions directly and factually. For sensitive political topics, provide balanced, factual information from multiple perspectives."
        
        messages = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": prompt}
        ]
        
        # Apply chat template for instruction-following models
        try:
            formatted_prompt = tokenizer.apply_chat_template(
                messages, 
                tokenize=False, 
                add_generation_prompt=True
            )
            print(f"[INFERENCE] Using chat template for {model_name}")
        except Exception as e:
            # Fallback for models without chat template
            print(f"[INFERENCE] Chat template failed for {model_name}, using fallback: {e}")
            formatted_prompt = f"System: {system_prompt}\n\nUser: {prompt}\n\nAssistant:"
        
        # Generate response with formatted prompt
        inputs = tokenizer(formatted_prompt, return_tensors="pt")
        input_ids = inputs["input_ids"].to(model.device)
        
        with torch.no_grad():
            outputs = model.generate(
                input_ids,
                max_new_tokens=max_tokens,
                temperature=temperature,
                do_sample=True if temperature > 0 else False,
                pad_token_id=tokenizer.eos_token_id,
                eos_token_id=tokenizer.eos_token_id
            )
        
        full_response = tokenizer.decode(outputs[0], skip_special_tokens=True)
        
        # FIXED: Better response extraction for chat templates
        response = ""
        
        # Method 1: Extract after "assistant" keyword (for chat templates)
        if "assistant" in full_response.lower():
            # Find the assistant section and extract everything after it
            assistant_parts = full_response.split("assistant")
            if len(assistant_parts) > 1:
                # Get everything after the last "assistant" occurrence
                response = assistant_parts[-1].strip()
                # Remove common prefixes like newlines, colons, etc.
                response = response.lstrip(": \n\t\r")
        
        # Method 2: Extract after formatted prompt (fallback)
        if not response and len(formatted_prompt) < len(full_response):
            response = full_response[len(formatted_prompt):].strip()
        
        # Method 3: Use full response if extraction failed
        if not response or len(response.strip()) < 5:
            response = full_response.strip()
        
        # Method 4: Final fallback
        if not response:
            response = "Response extraction failed"
        
        inference_time = time.time() - start_time
        
        execution_details = {
            "provider_id": f"modal-{region}",
            "region": region,
            "model": model_name,
            "started_at": start_time,
            "completed_at": time.time(),
            "duration": inference_time
        }

        receipt = {
            "schema_version": "v0.1.0",
            "execution_details": execution_details,
            "output": {
                "response": response,
                "prompt": prompt,
                "tokens_generated": len(tokenizer.encode(response)),
                "metadata": {
                    "temperature": temperature,
                    "max_tokens": max_tokens,
                    "full_response": full_response
                }
            },
            "provenance": {
                "provider": "modal",
                "architecture": "hf-transformers",
                "model_registry": model_name
            }
        }

        return {
            "success": True,
            "response": response,
            "model": model_name,
            "inference_time": inference_time,
            "region": region,
            "tokens_generated": len(tokenizer.encode(response)),
            "gpu_memory_used": torch.cuda.memory_allocated() if torch.cuda.is_available() else 0,
            "receipt": receipt
        }
        
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "region": region,
            "inference_time": time.time() - start_time
        }

# APAC Region Function
@app.function(
    image=image,
    gpu="A10G",
    volumes={"/models": models_volume},
    timeout=900,
    scaledown_window=600,
    region=["ap-southeast", "ap-northeast"],
    memory=16384,
    secrets=SECRETS,
    startup_timeout=1800,
)
def run_inference(
    model_name: str,
    prompt: str, 
    temperature: float = 0.1,
    max_tokens: int = 128
) -> Dict[str, Any]:
    """HF Transformers inference - APAC region"""
    if not MODEL_CACHE:
        preload_all_models()
    return run_inference_logic(model_name, prompt, "asia-southeast", temperature, max_tokens)

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

def _health_payload() -> Dict[str, Any]:
    ready_models = [name for name, cache in MODEL_CACHE.items() if cache.get("status") == "ready"] if MODEL_CACHE else []
    return {
        "status": "healthy",
        "timestamp": time.time(),
        "models_available": list(MODEL_REGISTRY.keys()),
        "models_ready": ready_models,
        "models_ready_count": len(ready_models),
        "region": "asia-southeast",
        "architecture": "hf-transformers",
        "cache_initialized": bool(MODEL_CACHE)
    }


@app.function(
    image=image,
    timeout=30,
    secrets=SECRETS,
)
def health_check() -> Dict[str, Any]:
    """Background health check callable by other Modal functions"""
    return _health_payload()

# Web endpoints for HTTP access
@app.function(
    image=image,
    gpu="A10G",
    volumes={"/models": models_volume},
    timeout=900,
    scaledown_window=600,
    region=["us-west"],
    memory=16384,
    secrets=SECRETS,
    startup_timeout=1800,
)
@modal.web_endpoint(method="POST")
def inference(item: dict):
    """HTTP inference endpoint"""
    if not MODEL_CACHE:
        preload_all_models()
    
    model_name = item.get("model", "llama3.2-1b")
    prompt = item.get("prompt", "")
    temperature = item.get("temperature", 0.1)
    max_tokens = item.get("max_tokens", 128)
    
    return run_inference_logic(model_name, prompt, "asia-southeast", temperature, max_tokens)

@app.function(
    image=image,
    timeout=30,
    secrets=SECRETS,
)
@modal.web_endpoint(method="GET")
def health():
    """HTTP health check endpoint"""
    return _health_payload()
