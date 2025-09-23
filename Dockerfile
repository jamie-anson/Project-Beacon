FROM python:3.9-slim

WORKDIR /app

# Install system dependencies for health check and Modal CLI
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

# Copy requirements.txt first for better Docker layer caching
COPY requirements.txt ./

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Install Modal CLI for EU/APAC function calls
RUN pip install --no-cache-dir modal

# Create Modal config directory
RUN mkdir -p /root/.modal

# Copy the hybrid router file
COPY hybrid_router.py ./

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD sh -c 'curl -f http://localhost:${PORT:-8080}/ready || exit 1'

# Start the application
CMD ["python3", "hybrid_router.py"]
