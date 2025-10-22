# LLM-Queryable Tracing System

**Date:** 2025-10-22  
**Purpose:** Enable LLMs to autonomously debug distributed system issues  
**Status:** 🎯 DESIGN READY

---

## Problem Statement

When debugging distributed system issues, we need:
1. **LLM to query traces** without human intervention
2. **Natural language → SQL** translation
3. **Structured diagnostic output** LLMs can interpret
4. **Root cause analysis** automation

**Example Scenario:**
```
User: "Why did execution 2193 timeout?"
LLM: *queries tracing system*
LLM: "Router completed successfully in 82s, but runner didn't receive 
      response for 2m36s. Root cause: Network timeout between router 
      and runner. Check Railway → Fly.io connectivity."
```

---

## Architecture Overview

```
┌─────────────┐
│ LLM Agent   │
│ (Windsurf)  │
└──────┬──────┘
       │
       │ Natural Language Query
       ▼
┌─────────────────────┐
│ Tracing Query API   │
│ (REST + MCP Server) │
└──────┬──────────────┘
       │
       │ Structured Query
       ▼
┌─────────────────────┐
│ Diagnostic Engine   │
│ (SQL + Analysis)    │
└──────┬──────────────┘
       │
       │ Results + Analysis
       ▼
┌─────────────────────┐
│ Postgres Traces DB  │
└─────────────────────┘
```

---

## Component 1: Diagnostic SQL Functions

### 1.1 Execution Trace with Gap Analysis

```sql
-- Enhanced trace query with automatic gap detection
CREATE FUNCTION diagnose_execution_trace(p_execution_id BIGINT)
RETURNS TABLE (
    service VARCHAR,
    operation VARCHAR,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,
    gap_after_ms INTEGER,
    status VARCHAR,
    error_message TEXT,
    is_anomaly BOOLEAN,
    anomaly_reason TEXT
) AS $$
WITH trace_data AS (
    SELECT 
        ts.service,
        ts.operation,
        ts.started_at,
        ts.completed_at,
        ts.duration_ms,
        ts.status,
        ts.error_message,
        LEAD(ts.started_at) OVER (ORDER BY ts.started_at) as next_started_at,
        -- Calculate average duration for this operation type
        AVG(ts.duration_ms) OVER (PARTITION BY ts.service, ts.operation) as avg_duration
    FROM trace_spans ts
    WHERE ts.execution_id = p_execution_id
)
SELECT 
    service,
    operation,
    started_at,
    completed_at,
    duration_ms,
    EXTRACT(EPOCH FROM (next_started_at - completed_at)) * 1000 as gap_after_ms,
    status,
    error_message,
    -- Flag anomalies
    CASE 
        WHEN duration_ms > avg_duration * 3 THEN TRUE
        WHEN EXTRACT(EPOCH FROM (next_started_at - completed_at)) * 1000 > 5000 THEN TRUE
        WHEN status IN ('failed', 'timeout') THEN TRUE
        ELSE FALSE
    END as is_anomaly,
    -- Explain anomaly
    CASE 
        WHEN duration_ms > avg_duration * 3 THEN 
            'Operation took ' || duration_ms || 'ms (3x longer than average ' || avg_duration || 'ms)'
        WHEN EXTRACT(EPOCH FROM (next_started_at - completed_at)) * 1000 > 5000 THEN
            'Gap of ' || EXTRACT(EPOCH FROM (next_started_at - completed_at)) * 1000 || 'ms before next operation'
        WHEN status IN ('failed', 'timeout') THEN
            'Operation failed: ' || COALESCE(error_message, 'Unknown error')
        ELSE NULL
    END as anomaly_reason
FROM trace_data
ORDER BY started_at;
$$ LANGUAGE sql;
```

### 1.2 Root Cause Identifier

