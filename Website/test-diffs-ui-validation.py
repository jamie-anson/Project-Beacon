#!/usr/bin/env python3
"""
Diffs UI Validation Test Suite
Tests the complete data flow from API to UI expectations
"""

import requests
import json
import time
from typing import Dict, Any, List

# Test configuration
MAIN_BACKEND = "https://beacon-runner-change-me.fly.dev"
HYBRID_ROUTER = "https://project-beacon-production.up.railway.app"
TEST_JOB_ID = "bias-detection-1758721736"

def test_execution_details_format():
    """Test that execution details have the format Portal UI expects"""
    print("🧪 Testing execution details format...")
    
    # Get basic executions first
    executions_url = f"{MAIN_BACKEND}/api/v1/jobs/{TEST_JOB_ID}/executions/all"
    response = requests.get(executions_url)
    
    if response.status_code != 200:
        print(f"❌ Failed to get executions: {response.status_code}")
        return False
    
    executions_data = response.json()
    basic_executions = executions_data.get("executions", [])
    
    if not basic_executions:
        print("❌ No executions found")
        return False
    
    print(f"✅ Found {len(basic_executions)} executions")
    
    # Test detailed execution data for each
    valid_executions = 0
    for exec_basic in basic_executions[:3]:  # Test first 3
        exec_id = exec_basic.get("id")
        if not exec_id:
            continue
            
        detail_url = f"{MAIN_BACKEND}/api/v1/executions/{exec_id}/details"
        detail_response = requests.get(detail_url)
        
        if detail_response.status_code == 200:
            detailed_exec = detail_response.json()
            
            # Check Portal UI requirements
            has_output = "output" in detailed_exec
            has_response = False
            response_preview = "No response"
            
            if has_output and isinstance(detailed_exec["output"], dict):
                output = detailed_exec["output"]
                if "response" in output:
                    has_response = True
                    response_preview = output["response"][:100] + "..."
            
            region = detailed_exec.get("region", "unknown")
            status = detailed_exec.get("status", "unknown")
            
            print(f"   Execution {exec_id} ({region}):")
            print(f"     Status: {status}")
            print(f"     Has output: {has_output}")
            print(f"     Has response: {has_response}")
            print(f"     Response preview: {response_preview}")
            
            if has_response:
                valid_executions += 1
        else:
            print(f"   ❌ Failed to get details for {exec_id}: {detail_response.status_code}")
    
    success_rate = (valid_executions / len(basic_executions)) * 100
    print(f"📊 Valid executions: {valid_executions}/{len(basic_executions)} ({success_rate:.1f}%)")
    
    return valid_executions > 0

def test_hybrid_router_data_structure():
    """Test that hybrid router returns data in expected format"""
    print("\n🧪 Testing hybrid router data structure...")
    
    url = f"{HYBRID_ROUTER}/api/v1/executions/{TEST_JOB_ID}/cross-region-diff"
    response = requests.get(url)
    
    if response.status_code != 200:
        print(f"❌ Hybrid router failed: {response.status_code}")
        return False
    
    data = response.json()
    
    # Check required fields
    required_fields = ["job_id", "total_regions", "executions", "analysis"]
    missing_fields = [field for field in required_fields if field not in data]
    
    if missing_fields:
        print(f"❌ Missing required fields: {missing_fields}")
        return False
    
    print("✅ All required fields present")
    
    # Check executions structure
    executions = data.get("executions", [])
    print(f"📊 Found {len(executions)} executions")
    
    # Check if executions have Portal UI requirements
    valid_for_ui = 0
    for exec in executions:
        has_output = "output" in exec
        has_response = False
        
        if has_output and isinstance(exec["output"], dict):
            if "response" in exec["output"]:
                has_response = True
        
        if has_response:
            valid_for_ui += 1
        
        region = exec.get("region", "unknown")
        print(f"   {region}: output={has_output}, response={has_response}")
    
    ui_ready_rate = (valid_for_ui / len(executions)) * 100 if executions else 0
    print(f"📊 UI-ready executions: {valid_for_ui}/{len(executions)} ({ui_ready_rate:.1f}%)")
    
    return ui_ready_rate > 0

