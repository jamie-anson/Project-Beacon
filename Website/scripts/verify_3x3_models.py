#!/usr/bin/env python3
"""
3x3 Model Verification Script
Tests all 3 models across all 3 regions via Hybrid Router
"""

import asyncio
import aiohttp
import json
import time
import ssl
from typing import Dict, List, Any

# Configuration
# HYBRID_ROUTER_BASE = "http://localhost:8080"  # Test locally first
HYBRID_ROUTER_BASE = "https://project-beacon-production.up.railway.app"  # Production
MODELS = ["llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"]
REGIONS = ["us-east", "eu-west", "asia-pacific"]
TEST_PROMPT = "What is artificial intelligence?"
MAX_TOKENS = 50

class ModelVerifier:
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip('/')
        self.results = {}
        
    async def test_single_inference(self, session: aiohttp.ClientSession, model: str, region: str) -> Dict[str, Any]:
        """Test inference for a single model/region combination"""
        payload = {
            "model": model,
            "prompt": TEST_PROMPT,
            "temperature": 0.1,
            "max_tokens": MAX_TOKENS,
            "region_preference": region,
            "cost_priority": True
        }
        
        start_time = time.time()
        
        try:
            async with session.post(
                f"{self.base_url}/inference",
                json=payload,
                timeout=aiohttp.ClientTimeout(total=300)  # 5 minutes for Mistral 7B
            ) as response:
                response_time = time.time() - start_time
                
                if response.status == 200:
                    data = await response.json()
                    return {
                        "status": "success" if data.get("success") else "failed",
                        "response_time": response_time,
                        "provider_used": data.get("provider_used"),
                        "inference_time": data.get("inference_time", 0),
                        "response_length": len(data.get("response") or ""),
                        "error": data.get("error"),
                        "response_preview": (data.get("response") or "")[:100] + "..." if len(data.get("response") or "") > 100 else (data.get("response") or "")
                    }
                else:
                    error_text = await response.text()
                    return {
                        "status": "http_error",
                        "response_time": response_time,
                        "http_status": response.status,
                        "error": error_text[:200]
                    }
                    
        except asyncio.TimeoutError:
            return {
                "status": "timeout",
                "response_time": time.time() - start_time,
                "error": "Request timeout"
            }
        except Exception as e:
            return {
                "status": "error",
                "response_time": time.time() - start_time,
                "error": str(e)
            }
    
    async def test_router_health(self, session: aiohttp.ClientSession) -> Dict[str, Any]:
        """Test router health and provider status"""
        try:
            async with session.get(f"{self.base_url}/health") as response:
                if response.status == 200:
                    return await response.json()
                else:
                    return {"status": "unhealthy", "error": f"HTTP {response.status}"}
        except Exception as e:
            return {"status": "error", "error": str(e)}
    
    async def test_models_endpoint(self, session: aiohttp.ClientSession) -> Dict[str, Any]:
        """Test models endpoint"""
        try:
            async with session.get(f"{self.base_url}/models") as response:
                if response.status == 200:
                    return await response.json()
                else:
                    return {"error": f"HTTP {response.status}"}
        except Exception as e:
            return {"error": str(e)}
    
    async def test_providers_endpoint(self, session: aiohttp.ClientSession) -> Dict[str, Any]:
        """Test providers endpoint"""
        try:
            async with session.get(f"{self.base_url}/providers") as response:
                if response.status == 200:
                    return await response.json()
                else:
                    return {"error": f"HTTP {response.status}"}
        except Exception as e:
            return {"error": str(e)}
    
    async def run_verification(self) -> Dict[str, Any]:
        """Run complete 3x3 verification"""
        print(f"🚀 Starting 3x3 Model Verification")
        print(f"📍 Router: {self.base_url}")
        print(f"🤖 Models: {', '.join(MODELS)}")
        print(f"🌍 Regions: {', '.join(REGIONS)}")
        print(f"💬 Test prompt: '{TEST_PROMPT}'")
        print("=" * 60)
        
        # Create SSL context that doesn't verify certificates for local testing
        ssl_context = ssl.create_default_context()
        ssl_context.check_hostname = False
        ssl_context.verify_mode = ssl.CERT_NONE
        
        connector = aiohttp.TCPConnector(ssl=ssl_context)
        async with aiohttp.ClientSession(connector=connector) as session:
            # Test router health
            print("🏥 Testing router health...")
            health = await self.test_router_health(session)
            print(f"   Status: {health.get('status', 'unknown')}")
            print(f"   Providers: {health.get('providers_healthy', 0)}/{health.get('providers_total', 0)}")
            
            # Test models endpoint
            print("\n📋 Testing models endpoint...")
            models_info = await self.test_models_endpoint(session)
            if "models" in models_info:
                print(f"   Available models: {len(models_info['models'])}")
                print(f"   Regions: {', '.join(models_info.get('regions_available', []))}")
            else:
                print(f"   Models endpoint not available (deployment pending)")
                print(f"   Error: {models_info.get('error', 'Unknown error')}")
            
            # Test providers endpoint
            print("\n🔧 Testing providers endpoint...")
            providers_info = await self.test_providers_endpoint(session)
            if "providers" in providers_info:
                healthy_providers = [p for p in providers_info["providers"] if p.get("healthy")]
                print(f"   Healthy providers: {len(healthy_providers)}")
                for provider in healthy_providers:
                    endpoint_type = provider.get('endpoint_type', 'unknown')
                    models_supported = provider.get('models_supported', ['unknown'])
                    print(f"     - {provider['name']} ({provider['region']}) - {endpoint_type} - models: {len(models_supported)}")
            else:
                print(f"   Providers endpoint error: {providers_info.get('error', 'Unknown error')}")
            
            print("\n🧪 Running 3x3 inference matrix...")
            print("   Model x Region combinations:")
            print("   Note: Testing with current router (may not have all 3 models yet)")
            
            # Run all combinations
            tasks = []
            for model in MODELS:
                for region in REGIONS:
                    task = self.test_single_inference(session, model, region)
                    tasks.append((model, region, task))
            
            # Execute all tests concurrently
            results = {}
            for model, region, task in tasks:
                print(f"   🔄 Testing {model} in {region}...")
                result = await task
                
                if model not in results:
                    results[model] = {}
                results[model][region] = result
                
                # Print immediate result
                status_emoji = "✅" if result["status"] == "success" else "❌"
                provider = result.get("provider_used", "unknown")
                response_time = result.get("response_time", 0)
                print(f"     {status_emoji} {model} @ {region}: {result['status']} ({response_time:.2f}s) via {provider}")
                
                if result["status"] != "success":
                    error_msg = result.get('error', 'Unknown error')
                    if len(error_msg) > 100:
                        error_msg = error_msg[:100] + "..."
                    print(f"       Error: {error_msg}")
        
        return {
            "timestamp": time.time(),
            "router_health": health,
            "models_info": models_info,
            "providers_info": providers_info,
            "inference_results": results,
            "summary": self._generate_summary(results)
        }
    
    def _generate_summary(self, results: Dict[str, Dict[str, Dict[str, Any]]]) -> Dict[str, Any]:
        """Generate summary statistics"""
        total_tests = len(MODELS) * len(REGIONS)
        successful_tests = 0
        failed_tests = 0
        total_response_time = 0
        
        success_by_model = {}
        success_by_region = {}
        
        for model, regions in results.items():
            success_by_model[model] = 0
            for region, result in regions.items():
                if region not in success_by_region:
                    success_by_region[region] = 0
                
                if result["status"] == "success":
                    successful_tests += 1
                    success_by_model[model] += 1
                    success_by_region[region] += 1
                    total_response_time += result.get("response_time", 0)
                else:
                    failed_tests += 1
        
        return {
            "total_tests": total_tests,
            "successful_tests": successful_tests,
            "failed_tests": failed_tests,
            "success_rate": (successful_tests / total_tests) * 100 if total_tests > 0 else 0,
            "avg_response_time": total_response_time / successful_tests if successful_tests > 0 else 0,
            "success_by_model": {model: f"{count}/{len(REGIONS)}" for model, count in success_by_model.items()},
            "success_by_region": {region: f"{count}/{len(MODELS)}" for region, count in success_by_region.items()},
            "status": "PASS" if successful_tests == total_tests else "PARTIAL" if successful_tests > 0 else "FAIL"
        }

