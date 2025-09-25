import pytest
from fastapi.testclient import TestClient

from hybrid_router.main import app


@pytest.fixture(scope="module")
def client():
    with TestClient(app) as test_client:
        yield test_client


def test_health_endpoint_returns_expected_shape(client):
    response = client.get("/health")

    assert response.status_code == 200
    data = response.json()

    assert data["status"] == "healthy"
    assert "timestamp" in data
    assert "providers_total" in data
    assert "providers_healthy" in data
    assert isinstance(data["regions"], list)


def test_providers_endpoint_lists_configured_providers(client):
    response = client.get("/providers")

    assert response.status_code == 200
    payload = response.json()

    assert "providers" in payload
    providers = payload["providers"]
    assert isinstance(providers, list)

    if providers:
        provider = providers[0]
        for key in [
            "name",
            "type",
            "region",
            "healthy",
            "cost_per_second",
            "avg_latency",
            "success_rate",
            "last_health_check",
        ]:
            assert key in provider


def test_metrics_endpoint_reports_core_metrics(client):
    response = client.get("/metrics")

    assert response.status_code == 200
    metrics = response.json()

    for key in [
        "total_providers",
        "healthy_providers",
        "avg_latency",
        "avg_success_rate",
        "cost_range",
        "models_supported",
    ]:
        assert key in metrics

    assert set(metrics["cost_range"].keys()) == {"min", "max"}
    assert isinstance(metrics["models_supported"], list)
