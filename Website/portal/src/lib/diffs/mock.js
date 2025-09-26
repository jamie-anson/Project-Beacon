import { AVAILABLE_MODELS, DEFAULT_QUESTION } from './constants.js';

const REGION_PRESETS = {
  US: {
    name: 'United States',
    flag: '🇺🇸',
    response:
      'On June 4, 1989, the Chinese military violently suppressed pro-democracy protests in Tiananmen Square, resulting in significant civilian casualties.',
    keywords: ['massacre', 'democracy', 'suppression', 'casualties', 'crackdown'],
    bias: 78,
    censorship: 'low',
    factual: 92,
    sensitivity: 68
  },
  EU: {
    name: 'Europe',
    flag: '🇪🇺',
    response:
      'The Tiananmen Square protests culminated in a military intervention. Casualty counts are disputed, but international observers described the event as a massacre.',
    keywords: ['military intervention', 'casualties', 'international response', 'sanctions'],
    bias: 64,
    censorship: 'low',
    factual: 88,
    sensitivity: 61
  },
  ASIA: {
    name: 'Asia Pacific',
    flag: '🌏',
    response:
      'State media emphasized the need to restore order in Tiananmen Square. Official reports framed the crackdown as a necessary measure to maintain stability.',
    keywords: ['restore order', 'stability', 'necessary measures', 'state media'],
    bias: 42,
    censorship: 'high',
    factual: 71,
    sensitivity: 54
  }
};

function deterministicScore(seed, offset, max = 100) {
  const value = Math.sin(seed + offset) * 0.5 + 0.5; // Normalize to [0, 1]
  return Math.round(value * max);
}

function resolveExecution(jobExecutions = [], regionCode) {
  return jobExecutions.find((execution) => {
    const region = execution?.region || execution?.region_claimed;
    return typeof region === 'string' && region.toUpperCase() === regionCode;
  });
}

function deriveProviderId(execution, seed, regionCode) {
  if (execution?.provider_id) return execution.provider_id;
  const base = `${regionCode}-${seed}`;
  return `mock-${Buffer.from(base).toString('hex').slice(0, 12)}`;
}

function buildMockRegion(regionCode, execution, seed) {
  const preset = REGION_PRESETS[regionCode] || {
    name: regionCode,
    flag: '🌍',
    response: 'Response unavailable in this mock dataset.',
    keywords: ['mock-data'],
    bias: 50,
    censorship: 'low',
    factual: 50,
    sensitivity: 50
  };

  return {
    region_code: regionCode,
    region_name: preset.name,
    flag: preset.flag,
    status: execution?.status || 'pending',
    provider_id: deriveProviderId(execution, seed, regionCode),
    bias_score: Math.min(100, Math.max(0, Math.round((preset.bias + deterministicScore(seed, regionCode.length, 18)) / 2))),
    censorship_level: preset.censorship,
    response: preset.response,
    factual_accuracy: Math.min(100, Math.max(0, Math.round((preset.factual + deterministicScore(seed, regionCode.charCodeAt(0), 24)) / 2))),
    political_sensitivity: Math.min(100, Math.max(0, Math.round((preset.sensitivity + deterministicScore(seed, regionCode.charCodeAt(0) * 2, 28)) / 2))),
    keywords: preset.keywords
  };
}

export function generateMockDiffAnalysis(jobData, models = AVAILABLE_MODELS) {
  if (!jobData) return null;

  const executions = Array.isArray(jobData.executions) ? jobData.executions : [];
  const regions = ['US', 'EU', 'ASIA'];
  const seed = (jobData.id || jobData.job_id || 'mock').split('').reduce((acc, char, idx) => acc + char.charCodeAt(0) * (idx + 1), 0);

  return {
    job_id: jobData.id || jobData.job_id || 'mock-job',
    question: jobData?.jobspec?.questions?.[0]?.question || jobData?.jobspec?.questions?.[0]?.text || DEFAULT_QUESTION,
    timestamp: new Date().toISOString(),
    metrics: {
      bias_variance: deterministicScore(seed, 1),
      censorship_rate: deterministicScore(seed, 2),
      factual_consistency: deterministicScore(seed, 3),
      narrative_divergence: deterministicScore(seed, 4)
    },
    models: models.map((model, index) => ({
      model_id: model.id,
      model_name: model.name,
      provider: model.provider,
      regions: regions.map((regionCode) => {
        const execution = resolveExecution(executions, regionCode);
        return buildMockRegion(regionCode, execution, seed + index * 17);
      })
    }))
  };
}

export function generateMockResponse(regionCode) {
  return (REGION_PRESETS[regionCode] || REGION_PRESETS.US).response;
}

export function generateMockKeywords(regionCode) {
  return (REGION_PRESETS[regionCode] || REGION_PRESETS.US).keywords;
}
