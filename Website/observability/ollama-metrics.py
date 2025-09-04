#!/usr/bin/env python3
"""
Basic Ollama metrics collector for Project Beacon
Exports request durations, model usage, and GPU utilization
"""

import requests
import time
import json
import subprocess
from datetime import datetime
from typing import Dict, Any

class OllamaMetrics:
    def __init__(self, ollama_url: str = "http://127.0.0.1:11434"):
        self.ollama_url = ollama_url
        self.metrics = []
        
    def get_gpu_stats(self) -> Dict[str, Any]:
        """Get GPU utilization stats (macOS Metal)"""
        try:
            # For macOS, use system_profiler for GPU info
            result = subprocess.run(
                ["system_profiler", "SPDisplaysDataType", "-json"],
                capture_output=True, text=True, timeout=5
            )
            if result.returncode == 0:
                data = json.loads(result.stdout)
                displays = data.get("SPDisplaysDataType", [])
                if displays:
                    gpu_info = displays[0]
                    return {
                        "gpu_name": gpu_info.get("sppci_model", "Unknown"),
                        "vram_total": gpu_info.get("sppci_vram", "Unknown"),
                        "timestamp": datetime.utcnow().isoformat()
                    }
        except Exception as e:
            print(f"GPU stats error: {e}")
        
        return {"gpu_available": False, "timestamp": datetime.utcnow().isoformat()}
    
    def get_ollama_models(self) -> Dict[str, Any]:
        """Get loaded models and their stats"""
        try:
            response = requests.get(f"{self.ollama_url}/api/tags", timeout=5)
            if response.status_code == 200:
                models = response.json().get("models", [])
                return {
                    "models_loaded": len(models),
                    "models": [
                        {
                            "name": m["name"],
                            "size_gb": round(m["size"] / (1024**3), 2),
                            "parameter_size": m["details"]["parameter_size"]
                        }
                        for m in models
                    ],
                    "timestamp": datetime.utcnow().isoformat()
                }
        except Exception as e:
            print(f"Ollama models error: {e}")
        
        return {"models_loaded": 0, "timestamp": datetime.utcnow().isoformat()}
    
    def test_inference_latency(self, model: str = "llama3.2:1b") -> Dict[str, Any]:
        """Test inference latency for monitoring"""
        start_time = time.time()
        try:
            response = requests.post(
                f"{self.ollama_url}/api/generate",
                json={
                    "model": model,
                    "prompt": "Hello",
                    "stream": False
                },
                timeout=30
            )
            
            duration = time.time() - start_time
            
            if response.status_code == 200:
                data = response.json()
                return {
                    "model": model,
                    "status": "success",
                    "duration_seconds": round(duration, 3),
                    "response_length": len(data.get("response", "")),
                    "timestamp": datetime.utcnow().isoformat()
                }
            else:
                return {
                    "model": model,
                    "status": "error",
                    "duration_seconds": round(duration, 3),
                    "error_code": response.status_code,
                    "timestamp": datetime.utcnow().isoformat()
                }
                
        except Exception as e:
            duration = time.time() - start_time
            return {
                "model": model,
                "status": "timeout",
                "duration_seconds": round(duration, 3),
                "error": str(e),
                "timestamp": datetime.utcnow().isoformat()
            }
    
    def collect_metrics(self) -> Dict[str, Any]:
        """Collect all metrics"""
        return {
            "gpu_stats": self.get_gpu_stats(),
            "ollama_models": self.get_ollama_models(),
            "inference_test": self.test_inference_latency(),
            "collection_timestamp": datetime.utcnow().isoformat()
        }
    
    def export_metrics(self, output_file: str = "/tmp/ollama_metrics.json"):
        """Export metrics to JSON file"""
        metrics = self.collect_metrics()
        with open(output_file, 'w') as f:
            json.dump(metrics, f, indent=2)
        print(f"Metrics exported to {output_file}")
        return metrics

if __name__ == "__main__":
    collector = OllamaMetrics()
    metrics = collector.export_metrics()
    
    # Print summary
    print("\n=== Ollama Metrics Summary ===")
    print(f"GPU: {metrics['gpu_stats'].get('gpu_name', 'Unknown')}")
    print(f"Models loaded: {metrics['ollama_models']['models_loaded']}")
    print(f"Inference test: {metrics['inference_test']['status']} ({metrics['inference_test']['duration_seconds']}s)")
