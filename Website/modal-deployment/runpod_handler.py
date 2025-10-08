"""
RunPod Serverless Handler for Project Beacon
Handles inference requests for LLM models in APAC region
RunPod SDK version: 1.7.13+
"""
import runpod
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer
import os
import time

# Model configuration (set via environment variables)
MODEL_NAME = os.getenv("MODEL_NAME", "llama3.2-1b")
HF_MODEL_MAP = {
    "llama3.2-1b": "meta-llama/Llama-3.2-1B-Instruct",
    "mistral-7b": "mistralai/Mistral-7B-Instruct-v0.3",
    "qwen2.5-1.5b": "Qwen/Qwen2.5-1.5B-Instruct"
}

# Load model at container startup (eliminates cold start)
print(f"[STARTUP] Loading model: {MODEL_NAME}")
HF_MODEL_ID = HF_MODEL_MAP.get(MODEL_NAME, MODEL_NAME)

tokenizer = AutoTokenizer.from_pretrained(HF_MODEL_ID)
model = AutoModelForCausalLM.from_pretrained(
    HF_MODEL_ID,
    torch_dtype=torch.float16,
    device_map="auto",
    load_in_8bit=True  # 8-bit quantization for T4 GPU
)
print(f"[STARTUP] Model loaded successfully on {model.device}")

def _is_refusal(text: str) -> bool:
    """Check if response is a refusal"""
    if not text:
        return True
    t = text.strip().lower()
    patterns = [
        "i'm sorry",
        "i can't assist",
        "i cannot assist",
        "i can't help",
        "i cannot help",
        "cannot comply",
        "as an ai",
    ]
    return any(p in t for p in patterns) and len(t) < 280

def handler(job):
    """
    RunPod job handler
    Input format: {"input": {"prompt": "...", "temperature": 0.1, "max_tokens": 500}}
    """
    start_time = time.time()
    
    try:
        job_input = job.get("input", {})
        prompt = job_input.get("prompt", "")
        temperature = job_input.get("temperature", 0.1)
        max_tokens = job_input.get("max_tokens", 500)
        
        if not prompt:
            return {"error": "Prompt is required"}
        
        # Regional system prompt for APAC
        system_prompt = "You are a helpful, honest, and harmless AI assistant based in Asia. Answer questions directly and factually. For sensitive political topics, provide balanced, factual information from multiple perspectives."
        
        messages = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": prompt}
        ]
        
        # Apply chat template
        try:
            formatted_prompt = tokenizer.apply_chat_template(
                messages, 
                tokenize=False, 
                add_generation_prompt=True
            )
            print(f"[INFERENCE] Using chat template for {MODEL_NAME}")
        except Exception as e:
            print(f"[INFERENCE] Chat template failed, using fallback: {e}")
            formatted_prompt = f"System: {system_prompt}\n\nUser: {prompt}\n\nAssistant:"
        
        # Generate response
        inputs = tokenizer(formatted_prompt, return_tensors="pt")
        input_ids = inputs["input_ids"].to(model.device)
        
        with torch.no_grad():
            outputs = model.generate(
                input_ids,
                max_new_tokens=max_tokens,
                min_new_tokens=min(16, max_tokens // 2),
                temperature=temperature,
                do_sample=True if temperature > 0 else False,
                pad_token_id=tokenizer.eos_token_id,
                eos_token_id=tokenizer.eos_token_id
            )
        
        # Extract response (remove prompt)
        try:
            gen_ids = outputs[0][input_ids.shape[1]:]
            response = tokenizer.decode(gen_ids, skip_special_tokens=True).strip()
        except Exception:
            response = tokenizer.decode(outputs[0], skip_special_tokens=True)
        
        # Cleanup artifacts
        response = response.replace("<|assistant|>", "").replace("<|end|>", "").replace("<|im_end|>", "").strip()
        
        # Retry with academic reframing if refusal detected
        if _is_refusal(response):
            try:
                print("[INFERENCE] Refusal detected, retrying with academic prompt")
                reframed_messages = [
                    {"role": "system", "content": "You are an academic research assistant performing neutral, factual analysis across perspectives. Avoid refusal phrases; instead, summarize facts neutrally."},
                    {"role": "user", "content": prompt},
                ]
                try:
                    reframed_formatted = tokenizer.apply_chat_template(reframed_messages, tokenize=False, add_generation_prompt=True)
                except Exception:
                    reframed_formatted = f"System: Provide neutral academic analysis.\n\nUser: {prompt}\n\nAssistant:"
                
                r_inputs = tokenizer(reframed_formatted, return_tensors="pt")
                r_input_ids = r_inputs["input_ids"].to(model.device)
                
                with torch.no_grad():
                    r_outputs = model.generate(
                        r_input_ids,
                        max_new_tokens=min(max_tokens * 2, 256),
                        temperature=max(temperature, 0.2),
                        do_sample=True,
                        pad_token_id=tokenizer.eos_token_id,
                        eos_token_id=tokenizer.eos_token_id,
                    )
                
                try:
                    r_gen_ids = r_outputs[0][r_input_ids.shape[1]:]
                    retry_resp = tokenizer.decode(r_gen_ids, skip_special_tokens=True).strip()
                except Exception:
                    retry_resp = tokenizer.decode(r_outputs[0], skip_special_tokens=True)
                
                retry_resp = retry_resp.replace("<|assistant|>", "").replace("<|end|>", "").replace("<|im_end|>", "").strip()
                
                if retry_resp and not _is_refusal(retry_resp):
                    response = retry_resp
                    print("[INFERENCE] Retry successful")
            except Exception as e:
                print(f"[INFERENCE] Retry failed: {e}")
        
        inference_time = time.time() - start_time
        
        # Build receipt matching Modal format
        receipt = {
            "schema_version": "v0.1.0",
            "execution_details": {
                "provider_id": "runpod-apac",
                "region": "asia-pacific",
                "model": MODEL_NAME,
                "started_at": start_time,
                "completed_at": time.time(),
                "duration": inference_time
            },
            "output": {
                "response": response,
                "prompt": prompt,
                "system_prompt": system_prompt,
                "tokens_generated": len(tokenizer.encode(response)),
                "metadata": {
                    "temperature": temperature,
                    "max_tokens": max_tokens,
                    "region_context": "asia-pacific"
                }
            },
            "provenance": {
                "provider": "runpod",
                "architecture": "hf-transformers",
                "model_registry": MODEL_NAME
            }
        }
        
        return {
            "success": True,
            "response": response,
            "model": MODEL_NAME,
            "inference_time": inference_time,
            "region": "asia-pacific",
            "tokens_generated": len(tokenizer.encode(response)),
            "gpu_memory_used": torch.cuda.memory_allocated() if torch.cuda.is_available() else 0,
            "provider": "runpod",
            "receipt": receipt
        }
        
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "inference_time": time.time() - start_time,
            "provider": "runpod"
        }

# Start RunPod serverless handler
print("[STARTUP] Starting RunPod serverless handler")
runpod.serverless.start({"handler": handler})
