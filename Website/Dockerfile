FROM python:3.9-slim

WORKDIR /app

# Install system dependencies for health checks
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       curl build-essential python3-dev libffi-dev libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install deps
COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code needed for the router service
COPY hybrid_router/ /app/hybrid_router/

# Expose port (modular app defaults to 8000; PORT env var will override)
EXPOSE 8000

# Robust health check: try /health then fall back to /ready
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD sh -c 'curl -fsS http://localhost:${PORT:-8000}/health || curl -fsS http://localhost:${PORT:-8000}/ready || exit 1'

# Start the modular FastAPI app
CMD ["python3", "-m", "hybrid_router.main"]
