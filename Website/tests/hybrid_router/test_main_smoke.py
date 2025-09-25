from fastapi.testclient import TestClient

from hybrid_router.main import app


def test_health_and_ready_endpoints():
    with TestClient(app) as client:
        r_ready = client.get("/ready")
        assert r_ready.status_code == 200
        assert r_ready.json().get("status") in ("ok", "healthy")

        r_health = client.get("/health")
        assert r_health.status_code == 200
        body = r_health.json()
        assert body["status"] == "healthy"
        assert "providers_total" in body
        assert "providers_healthy" in body
        assert isinstance(body.get("regions"), list)


def test_providers_and_metrics():
    with TestClient(app) as client:
        r_providers = client.get("/providers")
        assert r_providers.status_code == 200
        data = r_providers.json()
        assert "providers" in data
        assert isinstance(data["providers"], list)

        r_metrics = client.get("/metrics")
        assert r_metrics.status_code == 200
        metrics = r_metrics.json()
        assert set(metrics.keys()) >= {"total_providers", "healthy_providers", "avg_latency", "avg_success_rate", "cost_range"}
