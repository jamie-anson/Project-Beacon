"""Configuration settings for the hybrid router"""

import os
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)

# CORS origins
CORS_ORIGINS = [
    "https://project-beacon-portal.netlify.app",
    "https://projectbeacon.netlify.app",
    "http://localhost:3000",
    "http://localhost:5173",
    "http://127.0.0.1:3000",
    "http://127.0.0.1:5173"
]

# Server configuration
DEFAULT_PORT = 8080
HOST = "0.0.0.0"

def get_port() -> int:
    """Get server port from environment"""
    return int(os.getenv("PORT", DEFAULT_PORT))
