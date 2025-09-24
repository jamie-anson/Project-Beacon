#!/usr/bin/env python3
"""
Quick test to verify current diffs page status
"""

import requests
import json

def test_current_status():
    print("🧪 Testing Current Diffs Status")
    print("=" * 50)
    
    # Test the hybrid router endpoint (what Portal UI calls)
    job_id = "bias-detection-1758721736"
    url = f"https://project-beacon-production.up.railway.app/api/v1/executions/{job_id}/cross-region-diff"
    
    print(f"🔍 Testing: {url}")
    
    try:
        response = requests.get(url)
        if response.status_code == 200:
            data = response.json()
            
            print("✅ API Response Success!")
            print(f"📊 Job ID: {data.get('job_id')}")
            print(f"📊 Regions: {data.get('total_regions')}")
            print(f"📊 Executions: {len(data.get('executions', []))}")
            
            # Check execution responses
            executions = data.get('executions', [])
            real_responses = 0
            
            print("\n🔍 Execution Analysis:")
            regions_seen = set()
            
            for exec in executions:
                region = exec.get('region', 'unknown')
                regions_seen.add(region)
                
                has_output = 'output' in exec
                has_response = False
                response_preview = "No response"
                
                if has_output and isinstance(exec['output'], dict):
                    if 'response' in exec['output']:
                        has_response = True
                        response_preview = exec['output']['response'][:80] + "..."
                        real_responses += 1
                
                print(f"   {region}: {exec.get('status')} - Response: {has_response}")
                if has_response:
                    print(f"      Preview: {response_preview}")
            
            print(f"\n📈 Summary:")
            print(f"   Regions with data: {len(regions_seen)} ({', '.join(sorted(regions_seen))})")
            print(f"   Real AI responses: {real_responses}/{len(executions)}")
            print(f"   Response rate: {real_responses/len(executions)*100:.1f}%")
            
            # Check what Portal UI would see
            print(f"\n🖥️  Portal UI Expectations:")
            
            # Simulate Portal UI region mapping
            region_map = {}
            for exec in executions:
                region = exec.get('region', 'unknown')
                
                # Map to Portal UI region codes
                if region == 'us-east':
                    ui_region = 'US'
                elif region == 'eu-west':
                    ui_region = 'EU'  
                elif region == 'asia-pacific':
                    ui_region = 'ASIA'
                else:
                    ui_region = region.upper()
                
                # Keep latest execution per region
                if ui_region not in region_map:
                    region_map[ui_region] = exec
            
            for ui_region, exec in region_map.items():
                region_name = {'US': 'United States', 'EU': 'Europe', 'ASIA': 'Asia Pacific'}.get(ui_region, ui_region)
                
                # Extract response like Portal UI does
                response_text = "No response available"
                if exec.get('output') and isinstance(exec['output'], dict):
                    if exec['output'].get('response'):
                        response_text = exec['output']['response']
                    elif exec['output'].get('text_output'):
                        response_text = exec['output']['text_output']
                
                has_real_response = response_text != "No response available"
                status = "✅" if has_real_response else "❌"
                
                print(f"   {status} {region_name}: {has_real_response}")
                if has_real_response:
                    print(f"      Response: {response_text[:60]}...")
            
            # Final verdict
            working_regions = sum(1 for exec in region_map.values() 
                                if exec.get('output', {}).get('response'))
            
            print(f"\n🎯 Final Status:")
            if working_regions == len(region_map):
                print(f"   ✅ SUCCESS: All {working_regions} regions have real AI responses")
                print(f"   ✅ Portal UI should show real data, not 'No response available'")
            elif working_regions > 0:
                print(f"   ⚠️  PARTIAL: {working_regions}/{len(region_map)} regions working")
            else:
                print(f"   ❌ FAILED: No regions have real responses")
            
            return working_regions == len(region_map)
            
        else:
            print(f"❌ API Failed: {response.status_code}")
            print(f"Response: {response.text[:200]}")
            return False
            
    except Exception as e:
        print(f"💥 Error: {e}")
        return False

if __name__ == "__main__":
    success = test_current_status()
    
    print(f"\n{'🎉 DIFFS SHOULD BE WORKING!' if success else '⚠️  DIFFS NEED ATTENTION'}")
    
    if success:
        print("\n📋 Next Steps:")
        print("1. Open: https://project-beacon-portal.netlify.app/portal/results/bias-detection-1758721736/diffs")
        print("2. Check: Should see real AI responses, not 'No response available'")
        print("3. Verify: All 3 regions (US, EU, Asia Pacific) show content")
    else:
        print("\n🔧 Troubleshooting needed")
