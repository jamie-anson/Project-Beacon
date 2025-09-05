"""
Modal deployment for Project Beacon LLM inference
Supports Llama 3.2-1B, Mistral 7B, and Qwen 2.5-1.5B models
"""

import modal
import os
import json
from typing import Dict, Any, List

# Create Modal app
app = modal.App("project-beacon-inference")

# Define container image with Ollama and models
image = (
    modal.Image.debian_slim(python_version="3.11")
    .apt_install("curl", "wget")
    .run_commands(
        # Install Ollama
        "curl -fsSL https://ollama.com/install.sh | sh",
        # Set environment variables
        "export OLLAMA_HOST=0.0.0.0:11434",
    )
    .pip_install(
        "requests",
        "pydantic",
        "fastapi",
    )
)

# Shared volume for model storage
models_volume = modal.Volume.from_name("beacon-models", create_if_missing=True)

@app.function(
    image=image,
    gpu=modal.gpu.T4(),  # Start with T4 for cost efficiency
    volumes={"/models": models_volume},
    timeout=300,
    container_idle_timeout=60,
    allow_concurrent_inputs=10,
)
def setup_models():
    """Pre-load models into the container"""
    import subprocess
    import time
    
    # Start Ollama service
    ollama_process = subprocess.Popen(
        ["ollama", "serve"],
        env={**os.environ, "OLLAMA_HOST": "0.0.0.0:11434"}
    )
    
    # Wait for Ollama to start
    time.sleep(5)
    
    # Pull required models
    models = [
        "llama3.2:1b",
        "mistral:7b", 
        "qwen2.5:1.5b"
    ]
    
    for model in models:
        print(f"Pulling model: {model}")
        result = subprocess.run(
            ["ollama", "pull", model],
            capture_output=True,
            text=True
        )
        if result.returncode == 0:
            print(f"Successfully pulled {model}")
        else:
            print(f"Failed to pull {model}: {result.stderr}")
    
    return {"status": "Models loaded", "models": models}

@app.function(
    image=image,
    gpu=modal.gpu.T4(),
    volumes={"/models": models_volume},
    timeout=60,
    container_idle_timeout=300,  # Keep warm for 5 minutes
    allow_concurrent_inputs=5,
)
def run_inference(
    model: str,
    prompt: str,
    temperature: float = 0.1,
    max_tokens: int = 512
) -> Dict[str, Any]:
    """Run inference on specified model"""
    import subprocess
    import requests
    import time
    import json
    
    # Start Ollama service if not running
    try:
        response = requests.get("http://localhost:11434/api/tags", timeout=2)
    except:
        print("Starting Ollama service...")
        ollama_process = subprocess.Popen(
            ["ollama", "serve"],
            env={**os.environ, "OLLAMA_HOST": "0.0.0.0:11434"}
        )
        time.sleep(3)
    
    # Prepare request payload
    payload = {
        "model": model,
        "prompt": prompt,
        "stream": False,
        "options": {
            "temperature": temperature,
            "num_predict": max_tokens
        }
    }
    
    start_time = time.time()
    
    try:
        # Make inference request
        response = requests.post(
            "http://localhost:11434/api/generate",
            json=payload,
            timeout=45
        )
        
        if response.status_code == 200:
            result = response.json()
            inference_time = time.time() - start_time
            
            return {
                "success": True,
                "response": result.get("response", ""),
                "model": model,
                "inference_time": inference_time,
                "tokens_generated": len(result.get("response", "").split()),
                "metadata": {
                    "temperature": temperature,
                    "max_tokens": max_tokens,
                    "prompt_length": len(prompt)
                }
            }
        else:
            return {
                "success": False,
                "error": f"HTTP {response.status_code}: {response.text}",
                "model": model,
                "inference_time": time.time() - start_time
            }
            
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "model": model,
            "inference_time": time.time() - start_time
        }

@app.function(
    image=image,
    gpu=modal.gpu.A10(),  # Upgrade to A10 for batch processing
    volumes={"/models": models_volume},
    timeout=300,
    container_idle_timeout=600,  # Keep warm longer for batch jobs
    allow_concurrent_inputs=1,
)
def run_batch_inference(
    requests: List[Dict[str, Any]]
) -> List[Dict[str, Any]]:
    """Run batch inference for multiple requests"""
    import subprocess
    import requests as http_requests
    import time
    import json
    
    # Start Ollama service
    try:
        response = http_requests.get("http://localhost:11434/api/tags", timeout=2)
    except:
        print("Starting Ollama service...")
        ollama_process = subprocess.Popen(
            ["ollama", "serve"],
            env={**os.environ, "OLLAMA_HOST": "0.0.0.0:11434"}
        )
        time.sleep(5)
    
    results = []
    batch_start_time = time.time()
    
    for req in requests:
        model = req.get("model", "llama3.2:1b")
        prompt = req.get("prompt", "")
        temperature = req.get("temperature", 0.1)
        max_tokens = req.get("max_tokens", 512)
        
        # Run individual inference
        result = run_inference.local(model, prompt, temperature, max_tokens)
        result["batch_id"] = req.get("id", "unknown")
        results.append(result)
    
    batch_time = time.time() - batch_start_time
    
    return {
        "batch_results": results,
        "batch_time": batch_time,
        "batch_size": len(requests),
        "success_count": sum(1 for r in results if r.get("success", False))
    }

@app.function(
    image=image,
    timeout=30,
)
def health_check() -> Dict[str, Any]:
    """Health check endpoint for monitoring"""
    import subprocess
    import requests
    import time
    
    try:
        # Check if Ollama is responsive
        response = requests.get("http://localhost:11434/api/tags", timeout=5)
        if response.status_code == 200:
            models = response.json().get("models", [])
            return {
                "status": "healthy",
                "timestamp": time.time(),
                "models_loaded": len(models),
                "available_models": [m.get("name", "") for m in models]
            }
        else:
            return {
                "status": "unhealthy",
                "timestamp": time.time(),
                "error": f"Ollama returned {response.status_code}"
            }
    except Exception as e:
        return {
            "status": "unhealthy", 
            "timestamp": time.time(),
            "error": str(e)
        }

# Web endpoint for HTTP API access
@app.function(
    image=image,
    gpu=modal.gpu.T4(),
    volumes={"/models": models_volume},
    container_idle_timeout=300,
    allow_concurrent_inputs=10,
)
@modal.web_endpoint(method="POST")
def inference_api(item: Dict[str, Any]) -> Dict[str, Any]:
    """HTTP API endpoint for inference requests"""
    model = item.get("model", "llama3.2:1b")
    prompt = item.get("prompt", "")
    temperature = item.get("temperature", 0.1)
    max_tokens = item.get("max_tokens", 512)
    
    if not prompt:
        return {
            "success": False,
            "error": "Prompt is required"
        }
    
    return run_inference.local(model, prompt, temperature, max_tokens)

if __name__ == "__main__":
    # Local testing
    with app.run():
        # Setup models
        print("Setting up models...")
        setup_result = setup_models.remote()
        print(f"Setup result: {setup_result}")
        
        # Test inference
        print("Testing inference...")
        test_result = run_inference.remote(
            model="llama3.2:1b",
            prompt="What is artificial intelligence?",
            temperature=0.1,
            max_tokens=100
        )
        print(f"Test result: {test_result}")
        
        # Test health check
        print("Testing health check...")
        health_result = health_check.remote()
        print(f"Health result: {health_result}")