```sql
-- Automatically identify root cause of failures
CREATE FUNCTION identify_root_cause(p_execution_id BIGINT)
RETURNS TABLE (
    root_cause_type VARCHAR,
    affected_service VARCHAR,
    affected_operation VARCHAR,
    evidence TEXT,
    recommendation TEXT
) AS $$
WITH trace_analysis AS (
    SELECT 
        ts.service,
        ts.operation,
        ts.status,
        ts.duration_ms,
        ts.error_message,
        ts.started_at,
        ts.completed_at,
        LEAD(ts.started_at) OVER (ORDER BY ts.started_at) as next_started_at
    FROM trace_spans ts
    WHERE ts.execution_id = p_execution_id
)
SELECT 
    CASE 
        -- Network timeout detection
        WHEN EXTRACT(EPOCH FROM (next_started_at - completed_at)) > 60 THEN 'NETWORK_TIMEOUT'
        -- Service failure detection
        WHEN status = 'failed' AND error_message LIKE '%connection%' THEN 'CONNECTION_FAILURE'
        WHEN status = 'failed' AND error_message LIKE '%timeout%' THEN 'SERVICE_TIMEOUT'
        -- Performance degradation
        WHEN duration_ms > 120000 THEN 'PERFORMANCE_DEGRADATION'
        -- Modal specific issues
        WHEN service = 'modal' AND status = 'failed' THEN 'MODAL_EXECUTION_FAILURE'
        ELSE 'UNKNOWN'
    END as root_cause_type,
    service as affected_service,
    operation as affected_operation,
    CASE 
        WHEN EXTRACT(EPOCH FROM (next_started_at - completed_at)) > 60 THEN
            'Service ' || service || ' completed at ' || completed_at || 
            ' but next service started at ' || next_started_at || 
            ' (gap: ' || EXTRACT(EPOCH FROM (next_started_at - completed_at)) || 's)'
        WHEN status = 'failed' THEN
            'Service ' || service || ' failed with: ' || COALESCE(error_message, 'No error message')
        WHEN duration_ms > 120000 THEN
            'Operation took ' || duration_ms || 'ms (>2 minutes)'
        ELSE 'See trace_spans for details'
    END as evidence,
    CASE 
        WHEN EXTRACT(EPOCH FROM (next_started_at - completed_at)) > 60 THEN
            'Check network connectivity between ' || service || ' and next service. Review firewall rules and timeout configurations.'
        WHEN status = 'failed' AND error_message LIKE '%connection%' THEN
            'Verify service health and network connectivity. Check if service is running and accessible.'
        WHEN service = 'modal' AND status = 'failed' THEN
            'Check Modal dashboard for function logs. Verify GPU availability and model loading.'
        WHEN duration_ms > 120000 THEN
            'Investigate performance bottleneck in ' || service || '. Check resource utilization and scaling.'
        ELSE 'Review detailed trace spans for more information'
    END as recommendation
FROM trace_analysis
WHERE status = 'failed' 
   OR EXTRACT(EPOCH FROM (next_started_at - completed_at)) > 60
   OR duration_ms > 120000
LIMIT 1;
$$ LANGUAGE sql;
```

### 1.3 Pattern Matcher (Find Similar Issues)

```sql
-- Find similar execution patterns for comparison
CREATE FUNCTION find_similar_traces(
    p_execution_id BIGINT,
    p_limit INTEGER DEFAULT 5
)
RETURNS TABLE (
    similar_execution_id BIGINT,
    similarity_score FLOAT,
    status VARCHAR,
    total_duration_ms INTEGER,
    failure_point VARCHAR
) AS $$
WITH target_trace AS (
    SELECT 
        job_id,
        model_id,
        region,
        array_agg(service || '.' || operation ORDER BY started_at) as operation_sequence,
        SUM(duration_ms) as total_duration
    FROM trace_spans
    WHERE execution_id = p_execution_id
    GROUP BY job_id, model_id, region
)
SELECT 
    ts.execution_id as similar_execution_id,
    -- Calculate similarity based on operation sequence
    1.0 - (
        levenshtein(
            array_to_string(array_agg(ts.service || '.' || ts.operation ORDER BY ts.started_at), ','),
            array_to_string(tt.operation_sequence, ',')
        )::FLOAT / 
        GREATEST(
            array_length(array_agg(ts.service || '.' || ts.operation ORDER BY ts.started_at), 1),
            array_length(tt.operation_sequence, 1)
        )
    ) as similarity_score,
    MAX(CASE WHEN ts.status = 'failed' THEN 'failed' ELSE 'completed' END) as status,
    SUM(ts.duration_ms) as total_duration_ms,
    MAX(CASE WHEN ts.status = 'failed' THEN ts.service || '.' || ts.operation ELSE NULL END) as failure_point
FROM trace_spans ts
CROSS JOIN target_trace tt
WHERE ts.execution_id != p_execution_id
  AND ts.model_id = tt.model_id
  AND ts.region = tt.region
GROUP BY ts.execution_id, tt.operation_sequence
HAVING 1.0 - (
    levenshtein(
        array_to_string(array_agg(ts.service || '.' || ts.operation ORDER BY ts.started_at), ','),
        array_to_string(tt.operation_sequence, ',')
    )::FLOAT / 
    GREATEST(
        array_length(array_agg(ts.service || '.' || ts.operation ORDER BY ts.started_at), 1),
        array_length(tt.operation_sequence, 1)
    )
) > 0.7
ORDER BY similarity_score DESC
LIMIT p_limit;
$$ LANGUAGE sql;
```