async def main():
    verifier = ModelVerifier(HYBRID_ROUTER_BASE)
    results = await verifier.run_verification()
    
    print("\n" + "=" * 60)
    print("📊 VERIFICATION SUMMARY")
    print("=" * 60)
    
    summary = results["summary"]
    print(f"🎯 Overall Status: {summary['status']}")
    print(f"📈 Success Rate: {summary['success_rate']:.1f}% ({summary['successful_tests']}/{summary['total_tests']})")
    print(f"⏱️  Average Response Time: {summary['avg_response_time']:.2f}s")
    
    print(f"\n🤖 Success by Model:")
    for model, success in summary["success_by_model"].items():
        print(f"   {model}: {success}")
    
    print(f"\n🌍 Success by Region:")
    for region, success in summary["success_by_region"].items():
        print(f"   {region}: {success}")
    
    # Save detailed results
    output_file = f"3x3_verification_{int(time.time())}.json"
    with open(output_file, 'w') as f:
        json.dump(results, f, indent=2)
    
    print(f"\n💾 Detailed results saved to: {output_file}")
    
    # Exit with appropriate code
    if summary["status"] == "PASS":
        print("\n🎉 All tests passed! 3x3 model matrix is fully operational.")
        exit(0)
    elif summary["status"] == "PARTIAL":
        print(f"\n⚠️  Partial success: {summary['failed_tests']} tests failed.")
        print("\n💡 This may be expected if the router deployment is still in progress.")
        exit(1)
    else:
        print("\n💥 All tests failed!")
        print("\n💡 This may be expected if the router deployment is still in progress.")
        exit(2)

if __name__ == "__main__":
    asyncio.run(main())
