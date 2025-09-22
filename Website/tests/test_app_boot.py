import os
import sys
from pathlib import Path
from fastapi.testclient import TestClient

# Ensure imports resolve relative to Website/
WEBSITE_DIR = Path(__file__).resolve().parent.parent
if str(WEBSITE_DIR) not in sys.path:
    sys.path.insert(0, str(WEBSITE_DIR))

from hybrid_router.main import app  # noqa: E402


def test_app_imports_and_title():
    assert app.title == "Project Beacon Hybrid Router"


def test_health_endpoint_ok():
    with TestClient(app) as client:
        r = client.get("/health")
        assert r.status_code == 200
        data = r.json()
        assert "status" in data
        assert data["status"] in {"healthy", "ok", "ready"}