---

## Component 2: REST API for LLM Queries

### 2.1 API Endpoints

```python
# tracing_api.py - FastAPI service for LLM queries
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import asyncpg
from typing import List, Optional

app = FastAPI(title="Project Beacon Tracing API")

class TraceQuery(BaseModel):
    execution_id: int
    analysis_type: str  # "full", "root_cause", "similar", "gaps"

class DiagnosticResult(BaseModel):
    execution_id: int
    analysis_type: str
    findings: List[dict]
    root_cause: Optional[dict]
    recommendations: List[str]
    similar_cases: Optional[List[dict]]

@app.post("/api/v1/trace/diagnose")
async def diagnose_execution(query: TraceQuery) -> DiagnosticResult:
    """
    LLM-friendly endpoint for comprehensive trace analysis.
    
    Returns structured diagnostic data that LLMs can interpret.
    """
    async with db_pool.acquire() as conn:
        # Get full trace with anomaly detection
        trace_data = await conn.fetch(
            "SELECT * FROM diagnose_execution_trace($1)",
            query.execution_id
        )
        
        # Get root cause analysis
        root_cause = await conn.fetchrow(
            "SELECT * FROM identify_root_cause($1)",
            query.execution_id
        )
        
        # Find similar traces
        similar = await conn.fetch(
            "SELECT * FROM find_similar_traces($1, 5)",
            query.execution_id
        )
        
        # Build structured response
        findings = []
        recommendations = []
        
        for row in trace_data:
            if row['is_anomaly']:
                findings.append({
                    "service": row['service'],
                    "operation": row['operation'],
                    "anomaly": row['anomaly_reason'],
                    "duration_ms": row['duration_ms'],
                    "gap_after_ms": row['gap_after_ms']
                })
        
        if root_cause:
            recommendations.append(root_cause['recommendation'])
        
        return DiagnosticResult(
            execution_id=query.execution_id,
            analysis_type=query.analysis_type,
            findings=findings,
            root_cause=dict(root_cause) if root_cause else None,
            recommendations=recommendations,
            similar_cases=[dict(s) for s in similar]
        )

@app.get("/api/v1/trace/timeline/{execution_id}")
async def get_timeline(execution_id: int):
    """
    Get visual timeline data for execution.
    Returns data suitable for waterfall visualization.
    """
    async with db_pool.acquire() as conn:
        timeline = await conn.fetch("""
            SELECT 
                service,
                operation,
                EXTRACT(EPOCH FROM started_at) * 1000 as start_ms,
                EXTRACT(EPOCH FROM completed_at) * 1000 as end_ms,
                duration_ms,
                status,
                error_message
            FROM trace_spans
            WHERE execution_id = $1
            ORDER BY started_at
        """, execution_id)
        
        return {
            "execution_id": execution_id,
            "timeline": [dict(t) for t in timeline]
        }

@app.get("/api/v1/trace/summary/{execution_id}")
async def get_summary(execution_id: int):
    """
    Get high-level summary for quick LLM understanding.
    """
    async with db_pool.acquire() as conn:
        summary = await conn.fetchrow("""
            SELECT 
                COUNT(*) as total_spans,
                COUNT(*) FILTER (WHERE status = 'failed') as failed_spans,
                COUNT(*) FILTER (WHERE status = 'timeout') as timeout_spans,
                SUM(duration_ms) as total_duration_ms,
                MAX(duration_ms) as max_span_duration_ms,
                array_agg(DISTINCT service) as services_involved,
                MAX(CASE WHEN status = 'failed' THEN error_message END) as first_error
            FROM trace_spans
            WHERE execution_id = $1
        """, execution_id)
        
        return dict(summary)
```

### 2.2 Natural Language Query Endpoint

