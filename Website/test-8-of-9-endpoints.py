#!/usr/bin/env python3
"""
Test script for 8/9 endpoint graceful failure scenario
Tests all 9 model/region combinations with expected graceful failure for EU Mistral 7B
"""

import requests
import json
import time
from concurrent.futures import ThreadPoolExecutor, as_completed

# Test configuration
BASE_URL = "https://project-beacon-production.up.railway.app"
MODELS = ["llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"]
REGIONS = ["us-east", "eu-west", "asia-pacific"]

# Test matrix: 3 models × 3 regions = 9 endpoints
TEST_MATRIX = [
    (model, region) for model in MODELS for region in REGIONS
]

def test_endpoint(model, region):
    """Test a single model/region endpoint"""
    print(f"🧪 Testing {model} in {region}...")
    
    payload = {
        "model": model,
        "prompt": f"Test {model} in {region}",
        "temperature": 0.1,
        "max_tokens": 10,
        "region_preference": region
    }
    
    start_time = time.time()
    try:
        response = requests.post(
            f"{BASE_URL}/inference",
            json=payload,
            timeout=60
        )
        
        duration = time.time() - start_time
        
        if response.status_code == 200:
            result = response.json()
            success = result.get("success", False)
            error = result.get("error", "")
            test_mode = result.get("metadata", {}).get("test_mode", "")
            
            status = "✅ SUCCESS" if success else "❌ FAILED"
            if test_mode == "graceful_failure":
                status = "🧪 GRACEFUL_FAILURE"
            
            return {
                "model": model,
                "region": region,
                "status": status,
                "success": success,
                "duration": f"{duration:.2f}s",
                "error": error,
                "test_mode": test_mode,
                "response_data": result
            }
        else:
            return {
                "model": model,
                "region": region,
                "status": "❌ HTTP_ERROR",
                "success": False,
                "duration": f"{duration:.2f}s",
                "error": f"HTTP {response.status_code}: {response.text}",
                "test_mode": "",
                "response_data": None
            }
            
    except requests.exceptions.Timeout:
        return {
            "model": model,
            "region": region,
            "status": "⏰ TIMEOUT",
            "success": False,
            "duration": "60.00s+",
            "error": "Request timeout",
            "test_mode": "",
            "response_data": None
        }
    except Exception as e:
        return {
            "model": model,
            "region": region,
            "status": "💥 EXCEPTION",
            "success": False,
            "duration": f"{time.time() - start_time:.2f}s",
            "error": str(e),
            "test_mode": "",
            "response_data": None
        }

def main():
    print("🚀 Starting 8/9 Endpoint Graceful Failure Test")
    print("=" * 60)
    print(f"Testing {len(TEST_MATRIX)} endpoints:")
    for model, region in TEST_MATRIX:
        expected = "GRACEFUL_FAILURE" if (model == "mistral-7b" and region == "eu-west") else "SUCCESS"
        print(f"  • {model} @ {region} (expected: {expected})")
    print("=" * 60)
    
    # Run tests in parallel for faster execution
    results = []
    with ThreadPoolExecutor(max_workers=3) as executor:
        future_to_test = {
            executor.submit(test_endpoint, model, region): (model, region)
            for model, region in TEST_MATRIX
        }
        
        for future in as_completed(future_to_test):
            result = future.result()
            results.append(result)
    
    # Sort results by region then model for better display
    results.sort(key=lambda x: (x["region"], x["model"]))
    
    # Display results
    print("\n📊 Test Results:")
    print("=" * 80)
    
    success_count = 0
    graceful_failure_count = 0
    unexpected_failure_count = 0
    
    for result in results:
        model = result["model"]
        region = result["region"]
        status = result["status"]
        duration = result["duration"]
        error = result["error"]
        
        error_text = error[:40] if error else ""
        print(f"{status} {model:15} @ {region:15} ({duration:>8}) {error_text}")
        
        if result["success"]:
            success_count += 1
        elif result["test_mode"] == "graceful_failure":
            graceful_failure_count += 1
        else:
            unexpected_failure_count += 1
    
    # Summary
    print("=" * 80)
    print(f"📈 Summary:")
    print(f"  ✅ Successful endpoints:     {success_count}/9 ({success_count/9*100:.1f}%)")
    print(f"  🧪 Graceful failures:       {graceful_failure_count}/9 ({graceful_failure_count/9*100:.1f}%)")
    print(f"  ❌ Unexpected failures:     {unexpected_failure_count}/9 ({unexpected_failure_count/9*100:.1f}%)")
    print(f"  🎯 Total working endpoints:  {success_count + graceful_failure_count}/9 ({(success_count + graceful_failure_count)/9*100:.1f}%)")
    
    # Expected: 8 success + 1 graceful failure = 9 total working
    expected_success = 8
    expected_graceful = 1
    
    if success_count == expected_success and graceful_failure_count == expected_graceful:
        print("\n🎉 TEST PASSED: Perfect 8/9 success rate with graceful failure!")
        print("   Ready for cross-region diffs testing with partial data handling.")
    else:
        print(f"\n⚠️  TEST ISSUES: Expected {expected_success} success + {expected_graceful} graceful failure")
        print(f"   Got {success_count} success + {graceful_failure_count} graceful failure")
    
    # Save detailed results for analysis
    with open("8-of-9-test-results.json", "w") as f:
        json.dump(results, f, indent=2)
    print(f"\n💾 Detailed results saved to: 8-of-9-test-results.json")

if __name__ == "__main__":
    main()
