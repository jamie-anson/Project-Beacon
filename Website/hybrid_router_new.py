#!/usr/bin/env python3
"""
Entry point for the refactored Project Beacon Hybrid Router
"""

if __name__ == "__main__":
    from hybrid_router.main import app
    from hybrid_router.config import get_port, HOST
    import uvicorn
    
    port = get_port()
    uvicorn.run(app, host=HOST, port=port, proxy_headers=True, forwarded_allow_ips="*")
