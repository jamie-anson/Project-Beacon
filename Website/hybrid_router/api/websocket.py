"""WebSocket endpoints"""

import json
import time
import logging
from fastapi import APIRouter, WebSocket, WebSocketDisconnect

from ..models import ConnectionManager

logger = logging.getLogger(__name__)
router = APIRouter()

# WebSocket connection manager
manager = ConnectionManager()


@router.get("/ws")
async def websocket_http_hint():
    """Helpful hint for accidental HTTP GET requests on the WebSocket endpoint"""
    return {
        "status": "ok",
        "message": "This is a WebSocket endpoint. Connect using wss://<host>/ws (HTTP GET will not upgrade)."
    }


@router.websocket("/ws")
async def websocket_endpoint(websocket: WebSocket):
    """WebSocket endpoint for real-time updates"""
    await manager.connect(websocket)
    try:
        # Send initial connection message
        await manager.send_personal_message(json.dumps({
            "type": "connection",
            "status": "connected",
            "timestamp": time.time()
        }), websocket)
        
        # Keep connection alive and handle messages
        while True:
            try:
                # Wait for messages from client (optional)
                data = await websocket.receive_text()
                # Echo back for now (can be extended for specific functionality)
                await manager.send_personal_message(json.dumps({
                    "type": "echo",
                    "data": data,
                    "timestamp": time.time()
                }), websocket)
            except WebSocketDisconnect:
                break
            except Exception as e:
                logger.error(f"WebSocket error: {e}")
                break
    except WebSocketDisconnect:
        pass
    finally:
        manager.disconnect(websocket)
