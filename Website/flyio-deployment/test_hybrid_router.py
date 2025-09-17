"""
Test script for hybrid router deployment
Run this to test the Fly.io hybrid routing service
"""

import asyncio
import httpx
import json
import time
from typing import Dict, Any

class HybridRouterTester:
    def __init__(self, base_url: str = "https://beacon-hybrid-router.fly.dev"):
        self.base_url = base_url
        self.client = httpx.AsyncClient(timeout=30.0)
    
    async def test_health_check(self) -> Dict[str, Any]:
        """Test health check endpoint"""
        try:
            response = await self.client.get(f"{self.base_url}/health")
            return {
                "success": response.status_code == 200,
                "status_code": response.status_code,
                "data": response.json() if response.status_code == 200 else response.text
            }
        except Exception as e:
            return {"success": False, "error": str(e)}
    
    async def test_providers_list(self) -> Dict[str, Any]:
        """Test providers listing endpoint"""
        try:
            response = await self.client.get(f"{self.base_url}/providers")
            return {
                "success": response.status_code == 200,
                "status_code": response.status_code,
                "data": response.json() if response.status_code == 200 else response.text
            }
        except Exception as e:
            return {"success": False, "error": str(e)}
    
    async def test_inference(self, model: str = "llama3.2:1b", prompt: str = "What is AI?") -> Dict[str, Any]:
        """Test inference endpoint"""
        payload = {
            "model": model,
            "prompt": prompt,
            "temperature": 0.1,
            "max_tokens": 100,
            "cost_priority": True
        }
        
        try:
            response = await self.client.post(f"{self.base_url}/inference", json=payload)
            return {
                "success": response.status_code == 200,
                "status_code": response.status_code,
                "data": response.json() if response.status_code == 200 else response.text
            }
        except Exception as e:
            return {"success": False, "error": str(e)}
    
    async def test_metrics(self) -> Dict[str, Any]:
        """Test metrics endpoint"""
        try:
            response = await self.client.get(f"{self.base_url}/metrics")
            return {
                "success": response.status_code == 200,
                "status_code": response.status_code,
                "data": response.json() if response.status_code == 200 else response.text
            }
        except Exception as e:
            return {"success": False, "error": str(e)}
    
    async def test_env_endpoint(self) -> Dict[str, Any]:
        """Test environment variables endpoint"""
        try:
            response = await self.client.get(f"{self.base_url}/env")
            return {
                "success": response.status_code == 200,
                "status_code": response.status_code,
                "data": response.json() if response.status_code == 200 else response.text
            }
        except Exception as e:
            return {"success": False, "error": str(e)}
    
    async def test_modal_health(self) -> Dict[str, Any]:
        """Test Modal health endpoint"""
        try:
            response = await self.client.get(f"{self.base_url}/modal-health")
            return {
                "success": response.status_code == 200,
                "status_code": response.status_code,
                "data": response.json() if response.status_code == 200 else response.text
            }
        except Exception as e:
            return {"success": False, "error": str(e)}
    
    async def test_websocket_hint(self) -> Dict[str, Any]:
        """Test WebSocket HTTP hint endpoint (optional feature)"""
        try:
            response = await self.client.get(f"{self.base_url}/ws")
            return {
                "success": response.status_code == 200,
                "status_code": response.status_code,
                "data": response.json() if response.status_code == 200 else response.text,
                "websocket_enabled": response.status_code == 200
            }
        except Exception as e:
            return {"success": False, "error": str(e), "websocket_enabled": False}
    
    async def run_all_tests(self):
        """Run all tests"""
        print("🧪 Testing Project Beacon Hybrid Router...")
        print(f"📡 Base URL: {self.base_url}")
        print("ℹ️  Note: WebSocket functionality may be disabled in some deployments")
        print()
        
        # Test health check
        print("1️⃣ Testing health check...")
        health_result = await self.test_health_check()
        if health_result["success"]:
            print("✅ Health check passed")
            health_data = health_result["data"]
            print(f"   📊 Providers: {health_data.get('providers_healthy', 0)}/{health_data.get('providers_total', 0)} healthy")
            print(f"   🌍 Regions: {health_data.get('regions', [])}")
        else:
            print(f"❌ Health check failed: {health_result.get('error', 'Unknown error')}")
        print()
        
        # Test providers list
        print("2️⃣ Testing providers list...")
        providers_result = await self.test_providers_list()
        if providers_result["success"]:
            print("✅ Providers list retrieved")
            providers = providers_result["data"].get("providers", [])
            for provider in providers:
                status = "🟢" if provider["healthy"] else "🔴"
                print(f"   {status} {provider['name']} ({provider['type']}) - {provider['region']}")
        else:
            print(f"❌ Providers list failed: {providers_result.get('error', 'Unknown error')}")
        print()
        
        # Test metrics
        print("3️⃣ Testing metrics...")
        metrics_result = await self.test_metrics()
        if metrics_result["success"]:
            print("✅ Metrics retrieved")
            metrics = metrics_result["data"]
            print(f"   📈 Avg Latency: {metrics.get('avg_latency', 0):.3f}s")
            print(f"   📊 Success Rate: {metrics.get('avg_success_rate', 0):.1%}")
            cost_range = metrics.get('cost_range', {})
            print(f"   💰 Cost Range: ${cost_range.get('min', 0):.6f} - ${cost_range.get('max', 0):.6f}/sec")
        else:
            print(f"❌ Metrics failed: {metrics_result.get('error', 'Unknown error')}")
        print()
        
        # Test environment endpoint
        print("4️⃣ Testing environment endpoint...")
        env_result = await self.test_env_endpoint()
        if env_result["success"]:
            print("✅ Environment endpoint working")
            env_data = env_result["data"]
            golem_endpoints = env_data.get("GOLEM_PROVIDER_ENDPOINTS", "")
            modal_base = env_data.get("MODAL_API_BASE", "")
            print(f"   🔗 Golem Endpoints: {golem_endpoints[:50]}..." if golem_endpoints else "   🔗 Golem Endpoints: Not configured")
            print(f"   🚀 Modal Base: {modal_base}" if modal_base else "   🚀 Modal Base: Not configured")
        else:
            print(f"❌ Environment endpoint failed: {env_result.get('error', 'Unknown error')}")
        print()
        
        # Test Modal health endpoint
        print("5️⃣ Testing Modal health endpoint...")
        modal_health_result = await self.test_modal_health()
        if modal_health_result["success"]:
            print("✅ Modal health endpoint working")
            print(f"   📊 Status: {modal_health_result['data'].get('status', 'unknown')}")
        else:
            print(f"❌ Modal health failed: {modal_health_result.get('error', 'Unknown error')}")
        print()
        
        # Test WebSocket hint endpoint (optional)
        print("6️⃣ Testing WebSocket hint endpoint...")
        ws_hint_result = await self.test_websocket_hint()
        if ws_hint_result["success"]:
            print("✅ WebSocket hint endpoint working")
            print(f"   💬 Message: {ws_hint_result['data'].get('message', '')[:60]}...")
            print("   🔌 WebSocket functionality: ENABLED")
        else:
            websocket_enabled = ws_hint_result.get("websocket_enabled", False)
            if ws_hint_result.get("status_code") == 404:
                print("ℹ️  WebSocket endpoint not found (feature disabled)")
                print("   🔌 WebSocket functionality: DISABLED")
            else:
                print(f"❌ WebSocket hint failed: {ws_hint_result.get('error', 'Unknown error')}")
                print("   🔌 WebSocket functionality: UNKNOWN")
        print()

        # Test inference (only if providers are healthy)
        if health_result["success"] and health_result["data"].get("providers_healthy", 0) > 0:
            print("7️⃣ Testing inference...")
            inference_result = await self.test_inference()
            if inference_result["success"]:
                print("✅ Inference completed")
                data = inference_result["data"]
                print(f"   🤖 Provider: {data.get('provider_used', 'unknown')}")
                print(f"   ⏱️  Time: {data.get('inference_time', 0):.3f}s")
                print(f"   💰 Cost: ${data.get('cost_estimate', 0):.6f}")
                print(f"   📝 Response: {data.get('response', '')[:100]}...")
            else:
                print(f"❌ Inference failed: {inference_result.get('error', 'Unknown error')}")
        else:
            print("7️⃣ Skipping inference test (no healthy providers)")
        print()
        
        await self.client.aclose()
        print("🎉 Testing completed!")

async def main():
    # Test local deployment
    local_tester = HybridRouterTester("http://localhost:8080")
    print("Testing local deployment...")
    await local_tester.run_all_tests()
    
    print("\n" + "="*50 + "\n")
    
    # Test Fly.io deployment (if available)
    fly_tester = HybridRouterTester("https://beacon-hybrid-router.fly.dev")
    print("Testing Fly.io deployment...")
    await fly_tester.run_all_tests()

if __name__ == "__main__":
    asyncio.run(main())
