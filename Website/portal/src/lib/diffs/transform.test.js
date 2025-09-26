import { transformCrossRegionDiff, extractKeywordsFromResponse } from './transform.js';
import { AVAILABLE_MODELS, DEFAULT_QUESTION } from './constants.js';

describe('diffs transform utilities', () => {
  test('normalizes cross-region diff payloads with analysis metrics and keyword extraction', () => {
    const apiData = {
      job_id: 'job-123',
      question: { text: 'What happened in the 1989 protests?' },
      analysis: {
        bias_variance: 0.42,
        censorship_rate: 0.12,
        factual_consistency: 0.91,
        narrative_divergence: 0.33
      },
      generated_at: '2025-09-24T00:00:00Z',
      executions: [
        {
          region: 'US',
          status: 'completed',
          provider_id: 'provider_us',
          output: {
            responses: [
              {
                response: 'The government violently suppressed the protest, resulting in a massacre.'
              }
            ]
          }
        },
        {
          region: 'EU',
          status: 'completed',
          provider_id: 'provider_eu',
          output: {
            text_output: 'The incident is widely regarded as a massacre with heavy casualties.'
          }
        }
      ]
    };

    const jobData = {
      id: 'job-123',
      jobspec: {
        questions: [
          {
            question: 'Fallback job question?'
          }
        ]
      }
    };

    const result = transformCrossRegionDiff(apiData, jobData, AVAILABLE_MODELS);

    expect(result).not.toBeNull();
    expect(result.job_id).toBe('job-123');
    expect(result.question).toBe('What happened in the 1989 protests?');
    expect(result.metrics).toEqual({
      bias_variance: 42,
      censorship_rate: 12,
      factual_consistency: 91,
      narrative_divergence: 33
    });

    expect(result.models).toHaveLength(AVAILABLE_MODELS.length);
    result.models.forEach((model) => {
      expect(model.regions).toHaveLength(2);
    });

    const usRegion = result.models[0].regions.find((region) => region.region_code === 'US');
    expect(usRegion.provider_id).toBe('provider_us');
    expect(usRegion.status).toBe('completed');
    expect(usRegion.response).toContain('violently');
    expect(usRegion.keywords).toContain('violence');
  });

  test('falls back to job question text and default metrics when analysis data missing', () => {
    const apiData = {
      job_id: 'job-456',
      executions: [
        {
          region: 'ASIA',
          status: 'pending',
          provider_id: 'provider_apac',
          output: {
            output: 'Content unavailable.'
          }
        }
      ]
    };

    const jobData = {
      id: 'job-456',
      jobspec: {
        questions: [
          {
            text: 'Fallback question from job spec.'
          }
        ]
      }
    };

    const result = transformCrossRegionDiff(apiData, jobData);
    expect(result.job_id).toBe('job-456');
    expect(result.question).toBe('Fallback question from job spec.');
    expect(result.metrics.bias_variance).toBe(23); // default 0.23 -> 23
    expect(result.models[0].regions[0].region_code).toBe('ASIA');
  });

  test('returns null when api data is absent', () => {
    expect(transformCrossRegionDiff(null, null)).toBeNull();
  });

  test('extractKeywordsFromResponse detects categories via simple heuristics', () => {
    const keywords = extractKeywordsFromResponse('The government crackdown led to casualties and violence.');
    expect(keywords).toEqual(expect.arrayContaining(['government', 'violence']));
  });
});
