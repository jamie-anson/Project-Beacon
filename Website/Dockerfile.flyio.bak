# Dockerfile for Project Beacon hybrid routing service
FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python dependencies (support repo-root build context)
COPY Website/flyio-deployment/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY Website/flyio-deployment/ .

# Expose port (align with PORT=8080 default and Railway conventions)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application with uvicorn (respect Railway $PORT)
CMD ["sh", "-c", "uvicorn hybrid_router:app --host 0.0.0.0 --port ${PORT:-8080} --proxy-headers"]
