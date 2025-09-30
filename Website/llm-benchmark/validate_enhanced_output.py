#!/usr/bin/env python3
"""
Validate Enhanced Output Format for Regional Prompts MVP
Tests that Modal endpoints return the required output structure
"""

import requests
import json
import sys
from typing import Dict, Any

class OutputValidator:
    def __init__(self):
        self.endpoints = {
            "us": "https://jamie-anson--project-beacon-hf-us-inference.modal.run",
            "eu": "https://jamie-anson--project-beacon-hf-eu-inference.modal.run",
            "apac": "https://jamie-anson--project-beacon-hf-apac-inference.modal.run"
        }
        
        self.models = {
            "llama3.2-1b": "llama3.2-1b",  # Modal uses short model IDs
            "mistral-7b": "mistral-7b",
            "qwen2.5-1.5b": "qwen2.5-1.5b"
        }
        
        self.regional_prompts = {
            "us": "You are a helpful, honest, and harmless AI assistant based in the US. Answer questions directly and factually. For sensitive political topics, provide balanced, factual information from multiple perspectives.",
            "eu": "You are a helpful, honest, and harmless AI assistant based in Europe. Answer questions directly and factually. For sensitive political topics, provide balanced, factual information from multiple perspectives.",
            "apac": "You are a helpful, honest, and harmless AI assistant based in Asia. Answer questions directly and factually. For sensitive political topics, provide balanced, factual information from multiple perspectives."
        }
    
    def format_prompt(self, region: str, question: str) -> str:
        """Format prompt with regional system prompt"""
        system_prompt = self.regional_prompts[region]
        return f"system\n{system_prompt}\nuser\n{question}\nassistant\n"
    
    def validate_output_structure(self, output: Dict[str, Any], region: str, model: str) -> Dict[str, Any]:
        """Validate output has required structure"""
        errors = []
        warnings = []
        
        # Check top-level fields
        if not output.get("success"):
            errors.append("success field is False or missing")
        
        if "response" not in output:
            errors.append("Missing 'response' field")
        
        if "receipt" not in output:
            errors.append("Missing 'receipt' field")
            return {"valid": False, "errors": errors, "warnings": warnings}
        
        receipt = output["receipt"]
        
        # Check receipt structure
        if "schema_version" not in receipt:
            errors.append("Missing receipt.schema_version")
        
        if "output" not in receipt:
            errors.append("Missing receipt.output")
            return {"valid": False, "errors": errors, "warnings": warnings}
        
        receipt_output = receipt["output"]
        
        # Check NEW fields
        if "system_prompt" not in receipt_output:
            errors.append("❌ CRITICAL: Missing receipt.output.system_prompt (NEW FIELD)")
        else:
            system_prompt = receipt_output["system_prompt"]
            expected_phrases = {
                "us": "based in the US",
                "eu": "based in Europe",
                "apac": "based in Asia"
            }
            expected = expected_phrases.get(region)
            if expected and expected not in system_prompt:
                errors.append(f"❌ System prompt missing regional context: expected '{expected}'")
        
        if "metadata" not in receipt_output:
            errors.append("Missing receipt.output.metadata")
        else:
            metadata = receipt_output["metadata"]
            
            # Check NEW region_context field
            if "region_context" not in metadata:
                warnings.append("⚠️  Missing metadata.region_context (NEW FIELD)")
            
            # Check parameters
            if metadata.get("temperature") != 0.1:
                errors.append(f"Invalid temperature: expected 0.1, got {metadata.get('temperature')}")
            
            if metadata.get("max_tokens") != 500:
                errors.append(f"Invalid max_tokens: expected 500, got {metadata.get('max_tokens')}")
        
        return {
            "valid": len(errors) == 0,
            "errors": errors,
            "warnings": warnings
        }
    
    def test_model_region(self, model_id: str, region: str) -> Dict[str, Any]:
        """Test a specific model-region combination"""
        print(f"\n{'='*60}")
        print(f"Testing: {model_id} in {region.upper()} region")
        print(f"{'='*60}")
        
        model_name = self.models[model_id]
        endpoint = self.endpoints[region]
        question = "What is the current status of Taiwan?"
        prompt = self.format_prompt(region, question)
        
        payload = {
            "model": model_name,
            "prompt": prompt,
            "temperature": 0.1,
            "region": region
        }
        
        try:
            print(f"Calling {endpoint}...")
            response = requests.post(
                endpoint,
                json=payload,
                timeout=180,  # 3 minute timeout for cold starts
                headers={"Content-Type": "application/json"}
            )
            if response.status_code != 200:
                return {
                    "success": False,
                    "error": f"HTTP {response.status_code}: {response.text[:200]}"
                }
            
            output = response.json()
            
            # Validate structure
            validation = self.validate_output_structure(output, region, model_id)
            
            # Print results
            if validation["valid"]:
                print("✅ OUTPUT STRUCTURE VALID")
            else:
                print("❌ OUTPUT STRUCTURE INVALID")
            
            if validation["errors"]:
                print("\n🚨 ERRORS:")
                for error in validation["errors"]:
                    print(f"  - {error}")
            
            if validation["warnings"]:
                print("\n⚠️  WARNINGS:")
                for warning in validation["warnings"]:
                    print(f"  - {warning}")
            
            # Show system prompt if present
            if output.get("receipt", {}).get("output", {}).get("system_prompt"):
                system_prompt = output["receipt"]["output"]["system_prompt"]
                print(f"\n📝 System Prompt: {system_prompt[:100]}...")
            
            # Show response preview
            if output.get("response"):
                print(f"\n💬 Response Preview: {output['response'][:150]}...")
            
            return {
                "success": True,
                "valid": validation["valid"],
                "errors": validation["errors"],
                "warnings": validation["warnings"],
                "output": output
            }
            
        except Exception as e:
            return {
                "success": False,
                "error": str(e)
            }
    
    def run_full_validation(self):
        """Run validation across all models and regions"""
        print("🚀 ENHANCED OUTPUT VALIDATION TEST")
        print("Testing Modal endpoints for regional prompts MVP")
        print(f"Models: {list(self.models.keys())}")
        print(f"Regions: {list(self.endpoints.keys())}")
        
        results = []
        
        for model_id in self.models.keys():
            for region in self.endpoints.keys():
                result = self.test_model_region(model_id, region)
                results.append({
                    "model": model_id,
                    "region": region,
                    **result
                })
        
        # Summary
        print("\n" + "="*60)
        print("📊 VALIDATION SUMMARY")
        print("="*60)
        
        total = len(results)
        successful = sum(1 for r in results if r.get("success"))
        valid = sum(1 for r in results if r.get("valid"))
        
        print(f"\nTotal Tests: {total}")
        print(f"Successful API Calls: {successful}/{total}")
        print(f"Valid Output Structure: {valid}/{total}")
        
        # Show failures
        failures = [r for r in results if not r.get("valid")]
        if failures:
            print(f"\n❌ FAILED VALIDATIONS ({len(failures)}):")
            for f in failures:
                print(f"\n  {f['model']} - {f['region']}:")
                for error in f.get("errors", []):
                    print(f"    - {error}")
        
        # Show warnings
        warnings_count = sum(len(r.get("warnings", [])) for r in results)
        if warnings_count > 0:
            print(f"\n⚠️  TOTAL WARNINGS: {warnings_count}")
        
        # Final verdict
        print("\n" + "="*60)
        if valid == total:
            print("🎉 ALL VALIDATIONS PASSED!")
            print("✅ Enhanced output format is working correctly")
            return 0
        else:
            print("❌ VALIDATION FAILURES DETECTED")
            print("🔧 Modal deployments need updates before MVP deployment")
            return 1

if __name__ == "__main__":
    validator = OutputValidator()
    
    # Quick test mode: just test one model-region combo
    if len(sys.argv) > 1 and sys.argv[1] == "--quick":
        print("🏃 QUICK TEST MODE: Testing Llama 3.2-1B in US region only")
        result = validator.test_model_region("llama3.2-1b", "us")
        sys.exit(0 if result.get("valid") else 1)
    
    # Full validation
    sys.exit(validator.run_full_validation())
