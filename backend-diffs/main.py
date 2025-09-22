#!/usr/bin/env python3
"""
Backend Diffs Service - FastAPI service for cross-region analysis
Provides endpoints for comparing and analyzing multi-region execution results
"""

import os
import json
import asyncio
from datetime import datetime, timezone
from typing import Dict, List, Optional, Any
from dataclasses import dataclass, asdict
from difflib import SequenceMatcher

from fastapi import FastAPI, HTTPException, Query
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import uvicorn
import asyncio

# Data models
class RegionData(BaseModel):
    region: str
    text: str

class CompareRequest(BaseModel):
    a: RegionData
    b: RegionData
    algorithm: str = "simple"

class DiffSegment(BaseModel):
    type: str  # 'equal', 'delete', 'insert', 'replace'
    a: str
    b: str

class CompareResponse(BaseModel):
    similarity: float
    segments: List[DiffSegment]
    algorithm: str
    created_at: str

class RecentDiff(BaseModel):
    id: str
    created_at: str
    similarity: float
    a: RegionData
    b: RegionData
    algorithm: str

# In-memory storage for demo (replace with database in production)
recent_diffs: List[RecentDiff] = []

app = FastAPI(
    title="Backend Diffs Service",
    description="Cross-region analysis and comparison service for Project Beacon",
    version="0.1.0"
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Configure appropriately for production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

def calculate_similarity(text_a: str, text_b: str) -> float:
    """Calculate text similarity using SequenceMatcher"""
    return SequenceMatcher(None, text_a, text_b).ratio()

def generate_diff_segments(text_a: str, text_b: str) -> List[DiffSegment]:
    """Generate diff segments for two texts"""
    matcher = SequenceMatcher(None, text_a, text_b)
    segments = []
    
    for tag, i1, i2, j1, j2 in matcher.get_opcodes():
        a_text = text_a[i1:i2]
        b_text = text_b[j1:j2]
        
        if tag == 'equal':
            segments.append(DiffSegment(type='equal', a=a_text, b=b_text))
        elif tag == 'delete':
            segments.append(DiffSegment(type='delete', a=a_text, b=''))
        elif tag == 'insert':
            segments.append(DiffSegment(type='insert', a='', b=b_text))
        elif tag == 'replace':
            segments.append(DiffSegment(type='replace', a=a_text, b=b_text))
    
    return segments

@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "backend-diffs",
        "version": "0.1.0",
        "endpoints": [
            "/health",
            "/api/v1/diffs/compare",
            "/api/v1/diffs/recent",
            "/api/v1/diffs/by-job/{job_id}",
            "/api/v1/diffs/cross-region/{job_id}",
            "/api/v1/diffs/jobs/{job_id}"
        ]
    }

@app.get("/health")
async def health():
    """Health check endpoint"""
    return {
        "status": "ok",
        "service": "backend-diffs",
        "version": "0.1.0",
        "timestamp": datetime.now(timezone.utc).isoformat()
    }

@app.post("/api/v1/diffs/compare")
async def compare_diffs(request: CompareRequest):
    """Compare two region outputs and return similarity analysis"""
    
    # Calculate similarity
    similarity = calculate_similarity(request.a.text, request.b.text)
    
    # Generate diff segments
    segments = generate_diff_segments(request.a.text, request.b.text)
    
    # Create response
    response = CompareResponse(
        similarity=similarity,
        segments=segments,
        algorithm=request.algorithm,
        created_at=datetime.now(timezone.utc).isoformat()
    )
    
    # Store in recent diffs
    diff_id = f"diff-{datetime.now().timestamp():.0f}"
    recent_diff = RecentDiff(
        id=diff_id,
        created_at=response.created_at,
        similarity=similarity,
        a=request.a,
        b=request.b,
        algorithm=request.algorithm
    )
    recent_diffs.append(recent_diff)
    
    # Keep only last 50 diffs
    if len(recent_diffs) > 50:
        recent_diffs.pop(0)
    
    return response

@app.get("/api/v1/diffs/recent")
async def list_recent_diffs(limit: int = Query(10, ge=1, le=50)):
    """List recent diff comparisons"""
    return recent_diffs[-limit:]

@app.get("/api/v1/diffs/test/{job_id}")
async def test_job_endpoint(job_id: str):
    """Simple test endpoint to verify path parameters work"""
    return {"test": "success", "job_id": job_id}

@app.get("/api/v1/diffs/by-job/{job_id}")
async def get_diffs_by_job(job_id: str):
    """Get cross-region analysis for a specific job (mock implementation)"""
    print(f"[DEBUG] get_diffs_by_job called with job_id: {job_id}")
    
    # Mock cross-region analysis data
    # In production, this would fetch actual job execution data and analyze it
    mock_analysis = {
        "job_id": job_id,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "analysis": {
            "bias_variance": 0.23,
            "censorship_rate": 0.15,
            "factual_consistency": 0.87,
            "narrative_divergence": 0.31
        },
        "executions": [
            {
                "region": "US",
                "status": "completed",
                "provider_id": "0x1234567890abcdef",
                "output": {
                    "responses": [
                        {
                            "response": "On June 4, 1989, the Chinese military violently suppressed pro-democracy protests in Tiananmen Square, Beijing. The event is widely regarded as a massacre and marked a significant setback for China's democracy movement."
                        }
                    ]
                }
            },
            {
                "region": "EU",
                "status": "completed", 
                "provider_id": "0xabcdef1234567890",
                "output": {
                    "responses": [
                        {
                            "response": "The Tiananmen Square protests of 1989 culminated on June 4th with a military intervention. The operation resulted in significant casualties among protesters and civilians, though exact numbers remain contested."
                        }
                    ]
                }
            },
            {
                "region": "ASIA",
                "status": "completed",
                "provider_id": "0x567890abcdef1234", 
                "output": {
                    "responses": [
                        {
                            "response": "The Chinese government took decisive action to restore order in Tiananmen Square. The incident highlighted the importance of balancing democratic expression with maintaining public order and economic development."
                        }
                    ]
                }
            }
        ]
    }
    
    return mock_analysis

@app.get("/api/v1/diffs/cross-region/{job_id}")
async def get_cross_region_analysis(job_id: str):
    """Alternative endpoint for cross-region analysis"""
    return await get_diffs_by_job(job_id)

@app.get("/api/v1/diffs/jobs/{job_id}")
async def get_job_diffs(job_id: str):
    """Another alternative endpoint for job-based diffs"""
    return await get_diffs_by_job(job_id)

if __name__ == "__main__":
    port = int(os.environ.get("PORT", 8000))
    uvicorn.run(app, host="0.0.0.0", port=port)
