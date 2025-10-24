"""Distributed tracing for hybrid router"""

import os
import logging
import json
from typing import Optional, Dict, Any
from uuid import UUID, uuid4
from datetime import datetime
import asyncpg

logger = logging.getLogger(__name__)


class DBTracer:
    """Database-backed distributed tracing"""
    
    def __init__(self, db_pool: Optional[asyncpg.Pool] = None):
        self.db_pool = db_pool
        self.enabled = os.getenv("ENABLE_DB_TRACING", "false").lower() == "true"
        
        if self.enabled:
            logger.info("🔍 DBTracer: ENABLED (ENABLE_DB_TRACING=true)")
        else:
            logger.info("⚠️  DBTracer: DISABLED (ENABLE_DB_TRACING=%s)", os.getenv("ENABLE_DB_TRACING", ""))
    
    async def start_span(
        self,
        trace_id: str,
        parent_span_id: Optional[str],
        service: str,
        operation: str,
        metadata: Optional[Dict[str, Any]] = None
    ) -> str:
        """Start a new span and return span_id"""
        if not self.enabled or not self.db_pool:
            return str(uuid4())  # Return dummy span_id
        
        span_id = str(uuid4())
        
        try:
            async with self.db_pool.acquire() as conn:
                # Serialize metadata payload explicitly to JSON and cast to JSONB
                await conn.execute(
                    """
                    INSERT INTO trace_spans 
                    (trace_id, span_id, parent_span_id, service, operation, 
                     started_at, status, metadata)
                    VALUES ($1, $2, $3, $4, $5, NOW(), 'started', $6::jsonb)
                    """,
                    trace_id,
                    span_id,
                    parent_span_id,
                    service,
                    operation,
                    json.dumps(metadata or {})
                )
            logger.debug("✅ Span started: %s/%s", service, operation)
        except Exception as e:
            logger.error("❌ Failed to start span: %s", e)
        
        return span_id
    
    async def complete_span(
        self,
        span_id: str,
        status: str,
        error_message: Optional[str] = None,
        error_type: Optional[str] = None
    ):
        """Complete a span with final status"""
        if not self.enabled or not self.db_pool:
            return
        
        try:
            async with self.db_pool.acquire() as conn:
                await conn.execute("""
                    UPDATE trace_spans 
                    SET completed_at = NOW(),
                        duration_ms = EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000,
                        status = $2,
                        error_message = $3,
                        error_type = $4
                    WHERE span_id = $1
                """, span_id, status, error_message, error_type)
            logger.debug("✅ Span completed: %s (status=%s)", span_id, status)
        except Exception as e:
            logger.error("❌ Failed to complete span: %s", e)


async def create_db_pool() -> Optional[asyncpg.Pool]:
    """Create database connection pool for tracing"""
    database_url = os.getenv("DATABASE_URL")
    if not database_url:
        logger.warning("DATABASE_URL not set, tracing will be disabled")
        return None
    
    try:
        pool = await asyncpg.create_pool(
            database_url,
            min_size=1,
            max_size=5,
            command_timeout=5.0
        )
        logger.info("✅ Database pool created for tracing")
        return pool
    except Exception as e:
        logger.error("❌ Failed to create database pool: %s", e)
        return None
