"""Main FastAPI application"""

import logging
from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from .core import HybridRouter
from .api import health_router, inference_router, providers_router, websocket_router, maps_router
from .config import CORS_ORIGINS, get_port, HOST

logger = logging.getLogger(__name__)

# Global router instance
router_instance = HybridRouter()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Handle application lifespan events"""
    # Startup
    logger.info("Starting Project Beacon Hybrid Router...")
    await router_instance.health_check_providers()
    try:
        configured = [f"{p.name}({p.type.value},{p.region})@{p.endpoint}" for p in router_instance.providers]
        logger.info(f"Initialized with {len(router_instance.providers)} providers: {configured}")
    except Exception as e:
        logger.error(f"Error during startup: {e}")
    
    yield
    
    # Shutdown (if needed)
    logger.info("Shutting down Project Beacon Hybrid Router...")


# FastAPI app
app = FastAPI(
    title="Project Beacon Hybrid Router", 
    version="1.0.0",
    description="Routes inference requests between Golem providers and serverless GPU providers",
    lifespan=lifespan
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=CORS_ORIGINS,
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "DELETE", "OPTIONS"],
    allow_headers=["*"],
)

# Include API routers
app.include_router(health_router)
app.include_router(inference_router)
app.include_router(providers_router)
app.include_router(websocket_router)
app.include_router(maps_router)


if __name__ == "__main__":
    port = get_port()
    uvicorn.run(app, host=HOST, port=port, proxy_headers=True, forwarded_allow_ips="*")