def test_portal_ui_data_transformation():
    """Test the data transformation logic that Portal UI uses"""
    print("\n🧪 Testing Portal UI data transformation...")
    
    # Get data from hybrid router
    url = f"{HYBRID_ROUTER}/api/v1/executions/{TEST_JOB_ID}/cross-region-diff"
    response = requests.get(url)
    
    if response.status_code != 200:
        print(f"❌ Failed to get data: {response.status_code}")
        return False
    
    api_data = response.json()
    executions = api_data.get("executions", [])
    
    # Simulate Portal UI transformation logic
    region_map = {}
    for exec in executions:
        region = exec.get("region", "unknown")
        
        # Map region names like Portal UI does
        if region == "us-east":
            region_code = "US"
        elif region == "eu-west":
            region_code = "EU"
        elif region == "asia-pacific":
            region_code = "ASIA"
        else:
            region_code = region.upper()
        
        # Only keep the latest execution per region (like Portal UI)
        if region_code not in region_map:
            region_map[region_code] = exec
    
    print(f"📊 Region mapping: {list(region_map.keys())}")
    
    # Test response extraction for each region
    ui_models = []
    for region_code in ["US", "EU", "ASIA"]:
        if region_code not in region_map:
            print(f"   ❌ Missing region: {region_code}")
            continue
        
        exec = region_map[region_code]
        region_name = {"US": "United States", "EU": "Europe", "ASIA": "Asia Pacific"}[region_code]
        
        # Extract response like Portal UI does
        response = "No response available"
        if exec.get("output"):
            output = exec["output"]
            if isinstance(output, dict):
                if output.get("response"):
                    response = output["response"]
                elif output.get("text_output"):
                    response = output["text_output"]
                elif output.get("output"):
                    response = output["output"]
        
        ui_model = {
            "region_code": region_code,
            "region_name": region_name,
            "flag": {"US": "🇺🇸", "EU": "🇪🇺", "ASIA": "🌏"}[region_code],
            "status": exec.get("status", "completed"),
            "provider_id": exec.get("provider_id", "unknown"),
            "response": response,
            "has_real_response": response != "No response available"
        }
        
        ui_models.append(ui_model)
        
        print(f"   {region_name}:")
        print(f"     Status: {ui_model['status']}")
        print(f"     Provider: {ui_model['provider_id']}")
        print(f"     Has real response: {ui_model['has_real_response']}")
        print(f"     Response preview: {response[:100]}...")
    
    working_responses = len([m for m in ui_models if m["has_real_response"]])
    success_rate = (working_responses / len(ui_models)) * 100 if ui_models else 0
    
    print(f"📊 UI transformation result: {working_responses}/{len(ui_models)} regions with responses ({success_rate:.1f}%)")
    
    return success_rate > 0

def test_railway_deployment_readiness():
    """Test if Railway has deployed the latest changes"""
    print("\n🧪 Testing Railway deployment readiness...")
    
    # Check if the hybrid router is returning detailed execution data
    url = f"{HYBRID_ROUTER}/api/v1/executions/{TEST_JOB_ID}/cross-region-diff"
    response = requests.get(url)
    
    if response.status_code != 200:
        print(f"❌ Hybrid router not responding: {response.status_code}")
        return False
    
    data = response.json()
    executions = data.get("executions", [])
    
    if not executions:
        print("❌ No executions in response")
        return False
    
    # Check if executions have detailed output (sign of new deployment)
    detailed_count = 0
    for exec in executions:
        if "output" in exec and isinstance(exec["output"], dict) and "response" in exec["output"]:
            detailed_count += 1
    
    deployment_ready = detailed_count > 0
    print(f"📊 Executions with detailed output: {detailed_count}/{len(executions)}")
    
    if deployment_ready:
        print("✅ Railway deployment appears to be ready!")
    else:
        print("⏳ Railway deployment still in progress...")
    
    return deployment_ready

def main():
    """Run all tests"""
    print("🚀 Diffs UI Validation Test Suite")
    print("=" * 60)
    
    results = {}
    
    # Test 1: Execution details format
    results["execution_details"] = test_execution_details_format()
    
    # Test 2: Hybrid router data structure
    results["hybrid_router_structure"] = test_hybrid_router_data_structure()
    
    # Test 3: Portal UI transformation
    results["ui_transformation"] = test_portal_ui_data_transformation()
    
    # Test 4: Railway deployment readiness
    results["deployment_ready"] = test_railway_deployment_readiness()
    
    # Summary
    print("\n📊 TEST SUMMARY")
    print("=" * 60)
    
    passed = sum(results.values())
    total = len(results)
    
    for test_name, passed in results.items():
        status = "✅ PASS" if passed else "❌ FAIL"
        print(f"{status} {test_name.replace('_', ' ').title()}")
    
    print(f"\nOverall: {passed}/{total} tests passed ({passed/total*100:.1f}%)")
    
    if results.get("deployment_ready"):
        print("\n🎉 Ready to test diffs page!")
    else:
        print("\n⏳ Wait a few more minutes for Railway deployment")
    
    # Save results
    with open("diffs-ui-validation-results.json", "w") as f:
        json.dump({
            "timestamp": time.time(),
            "test_job_id": TEST_JOB_ID,
            "results": results,
            "summary": f"{passed}/{total} tests passed"
        }, f, indent=2)
    
    print(f"\n💾 Results saved to: diffs-ui-validation-results.json")

if __name__ == "__main__":
    main()
