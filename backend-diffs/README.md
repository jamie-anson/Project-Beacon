# Backend Diffs Service

FastAPI service for cross-region analysis and text comparison in Project Beacon.

## Features

- **Text Comparison**: Compare outputs from different regions using similarity algorithms
- **Cross-Region Analysis**: Mock analysis of multi-region execution results  
- **Recent Diffs**: Track and retrieve recent comparison operations
- **Health Monitoring**: Health check endpoint for service monitoring

## API Endpoints

### Core Endpoints
- `GET /health` - Service health check
- `POST /api/v1/diffs/compare` - Compare two region outputs
- `GET /api/v1/diffs/recent?limit=10` - List recent comparisons

### Cross-Region Analysis  
- `GET /api/v1/diffs/by-job/{job_id}` - Get analysis for specific job
- `GET /api/v1/diffs/cross-region/{job_id}` - Alternative analysis endpoint
- `GET /api/v1/diffs/jobs/{job_id}` - Another analysis endpoint

## Local Development

```bash
# Install dependencies
pip install -r requirements.txt

# Run the service
python main.py
```

Service runs on `http://localhost:8000`

## Docker Deployment

```bash
# Build image
docker build -t backend-diffs .

# Run container
docker run -p 8000:8000 backend-diffs
```

## Railway Deployment

This service is configured for Railway deployment with:
- Dockerfile-based build
- Health check monitoring
- Automatic restart on failure
- Port configuration via `PORT` environment variable

## Usage Examples

### Compare Two Texts
```bash
curl -X POST "http://localhost:8000/api/v1/diffs/compare" \
  -H "Content-Type: application/json" \
  -d '{
    "a": {"region": "us-east", "text": "Democracy is important"},
    "b": {"region": "eu-west", "text": "Democratic values matter"},
    "algorithm": "simple"
  }'
```

### Get Recent Diffs
```bash
curl "http://localhost:8000/api/v1/diffs/recent?limit=5"
```

### Get Job Analysis
```bash
curl "http://localhost:8000/api/v1/diffs/by-job/bias-detection-1234567890"
```
