"""API endpoints for the hybrid router"""

from .health import router as health_router
from .inference import router as inference_router
from .providers import router as providers_router
from .websocket import router as websocket_router

__all__ = ["health_router", "inference_router", "providers_router", "websocket_router"]