```python
@app.post("/api/v1/trace/query")
async def natural_language_query(query: str, execution_id: Optional[int] = None):
    """
    Accept natural language queries and translate to SQL.
    
    Examples:
    - "Why did execution 2193 timeout?"
    - "Show me all failed Modal calls in the last hour"
    - "What's the average duration for hybrid_router_call?"
    """
    
    # Simple pattern matching (can be enhanced with LLM)
    if "timeout" in query.lower():
        if execution_id:
            result = await conn.fetch(
                "SELECT * FROM identify_root_cause($1) WHERE root_cause_type LIKE '%TIMEOUT%'",
                execution_id
            )
        else:
            result = await conn.fetch("""
                SELECT execution_id, service, operation, error_message
                FROM trace_spans
                WHERE status = 'timeout'
                  AND started_at > NOW() - INTERVAL '1 hour'
                ORDER BY started_at DESC
                LIMIT 10
            """)
    
    elif "failed" in query.lower():
        result = await conn.fetch("""
            SELECT 
                execution_id,
                service,
                operation,
                error_message,
                started_at
            FROM trace_spans
            WHERE status = 'failed'
              AND started_at > NOW() - INTERVAL '1 hour'
            ORDER BY started_at DESC
            LIMIT 20
        """)
    
    elif "average" in query.lower() or "avg" in query.lower():
        # Extract operation name from query
        operation = extract_operation_name(query)
        result = await conn.fetch("""
            SELECT 
                service,
                operation,
                AVG(duration_ms) as avg_duration_ms,
                PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_ms) as p50_ms,
                PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_ms,
                COUNT(*) as sample_count
            FROM trace_spans
            WHERE operation LIKE $1
              AND started_at > NOW() - INTERVAL '24 hours'
            GROUP BY service, operation
        """, f"%{operation}%")
    
    return {"query": query, "results": [dict(r) for r in result]}
```

---

## Component 3: MCP Server for Windsurf Integration

### 3.1 MCP Server Implementation

```python
# mcp_tracing_server.py - Model Context Protocol server
from mcp.server import Server
from mcp.types import Tool, TextContent

server = Server("project-beacon-tracing")

@server.list_tools()
async def list_tools():
    return [
        Tool(
            name="diagnose_execution",
            description="Diagnose why an execution failed or had performance issues",
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
            description="Identify root cause of timeout in distributed system",
            inputSchema={
                "type": "object",
                "properties": {
                    "execution_id": {"type": "integer"}
                }
            }
        ),
        Tool(
            name="compare_with_successful",
            description="Compare failed execution with similar successful ones",
            inputSchema={
                "type": "object",
                "properties": {
                    "execution_id": {"type": "integer"}
                }
            }
        ),
        Tool(
            name="query_traces",
            description="Query trace database with natural language",
            inputSchema={
                "type": "object",
                "properties": {
                    "query": {
                        "type": "string",
                        "description": "Natural language query about traces"
                    },
                    "execution_id": {
                        "type": "integer",
                        "description": "Optional execution ID for context"
                    }
                }
            }
        )
    ]

@server.call_tool()
async def call_tool(name: str, arguments: dict):
    if name == "diagnose_execution":
        execution_id = arguments["execution_id"]
        
        # Query diagnostic functions
        async with db_pool.acquire() as conn:
            trace = await conn.fetch(
                "SELECT * FROM diagnose_execution_trace($1)",
                execution_id
            )
            root_cause = await conn.fetchrow(
                "SELECT * FROM identify_root_cause($1)",
                execution_id
            )
        
        # Format for LLM consumption
        output = f"## Execution {execution_id} Diagnostic Report\n\n"
        
        if root_cause:
            output += f"### Root Cause: {root_cause['root_cause_type']}\n"
            output += f"**Affected Service:** {root_cause['affected_service']}\n"
            output += f"**Evidence:** {root_cause['evidence']}\n\n"
            output += f"**Recommendation:** {root_cause['recommendation']}\n\n"
        
        output += "### Trace Timeline:\n\n"
        for span in trace:
            status_emoji = "✅" if span['status'] == 'completed' else "❌"
            output += f"{status_emoji} **{span['service']}.{span['operation']}** - {span['duration_ms']}ms"
            
            if span['is_anomaly']:
                output += f" ⚠️ ANOMALY: {span['anomaly_reason']}"
            
            if span['gap_after_ms'] and span['gap_after_ms'] > 1000:
                output += f"\n   └─ Gap: {span['gap_after_ms']}ms before next operation"
            
            output += "\n"
        
        return [TextContent(type="text", text=output)]
    
    elif name == "find_timeout_root_cause":
        execution_id = arguments["execution_id"]
        
        async with db_pool.acquire() as conn:
            root_cause = await conn.fetchrow(
                "SELECT * FROM identify_root_cause($1) WHERE root_cause_type LIKE '%TIMEOUT%'",
                execution_id
            )
        
        if not root_cause:
            return [TextContent(
                type="text",
                text=f"No timeout detected for execution {execution_id}"
            )]
        
        output = f"## Timeout Root Cause Analysis\n\n"
        output += f"**Type:** {root_cause['root_cause_type']}\n"
        output += f"**Location:** {root_cause['affected_service']}.{root_cause['affected_operation']}\n\n"
        output += f"**Evidence:**\n{root_cause['evidence']}\n\n"
        output += f"**Recommended Action:**\n{root_cause['recommendation']}\n"
        
        return [TextContent(type="text", text=output)]
    
    elif name == "compare_with_successful":
        execution_id = arguments["execution_id"]
        
        async with db_pool.acquire() as conn:
            similar = await conn.fetch(
                "SELECT * FROM find_similar_traces($1, 5)",
                execution_id
            )
        
        output = f"## Similar Executions Comparison\n\n"
        
        for sim in similar:
            status_emoji = "✅" if sim['status'] == 'completed' else "❌"
            output += f"{status_emoji} Execution {sim['similar_execution_id']} "
            output += f"(similarity: {sim['similarity_score']:.1%})\n"
            output += f"   Duration: {sim['total_duration_ms']}ms\n"
            
            if sim['failure_point']:
                output += f"   Failed at: {sim['failure_point']}\n"
            
            output += "\n"
        
        return [TextContent(type="text", text=output)]
    
    elif name == "query_traces":
        query = arguments["query"]
        execution_id = arguments.get("execution_id")
        
        # Call natural language query endpoint
        result = await natural_language_query(query, execution_id)
        
        output = f"## Query Results: {query}\n\n"
        
        for row in result["results"]:
            output += "- " + ", ".join(f"{k}: {v}" for k, v in row.items()) + "\n"
        
        return [TextContent(type="text", text=output)]
```

