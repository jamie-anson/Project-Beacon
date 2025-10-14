/**
 * Live Progress Data Transformation Helpers
 * 
 * Transforms flat execution array into question-centric hierarchy:
 * Question → Model → Region
 */

/**
 * Transform executions into question-centric data structure
 * 
 * @param {Object} activeJob - Job with executions array
 * @param {Array<string>} selectedRegions - Regions user selected (e.g., ['US', 'EU'])
 * @returns {Array<QuestionData>} - Transformed question-centric data
 */
export function transformExecutionsToQuestions(activeJob, selectedRegions) {
  if (!activeJob) return [];
  
  const jobSpec = activeJob.job || activeJob;
  const questions = jobSpec?.questions || [];
  // Models can be in jobSpec.models OR jobSpec.metadata.models (for cross-region jobs)
  const models = jobSpec?.models || jobSpec?.metadata?.models || [];
  const executions = activeJob?.executions || [];
  
  // Validation: Check for data integrity
  if (!Array.isArray(selectedRegions) || selectedRegions.length === 0) {
    console.error('[transformExecutionsToQuestions] Invalid selectedRegions:', selectedRegions);
    return [];
  }
  
  if (!Array.isArray(models) || models.length === 0) {
    console.warn('[transformExecutionsToQuestions] No models defined');
    return [];
  }
  
  // Debug logging to catch state reversion issues
  console.log('[transformExecutionsToQuestions]', {
    totalExecutions: executions.length,
    selectedRegions: selectedRegions,
    models: models.map(m => m.id || m),
    questions: questions,
    executionStatuses: executions.map(e => ({ 
      id: e.id, 
      question: e.question_id, 
      model: e.model_id, 
      region: e.region,
      status: e.status 
    }))
  });
  
  // Build question-centric structure
  return questions.map(questionId => {
    const questionExecs = executions.filter(e => e.question_id === questionId);
    
    // Debug: Log failed executions that might be missing
    const failedExecs = questionExecs.filter(e => e.status === 'failed');
    if (failedExecs.length > 0) {
      console.log(`[FAILED EXECUTIONS] Q:${questionId} has ${failedExecs.length} failed:`, 
        failedExecs.map(e => ({ id: e.id, model: e.model_id, region: e.region, status: e.status }))
      );
    }
    
    // Build model data for this question
    const modelData = models.map(model => {
      // Handle both object {id: "..."} and string "..." formats
      const modelId = typeof model === 'string' ? model : model.id;
      if (!modelId) {
        console.error('[transformExecutionsToQuestions] Invalid model:', model);
        return null;
      }
      
      const modelExecs = questionExecs.filter(e => e.model_id === modelId);
      
      // Debug: Log model executions, especially if any failed
      if (modelExecs.some(e => e.status === 'failed')) {
        console.log(`[MODEL EXECS WITH FAILURES] Q:${questionId} M:${modelId}:`, 
          modelExecs.map(e => ({ id: e.id, region: e.region, norm: normalizeRegion(e.region), status: e.status }))
        );
      }
      
      // Build region data for this model
      const regionData = selectedRegions.map(region => {
        // Find execution for this region with defensive checks
        const regionExec = modelExecs.find(e => {
          if (!e || !e.region) return false;
          const normalized = normalizeRegion(e.region);
          return normalized === region;
        });
        
        // Debug: Log when execution is not found
        if (!regionExec && modelExecs.length > 0) {
          console.warn(`[MISSING EXECUTION] Q:${questionId} M:${modelId} R:${region}`, {
            lookingFor: region,
            availableExecs: modelExecs.map(e => ({ 
              id: e.id, 
              region: e.region, 
              normalized: normalizeRegion(e.region),
              status: e.status,
              matches: normalizeRegion(e.region) === region
            }))
          });
        }
        
        return {
          region,
          execution: regionExec || null,
          status: regionExec?.status || 'pending'
        };
      });
      
      // Calculate model-level progress and status
      const modelProgress = calculateModelProgress(regionData);
      const modelStatus = calculateModelStatus(regionData);
      const modelDiffsEnabled = isModelComplete(regionData);
      
      return {
        modelId: modelId,
        modelName: (typeof model === 'object' ? model.name : null) || modelId,
        regions: regionData,
        progress: modelProgress,
        status: modelStatus,
        diffsEnabled: modelDiffsEnabled
      };
    }).filter(Boolean); // Remove any null entries from invalid models
    
    // Calculate question-level progress and status
    const questionProgress = calculateQuestionProgress(modelData, selectedRegions.length);
    const questionStatus = calculateQuestionStatus(modelData);
    const questionDiffsEnabled = isQuestionComplete(modelData);
    
    return {
      questionId,
      models: modelData,
      progress: questionProgress,
      status: questionStatus,
      diffsEnabled: questionDiffsEnabled
    };
  });
}

