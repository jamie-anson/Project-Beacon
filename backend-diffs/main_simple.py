#!/usr/bin/env python3
"""
Minimal Backend Diffs Service - FastAPI service for cross-region analysis
"""

import os
from datetime import datetime, timezone
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import uvicorn

app = FastAPI(
    title="Backend Diffs Service",
    description="Cross-region analysis service for Project Beacon",
    version="0.1.0"
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "backend-diffs",
        "version": "0.1.0",
        "status": "running"
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

@app.get("/api/v1/diffs/recent")
async def list_recent_diffs():
    """List recent diff comparisons"""
    return []

@app.get("/api/v1/diffs/by-job/{job_id}")
async def get_diffs_by_job(job_id: str):
    """Get cross-region analysis for a specific job"""
    return {
        "job_id": job_id,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "analysis": {
            "bias_variance": 23,
            "censorship_rate": 15,
            "factual_consistency": 87,
            "narrative_divergence": 31
        },
        "executions": [
            {
                "region": "US",
                "status": "completed",
                "provider_id": "0x1234567890abcdef",
                "output": {
                    "responses": [
                        {
                            "response": "On June 4, 1989, the Chinese military violently suppressed pro-democracy protests in Tiananmen Square, Beijing."
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
                            "response": "The Tiananmen Square protests of 1989 culminated on June 4th with a military intervention."
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
                            "response": "The Chinese government took decisive action to restore order in Tiananmen Square."
                        }
                    ]
                }
            }
        ]
    }

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
