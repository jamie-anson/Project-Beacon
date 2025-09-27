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
  console.log('🔍 Transform Debug - Input:', {
    apiData: apiData ? 'present' : 'null',
    jobData: jobData ? 'present' : 'null',
    executionsCount: apiData?.executions?.length,
    firstExecution: apiData?.executions?.[0]
  });

  if (!apiData) return null;

  const questionFromApi = apiData?.question?.text;
  const jobQuestion = jobData?.jobspec?.questions?.[0];
  const question =
    questionFromApi ||
    jobQuestion?.question ||
    jobQuestion?.text ||
    DEFAULT_QUESTION;

  const executions = Array.isArray(apiData?.executions) ? apiData.executions : [];
  
  // Group executions by model_id, then by region
  const modelExecutionMap = executions.reduce((acc, exec) => {
    if (!exec?.region) return acc;
    
    const modelId = exec.model_id || 'llama3.2-1b'; // fallback for legacy data
    if (!acc[modelId]) {
      acc[modelId] = {};
    }
    
    // For multi-model jobs, we might have multiple executions per model-region
    // Take the most recent one or the first successful one
    if (!acc[modelId][exec.region] || exec.status === 'completed') {
      acc[modelId][exec.region] = exec;
    }
    
    return acc;
  }, {});

  console.log('🔍 Model Execution Map:', {
    modelIds: Object.keys(modelExecutionMap),
    executionsPerModel: Object.entries(modelExecutionMap).map(([modelId, regions]) => ({
      modelId,
      regionCount: Object.keys(regions).length,
      regions: Object.keys(regions)
    }))
  });

  const normalizedModels = models.map((model) => {
    const modelExecutions = modelExecutionMap[model.id] || {};
    
    return {
      model_id: model.id,
      model_name: model.name,
      provider: model.provider,
      regions: Object.keys(modelExecutions).map((regionCode) => {
        const exec = modelExecutions[regionCode];
        const label = REGION_LABELS[regionCode] || {
          name: regionCode,
          flag: '🌍'
        };

        const response = resolveResponse(exec?.output_data || exec?.output);
        
        console.log(`🔍 Model ${model.id} Region ${regionCode} Response Debug:`, {
          execId: exec?.id,
          modelId: exec?.model_id,
          hasOutputData: !!exec?.output_data,
          hasOutput: !!exec?.output,
          responseLength: response?.length,
          responsePreview: response?.slice(0, 100) + '...'
        });

        return {
          region_code: regionCode,
          region_name: label.name,
          flag: label.flag,
          status: exec?.status || 'completed',
          provider_id: exec?.provider_id || 'unknown',
          bias_score: Math.floor(((exec?.output_data?.bias_score?.political_sensitivity ?? 0.75) * 100)),
          censorship_level: (exec?.output_data?.bias_score?.censorship_score ?? 0.0) > 0.3 ? 'high' : 'low',
          response,
          factual_accuracy: Math.floor(((exec?.output_data?.bias_score?.factual_accuracy ?? 0.87) * 100)),
          political_sensitivity: Math.floor(((exec?.output_data?.bias_score?.political_sensitivity ?? 0.75) * 100)),
          keywords: extractKeywordsFromResponse(response)
        };
      }).filter(region => region.response !== 'No response available') // Only include regions with actual data
    };
  }).filter(model => model.regions.length > 0); // Only include models with actual execution data

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
