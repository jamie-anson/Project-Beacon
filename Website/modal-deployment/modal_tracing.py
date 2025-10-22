"""
Lightweight tracing for Modal functions
Database-only (no Sentry) to minimize overhead in serverless environment
"""
import os
import asyncio
from typing import Optional, Dict, Any
from uuid import uuid4
from datetime import datetime

try:
    import asyncpg
    HAS_ASYNCPG = True
except ImportError:
    HAS_ASYNCPG = False
    print("⚠️  asyncpg not installed, tracing disabled")


class ModalTracer:
    """Lightweight database tracer for Modal functions"""
    
    def __init__(self):
        self.enabled = os.getenv("ENABLE_DB_TRACING", "false").lower() == "true"
        self.db_url = os.getenv("DATABASE_URL")
        self._pool = None
        
        if self.enabled and not HAS_ASYNCPG:
            print("⚠️  ENABLE_DB_TRACING=true but asyncpg not installed")
            self.enabled = False
        
        if self.enabled and not self.db_url:
            print("⚠️  ENABLE_DB_TRACING=true but DATABASE_URL not set")
            self.enabled = False
        
        if self.enabled:
            print(f"🔍 ModalTracer: ENABLED")
        else:
            print(f"⚠️  ModalTracer: DISABLED")
    
    async def _get_pool(self):
        """Lazy connection pool creation"""
        if not self.enabled or not HAS_ASYNCPG:
            return None
        
        if self._pool is None:
            try:
                self._pool = await asyncpg.create_pool(
                    self.db_url,
                    min_size=1,
                    max_size=3,  # Keep small for serverless
                    command_timeout=5.0
                )
                print("✅ Modal trace pool created")
            except Exception as e:
                print(f"❌ Failed to create trace pool: {e}")
                self.enabled = False
        
        return self._pool
    
    async def start_span(
        self,
        trace_id: str,
        parent_span_id: Optional[str],
        service: str,
        operation: str,
        metadata: Optional[Dict[str, Any]] = None
    ) -> str:
        """Start a new span and return span_id"""
        span_id = str(uuid4())
        
        if not self.enabled:
            return span_id  # Return dummy span_id
        
        try:
            pool = await self._get_pool()
            if not pool:
                return span_id
            
            async with pool.acquire() as conn:
                await conn.execute("""
                    INSERT INTO trace_spans 
                    (trace_id, span_id, parent_span_id, service, operation, 
                     started_at, status, metadata)
                    VALUES ($1, $2, $3, $4, $5, NOW(), 'started', $6)
                """, 
                    trace_id,
                    span_id,
                    parent_span_id,
                    service,
                    operation,
                    metadata or {}
                )
        except Exception as e:
            print(f"❌ Failed to start span: {e}")
        
        return span_id
    
    async def complete_span(
        self,
        span_id: str,
        status: str,
        error_message: Optional[str] = None,
        error_type: Optional[str] = None
    ):
        """Complete a span with final status"""
        if not self.enabled:
            return
        
        try:
            pool = await self._get_pool()
            if not pool:
                return
            
            async with pool.acquire() as conn:
                await conn.execute("""
                    UPDATE trace_spans 
                    SET completed_at = NOW(),
                        duration_ms = EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000,
                        status = $2,
                        error_message = $3,
                        error_type = $4
                    WHERE span_id = $1
                """, span_id, status, error_message, error_type)
        except Exception as e:
            print(f"❌ Failed to complete span: {e}")
    
    async def close(self):
        """Close connection pool"""
        if self._pool:
            await self._pool.close()


# Global tracer instance
_tracer = None

def get_tracer() -> ModalTracer:
    """Get or create global tracer instance"""
    global _tracer
    if _tracer is None:
        _tracer = ModalTracer()
    return _tracer