---

## Component 4: LLM Usage Examples

### Example 1: Debugging Timeout

```python
# In Windsurf chat
User: "Execution 2193 timed out, what happened?"

# LLM calls MCP tool
diagnose_execution(execution_id=2193)

# Returns:
"""
## Execution 2193 Diagnostic Report

### Root Cause: NETWORK_TIMEOUT
**Affected Service:** router
**Evidence:** Service router completed at 2025-10-22 16:22:34.891 
but next service started at 2025-10-22 16:25:11.008 (gap: 156s)

**Recommendation:** Check network connectivity between router and 
runner. Review firewall rules and timeout configurations.

### Trace Timeline:

✅ **runner.execute_question** - 238901ms
✅ **runner.hybrid_router_call** - 238893ms ⚠️ ANOMALY: Gap of 156117ms before next operation
✅ **router.inference_request** - 82549ms
✅ **router.modal_call** - 82334ms
✅ **modal.inference** - 77333ms
"""

# LLM interprets and responds:
"The timeout occurred between the router and runner services. The router 
successfully completed the Modal inference in 82 seconds, but the runner 
didn't receive the response for an additional 2 minutes and 36 seconds. 

This suggests a network timeout issue between Railway (router) and Fly.io 
(runner). I recommend:

1. Check Railway → Fly.io network connectivity
2. Review timeout settings in runner's HTTP client
3. Check if Fly.io had any network incidents around 16:22-16:25 UTC"
```

### Example 2: Performance Investigation

```python
User: "Why are Modal calls taking so long today?"

# LLM calls MCP tool
query_traces(query="average duration for modal.inference in last 24 hours")

# Returns performance metrics
# LLM analyzes and responds with insights
```

### Example 3: Pattern Detection

```python
User: "Are there other executions failing the same way as 2193?"

# LLM calls MCP tool
compare_with_successful(execution_id=2193)

# Returns similar traces
# LLM identifies pattern and suggests fix
```

---

## Implementation Plan

### Week 1: SQL Functions
- [ ] Create diagnostic SQL functions
- [ ] Add indexes for performance
- [ ] Test with existing trace data

### Week 2: REST API
- [ ] Build FastAPI tracing service
- [ ] Add natural language query endpoint
- [ ] Deploy alongside runner app

### Week 3: MCP Server
- [ ] Implement MCP server
- [ ] Register with Windsurf
- [ ] Test LLM integration

### Week 4: Documentation & Training
- [ ] Document query patterns
- [ ] Create example queries
- [ ] Train LLM with common scenarios

---

## Benefits

1. **Autonomous Debugging** - LLM can diagnose issues without human intervention
2. **Faster Resolution** - Root cause identified in seconds, not hours
3. **Pattern Recognition** - Automatically find similar failures
4. **Natural Language** - No need to write complex SQL queries
5. **Context Aware** - LLM has full trace context for better analysis

---

## Success Criteria

- [ ] LLM can diagnose timeout root cause in <30 seconds
- [ ] Natural language queries return accurate results
- [ ] MCP integration works seamlessly in Windsurf
- [ ] Root cause identification accuracy >90%
- [ ] Query response time <2 seconds

---

**Status:** 🎯 Ready for implementation  
**Estimated Effort:** 3-4 weeks  
**Priority:** HIGH - Enables autonomous debugging
