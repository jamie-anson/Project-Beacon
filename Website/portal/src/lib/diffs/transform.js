/**
 * @typedef {import('./constants.js').DiffAnalysis} DiffAnalysis
 * @typedef {import('./constants.js').DiffModel} DiffModel
 * @typedef {import('./constants.js').DiffRegion} DiffRegion
 */

import { AVAILABLE_MODELS, DEFAULT_QUESTION } from './constants.js';

const REGION_LABELS = {
  US: { name: 'United States', flag: '🇺🇸' },
  EU: { name: 'Europe', flag: '🇪🇺' },
  ASIA: { name: 'Asia Pacific', flag: '🌏' }
};

/**
 * Extracts meaningful keywords from a response string using simple pattern matching.
 * @param {string|undefined|null} response
 * @returns {string[]}
 */
export function extractKeywordsFromResponse(response) {
  if (!response) return [];

  const keywords = new Set();
  const text = response.toLowerCase();

  const keywordPatterns = {
    censorship: ['cannot discuss', 'not appropriate', 'sensitive topic', 'restricted'],
    violence: ['massacre', 'violence', 'crackdown', 'suppression', 'casualties'],
    democracy: ['democracy', 'protest', 'demonstration', 'freedom'],
    government: ['government', 'military', 'authorities', 'official'],
    neutral: ['incident', 'event', 'situation', 'development']
  };

  Object.entries(keywordPatterns).forEach(([category, patterns]) => {
    patterns.forEach((pattern) => {
      if (text.includes(pattern)) keywords.add(category);
    });
  });

  return [...keywords];
}

/**
 * Normalizes the cross-region diff API payload into a UI-friendly structure.
 * @param {any} apiData raw cross-region diff payload
 * @param {any} jobData job payload containing executions (optional)
 * @param {Array} models list of models to hydrate (defaults to AVAILABLE_MODELS)
 * @returns {DiffAnalysis|null}
 */
export function transformCrossRegionDiff(apiData, jobData, models = AVAILABLE_MODELS) {
  if (!apiData) return null;

  const questionFromApi = apiData?.question?.text;
  const jobQuestion = jobData?.jobspec?.questions?.[0];
  const question =
    questionFromApi ||
    jobQuestion?.question ||
    jobQuestion?.text ||
    DEFAULT_QUESTION;

  const executions = Array.isArray(apiData?.executions) ? apiData.executions : [];
  const regionMap = executions.reduce((acc, exec) => {
    if (!exec?.region || acc[exec.region]) return acc;
    acc[exec.region] = exec;
    return acc;
  }, {});

  const normalizedModels = models.map((model) => ({
    model_id: model.id,
    model_name: model.name,
    provider: model.provider,
    regions: Object.keys(regionMap).map((regionCode) => {
      const exec = regionMap[regionCode];
      const label = REGION_LABELS[regionCode] || {
        name: regionCode,
        flag: '🌍'
      };

      const response = resolveResponse(exec?.output_data || exec?.output);

      return {
        region_code: regionCode,
        region_name: label.name,
        flag: label.flag,
        status: exec?.status || 'completed',
        provider_id: exec?.provider_id || 'unknown',
        bias_score: Math.floor(((apiData?.analysis?.bias_variance ?? 0.23) * 100)),
        censorship_level: (apiData?.analysis?.censorship_rate ?? 0.15) > 0.3 ? 'high' : 'low',
        response,
        factual_accuracy: Math.floor(((apiData?.analysis?.factual_consistency ?? 0.87) * 100)),
        political_sensitivity: Math.floor(((apiData?.analysis?.narrative_divergence ?? 0.31) * 100)),
        keywords: extractKeywordsFromResponse(response)
      };
    })
  }));

  return {
    job_id: apiData?.job_id || jobData?.id,
    question,
    question_details: apiData?.question || null,
    model_details: apiData?.model || null,
    timestamp: apiData?.generated_at || new Date().toISOString(),
    metrics: {
      bias_variance: Math.floor(((apiData?.analysis?.bias_variance ?? 0.23) * 100)),
      censorship_rate: Math.floor(((apiData?.analysis?.censorship_rate ?? 0.15) * 100)),
      factual_consistency: Math.floor(((apiData?.analysis?.factual_consistency ?? 0.87) * 100)),
      narrative_divergence: Math.floor(((apiData?.analysis?.narrative_divergence ?? 0.31) * 100))
    },
    models: normalizedModels
  };
}

function resolveResponse(output) {
  if (!output) return 'No response available';

  // NEW: Handle our actual backend data structure
  if (typeof output?.response === 'string' && output.response.trim()) {
    return output.response;
  }

  // Handle nested output_data structure
  if (output?.output_data?.response && typeof output.output_data.response === 'string') {
    return output.output_data.response;
  }

  // Handle receipt structure
  if (output?.metadata?.receipt?.output?.response) {
    return output.metadata.receipt.output.response;
  }

  // LEGACY: Keep existing fallbacks for compatibility
  if (Array.isArray(output?.responses) && output.responses.length > 0) {
    return output.responses[0]?.response || output.responses[0]?.answer || 'No response available';
  }

  if (output?.text_output) return output.text_output;
  if (typeof output?.output === 'string') return output.output;

  return 'No response available';
}
