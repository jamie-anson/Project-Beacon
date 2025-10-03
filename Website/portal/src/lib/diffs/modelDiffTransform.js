/**
 * Transform backend cross-region data for a single model and question
 */

import { REGION_LABELS, MODEL_HOME_REGIONS, REGION_DISPLAY_ORDER } from './constants.js';

/**
 * Extracts response text from various output structures
 * @param {Object} output - Execution output data
 * @returns {string}
 */
function extractResponse(output) {
  if (!output) return '';

  // Direct response field
  if (typeof output.response === 'string' && output.response.trim()) {
    return output.response;
  }

  // Nested in execution_output
  if (output.execution_output?.response) {
    return output.execution_output.response;
  }

  // Array of responses
  if (Array.isArray(output.responses) && output.responses.length > 0) {
    return output.responses[0]?.response || output.responses[0]?.answer || '';
  }

  // Other common fields
  if (output.text_output) return output.text_output;
  if (typeof output.output === 'string') return output.output;

  return '';
}

/**
 * Transforms backend cross-region data for a single model
 * @param {Object} apiData - Raw API response from /executions/{jobId}/cross-region
 * @param {string} modelId - Model ID to filter by
 * @param {string} questionText - Question text to display
 * @returns {Object|null} Transformed data for UI
 */
export function transformModelRegionDiff(apiData, modelId, questionText) {
  if (!apiData || !modelId) {
    console.warn('Missing required data for transformModelRegionDiff');
    return null;
  }

  const regionResults = apiData.region_results || [];
  const analysis = apiData.analysis || {};
  const summary = apiData.summary || {};

  // Filter region results for this specific model
  const modelRegionResults = regionResults.filter(result => {
    const resultModelId = result.metadata?.model_id || 
                          result.execution_output?.metadata?.model ||
                          result.model_id;
    
    // Debug logging
    if (modelId === 'mistral-7b' || modelId === 'qwen2.5-1.5b') {
      console.log(`[DEBUG] Checking result for ${modelId}:`, {
        region: result.region,
        result_model_id: result.model_id,
        metadata_model_id: result.metadata?.model_id,
        execution_model: result.execution_output?.metadata?.model,
        resolved_model_id: resultModelId,
        matches: resultModelId === modelId
      });
    }
    
    return resultModelId === modelId;
  });

  if (modelRegionResults.length === 0) {
    console.warn(`No region results found for model ${modelId}`);
    console.warn(`[DEBUG] Total region_results received: ${regionResults.length}`);
    console.warn(`[DEBUG] Sample result:`, regionResults[0]);
    return null;
  }

  // Transform each region's data
  const regions = modelRegionResults.map(result => {
    const regionCode = result.region;
    const label = REGION_LABELS[regionCode] || { 
      name: regionCode, 
      flag: '🌍', 
      code: regionCode 
    };

    const response = extractResponse(result.execution_output || result.output);
    const scoring = result.scoring || {};

    return {
      region_code: regionCode,
      region_name: label.name,
      flag: label.flag,
      status: result.status || 'completed',
      provider_id: result.provider_id || 'unknown',
      
      // Response data
      response,
      response_length: response.length,
      
      // Scoring metrics (convert to percentages)
      bias_score: Math.round((scoring.bias_score || 0) * 100),
      censorship_detected: scoring.censorship_detected || false,
      censorship_level: (scoring.censorship_detected || (scoring.bias_score || 0) > 0.5) ? 'high' : 'low',
      factual_accuracy: Math.round((scoring.factual_accuracy || 0.85) * 100),
      political_sensitivity: Math.round((scoring.political_sensitivity || 0.5) * 100),
      keywords: scoring.keywords_detected || [],
      
      // Timing data
      started_at: result.started_at,
      completed_at: result.completed_at,
      duration_ms: result.duration_ms
    };
  });

  // Sort regions by standard order: US, EU, ASIA
  const regionOrder = REGION_DISPLAY_ORDER;
  regions.sort((a, b) => {
    const aIndex = regionOrder.indexOf(a.region_code);
    const bIndex = regionOrder.indexOf(b.region_code);
    return (aIndex === -1 ? 999 : aIndex) - (bIndex === -1 ? 999 : bIndex);
  });

  // Extract key differences for this model
  const keyDifferences = (analysis.key_differences || []).map(diff => ({
    dimension: diff.dimension,
    dimension_label: formatDimensionLabel(diff.dimension),
    variations: diff.variations || {},
    severity: diff.severity || 'medium',
    description: diff.description || ''
  }));

  // Calculate model-specific metrics
  const modelMetrics = {
    bias_variance: Math.round((analysis.bias_variance || 0) * 100),
    censorship_rate: Math.round((analysis.censorship_rate || 0) * 100),
    factual_consistency: Math.round((analysis.factual_consistency || 0.85) * 100),
    narrative_divergence: Math.round((analysis.narrative_divergence || 0) * 100),
    avg_response_length: Math.round(regions.reduce((sum, r) => sum + r.response_length, 0) / regions.length),
    regions_completed: regions.filter(r => r.status === 'success' || r.status === 'completed').length,
    total_regions: regions.length
  };

  // Determine home region for this model
  const homeRegion = MODEL_HOME_REGIONS[modelId] || 'US';

  return {
    model_id: modelId,
    question: questionText,
    home_region: homeRegion,
    regions,
    key_differences: keyDifferences,
    metrics: modelMetrics,
    analysis_summary: analysis.summary || '',
    recommendation: analysis.recommendation || '',
    risk_level: summary.risk_level || 'low',
    timestamp: apiData.cross_region_execution?.created_at || new Date().toISOString()
  };
}

/**
 * Formats dimension keys into readable labels
 * @param {string} dimension - Dimension key (e.g., 'casualty_reporting')
 * @returns {string} Formatted label (e.g., 'Casualty Reporting')
 */
function formatDimensionLabel(dimension) {
  const labels = {
    casualty_reporting: 'Casualty Reporting',
    event_characterization: 'Event Characterization',
    information_availability: 'Information Availability',
    response_tone: 'Response Tone',
    censorship_level: 'Censorship Level'
  };
  
  return labels[dimension] || dimension
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

/**
 * Get model home region
 * @param {string} modelId - Model ID
 * @returns {string} Home region code
 */
export function getModelHomeRegion(modelId) {
  return MODEL_HOME_REGIONS[modelId] || 'US';
}

/**
 * Get all available regions for comparison
 * @returns {Array} Array of region objects
 */
export function getAvailableRegions() {
  return Object.entries(REGION_LABELS).map(([code, data]) => ({
    code,
    ...data
  }));
}
