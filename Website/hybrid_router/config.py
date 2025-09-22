"""Configuration settings for the hybrid router (updated)

This file mirrors `hybrid_router/config.py` but fixes two issues:
- Sets DEFAULT_PORT to 8000 to match Dockerfile.railway (EXPOSE 8000)
- Adds get_port() to respect the PORT environment variable
"""

import os
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)

# CORS origins
CORS_ORIGINS = [
    "https://projectbeacon.netlify.app",
    "https://project-beacon-portal.netlify.app",
    "http://localhost:3000",
    "http://localhost:5173",
    "http://localhost:8787",
    "http://127.0.0.1:3000",
    "http://127.0.0.1:5173",
    "http://127.0.0.1:8787",
]

# Server configuration
DEFAULT_PORT = 8000
HOST = "0.0.0.0"

def get_port() -> int:
    """Return the port to bind the server to.

    Reads the PORT environment variable, falling back to DEFAULT_PORT.
    If PORT is set but invalid, logs a warning and falls back.
    """
    raw = os.getenv("PORT")
    if raw is None or raw == "":
        return DEFAULT_PORT
    try:
        return int(raw)
    except ValueError:
        logging.warning("Invalid PORT env var '%s'; falling back to DEFAULT_PORT=%s", raw, DEFAULT_PORT)
        return DEFAULT_PORT

def get_google_maps_api_key() -> str:
    """Get Google Maps API key from environment"""
    key = os.getenv("GOOGLE_MAPS_API_KEY")
    if not key:
        logging.warning("GOOGLE_MAPS_API_KEY environment variable not set")
    return key
