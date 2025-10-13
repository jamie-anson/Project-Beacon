"""Region-based queue system for managing GPU resource allocation

This module implements per-region queues to prevent GPU limit exhaustion:
- Each region (US, EU, ASIA) has its own queue
- Jobs are processed sequentially per region
- Prevents parallel execution from exceeding Modal's 10 GPU limit
"""

import asyncio
import logging
from typing import Dict, Optional, Any
from datetime import datetime
from dataclasses import dataclass, field

logger = logging.getLogger(__name__)


@dataclass
class QueuedJob:
    """Represents a job waiting in a region queue"""
    job_id: str
    model: str
    prompt: str
    temperature: float
    max_tokens: int
    region: str
    queued_at: datetime = field(default_factory=datetime.utcnow)
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "job_id": self.job_id,
            "model": self.model,
            "region": self.region,
            "queued_at": self.queued_at.isoformat(),
            "started_at": self.started_at.isoformat() if self.started_at else None,
            "completed_at": self.completed_at.isoformat() if self.completed_at else None,
        }


class RegionQueue:
    """Queue for a single region with sequential processing"""
    
    def __init__(self, region: str):
        self.region = region
        self.queue: asyncio.Queue = asyncio.Queue()
        self.processing = False
        self.current_job: Optional[QueuedJob] = None
        self.completed_count = 0
        self.failed_count = 0
        
    async def enqueue(self, job: QueuedJob):
        """Add a job to the queue"""
        await self.queue.put(job)
        logger.info(f"[{self.region}] Enqueued job {job.job_id}, queue size: {self.queue.qsize()}")
        
    def get_status(self) -> Dict[str, Any]:
        """Get current queue status"""
        return {
            "region": self.region,
            "queue_size": self.queue.qsize(),
            "processing": self.processing,
            "current_job": self.current_job.to_dict() if self.current_job else None,
            "completed": self.completed_count,
            "failed": self.failed_count,
        }


class RegionQueueManager:
    """Manages multiple region queues for GPU resource control"""
    
    def __init__(self):
        self.queues: Dict[str, RegionQueue] = {
            "US": RegionQueue("US"),
            "EU": RegionQueue("EU"),
            "ASIA": RegionQueue("ASIA"),
        }
        self.workers_started = False
        
    async def enqueue_job(self, job: QueuedJob):
        """Enqueue a job to the appropriate region queue"""
        region = job.region.upper()
        if region not in self.queues:
            raise ValueError(f"Unknown region: {region}")
        
        await self.queues[region].enqueue(job)
        
    async def process_queue(self, region: str, inference_func):
        """Process jobs from a region queue sequentially"""
        queue = self.queues[region]
        logger.info(f"[{region}] Queue worker started")
        
        while True:
            try:
                # Wait for next job
                job: QueuedJob = await queue.queue.get()
                queue.processing = True
                queue.current_job = job
                job.started_at = datetime.utcnow()
                
                logger.info(f"[{region}] Processing job {job.job_id} (model: {job.model})")
                
                try:
                    # Execute inference
                    result = await inference_func(
                        model=job.model,
                        prompt=job.prompt,
                        temperature=job.temperature,
                        max_tokens=job.max_tokens,
                        region=region,
                    )
                    
                    job.completed_at = datetime.utcnow()
                    queue.completed_count += 1
                    
                    duration = (job.completed_at - job.started_at).total_seconds()
                    logger.info(f"[{region}] Completed job {job.job_id} in {duration:.2f}s")
                    
                except Exception as e:
                    job.completed_at = datetime.utcnow()
                    queue.failed_count += 1
                    logger.error(f"[{region}] Failed job {job.job_id}: {e}")
                    
                finally:
                    queue.current_job = None
                    queue.processing = False
                    queue.queue.task_done()
                    
            except Exception as e:
                logger.error(f"[{region}] Queue worker error: {e}")
                await asyncio.sleep(1)  # Brief pause before retrying
                
    async def start_workers(self, inference_func):
        """Start queue workers for all regions"""
        if self.workers_started:
            logger.warning("Queue workers already started")
            return
            
        self.workers_started = True
        
        # Start a worker for each region
        tasks = []
        for region in self.queues.keys():
            task = asyncio.create_task(self.process_queue(region, inference_func))
            tasks.append(task)
            logger.info(f"Started queue worker for {region}")
            
        # Keep workers running
        await asyncio.gather(*tasks)
        
    def get_all_statuses(self) -> Dict[str, Any]:
        """Get status of all region queues"""
        return {
            region: queue.get_status()
            for region, queue in self.queues.items()
        }
        
    def get_queue_size(self, region: str) -> int:
        """Get current queue size for a region"""
        return self.queues[region.upper()].queue.qsize()


# Global queue manager instance
queue_manager = RegionQueueManager()
