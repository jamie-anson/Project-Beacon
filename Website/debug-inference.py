#!/usr/bin/env python3
"""
Debug script to test inference endpoint with detailed error reporting
"""

import requests
import json

def test_inference_debug():
    """Test inference with detailed error reporting"""
    
    # Test the graceful failure case first (should be fastest)
    payload = {
        "model": "mistral-7b",
        "prompt": "Test",
        "temperature": 0.1,
        "max_tokens": 5,
        "region_preference": "eu-west"
    }
    
    print("🧪 Testing EU Mistral 7B (graceful failure case)...")
    print(f"Payload: {json.dumps(payload, indent=2)}")
    
    try:
        response = requests.post(
            "https://project-beacon-production.up.railway.app/inference",
            json=payload,
            timeout=10
        )
        
        print(f"Status Code: {response.status_code}")
        print(f"Headers: {dict(response.headers)}")
        
        if response.status_code == 200:
            result = response.json()
            print(f"Success Response: {json.dumps(result, indent=2)}")
        else:
            print(f"Error Response: {response.text}")
            
    except requests.exceptions.Timeout:
        print("❌ Request timed out")
    except Exception as e:
        print(f"❌ Exception: {e}")

if __name__ == "__main__":
    test_inference_debug()
