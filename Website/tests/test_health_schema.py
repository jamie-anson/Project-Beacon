import sys
from pathlib import Path
from time import perf_counter
from fastapi.testclient import TestClient

# Ensure Website on path
WEBSITE_DIR = Path(__file__).resolve().parent.parent
if str(WEBSITE_DIR) not in sys.path:
    sys.path.insert(0, str(WEBSITE_DIR))

from hybrid_router.main import app  # noqa: E402

client = TestClient(app)


def test_health_schema_and_latency():
    t0 = perf_counter()
    r = client.get('/health')
    t1 = perf_counter()

    assert r.status_code == 200
    data = r.json()

    # Required keys
    for key in [
        'status', 'timestamp', 'providers_total', 'providers_healthy', 'regions'
    ]:
        assert key in data, f"missing key: {key}"

    # Types
    assert isinstance(data['status'], str)
    assert isinstance(data['timestamp'], (int, float))
    assert isinstance(data['providers_total'], int)
    assert isinstance(data['providers_healthy'], int)
    assert isinstance(data['regions'], list)

    # Should be fast (< 500ms in test environment)
    assert (t1 - t0) < 0.5, f"/health too slow: {(t1 - t0):.3f}s"
