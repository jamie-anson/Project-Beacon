"""API endpoints for the hybrid router"""

from .health import router as health_router
from .inference import router as inference_router
from .providers import router as providers_router
from .websocket import router as websocket_router
import logging

try:
    from .maps import router as maps_router
except Exception as e:
    logging.getLogger(__name__).warning("Maps router unavailable: %s", e)
    maps_router = None

__all__ = ["health_router", "inference_router", "providers_router", "websocket_router", "maps_router"]
