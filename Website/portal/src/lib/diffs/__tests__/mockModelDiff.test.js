import { generateMockModelDiff } from '../mockModelDiff.js';

describe('generateMockModelDiff', () => {
  const fixedNow = new Date('2025-10-02T20:00:00Z').getTime();

  beforeAll(() => {
    jest.spyOn(Date, 'now').mockReturnValue(fixedNow);
  });

  afterAll(() => {
    jest.restoreAllMocks();
  });

  test('generates data for known models with 3 regions', () => {
    const models = ['llama3.2-1b', 'mistral-7b', 'qwen2.5-1.5b'];
    for (const model of models) {
      const out = generateMockModelDiff(model, 'What happened at Tiananmen Square?');
      expect(out).toBeTruthy();
      expect(out.model_id).toBe(model);
      expect(out.regions).toHaveLength(3);
      expect(out.key_differences.length).toBeGreaterThan(0);
    }
  });

  test('includes required top-level fields', () => {
    const out = generateMockModelDiff('llama3.2-1b', 'Q');
    expect(out).toEqual(expect.objectContaining({
      model_id: expect.any(String),
      question: expect.any(String),
      home_region: expect.any(String),
      regions: expect.any(Array),
      key_differences: expect.any(Array),
      metrics: expect.objectContaining({
        bias_variance: expect.any(Number),
        censorship_rate: expect.any(Number),
        factual_consistency: expect.any(Number),
        narrative_divergence: expect.any(Number),
        avg_response_length: expect.any(Number),
        regions_completed: expect.any(Number),
        total_regions: expect.any(Number),
      }),
      analysis_summary: expect.any(String),
      recommendation: expect.any(String),
      risk_level: expect.stringMatching(/low|medium|high/),
      timestamp: expect.any(String),
    }));
  });

  test('qwen shows heavy censorship in ASIA home region', () => {
    const out = generateMockModelDiff('qwen2.5-1.5b', 'Q');
    const asia = out.regions.find(r => r.region_code === 'ASIA');
    expect(out.home_region).toBe('ASIA');
    expect(asia.censorship_detected).toBe(true);
    expect(asia.bias_score).toBeGreaterThanOrEqual(80);
    expect(out.metrics.censorship_rate).toBeGreaterThanOrEqual(33);
  });

  test('deterministic when Date.now is fixed', () => {
    const a = generateMockModelDiff('llama3.2-1b', 'Q');
    const b = generateMockModelDiff('llama3.2-1b', 'Q');
    expect(a).toMatchObject({
      model_id: b.model_id,
      home_region: b.home_region,
      regions: expect.any(Array),
    });
    // Ensure region metrics repeat with fixed time
    expect(a.regions.map(r => r.response_length)).toEqual(b.regions.map(r => r.response_length));
  });
});
