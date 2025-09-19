from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from .core.config import settings
from .api.v1 import router as api_router
from .dao.postgres import create_engine_and_session, create_tables

app = FastAPI(title="Project Beacon Backend", version="0.1.0")

# CORS (adjust origins as needed)
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.on_event("startup")
async def _startup_db():
    # If a DATABASE_URL is provided, ensure tables exist
    if settings.database_url:
        engine, _ = create_engine_and_session(settings.database_url)
        await create_tables(engine)


@app.get("/health")
async def health():
    return {"status": "ok", "service": "backend", "version": "0.1.0"}


# Mount API routers
app.include_router(api_router, prefix="/api/v1")


# Uvicorn entrypoint for local run: `python -m backend.app.main`
if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "backend.app.main:app",
        host="0.0.0.0",
        port=settings.port,
        reload=True,
    )
