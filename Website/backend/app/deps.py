from __future__ import annotations
from functools import lru_cache
from .dao.base import DiffDAO, InMemoryDiffDAO
from .services.compute import ComputeService
from .core.config import settings
from .dao.postgres import PostgresDiffDAO


@lru_cache(maxsize=1)
def get_dao() -> DiffDAO:
    if settings.database_url:
        return PostgresDiffDAO(settings.database_url)
    return InMemoryDiffDAO()


def get_compute_service() -> ComputeService:
    return ComputeService(get_dao())
