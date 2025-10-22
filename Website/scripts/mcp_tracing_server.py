#!/usr/bin/env python3
"""
MCP Server for Distributed Tracing Queries
Enables LLMs to autonomously query and analyze execution traces
"""
import os
import json
import asyncio
from typing import Any, Dict, List
import asyncpg

# MCP Server imports
try:
    from mcp.server import Server
    from mcp.server.stdio import stdio_server
    from mcp.types import Tool, TextContent
    HAS_MCP = True
except ImportError:
    print("⚠️  MCP SDK not installed: pip install mcp")
    HAS_MCP = False
    exit(1)

# Database connection
DATABASE_URL = os.getenv("DATABASE_URL")
if not DATABASE_URL:
    print("❌ DATABASE_URL environment variable required")
    exit(1)

# Create MCP server
server = Server("project-beacon-tracing")

# Database pool (created on startup)
db_pool = None


async def init_db_pool():
    """Initialize database connection pool"""
    global db_pool
    try:
        db_pool = await asyncpg.create_pool(
            DATABASE_URL,
            min_size=1,
            max_size=5,
            command_timeout=30.0
        )
        print("✅ Database pool initialized")
    except Exception as e:
        print(f"❌ Failed to initialize database pool: {e}")
        raise


@server.list_tools()
async def list_tools() -> List[Tool]:
    """List available tracing tools"""
    return [
        Tool(
            name="diagnose_execution",
            description="Diagnose a failed execution by analyzing its trace spans. Returns root cause analysis, timing breakdown, and recommendations.",
            inputSchema={
                "type": "object",
                "properties": {
                    "execution_id": {
                        "type": "integer",
                        "description": "The execution ID to diagnose"
                    }
                },
                "required": ["execution_id"]
            }
        ),
        Tool(
            name="find_timeout_root_cause",
            description="Find the root cause of timeout failures by analyzing trace gaps and slow spans.",
            inputSchema={
                "type": "object",
                "properties": {
                    "job_id": {
                        "type": "string",
                        "description": "The job ID to analyze for timeouts"
                    }
                },
                "required": ["job_id"]
            }
        ),
        Tool(
            name="compare_with_successful",
            description="Compare a failed execution with similar successful executions to identify differences.",
            inputSchema={
                "type": "object",
                "properties": {
                    "execution_id": {
                        "type": "integer",
                        "description": "The failed execution ID to compare"
                    }
                },
                "required": ["execution_id"]
            }
        ),
        Tool(
            name="get_recent_failures",
            description="Get recent failed executions with their error patterns.",
            inputSchema={
                "type": "object",
                "properties": {
                    "limit": {
                        "type": "integer",
                        "description": "Number of failures to return (default: 10)",
                        "default": 10
                    }
                }
            }
        ),
        Tool(
            name="analyze_performance",
            description="Analyze performance metrics for a specific model or region.",
            inputSchema={
                "type": "object",
                "properties": {
                    "model_id": {
                        "type": "string",
                        "description": "Model ID to analyze (optional)"
                    },
                    "region": {
                        "type": "string",
                        "description": "Region to analyze (optional)"
                    }
                }
            }
        ),
        Tool(
            name="find_trace_gaps",
            description="Find gaps in distributed traces where spans are missing or delayed.",
            inputSchema={
                "type": "object",
                "properties": {
                    "trace_id": {
                        "type": "string",
                        "description": "Trace ID to analyze for gaps"
                    }
                },
                "required": ["trace_id"]
            }
        )
    ]


@server.call_tool()
async def call_tool(name: str, arguments: Dict[str, Any]) -> List[TextContent]:
    """Handle tool calls"""
    
    if not db_pool:
        return [TextContent(
            type="text",
            text="Error: Database pool not initialized"
        )]
    
    try:
        if name == "diagnose_execution":
            result = await diagnose_execution(arguments["execution_id"])
        elif name == "find_timeout_root_cause":
            result = await find_timeout_root_cause(arguments["job_id"])
        elif name == "compare_with_successful":
            result = await compare_with_successful(arguments["execution_id"])
        elif name == "get_recent_failures":
            limit = arguments.get("limit", 10)
            result = await get_recent_failures(limit)
        elif name == "analyze_performance":
            result = await analyze_performance(
                arguments.get("model_id"),
                arguments.get("region")
            )
        elif name == "find_trace_gaps":
            result = await find_trace_gaps(arguments["trace_id"])
        else:
            result = {"error": f"Unknown tool: {name}"}
        
        return [TextContent(
            type="text",
            text=json.dumps(result, indent=2)
        )]
    
    except Exception as e:
        return [TextContent(
            type="text",
            text=f"Error: {str(e)}"
        )]


async def diagnose_execution(execution_id: int) -> Dict[str, Any]:
    """Diagnose a failed execution using SQL diagnostic function"""
    async with db_pool.acquire() as conn:
        result = await conn.fetchrow(
            "SELECT * FROM diagnose_execution_trace($1)",
            execution_id
        )
        
        if not result:
            return {"error": f"No trace data found for execution {execution_id}"}
        
        return dict(result)


