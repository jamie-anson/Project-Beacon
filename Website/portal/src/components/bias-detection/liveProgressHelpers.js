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
  const models = jobSpec?.models || [];
  const executions = activeJob?.executions || [];
  
  // Debug logging to catch state reversion issues
  console.log('[transformExecutionsToQuestions]', {
    totalExecutions: executions.length,
    executionStatuses: executions.map(e => ({ 
      id: e.id, 
      question: e.question_id, 
      model: e.model_id, 
      region: e.region, // ← CRITICAL: Need to see if region is present
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
      const modelExecs = questionExecs.filter(e => e.model_id === model.id);
      
      // Debug: Log model executions, especially if any failed
      if (modelExecs.some(e => e.status === 'failed')) {
        console.log(`[MODEL EXECS WITH FAILURES] Q:${questionId} M:${model.id}:`, 
          modelExecs.map(e => ({ id: e.id, region: e.region, norm: normalizeRegion(e.region), status: e.status }))
        );
      }
      
      // Build region data for this model
      const regionData = selectedRegions.map(region => {
        const regionExec = modelExecs.find(e => normalizeRegion(e.region) === region);
        
        // Debug: Log when execution is not found
        if (!regionExec) {
          console.warn(`[MISSING EXECUTION] Q:${questionId} M:${model.id} R:${region} - Not found in modelExecs:`, 
            modelExecs.map(e => ({ id: e.id, region: e.region, normalized: normalizeRegion(e.region) }))
          );
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
        modelId: model.id,
        modelName: model.name || model.id,
        regions: regionData,
        progress: modelProgress,
        status: modelStatus,
        diffsEnabled: modelDiffsEnabled
      };
    });
    
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
 * Check if question is complete (all models completed)
 */
function isQuestionComplete(modelData) {
  return modelData.every(m => m.diffsEnabled);
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
    case 'failed':
      return 'Failed';
    case 'cancelled':
      return 'Cancelled';
    case 'pending':
    default:
      return 'Pending';
  }
}
