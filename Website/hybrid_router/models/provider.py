"""Provider models and types"""

from dataclasses import dataclass
from enum import Enum


class ProviderType(Enum):
    GOLEM = "golem"
    MODAL = "modal"
    # RUNPOD = "runpod"  # Removed - not using RunPod


@dataclass
class Provider:
    name: str
    type: ProviderType
    endpoint: str
    region: str
    cost_per_second: float
    max_concurrent: int
    healthy: bool = True
    last_health_check: float = 0
    avg_latency: float = 0
    success_rate: float = 1.0
