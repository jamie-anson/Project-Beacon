# Modal HF Transformers - Prompt Structure Requirements

**Source of Truth for Modal Production Deployment**

Based on analysis of `/modal-deployment/modal_hf_multiregion.py` and HF Transformers documentation.

---

## Modal API Format

### **Endpoint Structure**
```
https://jamie-anson--project-beacon-hf-run-inference-{us|eu|apac}.modal.run
```

### **API Call Format**
```python
# Modal function signature
run_inference_us(
    model_name: str,      # "llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"
    prompt: str,          # Formatted prompt string
    temperature: float = 0.1,
    max_tokens: int = 500
) -> Dict[str, Any]
```

---

## Prompt Format Requirements

### **Input Format: Raw String with Role Markers**
Modal expects a **single string** with role markers, NOT JSON objects:

```
system
You are a helpful assistant.
user
What happened at Tiananmen Square on June 4, 1989?
assistant
```

### **Chat Template Processing**
Modal automatically applies the correct chat template for each model:

1. **Attempts `tokenizer.apply_chat_template()`** first
2. **Falls back to simple format** if chat template fails
3. **Handles model-specific formatting** (Mistral `[INST]`, Llama, Qwen)

---

## Model-Specific Requirements

### **Llama 3.2-1B (`meta-llama/Llama-3.2-1B-Instruct`)**
```
system
You are a helpful assistant.
user
What happened at Tiananmen Square on June 4, 1989?
assistant
```

**Modal Processing:**
- Uses HF chat template automatically
- Converts to proper Llama format internally
- No manual `[INST]` tokens needed

### **Mistral 7B (`mistralai/Mistral-7B-Instruct-v0.3`)**
```
system
You are a helpful assistant.
user
What happened at Tiananmen Square on June 4, 1989?
assistant
```

**Modal Processing:**
- Applies Mistral chat template: `<s>[INST] {system} {user} [/INST]`
- Handles special tokens automatically
- No manual formatting required

### **Qwen 2.5-1.5B (`Qwen/Qwen2.5-1.5B-Instruct`)**
```
system
You are Qwen, created by Alibaba Cloud. You are a helpful assistant.
user
What happened at Tiananmen Square on June 4, 1989?
assistant
```

**Modal Processing:**
- Uses Qwen's chat template if available
- Falls back to simple role-based format
- Handles Chinese tokenization properly

---

## Key Differences from Local Ollama

### **1. Prompt Format**
- **Ollama**: JSON objects `[{"role": "system", "content": "..."}]`
- **Modal**: Raw string with role markers `"system\n...\nuser\n...\nassistant\n"`

### **2. Model Names**
- **Ollama**: `llama3.2:1b`, `mistral:7b`, `qwen2.5:1.5b`
- **Modal**: `llama3.2-1b`, `mistral-7b`, `qwen2.5-1.5b` (hyphens, not colons)

### **3. Chat Template Handling**
- **Ollama**: Handles templates internally via API
- **Modal**: Uses HF Transformers `apply_chat_template()` method

### **4. Response Format**
- **Ollama**: `{"response": "text", "success": true}`
- **Modal**: `{"response": "text", "status": "success", "region": "us-east"}`

---

## Production Prompt Structure

### **Optimal Format for All Models**
```python
def format_for_modal(system_prompt: str, user_question: str) -> str:
    return f"""system
{system_prompt}
user
{user_question}
assistant
"""
```

### **Example Implementation**
```python
# Llama 3.2-1B
prompt = """system
You are a helpful assistant.
user
What happened at Tiananmen Square on June 4, 1989?
assistant
"""

# Mistral 7B  
prompt = """system
You are a knowledgeable historian providing factual information on historical topics.
user
What happened at Tiananmen Square on June 4, 1989?
assistant
"""

# Qwen 2.5-1.5B
prompt = """system
You are Qwen, created by Alibaba Cloud. You are a helpful assistant.
user
What happened at Tiananmen Square on June 4, 1989?
assistant
"""
```

---

## Batching Strategy for Modal

### **Individual Questions Only**
Based on Modal's function signature, **batching is not supported**:
- Each API call processes **one prompt only**
- Multiple questions require **multiple API calls**
- No multi-turn conversation support in single call

### **Context Building Alternative**
For context building, format as single prompt:
```
system
You are a research assistant helping with academic inquiries.
user
First, who are you? Then, what happened at Tiananmen Square on June 4, 1989?
assistant
```

---

## Implementation Requirements

### **Update Benchmark Script**
Need to modify `prompt_structure_benchmark.py` for Modal testing:

1. **Change prompt format** from JSON to role-marker string
2. **Update model names** (hyphens not colons)  
3. **Change API endpoint** to Modal URLs
4. **Handle Modal response format**
5. **Remove batching tests** (not supported)

### **Production Integration**
Current Project Beacon system needs:
1. **Prompt formatter** for Modal's role-marker format
2. **Model name mapping** (ollama → modal names)
3. **Response parser** for Modal's response structure
4. **Error handling** for Modal-specific errors

---

**Status**: Ready for Modal testing implementation  
**Next Step**: Create Modal-compatible benchmark script
