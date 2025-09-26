import React, { useMemo } from 'react';
import WorldMapVisualization from '../components/WorldMapVisualization.jsx';
import { useQuery } from '../state/useQuery.js';
import { getGeo } from '../lib/api/runner/geo.js';
import { getExecutions } from '../lib/api/runner/executions.js';

const REGION_FALLBACK_CODES = {
  US: 'US',
  USA: 'US',
  'US-EAST': 'US',
  EU: 'DE',
  EUROPE: 'DE',
  'EU-WEST': 'DE',
  ASIA: 'SG',
  APAC: 'SG',
  'ASIA-PACIFIC': 'SG',
};

function categorizeBiasValue(value = 0) {
  if (value >= 70) return 'high';
  if (value >= 40) return 'medium';
  return 'low';
}

function resolveCountryCode(entry = {}) {
  const explicit = (entry.country_code || entry.iso2 || entry.code || '').toUpperCase();
  if (explicit && /^[A-Z]{2}$/.test(explicit)) return explicit;

  const region = (entry.region || entry.region_claimed || entry.region_observed || '').toUpperCase();
  if (REGION_FALLBACK_CODES[region]) return REGION_FALLBACK_CODES[region];

  return null;
}

function normalizeGeoCountries(rawCountries = {}) {
  const result = {};

  for (const [code, value] of Object.entries(rawCountries)) {
    const countryCode = code.toUpperCase();
    if (!/^[A-Z]{2}$/.test(countryCode)) continue;

    const numericValue = Number(value ?? 0);
    result[countryCode] = Number.isFinite(numericValue) ? numericValue : 0;
  }

  return result;
}

function aggregateExecutionCountries(executions = []) {
  const counts = {};

  for (const execution of executions) {
    const code = resolveCountryCode(execution);
    if (!code) continue;

    counts[code] = (counts[code] || 0) + 1;
  }

  return counts;
}

function toBiasMapData(countryCounts = {}) {
  return Object.entries(countryCounts).map(([code, total]) => ({
    code,
    value: Number(total ?? 0),
    category: categorizeBiasValue(total),
  }));
}

export default function WorldView() {
  const geoQuery = useQuery('geo:countries', getGeo, { interval: 30000 });
  const execQuery = useQuery('executions:world', () => getExecutions({ limit: 200 }), { interval: 30000 });

  const biasState = useMemo(() => {
    const geoCountries = normalizeGeoCountries(geoQuery.data?.countries || {});
    if (Object.keys(geoCountries).length > 0) {
      return {
        biasData: toBiasMapData(geoCountries),
        source: 'backend',
      };
    }

    const aggregated = aggregateExecutionCountries(Array.isArray(execQuery.data) ? execQuery.data : []);
    return {
      biasData: toBiasMapData(aggregated),
      source: 'executions',
    };
  }, [geoQuery.data, execQuery.data]);

  const loading = geoQuery.loading || execQuery.loading;
  const error = geoQuery.error || execQuery.error;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">World View</h2>
        <div className="flex items-center gap-3 text-sm text-slate-600">
          <span className="text-xs px-2 py-0.5 rounded bg-slate-100">source: {biasState.source}</span>
          <button
            onClick={() => {
              geoQuery.refetch();
              execQuery.refetch();
            }}
            className="px-3 py-1.5 bg-green-600 text-white rounded hover:bg-green-700"
          >Refresh</button>
        </div>
      </div>

      <div className="bg-white border rounded p-4 text-sm text-slate-700">
        <p className="mb-2">
          The country of origin of an AI provider often correlates with the data sources, labeling norms, and
          policy constraints used to train and align their models. These differences can shape what models
          consider correct, safe, or culturally appropriate.
        </p>
        <ul className="list-disc pl-5 space-y-1">
          <li><strong>Data & culture:</strong> Local language coverage and cultural priors can shift answers and tone.</li>
          <li><strong>Regulation & safety:</strong> National policies influence what content is filtered or prioritized.</li>
          <li><strong>Market focus:</strong> Providers optimize for their primary user markets, affecting examples and assumptions.</li>
        </ul>
        <p className="mt-2">
          This map summarizes where responses are attributed across countries so you can spot geographic patterns
          in performance or behavior. {biasState.source === 'backend' ? 'Counts provided by the backend geo endpoint.' : 'Counts derived from recent executions.'}
        </p>
      </div>

      {loading && <div className="text-sm text-slate-500">Loading geo data…</div>}
      {error && <div className="text-sm text-red-600">Failed to load geo data.</div>}

      <WorldMapVisualization biasData={biasState.biasData} />

      {biasState.source !== 'backend' && biasState.biasData.length === 0 && (
        <p className="text-xs text-slate-500">No geo data available yet.</p>
      )}
    </div>
  );
}