async def find_timeout_root_cause(job_id: str) -> Dict[str, Any]:
    """Find root cause of timeout failures"""
    async with db_pool.acquire() as conn:
        # Get all executions for this job
        executions = await conn.fetch("""
            SELECT id, region, model_id, status, 
                   EXTRACT(EPOCH FROM (completed_at - created_at)) * 1000 as duration_ms
            FROM executions
            WHERE job_id = $1
            ORDER BY created_at
        """, job_id)
        
        if not executions:
            return {"error": f"No executions found for job {job_id}"}
        
        # Analyze each execution's traces
        analysis = []
        for exec in executions:
            spans = await conn.fetch("""
                SELECT service, operation, duration_ms, status, error_message
                FROM trace_spans
                WHERE execution_id = $1
                ORDER BY started_at
            """, exec['id'])
            
            analysis.append({
                "execution_id": exec['id'],
                "region": exec['region'],
                "model": exec['model_id'],
                "status": exec['status'],
                "duration_ms": float(exec['duration_ms']) if exec['duration_ms'] else None,
                "span_count": len(spans),
                "spans": [dict(s) for s in spans]
            })
        
        return {
            "job_id": job_id,
            "total_executions": len(executions),
            "analysis": analysis
        }


async def compare_with_successful(execution_id: int) -> Dict[str, Any]:
    """Compare failed execution with similar successful ones"""
    async with db_pool.acquire() as conn:
        result = await conn.fetchrow(
            "SELECT * FROM find_similar_traces($1)",
            execution_id
        )
        
        if not result:
            return {"error": f"No similar traces found for execution {execution_id}"}
        
        return dict(result)


async def get_recent_failures(limit: int = 10) -> Dict[str, Any]:
    """Get recent failed executions"""
    async with db_pool.acquire() as conn:
        failures = await conn.fetch("""
            SELECT 
                e.id,
                e.job_id,
                e.region,
                e.model_id,
                e.status,
                e.created_at,
                COALESCE(
                    (SELECT error_message FROM trace_spans 
                     WHERE execution_id = e.id AND error_message IS NOT NULL 
                     LIMIT 1),
                    'No error message'
                ) as error_message
            FROM executions e
            WHERE e.status = 'failed'
            ORDER BY e.created_at DESC
            LIMIT $1
        """, limit)
        
        return {
            "total_failures": len(failures),
            "failures": [dict(f) for f in failures]
        }


async def analyze_performance(model_id: str = None, region: str = None) -> Dict[str, Any]:
    """Analyze performance metrics"""
    async with db_pool.acquire() as conn:
        where_clauses = []
        params = []
        param_idx = 1
        
        if model_id:
            where_clauses.append(f"model_id = ${param_idx}")
            params.append(model_id)
            param_idx += 1
        
        if region:
            where_clauses.append(f"region = ${param_idx}")
            params.append(region)
            param_idx += 1
        
        where_sql = f"WHERE {' AND '.join(where_clauses)}" if where_clauses else ""
        
        stats = await conn.fetchrow(f"""
            SELECT 
                COUNT(*) as total_executions,
                COUNT(*) FILTER (WHERE status = 'completed') as successful,
                COUNT(*) FILTER (WHERE status = 'failed') as failed,
                AVG(EXTRACT(EPOCH FROM (completed_at - created_at))) as avg_duration_sec,
                PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (completed_at - created_at))) as p95_duration_sec
            FROM executions
            {where_sql}
        """, *params)
        
        return dict(stats)


async def find_trace_gaps(trace_id: str) -> Dict[str, Any]:
    """Find gaps in distributed trace"""
    async with db_pool.acquire() as conn:
        spans = await conn.fetch("""
            SELECT 
                span_id,
                parent_span_id,
                service,
                operation,
                started_at,
                completed_at,
                duration_ms,
                status
            FROM trace_spans
            WHERE trace_id = $1
            ORDER BY started_at
        """, trace_id)
        
        if not spans:
            return {"error": f"No spans found for trace {trace_id}"}
        
        # Analyze gaps between spans
        gaps = []
        for i in range(len(spans) - 1):
            current = spans[i]
            next_span = spans[i + 1]
            
            if current['completed_at'] and next_span['started_at']:
                gap_ms = (next_span['started_at'] - current['completed_at']).total_seconds() * 1000
                if gap_ms > 100:  # Gap > 100ms
                    gaps.append({
                        "after_span": current['span_id'],
                        "before_span": next_span['span_id'],
                        "gap_ms": gap_ms
                    })
        
        return {
            "trace_id": trace_id,
            "total_spans": len(spans),
            "spans": [dict(s) for s in spans],
            "gaps": gaps
        }


async def main():
    """Run MCP server"""
    print("🚀 Starting Project Beacon Tracing MCP Server...")
    
    # Initialize database
    await init_db_pool()
    
    # Run server
    async with stdio_server() as (read_stream, write_stream):
        await server.run(
            read_stream,
            write_stream,
            server.create_initialization_options()
        )


if __name__ == "__main__":
    asyncio.run(main())
