"""Configuration settings for the hybrid router"""

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
    "http://127.0.0.1:8787"
]

# Server configuration
DEFAULT_PORT = 8080
HOST = "0.0.0.0"

def get_google_maps_api_key() -> str:
    """Get Google Maps API key from environment"""
    key = os.getenv("GOOGLE_MAPS_API_KEY")
    if not key:
        logging.warning("GOOGLE_MAPS_API_KEY environment variable not set")
    return key
