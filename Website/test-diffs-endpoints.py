#!/usr/bin/env python3
"""
Manual test script for diffs view endpoints
Tests all the API endpoints that the Portal UI tries to call
"""

import requests
import json
import time
from typing import Dict, Any

# Test configuration
MAIN_BACKEND = "https://beacon-runner-change-me.fly.dev"
HYBRID_ROUTER = "https://project-beacon-production.up.railway.app"
DIFFS_BACKEND = "https://project-beacon-portal.netlify.app/backend-diffs"
TEST_JOB_ID = "bias-detection-1758721736"

def test_endpoint(url: str, name: str) -> Dict[str, Any]:
    """Test a single endpoint and return results"""
    print(f"🧪 Testing {name}: {url}")
    
    try:
        start_time = time.time()
        response = requests.get(url, timeout=10)
        duration = time.time() - start_time
        
        result = {
            "name": name,
            "url": url,
            "status_code": response.status_code,
            "duration_ms": round(duration * 1000, 2),
            "success": response.status_code == 200,
            "content_length": len(response.content),
            "headers": dict(response.headers)
        }
        
        if response.status_code == 200:
            try:
                data = response.json()
                result["has_json"] = True
                result["json_keys"] = list(data.keys()) if isinstance(data, dict) else "array"
                print(f"   ✅ SUCCESS ({response.status_code}) - {result['duration_ms']}ms - {result['content_length']} bytes")
                if isinstance(data, dict) and len(data) < 5:
                    print(f"   📄 Data: {json.dumps(data, indent=2)[:200]}...")
            except:
                result["has_json"] = False
                print(f"   ✅ SUCCESS ({response.status_code}) - {result['duration_ms']}ms - Non-JSON response")
        else:
            result["has_json"] = False
            result["error_text"] = response.text[:200]
            print(f"   ❌ FAILED ({response.status_code}) - {result['duration_ms']}ms - {response.text[:100]}")
            
    except requests.exceptions.Timeout:
        result = {"name": name, "url": url, "error": "timeout", "success": False}
        print(f"   ⏰ TIMEOUT - Request took longer than 10s")
    except requests.exceptions.RequestException as e:
        result = {"name": name, "url": url, "error": str(e), "success": False}
        print(f"   💥 ERROR - {str(e)}")
    
    return result

def main():
    print("🚀 Starting Diffs View Endpoint Testing")
    print("=" * 60)
    
    # Test endpoints in order of Portal UI priority
    endpoints = [
        # Diffs backend endpoints (tried first)
        (f"{DIFFS_BACKEND}/api/v1/diffs/recent?limit=10", "Diffs Backend - Recent"),
        (f"{DIFFS_BACKEND}/api/v1/diffs/by-job/{TEST_JOB_ID}", "Diffs Backend - By Job"),
        (f"{DIFFS_BACKEND}/api/v1/diffs/cross-region/{TEST_JOB_ID}", "Diffs Backend - Cross Region"),
        (f"{DIFFS_BACKEND}/api/v1/diffs/jobs/{TEST_JOB_ID}", "Diffs Backend - Jobs"),
        
        # Hybrid router endpoints (temporary)
        (f"{HYBRID_ROUTER}/api/v1/executions/{TEST_JOB_ID}/cross-region-diff", "Hybrid Router - Cross Region Diff"),
        
        # Main backend endpoints (fallback)
        (f"{MAIN_BACKEND}/api/v1/executions/{TEST_JOB_ID}/cross-region-diff", "Main Backend - Cross Region Diff"),
        (f"{MAIN_BACKEND}/api/v1/executions/{TEST_JOB_ID}/regions", "Main Backend - Regions"),
        (f"{MAIN_BACKEND}/api/v1/executions/{TEST_JOB_ID}/cross-region", "Main Backend - Cross Region"),
        (f"{MAIN_BACKEND}/api/v1/executions/{TEST_JOB_ID}/diff-analysis", "Main Backend - Diff Analysis"),
        
        # Individual execution endpoints
        (f"{MAIN_BACKEND}/api/v1/executions/637/details", "Main Backend - Execution Details"),
        (f"{MAIN_BACKEND}/api/v1/jobs/{TEST_JOB_ID}/executions/all", "Main Backend - Job Executions"),
    ]
    
    results = []
    for url, name in endpoints:
        result = test_endpoint(url, name)
        results.append(result)
        print()  # Empty line for readability
    
    # Summary
    print("📊 TEST SUMMARY")
    print("=" * 60)
    
    successful = [r for r in results if r.get("success")]
    failed = [r for r in results if not r.get("success")]
    
    print(f"✅ Successful endpoints: {len(successful)}/{len(results)}")
    print(f"❌ Failed endpoints: {len(failed)}/{len(results)}")
    print()
    
    if successful:
        print("🎉 WORKING ENDPOINTS:")
        for result in successful:
            print(f"   ✅ {result['name']} ({result.get('status_code', 'N/A')})")
    
    if failed:
        print("\n💥 FAILED ENDPOINTS:")
        for result in failed:
            error = result.get('error', f"HTTP {result.get('status_code', 'N/A')}")
            print(f"   ❌ {result['name']} - {error}")
    
    # Save detailed results
    with open('diffs-endpoint-test-results.json', 'w') as f:
        json.dump({
            "test_timestamp": time.time(),
            "test_job_id": TEST_JOB_ID,
            "summary": {
                "total_endpoints": len(results),
                "successful": len(successful),
                "failed": len(failed),
                "success_rate": len(successful) / len(results) * 100
            },
            "results": results
        }, f, indent=2)
    
    print(f"\n💾 Detailed results saved to: diffs-endpoint-test-results.json")
    
    # Recommendations
    print("\n🎯 RECOMMENDATIONS:")
    if len(successful) == 0:
        print("   🚨 No endpoints working - check backend deployments")
    elif len(successful) < len(results) / 2:
        print("   ⚠️  Most endpoints failing - focus on backend integration")
    else:
        print("   ✅ Some endpoints working - focus on Portal UI integration")

if __name__ == "__main__":
    main()