/**
 * Normalize region name to match selected regions format
 * Maps database region names to UI region codes
 */
function normalizeRegion(region) {
  const r = String(region || '').toLowerCase();
  if (r.includes('us') || r.includes('united') || r === 'us-east') return 'US';
  if (r.includes('eu') || r.includes('europe') || r === 'eu-west') return 'EU';
  if (r.includes('asia') || r.includes('apac') || r.includes('pacific') || r === 'asia-pacific') return 'ASIA';
  return region;
}

/**
 * Calculate progress for a model (completed regions / total regions)
 */
function calculateModelProgress(regionData) {
  const completed = regionData.filter(r => r.status === 'completed').length;
  const total = regionData.length;
  return total > 0 ? completed / total : 0;
}

/**
 * Calculate status for a model
 * Backend can return: 'completed', 'failed', 'cancelled', 'duplicate_skipped'
 * 
 * Priority:
 * 1. Failed: if ANY region failed/cancelled
 * 2. Complete: if ALL regions completed OR duplicate_skipped
 * 3. Pending: if any region still pending (no execution record)
 */
function calculateModelStatus(regionData) {
  const statuses = regionData.map(r => r.status);
  
  // Any failure/cancellation = failed
  if (statuses.some(s => s === 'failed' || s === 'cancelled')) {
    return 'failed';
  }
  
  // All completed or skipped = completed
  const completedOrSkipped = statuses.every(s => 
    s === 'completed' || s === 'duplicate_skipped'
  );
  if (completedOrSkipped) {
    return 'completed';
  }
  
  // Default: pending (waiting for execution records to be created)
  return 'pending';
}

/**
 * Check if model is complete (all regions completed)
 */
function isModelComplete(regionData) {
  return regionData.every(r => r.status === 'completed');
}

/**
 * Calculate progress for a question (completed executions / total expected)
 */
function calculateQuestionProgress(modelData, numRegions) {
  const totalExpected = modelData.length * numRegions;
  const completed = modelData.reduce((sum, model) => {
    return sum + model.regions.filter(r => r.status === 'completed').length;
  }, 0);
  
  return totalExpected > 0 ? completed / totalExpected : 0;
}

/**
 * Calculate status for a question
 * - Failed: if any model failed
 * - Complete: if all models completed
 * - Processing: if any model processing
 * - Pending: otherwise
 */
function calculateQuestionStatus(modelData) {
  const statuses = modelData.map(m => m.status);
  
  if (statuses.some(s => s === 'failed')) return 'failed';
  if (statuses.every(s => s === 'completed')) return 'completed';
  if (statuses.some(s => s === 'processing')) return 'processing';
  return 'pending';
}

/**
 * Check if question is complete (at least 2 models completed for bias detection)
 */
function isQuestionComplete(modelData) {
  const completedModels = modelData.filter(m => m.diffsEnabled);
  return completedModels.length >= 2;
}

/**
 * Get execution for specific question/model/region combination
 */
export function getExecution(executions, questionId, modelId, region) {
  return executions.find(e => 
    e.question_id === questionId &&
    e.model_id === modelId &&
    normalizeRegion(e.region) === region
  );
}

/**
 * Format progress as percentage string
 */
export function formatProgress(progress) {
  return `${Math.round(progress * 100)}%`;
}

/**
 * Get status color class (Catppuccin Mocha)
 */
export function getStatusColor(status) {
  switch (status?.toLowerCase()) {
    case 'completed':
    case 'complete':
      return 'bg-green-900/20 text-green-400 border-green-700';
    case 'processing':
    case 'running':
    case 'retrying':
      return 'bg-yellow-900/20 text-yellow-400 border-yellow-700';
    case 'failed':
      return 'bg-red-900/20 text-red-400 border-red-700';
    case 'cancelled':
      return 'bg-orange-900/20 text-orange-400 border-orange-700';
    case 'pending':
    default:
      return 'bg-gray-900/20 text-gray-400 border-gray-700';
  }
}

/**
 * Get status display text
 */
export function getStatusText(status) {
  switch (status?.toLowerCase()) {
    case 'completed':
    case 'complete':
      return 'Complete';
    case 'processing':
    case 'running':
      return 'Processing';
    case 'retrying':
      return 'Retrying';
    case 'failed':
      return 'Failed';
    case 'cancelled':
      return 'Cancelled';
    case 'pending':
    default:
      return 'Pending';
  }
}
