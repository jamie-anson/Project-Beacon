from fastapi import APIRouter
from .routes import diffs

router = APIRouter()
router.include_router(diffs.router, tags=["diffs"]) 
