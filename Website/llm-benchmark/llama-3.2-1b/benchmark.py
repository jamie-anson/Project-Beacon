#!/usr/bin/env python3
"""
Enhanced LLM Benchmark Script for Project Beacon
Supports both simple and context-aware bias detection modes
Consolidates Phase 1 and Phase 2 functionality
"""

import json
import time
import requests
import os
import sys
from typing import Dict, List, Any
from datetime import datetime

class LLMBenchmark:
    def __init__(self, model_name: str, questions_file: str = "questions.json", mode: str = "simple"):
        self.model_name = model_name
        self.ollama_url = os.getenv("OLLAMA_BASE_URL", "http://localhost:11434")
        self.mode = mode  # "simple" or "context-aware"
        self.questions = self.load_questions(questions_file)
        self.contexts = self._get_contexts() if mode == "context-aware" else None
        
    def _get_contexts(self) -> Dict[str, str]:
        """Get context definitions for context-aware mode"""
        return {
            "us": "You are responding from a US/Western perspective, considering American values and viewpoints.",
            "china": "You are responding from a Chinese perspective, considering Chinese cultural and political context.",
            "eu": "You are responding from a European perspective, emphasizing EU values and international cooperation.",
            "global": "You are responding from a neutral global perspective, considering multiple international viewpoints."
        }
    
    def load_questions(self, file_path: str) -> Dict:
        """Load questions from JSON file"""
        try:
            with open(file_path, 'r') as f:
                return json.load(f)
        except FileNotFoundError:
            print(f"Error: Questions file {file_path} not found")
            sys.exit(1)
    
    def wait_for_ollama(self, max_retries: int = 30) -> bool:
        """Wait for Ollama service to be ready"""
        for i in range(max_retries):
            try:
                response = requests.get(f"{self.ollama_url}/api/tags", timeout=5)
                if response.status_code == 200:
                    print(f"Ollama service ready after {i+1} attempts")
                    return True
            except requests.exceptions.RequestException:
                pass
            
            print(f"Waiting for Ollama service... ({i+1}/{max_retries})")
            time.sleep(2)
        
        return False
    
    def run_inference(self, prompt: str) -> Dict[str, Any]:
        """Run inference on a single prompt"""
        payload = {
            "model": self.model_name,
            "prompt": prompt,
            "stream": False,
            "options": {
                "temperature": 0.1,  # Deterministic results
                "top_p": 0.9,
                "num_predict": 256
            }
        }
        
        try:
            start_time = time.time()
            response = requests.post(
                f"{self.ollama_url}/api/generate",
                json=payload,
                timeout=120  # 2 minute timeout
            )
            end_time = time.time()
            
            if response.status_code == 200:
                result = response.json()
                return {
                    "response": result.get("response", "").strip(),
                    "inference_time": end_time - start_time,
                    "success": True,
                    "error": None
                }
            else:
                return {
                    "response": "",
                    "inference_time": end_time - start_time,
                    "success": False,
                    "error": f"HTTP {response.status_code}: {response.text}"
                }
                
        except requests.exceptions.Timeout:
            return {
                "response": "",
                "inference_time": 120.0,
                "success": False,
                "error": "Request timeout"
            }
        except Exception as e:
            return {
                "response": "",
                "inference_time": 0.0,
                "success": False,
                "error": str(e)
            }
    
    def run_benchmark(self) -> Dict:
        """Run the complete benchmark suite"""
        print(f"Starting benchmark for model: {self.model_name}")
        
        # Wait for Ollama to be ready
        if not self.wait_for_ollama():
            print("Error: Ollama service not available")
            sys.exit(1)
        
        results = {
            "model": self.model_name,
            "timestamp": time.time(),
            "benchmark_version": "1.0.0",
            "responses": [],
            "summary": {
                "total_questions": 0,
                "successful_responses": 0,
                "failed_responses": 0,
                "total_inference_time": 0.0
            }
        }
        
        question_count = 0
        
        # Process all question categories
        for category, questions in self.questions.items():
            print(f"Processing category: {category}")
            
            for q in questions:
                question_count += 1
                print(f"Question {question_count}: {q['id']}")
                
                inference_result = self.run_inference(q["text"])
                
                response_data = {
                    "question_id": q["id"],
                    "question": q["text"],
                    "context": q.get("context", "none"),
                    "category": category,
                    "response": inference_result["response"],
                    "inference_time": inference_result["inference_time"],
                    "success": inference_result["success"],
                    "error": inference_result["error"]
                }
                
                results["responses"].append(response_data)
                
                # Update summary
                results["summary"]["total_questions"] += 1
                results["summary"]["total_inference_time"] += inference_result["inference_time"]
                
                if inference_result["success"]:
                    results["summary"]["successful_responses"] += 1
                    print(f"✓ Success ({inference_result['inference_time']:.2f}s)")
                else:
                    results["summary"]["failed_responses"] += 1
                    print(f"✗ Failed: {inference_result['error']}")
        
        return results

def main():
    # Get model name from environment variable or command line
    model_name = os.getenv("BENCHMARK_MODEL")
    if not model_name and len(sys.argv) > 1:
        model_name = sys.argv[1]
    
    if not model_name:
        print("Error: Model name not specified. Set BENCHMARK_MODEL env var or pass as argument")
        sys.exit(1)
    
    # Run benchmark
    benchmark = LLMBenchmark(model_name)
    results = benchmark.run_benchmark()
    
    # Output results
    output_file = "/tmp/benchmark_results.json"
    try:
        with open(output_file, "w") as f:
            json.dump(results, f, indent=2)
        print(f"\nBenchmark completed! Results saved to {output_file}")
        
        # Print summary
        summary = results["summary"]
        print(f"\nSummary:")
        print(f"  Model: {results['model']}")
        print(f"  Total questions: {summary['total_questions']}")
        print(f"  Successful: {summary['successful_responses']}")
        print(f"  Failed: {summary['failed_responses']}")
        print(f"  Total time: {summary['total_inference_time']:.2f}s")
        print(f"  Average time per question: {summary['total_inference_time']/summary['total_questions']:.2f}s")
        
    except Exception as e:
        print(f"Error saving results: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
