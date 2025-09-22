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

# Optimized image with HF transformers
image = (
    modal.Image.debian_slim(python_version="3.11")
    .pip_install(
        "torch>=2.0.0",
        "transformers>=4.35.0", 
        "accelerate>=0.24.0",
        "bitsandbytes>=0.41.0",  # For 8-bit quantization
        "fastapi>=0.104.0",
        "pydantic>=2.0.0"
    )
    .apt_install("git")  # For model downloads
)

# Persistent volume for model caching
models_volume = modal.Volume.from_name("beacon-hf-models", create_if_missing=True)

# Model configurations
MODELS = {
    "llama3.2-1b": "meta-llama/Llama-3.2-1B-Instruct",
    "mistral-7b": "mistralai/Mistral-7B-Instruct-v0.3",
    "qwen2.5-1.5b": "Qwen/Qwen2.5-1.5B-Instruct"
}

def load_model_and_tokenizer(model_name: str, model_path: str):
    """Load or download model and tokenizer"""
    import torch
    from transformers import AutoTokenizer, AutoModelForCausalLM
    
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
        tokenizer = AutoTokenizer.from_pretrained(hf_model_name)
        model = AutoModelForCausalLM.from_pretrained(
            hf_model_name,
            torch_dtype=torch.float16,
            device_map="auto",
            load_in_8bit=True
        )
        # Cache for future use
        tokenizer.save_pretrained(model_path)
        model.save_pretrained(model_path)
        print(f"Model cached to {model_path}")
    
    return tokenizer, model

def run_inference_logic(model_name: str, prompt: str, region: str, temperature: float = 0.1, max_tokens: int = 512):
    """Shared inference logic"""
    import torch
    
    start_time = time.time()
    
    try:
        # Validate model
        if model_name not in MODELS:
            return {"status": "error", "error": f"Unknown model: {model_name}", "region": region}
        
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

# US Region Function
@app.function(
    image=image,
    gpu="T4",
    volumes={"/models": models_volume},
    timeout=300,
    container_idle_timeout=600,  # Keep warm for 10 minutes
    region=["us-east", "us-central", "us-west"],
    memory=8192  # 8GB RAM
)
def run_inference_us(
    model_name: str,
    prompt: str, 
    temperature: float = 0.1,
    max_tokens: int = 512
) -> Dict[str, Any]:
    """HF Transformers inference - US region"""
    return run_inference_logic(model_name, prompt, "us-east", temperature, max_tokens)

# EU Region Function
@app.function(
    image=image,
    gpu="T4",
    volumes={"/models": models_volume},
    timeout=300,
    container_idle_timeout=600,
    region=["eu-west", "eu-north"],
    memory=8192
)
def run_inference_eu(
    model_name: str,
    prompt: str,
    temperature: float = 0.1,
    max_tokens: int = 512
) -> Dict[str, Any]:
    """HF Transformers inference - EU region"""
    return run_inference_logic(model_name, prompt, "eu-west", temperature, max_tokens)

# APAC Region Function
@app.function(
    image=image,
    gpu="T4",
    volumes={"/models": models_volume},
    timeout=300,
    container_idle_timeout=600,
    region=["ap-southeast", "ap-northeast"],
    memory=8192
)
def run_inference_apac(
    model_name: str,
    prompt: str,
    temperature: float = 0.1,
    max_tokens: int = 512
) -> Dict[str, Any]:
    """HF Transformers inference - APAC region"""
    return run_inference_logic(model_name, prompt, "asia-pacific", temperature, max_tokens)

# Health check function
@app.function(
    image=image,
    timeout=30,
)
def health_check() -> Dict[str, Any]:
    """Simple health check that always returns healthy"""
    return {
        "status": "healthy",
        "timestamp": time.time(),
        "models_available": list(MODELS.keys()),
        "regions": ["us-east", "eu-west", "asia-pacific"],
        "architecture": "hf-transformers"
    }

# Web endpoints for HTTP access
@app.function(
    image=image,
    timeout=30,
)
@modal.web_endpoint(method="POST")
def inference_api(item: Dict[str, Any]) -> Dict[str, Any]:
    """HTTP API endpoint for inference requests"""
    model_name = item.get("model", "llama3.2-1b")
    prompt = item.get("prompt", "")
    temperature = item.get("temperature", 0.1)
    max_tokens = item.get("max_tokens", 512)
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
)
@modal.web_endpoint(method="GET", label="hf-health")
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
        
        # Test US inference
        us_result = run_inference_us.remote(
            model_name="llama3.2-1b",
            prompt="What is artificial intelligence?",
            max_tokens=50
        )
        print(f"US inference: {us_result}")
