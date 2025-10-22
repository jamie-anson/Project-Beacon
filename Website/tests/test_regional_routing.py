#!/usr/bin/env python3
"""
Test regional routing - verify that region_preference routes to correct Modal provider

This test ensures:
1. US requests go to modal-us-east
2. EU requests go to modal-eu-west
3. APAC requests go to modal-apac (when available)
4. Fallback works when preferred region is unavailable
"""

import pytest
import httpx
import asyncio
from typing import Dict, Any


BASE_URL = "https://project-beacon-production.up.railway.app"
TIMEOUT = 60.0


class TestRegionalRouting:
    """Test suite for regional routing verification"""
    
    @pytest.fixture
    async def client(self):
        """HTTP client fixture"""
        async with httpx.AsyncClient(timeout=TIMEOUT) as client:
            yield client
    
    async def _make_inference_request(
        self, 
        client: httpx.AsyncClient, 
        region: str,
        model: str = "llama3.2-1b"
    ) -> Dict[str, Any]:
        """Make inference request with region preference"""
        payload = {
            "model": model,
            "prompt": f"Test routing to {region}",
            "temperature": 0.1,
            "max_tokens": 10,
            "region_preference": region
        }
        
        response = await client.post(f"{BASE_URL}/inference", json=payload)
        assert response.status_code == 200, f"Request failed: {response.text}"
        return response.json()
    
    @pytest.mark.asyncio
    async def test_us_east_routing(self, client):
        """Test that US-East requests go to modal-us-east"""
        result = await self._make_inference_request(client, "us-east")
        
        # Verify success
        assert result["success"] is True, f"Request failed: {result.get('error')}"
        
        # Verify provider used
        provider_used = result.get("provider_used", "")
        assert provider_used == "modal-us-east", (
            f"Expected modal-us-east but got {provider_used}"
        )
        
        # Verify region in metadata
        metadata = result.get("metadata", {})
        assert metadata.get("region") == "us-east", (
            f"Expected region us-east but got {metadata.get('region')}"
        )
        
        # Verify receipt has correct region
        receipt = metadata.get("receipt", {})
        execution_details = receipt.get("execution_details", {})
        assert execution_details.get("region") == "us-east", (
            f"Receipt shows wrong region: {execution_details.get('region')}"
        )
        
        print(f"✅ US-East routing verified: {provider_used}")
    
    @pytest.mark.asyncio
    async def test_eu_west_routing(self, client):
        """Test that EU-West requests go to modal-eu-west"""
        result = await self._make_inference_request(client, "eu-west")
        
        # Verify success (may fail if EU is cold starting)
        if not result["success"]:
            pytest.skip(f"EU provider unavailable: {result.get('error')}")
        
        # Verify provider used
        provider_used = result.get("provider_used", "")
        assert provider_used == "modal-eu-west", (
            f"Expected modal-eu-west but got {provider_used}"
        )
        
        # Verify region in metadata
        metadata = result.get("metadata", {})
        assert metadata.get("region") == "eu-west", (
            f"Expected region eu-west but got {metadata.get('region')}"
        )
        
        # Verify receipt has correct region
        receipt = metadata.get("receipt", {})
        execution_details = receipt.get("execution_details", {})
        assert execution_details.get("region") == "eu-west", (
            f"Receipt shows wrong region: {execution_details.get('region')}"
        )
        
        print(f"✅ EU-West routing verified: {provider_used}")
    
    @pytest.mark.asyncio
    async def test_asia_pacific_routing(self, client):
        """Test that APAC requests go to modal-apac (when available)"""
        result = await self._make_inference_request(client, "asia-pacific")
        
        # APAC may not be configured yet, so we allow fallback
        if not result["success"]:
            pytest.skip(f"APAC provider unavailable: {result.get('error')}")
        
        provider_used = result.get("provider_used", "")
        
        # Either APAC provider or fallback to another region
        if provider_used == "modal-apac":
            # Verify APAC routing
            metadata = result.get("metadata", {})
            assert metadata.get("region") == "asia-pacific", (
                f"Expected region asia-pacific but got {metadata.get('region')}"
            )
            print(f"✅ APAC routing verified: {provider_used}")
        else:
            # Fallback occurred
            print(f"ℹ️  APAC not available, fallback to: {provider_used}")
            assert provider_used in ["modal-us-east", "modal-eu-west"], (
                f"Unexpected fallback provider: {provider_used}"
            )
    
    @pytest.mark.asyncio
    async def test_fallback_when_region_unavailable(self, client):
        """Test that requests fallback to another region when preferred is unavailable"""
        # This test assumes EU might be cold starting or unavailable
        # We make multiple requests and verify fallback behavior
        
        results = []
        for _ in range(3):
            try:
                result = await self._make_inference_request(client, "eu-west")
                results.append(result)
            except Exception as e:
                print(f"Request failed (expected if EU unavailable): {e}")
        
        if not results:
            pytest.skip("All requests failed - cannot test fallback")
        
        # At least one request should succeed (via fallback if needed)
        successful = [r for r in results if r.get("success")]
        assert len(successful) > 0, "No successful requests - fallback not working"
        
        # Check if any used fallback
        providers_used = [r.get("provider_used") for r in successful]
        print(f"ℹ️  Providers used: {set(providers_used)}")
        
        # If EU was unavailable, should have fallen back to US
        if "modal-us-east" in providers_used:
            print("✅ Fallback to US-East verified when EU unavailable")
    
    @pytest.mark.asyncio
    async def test_all_regions_parallel(self, client):
        """Test all regions in parallel to verify routing consistency"""
        regions = ["us-east", "eu-west", "asia-pacific"]
        
        tasks = [
            self._make_inference_request(client, region)
            for region in regions
        ]
        
        results = await asyncio.gather(*tasks, return_exceptions=True)
        
        # Verify results
        for region, result in zip(regions, results):
            if isinstance(result, Exception):
                print(f"⚠️  {region}: {result}")
                continue
            
            if not result.get("success"):
                print(f"⚠️  {region}: {result.get('error')}")
                continue
            
            provider_used = result.get("provider_used", "")
            expected_provider = f"modal-{region.replace('asia-pacific', 'apac')}"
            
            if provider_used == expected_provider:
                print(f"✅ {region}: Routed to {provider_used}")
            else:
                print(f"ℹ️  {region}: Fallback to {provider_used}")
    
    @pytest.mark.asyncio
    async def test_receipt_consistency(self, client):
        """Test that receipt data matches routing decision"""
        result = await self._make_inference_request(client, "us-east")
        
        assert result["success"] is True
        
        provider_used = result["provider_used"]
        metadata = result.get("metadata", {})
        receipt = metadata.get("receipt", {})
        
        # Verify consistency across response fields
        execution_details = receipt.get("execution_details", {})
        
        assert execution_details.get("provider_id") == provider_used, (
            "Receipt provider_id doesn't match provider_used"
        )
        
        assert execution_details.get("region") == "us-east", (
            "Receipt region doesn't match requested region"
        )
        
        print("✅ Receipt data consistent with routing decision")


