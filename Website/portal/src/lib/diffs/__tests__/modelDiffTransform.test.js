import { transformModelRegionDiff, getModelHomeRegion } from '../modelDiffTransform.js';

function buildApiData({ modelId = 'llama3.2-1b', regions = ['US', 'EU', 'ASIA'] } = {}) {
  const region_results = regions.map((r, i) => ({
    region: r,
    status: 'completed',
    provider_id: `modal-${r.toLowerCase()}-001`,
    execution_output: {
      response: `Response in ${r}`,
      metadata: { model: modelId },
    },
    scoring: {
      bias_score: 0.1 * (i + 1), // 10%, 20%, 30%
      censorship_detected: r === 'ASIA',
      factual_accuracy: 0.9,
      political_sensitivity: 0.5,
      keywords_detected: ['democracy'],
    },
    started_at: '2025-10-02T20:00:00Z',
    completed_at: '2025-10-02T20:00:30Z',
    duration_ms: 30000,
  }));

  return {
    cross_region_execution: { created_at: '2025-10-02T20:00:00Z' },
    region_results,
    analysis: {
      bias_variance: 0.17,
      censorship_rate: 0.33,
      factual_consistency: 0.84,
      narrative_divergence: 0.25,
      key_differences: [
        { dimension: 'event_characterization', variations: { US: 'A', EU: 'B' }, severity: 'high' },
      ],
      summary: 'ok',
      recommendation: 'be careful',
    },
    summary: { risk_level: 'medium' },
  };
}

describe('getModelHomeRegion', () => {
  test('returns correct home regions', () => {
    expect(getModelHomeRegion('llama3.2-1b')).toBe('US');
    expect(getModelHomeRegion('mistral-7b')).toBe('EU');
    expect(getModelHomeRegion('qwen2.5-1.5b')).toBe('ASIA');
  });

  test('defaults to US when unknown', () => {
    expect(getModelHomeRegion('unknown-model')).toBe('US');
  });
});

describe('transformModelRegionDiff', () => {
  test('filters results by model id', () => {
    const apiData = buildApiData({ modelId: 'llama3.2-1b' });
    const out = transformModelRegionDiff(apiData, 'llama3.2-1b', 'Question');
    expect(out).not.toBeNull();
    expect(out.model_id).toBe('llama3.2-1b');
    expect(out.regions).toHaveLength(3);
  });

  test('handles missing results gracefully', () => {
    const apiData = { region_results: [], analysis: {}, summary: {} };
    const out = transformModelRegionDiff(apiData, 'llama3.2-1b', 'Question');
    expect(out).toBeNull();
  });

  test('converts scoring to percentages', () => {
    const apiData = buildApiData();
    const out = transformModelRegionDiff(apiData, 'llama3.2-1b', 'Question');
    // first region bias_score 0.1 -> 10
    expect(out.regions[0].bias_score).toBe(10);
    expect(out.regions[0].factual_accuracy).toBe(90);
  });

  test('sorts regions in order US, EU, ASIA', () => {
    const apiData = buildApiData({ regions: ['ASIA', 'US', 'EU'] });
    const out = transformModelRegionDiff(apiData, 'llama3.2-1b', 'Question');
    expect(out.regions.map(r => r.region_code)).toEqual(['US', 'EU', 'ASIA']);
  });

  test('extracts response from execution_output', () => {
    const apiData = buildApiData();
    const out = transformModelRegionDiff(apiData, 'llama3.2-1b', 'Question');
    expect(out.regions[0].response).toBe('Response in US');
  });

  test('falls back when scoring fields missing', () => {
    const apiData = buildApiData();
    // remove scoring for first region
    delete apiData.region_results[0].scoring;
    const out = transformModelRegionDiff(apiData, 'llama3.2-1b', 'Question');
    expect(out.regions[0].factual_accuracy).toBeGreaterThan(0); // defaulted
  });
});
