FROM python:3.9-slim

WORKDIR /app

# Install system dependencies for health check
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

# Copy the hybrid router files (Railway sees entire repo context)
COPY hybrid_router_new.py requirements.txt ./
COPY hybrid_router/ ./hybrid_router/

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD sh -c 'curl -f http://localhost:${PORT:-8000}/ready || exit 1'

# Start the application
CMD ["python3", "hybrid_router_new.py"]
