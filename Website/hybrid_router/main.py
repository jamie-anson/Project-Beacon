"""Main FastAPI application"""

import logging
import os
from contextlib import asynccontextmanager

import asyncio
import uvicorn
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

# Sentry for error tracking
import sentry_sdk
from sentry_sdk.integrations.fastapi import FastApiIntegration
from sentry_sdk.integrations.logging import LoggingIntegration

from .core import HybridRouter
from .core.region_queue import queue_manager
from .api import health_router, inference_router, providers_router, websocket_router, maps_router, queue_router, debug_router
from .config import CORS_ORIGINS, get_port, HOST
from .tracing import DBTracer, create_db_pool

logger = logging.getLogger(__name__)

# Initialize Sentry
sentry_dsn = os.getenv("SENTRY_DSN")
if sentry_dsn:
    sentry_sdk.init(
        dsn=sentry_dsn,
        environment=os.getenv("RAILWAY_ENVIRONMENT", "development"),
        release=f"router@{os.getenv('RAILWAY_GIT_COMMIT_SHA', 'dev')[:7]}",
        traces_sample_rate=0.2,  # 20% of transactions
        integrations=[
            FastApiIntegration(),
            LoggingIntegration(
                level=logging.INFO,
                event_level=logging.ERROR
            ),
        ],
        before_send=lambda event, hint: {
            **event,
            "tags": {**event.get("tags", {}), "service": "router"}
        }
    )
    logger.info("✅ Sentry initialized for router")
else:
    logger.info("⚠️  Sentry disabled (no SENTRY_DSN)")

# Global router instance
router_instance = HybridRouter()


@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("Starting Project Beacon Hybrid Router...")
    port = get_port()
    logger.info("Binding to %s:%s", HOST, port)

    # Initialize database tracing
    db_pool = await create_db_pool()
    db_tracer = DBTracer(db_pool)
    app.state.db_tracer = db_tracer

    try:
        # Run provider checks in the background so startup does NOT block
        asyncio.create_task(router_instance.health_check_providers())
        
        # DISABLED: Region queue workers causing asyncio event loop conflicts
        # The main /inference endpoint doesn't use queues anyway
        # Only /inference/queued uses queues, which is not used by runner
        # TODO: Fix asyncio event loop issue before re-enabling
        # logger.info("Starting region queue workers...")
        # asyncio.create_task(queue_manager.start_workers(router_instance.run_inference))
        # logger.info("Region queue workers started for US, EU, ASIA")
        logger.info("Region queue workers DISABLED (asyncio event loop fix pending)")
    except Exception as e:
        logger.exception("Initialization during startup failed: %s", e)

    yield

    logger.info("Shutting down Project Beacon Hybrid Router...")
    if db_pool:
        await db_pool.close()


# FastAPI app
app = FastAPI(
    title="Project Beacon Hybrid Router", 
    version="1.0.0",
    description="Routes inference requests between Golem providers and serverless GPU providers",
    lifespan=lifespan
)

# Store router_instance in app state for dependency injection
app.state.router_instance = router_instance

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
app.include_router(queue_router)
app.include_router(debug_router)
if maps_router is not None:
    app.include_router(maps_router)
else:
    logger.warning("Maps router not included because it failed to import.")


if __name__ == "__main__":
    port = get_port()
    uvicorn.run(app, host=HOST, port=port, proxy_headers=True, forwarded_allow_ips="*")