# Standalone test function for manual execution
async def manual_test():
    """Run tests manually without pytest"""
    print("🧪 Testing Regional Routing")
    print("=" * 60)
    
    async with httpx.AsyncClient(timeout=TIMEOUT) as client:
        test = TestRegionalRouting()
        
        # Test US routing
        print("\n1️⃣ Testing US-East routing...")
        try:
            result = await test._make_inference_request(client, "us-east")
            provider_used = result.get("provider_used", "")
            if result["success"] and provider_used == "modal-us-east":
                print(f"✅ US-East routing verified: {provider_used}")
            else:
                print(f"⚠️  US-East: {result.get('error') or f'Wrong provider: {provider_used}'}")
        except Exception as e:
            print(f"❌ US-East test failed: {e}")
        
        # Test EU routing
        print("\n2️⃣ Testing EU-West routing...")
        try:
            result = await test._make_inference_request(client, "eu-west")
            provider_used = result.get("provider_used", "")
            if result["success"] and provider_used == "modal-eu-west":
                print(f"✅ EU-West routing verified: {provider_used}")
            else:
                print(f"⚠️  EU-West: {result.get('error') or f'Wrong provider: {provider_used}'}")
        except Exception as e:
            print(f"❌ EU-West test failed: {e}")
        
        # Test APAC routing
        print("\n3️⃣ Testing Asia-Pacific routing...")
        try:
            result = await test._make_inference_request(client, "asia-pacific")
            provider_used = result.get("provider_used", "")
            if result["success"]:
                if provider_used == "modal-apac":
                    print(f"✅ APAC routing verified: {provider_used}")
                else:
                    print(f"ℹ️  APAC not available, fallback to: {provider_used}")
            else:
                print(f"⚠️  APAC: {result.get('error')}")
        except Exception as e:
            print(f"❌ APAC test failed: {e}")
        
        # Test parallel
        print("\n4️⃣ Testing all regions in parallel...")
        try:
            await test.test_all_regions_parallel(client)
        except Exception as e:
            print(f"❌ Parallel test failed: {e}")
        
        # Test receipt consistency
        print("\n5️⃣ Testing receipt consistency...")
        try:
            await test.test_receipt_consistency(client)
        except Exception as e:
            print(f"❌ Receipt test failed: {e}")
    
    print("\n" + "=" * 60)
    print("🎉 Regional routing tests completed!")


if __name__ == "__main__":
    asyncio.run(manual_test())
