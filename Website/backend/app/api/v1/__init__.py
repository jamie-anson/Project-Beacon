from fastapi import APIRouter
from .routes import diffs, executions

router = APIRouter()
router.include_router(diffs.router, tags=["diffs"])
router.include_router(executions.router, tags=["executions"]) 
