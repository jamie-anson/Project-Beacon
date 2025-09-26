import { hybridFetch } from '../http.js';

export async function getInfrastructureHealth() {
  const [healthRes, providersRes] = await Promise.allSettled([
    hybridFetch('/health'),
    hybridFetch('/providers'),
  ]);

  const health = healthRes.status === 'fulfilled' ? healthRes.value : null;
  const providersArr = providersRes.status === 'fulfilled' && Array.isArray(providersRes.value?.providers)
    ? providersRes.value.providers
    : [];

  const providersHealthy = providersArr.filter((p) => p?.healthy).length;
  const providersTotal = providersArr.length;
  const derivedOverall = providersTotal === 0
    ? (health?.status || 'unknown')
    : (providersHealthy === providersTotal ? 'healthy' : (providersHealthy > 0 ? 'degraded' : 'down'));
  const overall_status = String(health?.status || derivedOverall || 'unknown').toLowerCase();

  const services = {
    router: {
      status: overall_status,
      response_time_ms: null,
      error: healthRes.status === 'rejected' ? (healthRes.reason?.message || String(healthRes.reason || '')) : null,
    },
  };

  for (const provider of providersArr) {
    const key = `${String(provider.type || 'provider')}_${String(provider.region || 'unknown')}`;
    services[key] = {
      status: provider.healthy ? 'healthy' : 'down',
      response_time_ms: Number.isFinite(provider?.avg_latency) ? Math.round(Number(provider.avg_latency) * 1000) : null,
      error: null,
    };
  }

  const values = Object.values(services);
  const healthy_services = values.filter((s) => s.status === 'healthy').length;
  const degraded_services = values.filter((s) => s.status === 'degraded').length;
  const down_services = values.filter((s) => s.status === 'down').length;

  return {
    overall_status,
    services,
    healthy_services,
    degraded_services,
    down_services,
    total_services: Object.keys(services).length,
    last_checked: new Date().toISOString(),
  };
}
