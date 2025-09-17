"""
WebSocket test for hybrid router
Tests the real-time WebSocket functionality
"""

import asyncio
import json
import time
import websockets
from typing import Dict, Any

class WebSocketTester:
    def __init__(self, base_url: str = "wss://beacon-hybrid-router.fly.dev"):
        self.base_url = base_url
    
    async def test_websocket_connection(self) -> Dict[str, Any]:
        """Test WebSocket connection and basic functionality"""
        try:
            ws_url = f"{self.base_url}/ws"
            print(f"🔌 Connecting to: {ws_url}")
            
            async with websockets.connect(ws_url, timeout=10) as websocket:
                # Wait for initial connection message
                initial_message = await asyncio.wait_for(websocket.recv(), timeout=5)
                initial_data = json.loads(initial_message)
                
                print(f"📨 Received initial message: {initial_data}")
                
                # Send a test message
                test_message = {"test": "hello", "timestamp": time.time()}
                await websocket.send(json.dumps(test_message))
                
                # Wait for echo response
                echo_response = await asyncio.wait_for(websocket.recv(), timeout=5)
                echo_data = json.loads(echo_response)
                
                print(f"📨 Received echo: {echo_data}")
                
                return {
                    "success": True,
                    "initial_message": initial_data,
                    "echo_response": echo_data,
                    "connection_type": initial_data.get("type"),
                    "echo_type": echo_data.get("type")
                }
                
        except asyncio.TimeoutError:
            return {"success": False, "error": "WebSocket connection timeout", "websocket_available": False}
        except websockets.exceptions.ConnectionClosed as e:
            return {"success": False, "error": f"WebSocket connection closed: {e}", "websocket_available": False}
        except websockets.exceptions.InvalidStatusCode as e:
            if e.status_code == 404:
                return {"success": False, "error": "WebSocket endpoint not found (feature disabled)", "websocket_available": False}
            return {"success": False, "error": f"WebSocket invalid status: {e}", "websocket_available": False}
        except Exception as e:
            return {"success": False, "error": str(e), "websocket_available": False}
    
    async def test_websocket_persistence(self) -> Dict[str, Any]:
        """Test WebSocket connection persistence"""
        try:
            ws_url = f"{self.base_url}/ws"
            
            async with websockets.connect(ws_url, timeout=10) as websocket:
                # Send multiple messages over time
                messages_sent = []
                messages_received = []
                
                for i in range(3):
                    if i == 0:
                        # Skip initial connection message
                        await websocket.recv()
                    
                    message = {"test": f"message_{i}", "timestamp": time.time()}
                    await websocket.send(json.dumps(message))
                    messages_sent.append(message)
                    
                    # Wait for response
                    response = await asyncio.wait_for(websocket.recv(), timeout=5)
                    response_data = json.loads(response)
                    messages_received.append(response_data)
                    
                    # Wait between messages
                    await asyncio.sleep(1)
                
                return {
                    "success": True,
                    "messages_sent": len(messages_sent),
                    "messages_received": len(messages_received),
                    "all_echoed": all(
                        sent["test"] in str(received.get("data", ""))
                        for sent, received in zip(messages_sent, messages_received)
                    )
                }
                
        except Exception as e:
            return {"success": False, "error": str(e)}

async def main():
    print("🧪 Testing Project Beacon Hybrid Router WebSocket...")
    print()
    
    # Test with Fly.io deployment
    fly_tester = WebSocketTester("wss://beacon-hybrid-router.fly.dev")
    
    print("1️⃣ Testing WebSocket connection...")
    connection_result = await fly_tester.test_websocket_connection()
    if connection_result["success"]:
        print("✅ WebSocket connection successful")
        print(f"   🔗 Connection type: {connection_result.get('connection_type', 'unknown')}")
        print(f"   📡 Echo type: {connection_result.get('echo_type', 'unknown')}")
        print(f"   ⏰ Initial message: {connection_result.get('initial_message', {}).get('status', 'unknown')}")
    else:
        websocket_available = connection_result.get("websocket_available", True)
        if not websocket_available and "not found" in connection_result.get("error", "").lower():
            print("ℹ️  WebSocket endpoint not available (feature disabled)")
            print("   🔌 This is expected if WebSocket functionality is turned off")
        else:
            print(f"❌ WebSocket connection failed: {connection_result.get('error', 'Unknown error')}")
    print()
    
    if connection_result["success"]:
        print("2️⃣ Testing WebSocket persistence...")
        persistence_result = await fly_tester.test_websocket_persistence()
        if persistence_result["success"]:
            print("✅ WebSocket persistence test passed")
            print(f"   📤 Messages sent: {persistence_result.get('messages_sent', 0)}")
            print(f"   📥 Messages received: {persistence_result.get('messages_received', 0)}")
            print(f"   🔄 All echoed: {persistence_result.get('all_echoed', False)}")
        else:
            print(f"❌ WebSocket persistence failed: {persistence_result.get('error', 'Unknown error')}")
    else:
        print("2️⃣ Skipping persistence test (connection failed)")
    print()
    
    print("🎉 WebSocket testing completed!")

if __name__ == "__main__":
    asyncio.run(main())
