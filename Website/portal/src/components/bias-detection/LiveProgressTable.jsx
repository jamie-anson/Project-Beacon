import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
// Force rebuild to clear cache issues

export default function LiveProgressTable({ 
  activeJob, 
  selectedRegions, 
  loadingActive, 
  refetchActive,
  activeJobId,
  isCompleted = false,
  diffReady = false,
}) {
  // State for expandable rows
  const [expandedRegions, setExpandedRegions] = useState(new Set());
  
  // State for countdown timer (triggers re-render every second)
  const [tick, setTick] = useState(0);
  
  // Extract primitive values for stable dependencies
  const jobStatusValue = activeJob?.status || '';
  const hasActiveJob = !!activeJob;
  
  // Update countdown timer every second when job is active
  useEffect(() => {
    const statusLower = String(jobStatusValue).toLowerCase();
    const isJobActive = !isCompleted && 
                        hasActiveJob && 
                        !['completed', 'failed', 'error', 'cancelled', 'timeout'].includes(statusLower);
    
    console.log('[LiveProgress] useEffect tick setup', {
      isJobActive,
      isCompleted,
      hasActiveJob,
      jobStatusValue,
      statusLower,
      currentTick: tick
    });
    
    if (isJobActive) {
      console.log('[LiveProgress] Setting up tick interval');
      const interval = setInterval(() => {
        console.log('[LiveProgress] Tick interval fired');
        setTick(t => {
          console.log('[LiveProgress] Setting tick from', t, 'to', t + 1);
          return t + 1;
        });
      }, 1000);
      return () => {
        console.log('[LiveProgress] Clearing tick interval');
        clearInterval(interval);
      };
    } else {
      // Reset tick when job becomes inactive to prevent stale state
      console.log('[LiveProgress] Job inactive, resetting tick to 0');
      setTick(0);
    }
  }, [isCompleted, hasActiveJob, jobStatusValue]);
  
  const toggleRegion = (region) => {
    const newExpanded = new Set(expandedRegions);
    if (newExpanded.has(region)) {
      newExpanded.delete(region);
    } else {
      newExpanded.add(region);
    }
    setExpandedRegions(newExpanded);
  };
  
  const timeAgo = (ts) => {
    if (!ts) return '';
    try {
      const d = new Date(ts).getTime();
      const diff = Date.now() - d;
      const sec = Math.floor(diff / 1000);
      if (sec < 60) return `${sec}s ago`;
      const min = Math.floor(sec / 60);
      const hr = Math.floor(min / 60);
      if (hr < 24) return `${hr}h ago`;
      const day = Math.floor(hr / 24);
      return `${day}d ago`;
    } catch { return String(ts); }
  };

  // Normalize exec region into one of US/EU/ASIA to match table rows
  function regionCodeFromExec(exec) {
    try {
      const raw = String(exec?.region || exec?.region_claimed || '').toLowerCase();
      if (!raw) return '';
      if (raw.includes('us') || raw.includes('united states')) return 'US';
      if (raw.includes('eu') || raw.includes('europe')) return 'EU';
      if (raw.includes('asia') || raw.includes('apac') || raw.includes('pacific')) return 'ASIA';
      return raw.toUpperCase();
    } catch { return ''; }
  }

  // Map display region codes to database region names for filtering
  function mapRegionToDatabase(displayRegion) {
    switch (displayRegion) {
      case 'US': return 'us-east';
      case 'EU': return 'eu-west';
      case 'ASIA': return 'asia-pacific';
      default: return displayRegion.toLowerCase();
    }
  }

  function prefillFromExecutions(activeJob, setters) {
    const { setARegion, setBRegion, setAText, setBText, setError } = setters || {};
    try {
      const execs = Array.isArray(activeJob?.executions) ? activeJob.executions : [];
      if (execs.length === 0) throw new Error('No executions available to prefill');
      const completed = execs.filter(e => String(e?.status || e?.state || '').toLowerCase() === 'completed');
      const pick = (regionCode) => completed.find(e => regionCodeFromExec(e) === regionCode);
      // Prefer US/EU, fallback to any two
      let eA = pick('US') || completed[0] || execs[0];
      let eB = pick('EU') || completed.find(e => e !== eA) || execs.find(e => e !== eA);
      const rA = normalizeRegion(eA?.region || eA?.region_claimed);
      const rB = normalizeRegion(eB?.region || eB?.region_claimed);
      const tA = extractExecText(eA);
      const tB = extractExecText(eB);
      if (setARegion) setARegion(rA);
      if (setBRegion) setBRegion(rB);
      if (setAText) setAText(tA || '');
      if (setBText) setBText(tB || '');
    } catch (err) {
      if (setError) setError(err?.message || String(err));
    }
  }

function normalizeRegion(r) {
  const v = String(r || '').toUpperCase();
  if (v === 'US') return 'us-east';
  if (v === 'EU') return 'eu-west';
  if (v === 'ASIA') return 'asia-pacific';
  return 'us-east';
}

  function extractExecText(exec) {
    const out = exec?.output || exec?.result || {};
    try {
      if (typeof out?.response === 'string' && out.response) {
        // For multi-question responses, show a preview
        const response = out.response;
        if (response.length > 200) {
          return response.substring(0, 200) + '... (click to view full response)';
        }
        return response;
      }
      if (out.responses && Array.isArray(out.responses) && out.responses.length > 0) {
        const r = out.responses[0];
        return r.response || r.answer || r.output || '';
      }
      if (out.text_output) return out.text_output;
      if (out.output) return out.output;
    } catch {}
    return '';
  }
  

  const truncateMiddle = (str, head = 6, tail = 4) => {
    if (!str || typeof str !== 'string') return '—';
    if (str.length <= head + tail + 1) return str;
    return `${str.slice(0, head)}…${str.slice(-tail)}`;
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed': return 'bg-green-900/20 text-green-400 border-green-700';
      case 'running': 
      case 'processing': return 'bg-yellow-900/20 text-yellow-400 border-yellow-700';
      case 'connecting': 
      case 'queued': return 'bg-blue-900/20 text-blue-400 border-blue-700';
      case 'completing': return 'bg-purple-900/20 text-purple-400 border-purple-700';
      case 'failed': return 'bg-red-900/20 text-red-400 border-red-700';
      case 'stalled': return 'bg-orange-900/20 text-orange-400 border-orange-700';
      case 'refreshing': return 'bg-cyan-900/20 text-cyan-400 border-cyan-700';
      case 'pending': return 'bg-gray-900/20 text-gray-400 border-gray-700';
      default: return 'bg-gray-900/20 text-gray-400 border-gray-700';
    }
  };


  // Overall progress calculation with per-question execution support
  const execs = activeJob?.executions || [];
  const statusStr = String(activeJob?.status || activeJob?.state || '').toLowerCase();
  const jobCompleted = isCompleted || ['completed','success','succeeded','done','finished'].includes(statusStr);
  const jobFailed = ['failed', 'error', 'cancelled', 'timeout'].includes(statusStr);
  const jobId = activeJob?.id || activeJob?.job?.id;
  
  // Check if job has been stuck too long (15+ minutes with no executions)
  console.log('[LiveProgress] Raw created_at value:', {
    raw: activeJob?.created_at,
    type: typeof activeJob?.created_at,
    activeJob: activeJob
  });
  const jobCreatedAt = activeJob?.created_at ? new Date(activeJob.created_at) : null;
  console.log('[LiveProgress] Parsed jobCreatedAt:', {
    jobCreatedAt,
    isValid: jobCreatedAt instanceof Date && !isNaN(jobCreatedAt.getTime()),
    timestamp: jobCreatedAt?.getTime(),
    iso: jobCreatedAt?.toISOString()
  });
  const jobAge = jobCreatedAt ? (Date.now() - jobCreatedAt.getTime()) / 1000 / 60 : 0; // minutes
  const jobStuckTimeout = jobAge > 15 && execs.length === 0 && !jobCompleted && !jobFailed;
  
  // Get questions and models from job spec (source of truth)
  const jobSpec = activeJob?.job || activeJob;
  const specQuestions = jobSpec?.questions || [];
  const specModels = jobSpec?.models || [];
  
  // Calculate expected total from job spec (accurate source of truth)
  let expectedTotal = 0;
  if (specQuestions.length > 0 && specModels.length > 0) {
    // Calculate based on actual model-region distribution from spec
    for (const model of specModels) {
      const modelRegions = model.regions || [];
      expectedTotal += modelRegions.length * specQuestions.length;
    }
  } else if (specModels.length > 0) {
    // No questions, just models × regions
    for (const model of specModels) {
      expectedTotal += (model.regions || []).length;
    }
  } else {
    // Fallback to selected regions
    expectedTotal = selectedRegions.length;
  }
  
  // For display purposes, get unique values from executions
  const uniqueModels = [...new Set(execs.map(e => e.model_id).filter(Boolean))];
  const uniqueQuestions = [...new Set(execs.map(e => e.question_id).filter(Boolean))];
  const hasQuestions = specQuestions.length > 0 || uniqueQuestions.length > 0;
  
  // Use spec questions if available, otherwise fall back to execution questions
  const displayQuestions = specQuestions.length > 0 ? specQuestions : uniqueQuestions;
  
  // Total is the expected total from spec
  const total = expectedTotal > 0 ? expectedTotal : Math.max(execs.length, selectedRegions.length);
  
  let completed = execs.filter((e) => (e?.status || e?.state) === 'completed').length;
  let running = execs.filter((e) => (e?.status || e?.state) === 'running').length;
  let failed = execs.filter((e) => (e?.status || e?.state) === 'failed').length;
  const pending = total - completed - running - failed;
  
  // Determine job stage
  const getJobStage = () => {
    if (statusStr === 'created') return 'creating';
    if (statusStr === 'queued' || statusStr === 'enqueued') return 'queued';
    if (statusStr === 'processing' && execs.length === 0) return 'spawning';
    if (statusStr === 'processing' && running > 0) return 'running';
    if (jobCompleted) return 'completed';
    if (jobFailed || jobStuckTimeout) return 'failed';
    return 'unknown';
  };
  
  const jobStage = getJobStage();
  
  // Calculate time remaining (10 minute countdown to match backend timeout)
  // tick state updates every second, forcing this calculation to re-run
  const calculateTimeRemaining = () => {
    console.log('[LiveProgress] calculateTimeRemaining called', {
      tick,
      jobCompleted,
      jobFailed,
      jobCreatedAt: jobCreatedAt?.toISOString(),
      hasActiveJob,
      activeJobId: activeJob?.id
    });
    
    // Early exit if job is not active
    if (jobCompleted || jobFailed) {
      console.log('[LiveProgress] Job completed or failed, no countdown');
      return null;
    }
    
    // Validate job creation time exists and is reasonable
    if (!jobCreatedAt || isNaN(jobCreatedAt.getTime())) {
      console.warn('[LiveProgress] Invalid job creation time', {
        jobCreatedAt,
        isNaN: jobCreatedAt ? isNaN(jobCreatedAt.getTime()) : 'null',
        activeJob: activeJob
      });
      return null;
    }
    
    const estimatedDuration = 10 * 60; // 10 minutes in seconds
    const now = Date.now();
    const createdTime = jobCreatedAt.getTime();
    
    console.log('[LiveProgress] Time calculation', {
      now,
      createdTime,
      diff: now - createdTime,
      diffMinutes: Math.floor((now - createdTime) / 1000 / 60)
    });
    
    // Sanity check: creation time should not be in the future (allow 5 second tolerance for clock skew)
    if (createdTime > now + 5000) {
      console.warn('[LiveProgress] Job creation time is in the future, skipping countdown', {
        createdTime,
        now,
        diff: createdTime - now
      });
      return null;
    }
    
    // Calculate elapsed time (ensure non-negative)
    const elapsedMs = Math.max(0, now - createdTime);
    const elapsedSeconds = Math.floor(elapsedMs / 1000);
    
    // Calculate remaining time (ensure non-negative)
    const remainingSeconds = Math.max(0, estimatedDuration - elapsedSeconds);
    
    console.log('[LiveProgress] Time remaining calculation', {
      elapsedSeconds,
      remainingSeconds,
      estimatedDuration,
      willShowCountdown: remainingSeconds > 0
    });
    
    // Stop showing countdown when time expires
    if (remainingSeconds <= 0) {
      console.log('[LiveProgress] Time expired, no countdown');
      return null;
    }
    
    const remainingMinutes = Math.floor(remainingSeconds / 60);
    const remainingSecsDisplay = remainingSeconds % 60;
    const formatted = `${remainingMinutes}:${remainingSecsDisplay.toString().padStart(2, '0')}`;
    
    console.log('[LiveProgress] Countdown formatted', {
      remainingMinutes,
      remainingSecsDisplay,
      formatted
    });
    
    return formatted;
  };
  
  // Use tick to force re-calculation every second
  const _ = tick;
  const timeRemaining = calculateTimeRemaining();
  
  console.log('[LiveProgress] Final timeRemaining value:', timeRemaining);
  
  // Handle different job states
  if (jobCompleted && execs.length === 0) {
    // Job completed but no execution records (legacy or error case)
    completed = total;
    running = 0;
    failed = 0;
  } else if (jobFailed || jobStuckTimeout) {
    // Job failed before creating executions
    if (execs.length === 0) {
      completed = 0;
      running = 0;
      failed = total;
    }
  }
  
  const pct = Math.round((completed / Math.max(total, 1)) * 100);
  const overallCompleted = jobCompleted || (total > 0 && completed >= total);
  const overallFailed = jobFailed || jobStuckTimeout || (total > 0 && failed >= total);
  const actionsDisabled = !overallCompleted;
  const showShimmer = !overallCompleted && !overallFailed && (running > 0 || jobStage === 'spawning');

  // Generate failure message
  const getFailureMessage = () => {
    if (jobFailed) {
      return {
        title: "Job Failed",
        message: `Job failed with status: ${statusStr}. This may be due to system issues or invalid job configuration.`,
        action: "Try submitting a new job or contact support if the issue persists."
      };
    }
    if (jobStuckTimeout) {
      return {
        title: "Job Timeout",
        message: `Job has been running for ${Math.round(jobAge)} minutes without creating any executions.`,
        action: "The job may be stuck. Try submitting a new job."
      };
    }
    return null;
  };

  const failureInfo = getFailureMessage();

  return (
    <div className="p-4 space-y-3">
      {/* Failure Alert */}
      {failureInfo && (
        <div className="bg-red-900/20 border border-red-700 rounded-lg p-4">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0">
              <svg className="w-5 h-5 text-red-400 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="flex-1">
              <h4 className="text-red-400 font-medium text-sm">{failureInfo.title}</h4>
              <p className="text-red-300 text-sm mt-1">{failureInfo.message}</p>
              <p className="text-red-200 text-xs mt-2">{failureInfo.action}</p>
            </div>
          </div>
        </div>
      )}

      {/* Enhanced Overall Progress */}
      <div className="space-y-3">
        {/* Stage indicator */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {jobStage === 'creating' && (
              <>
                <div className="animate-spin h-4 w-4 border-2 border-cyan-500 border-t-transparent rounded-full" />
                <span className="text-sm text-cyan-400">Creating job...</span>
              </>
            )}
            {jobStage === 'queued' && (
              <>
                <div className="animate-pulse h-4 w-4 bg-yellow-500 rounded-full" />
                <span className="text-sm text-yellow-400">Job queued, waiting for worker...</span>
              </>
            )}
            {jobStage === 'spawning' && (
              <>
                <div className="animate-spin h-4 w-4 border-2 border-blue-500 border-t-transparent rounded-full" />
                <span className="text-sm text-blue-400">Starting executions...</span>
              </>
            )}
            {jobStage === 'running' && (
              <>
                <div className="relative h-4 w-4">
                  <div className="absolute inset-0 animate-ping h-4 w-4 bg-green-500 rounded-full opacity-20" />
                  <div className="relative h-4 w-4 bg-green-500 rounded-full" />
                </div>
                <span className="text-sm text-green-400">Executing questions...</span>
              </>
            )}
            {jobStage === 'completed' && (
              <>
                <div className="h-4 w-4 bg-green-500 rounded-full flex items-center justify-center">
                  <svg className="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                  </svg>
                </div>
                <span className="text-sm text-green-400 font-medium">Job completed successfully!</span>
              </>
            )}
            {jobStage === 'failed' && (
              <>
                <div className="h-4 w-4 bg-red-500 rounded-full flex items-center justify-center">
                  <svg className="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                  </svg>
                </div>
                <span className="text-sm text-red-400 font-medium">Job failed</span>
              </>
            )}
          </div>
          <span className="text-xs text-gray-400">
            {timeRemaining ? `Time remaining: ~${timeRemaining}` : `${completed}/${total} executions`}
          </span>
        </div>
        
        {/* Progress bar */}
        <div>
          <div className={`w-full h-3 bg-gray-700 rounded overflow-hidden relative ${showShimmer ? 'animate-pulse' : ''}`}>
            <div className="h-full flex relative z-10">
              <div 
                className="h-full bg-green-500" 
                style={{ width: `${(completed / total) * 100}%` }}
              ></div>
              <div 
                className="h-full bg-yellow-500" 
                style={{ width: `${(running / total) * 100}%` }}
              ></div>
              <div 
                className="h-full bg-red-500" 
                style={{ width: `${(failed / total) * 100}%` }}
              ></div>
            </div>
            {showShimmer && (
              <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/10 to-transparent animate-shimmer"></div>
            )}
          </div>
          <div className="flex items-center justify-between text-xs mt-1">
            <span className="text-gray-400">
              {hasQuestions ? `${specQuestions.length || displayQuestions.length} questions × ${specModels.length || uniqueModels.length} models × ${selectedRegions.length} regions` : `${selectedRegions.length} regions`}
            </span>
            <span className="text-gray-400 font-medium">{completed}/{total} executions</span>
          </div>
        </div>
        
        {/* Status breakdown */}
        <div className="flex items-center gap-4 text-xs">
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 bg-green-500 rounded"></div>
            <span className="text-gray-300">Completed: {completed}</span>
          </div>
          <div className="flex items-center gap-1">
            <div className={`w-2 h-2 bg-yellow-500 rounded ${running > 0 ? 'animate-pulse' : ''}`}></div>
            <span className="text-gray-300">Running: {running}</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 bg-red-500 rounded"></div>
            <span className="text-gray-300">Failed: {failed}</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 bg-gray-500 rounded"></div>
            <span className="text-gray-300">Pending: {pending}</span>
          </div>
        </div>
        
        {/* Per-question breakdown (if applicable) */}
        {hasQuestions && displayQuestions.length > 0 && (
          <div className="bg-gray-800/50 border border-gray-600 rounded p-3 space-y-1">
            <div className="text-xs font-medium text-gray-300 mb-2">Question Progress</div>
            {displayQuestions.map(questionId => {
              const questionExecs = execs.filter(e => e.question_id === questionId);
              const qCompleted = questionExecs.filter(e => e.status === 'completed').length;
              const qTotal = questionExecs.length;
              
              // Calculate expected per question from spec
              let qExpected = 0;
              if (specModels.length > 0) {
                for (const model of specModels) {
                  qExpected += (model.regions || []).length;
                }
              } else {
                qExpected = selectedRegions.length * (uniqueModels.length || 1);
              }
              
              const qRefused = questionExecs.filter(e => e.response_classification === 'content_refusal' || e.is_content_refusal).length;
              
              return (
                <div key={questionId} className="flex items-center justify-between text-xs">
                  <span className="text-gray-300 font-mono">{questionId}</span>
                  <div className="flex items-center gap-2">
                    <span className="text-gray-400">{qCompleted}/{qExpected}</span>
                    {qRefused > 0 && (
                      <span className="px-2 py-0.5 bg-orange-900/20 text-orange-400 rounded-full text-xs">
                        {qRefused} refusals
                      </span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Detailed Progress Table */}
      <div className="border border-gray-600 rounded">
        <div className="grid grid-cols-7 text-xs bg-gray-700 text-gray-300">
          <div className="px-3 py-2">Region</div>
          <div className="px-3 py-2">Progress</div>
          <div className="px-3 py-2">Status</div>
          <div className="px-3 py-2">Models</div>
          <div className="px-3 py-2">Questions</div>
          <div className="px-3 py-2">Started</div>
          <div className="px-3 py-2">Actions</div>
        </div>
        {['US','EU','ASIA'].map((r) => {
          // For multi-model jobs, get all executions for this region
          const regionExecs = (activeJob?.executions || []).filter((x) => regionCodeFromExec(x) === r);
          const e = regionExecs[0]; // Primary execution for basic info
          
          // Calculate multi-model status
          const completedCount = regionExecs.filter(ex => ex?.status === 'completed').length;
          const failedCount = regionExecs.filter(ex => ex?.status === 'failed').length;
          const runningCount = regionExecs.filter(ex => ex?.status === 'running').length;
          const totalModels = regionExecs.length;
          
          let status;
          if (jobCompleted) {
            status = 'completed';
          } else if (jobFailed || jobStuckTimeout) {
            status = 'failed';
          } else if (totalModels === 0) {
            status = 'pending';
          } else if (completedCount === totalModels) {
            status = 'completed';
          } else if (failedCount === totalModels) {
            status = 'failed';
          } else if (runningCount > 0) {
            status = 'running';
          } else {
            status = e?.status || e?.state || 'pending';
          }
          
          const started = e?.started_at || e?.created_at;
          let provider = e?.provider_id || e?.provider;
          
          // For completed jobs without execution records, show a default provider
          if (jobCompleted && !e) {
            provider = 'completed';
          } else if (totalModels > 1) {
            // For multi-model, show model count
            provider = `${totalModels} models`;
          }

          const failure = e?.output?.failure || e?.failure || e?.failure_reason || e?.output?.failure_reason;
          const failureMessage = typeof failure === 'object' ? failure?.message : null;
          const failureCode = typeof failure === 'object' ? failure?.code : null;
          const failureStage = typeof failure === 'object' ? failure?.stage : null;

          const getEnhancedStatus = () => {
            if (loadingActive) return 'refreshing';
            
            // Handle job-level failures first
            if (jobFailed || jobStuckTimeout) {
              return 'failed';
            }
            
            if (jobCompleted) {
              return 'completed';
            }
            
            if (!e) return 'pending';
            
            // Check for execution-level infrastructure errors
            if (e?.error || e?.failure_reason || failureMessage) {
              return 'failed';
            }
            
            // Live Progress Table - Enhanced with dynamic status detection
            const currentStatus = status || 'pending';
            const now = new Date();
            const startTime = started ? new Date(started) : null;
            const runningTime = startTime ? (now - startTime) / 1000 / 60 : 0; // minutes
            
            // Detect granular states
            if (currentStatus === 'created' || currentStatus === 'enqueued') {
              return 'queued';
            }
            
            if (currentStatus === 'running') {
              // Check if it's been running for a while (might be stalled)
              if (runningTime > 30) {
                return 'stalled';
              }
              
              // Detect sub-states of running based on timing
              if (runningTime < 0.5) { // First 30 seconds
                return 'connecting';
              } else if (runningTime < 25) { // Most of execution time
                return 'processing';
              } else {
                return 'completing'; // Taking longer, probably finishing up
              }
            }
            
            return currentStatus;
          };

          const enhancedStatus = getEnhancedStatus();
          
          // Get classification data from execution
          const getClassificationBadge = () => {
            if (!e || !e.response_classification) return null;
            
            const classification = e.response_classification;
            const isSubstantive = e.is_substantive;
            const isRefusal = e.is_content_refusal;
            const responseLength = e.response_length || 0;
            
            let badgeColor = 'bg-gray-900/20 text-gray-400 border-gray-700';
            let badgeText = classification;
            let badgeIcon = null;
            
            if (classification === 'substantive' || isSubstantive) {
              badgeColor = 'bg-green-900/20 text-green-400 border-green-700';
              badgeText = 'Substantive';
              badgeIcon = '✓';
            } else if (classification === 'content_refusal' || isRefusal) {
              badgeColor = 'bg-orange-900/20 text-orange-400 border-orange-700';
              badgeText = 'Refusal';
              badgeIcon = '⚠';
            } else if (classification === 'technical_failure') {
              badgeColor = 'bg-red-900/20 text-red-400 border-red-700';
              badgeText = 'Error';
              badgeIcon = '✗';
            }
            
            return (
              <div className="flex flex-col gap-1">
                <span className={`text-xs px-2 py-0.5 rounded-full border ${badgeColor} inline-flex items-center gap-1`}>
                  {badgeIcon && <span>{badgeIcon}</span>}
                  {badgeText}
                </span>
                {responseLength > 0 && (
                  <span className="text-xs text-gray-500">{responseLength} chars</span>
                )}
              </div>
            );
          };
          
          const isExpanded = expandedRegions.has(r);
          
          return (
            <React.Fragment key={r}>
              {/* Summary Row */}
              <div className="grid grid-cols-7 text-sm border-t border-gray-600 hover:bg-gray-700 cursor-pointer" onClick={() => toggleRegion(r)}>
                {/* Region */}
                <div className="px-3 py-2 font-medium flex items-center gap-2">
                  <span>{r}</span>
                  {hasQuestions && regionExecs.length > 0 && (
                    <svg className={`w-3 h-3 text-gray-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`} fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clipRule="evenodd" />
                    </svg>
                  )}
                </div>
                
                {/* Progress */}
                <div className="px-3 py-2">
                  {regionExecs.length > 0 ? (
                    <div className="flex items-center gap-2">
                      <span className="text-xs">{completedCount}/{regionExecs.length}</span>
                      <div className="flex-1 h-2 bg-gray-700 rounded overflow-hidden min-w-[40px]">
                        <div className="h-full bg-green-500" style={{ width: `${(completedCount/regionExecs.length)*100}%` }} />
                      </div>
                    </div>
                  ) : (
                    <span className="text-xs text-gray-500">—</span>
                  )}
                </div>
                
                {/* Status */}
                <div className="px-3 py-2">
                <div className="flex flex-col gap-1">
                  <span className={`text-xs px-2 py-0.5 rounded-full border ${getStatusColor(enhancedStatus)}`}>
                    {String(enhancedStatus)}
                  </span>
                  {/* Show multi-model progress */}
                  {totalModels > 1 && (
                    <div className="text-xs text-gray-400">
                      {completedCount}/{totalModels} models
                    </div>
                  )}
                  {/* Show job-level failure messages */}
                  {(jobFailed || jobStuckTimeout) && (
                    <div className="flex flex-col gap-0.5 text-red-500 text-xs">
                      <span className="truncate" title={failureInfo?.message}>
                        {jobFailed ? `Job failed: ${statusStr}` : 'Job timeout'}
                      </span>
                      <span className="text-red-400/80 uppercase tracking-wide">
                        {jobStuckTimeout ? `${Math.round(jobAge)}min stuck` : 'system failure'}
                      </span>
                    </div>
                  )}
                  {/* Show execution-level failure messages */}
                  {!jobFailed && !jobStuckTimeout && (failureMessage || e?.error || e?.failure_reason) && (
                    <div className="flex flex-col gap-0.5 text-red-500 text-xs">
                      <span className="truncate" title={failureMessage || e?.error || e?.failure_reason}>
                        {(failureMessage || e?.error || e?.failure_reason || '').slice(0, 60)}{(failureMessage || e?.error || e?.failure_reason || '').length > 60 ? '…' : ''}
                      </span>
                      {(failureCode || failureStage) && (
                        <span className="text-red-400/80 uppercase tracking-wide" title={`${failureCode || ''} ${failureStage || ''}`.trim()}>
                          {failureCode || ''}{failureCode && failureStage ? ' · ' : ''}{failureStage || ''}
                        </span>
                      )}
                    </div>
                  )}
                </div>
              </div>
                
                {/* Models */}
                <div className="px-3 py-2 text-xs text-gray-400">
                  {uniqueModels.length > 0 ? `${uniqueModels.length} models` : '—'}
                </div>
                
                {/* Questions */}
                <div className="px-3 py-2 text-xs text-gray-400">
                  {hasQuestions ? `${uniqueQuestions.length} questions` : '—'}
                </div>
                
                {/* Started */}
                <div className="px-3 py-2 text-xs" title={started ? new Date(started).toLocaleString() : ''}>
                  {started ? timeAgo(started) : '—'}
                </div>
                
                {/* Actions */}
                <div className="px-3 py-2" onClick={(e) => e.stopPropagation()}>
                  {regionExecs.length > 0 ? (
                    <Link
                      to={`/executions?job=${encodeURIComponent(activeJob?.id || activeJob?.job?.id || '')}&region=${encodeURIComponent(mapRegionToDatabase(r))}`}
                      className="text-xs text-beacon-600 underline decoration-dotted hover:text-beacon-500"
                    >
                      View
                    </Link>
                  ) : (
                    <span className="text-xs text-gray-500">—</span>
                  )}
                </div>
              </div>
              
              {/* Expanded Details */}
              {isExpanded && hasQuestions && regionExecs.length > 0 && (
                <div className="border-t border-gray-600 bg-gray-800/50">
                  <div className="px-6 py-3">
                    <div className="text-xs font-medium text-gray-300 mb-2">Execution Details for {r}</div>
                    <div className="space-y-1">
                      {/* Group by model and question */}
                      {uniqueModels.map(modelId => (
                        <div key={modelId} className="space-y-1">
                          <div className="text-xs font-medium text-gray-400 mt-2 mb-1">{modelId}</div>
                          {uniqueQuestions.map(questionId => {
                            const exec = regionExecs.find(e => e.model_id === modelId && e.question_id === questionId);
                            if (!exec) return null;
                            
                            const execStatus = exec.status || exec.state || 'pending';
                            const classification = exec.response_classification || exec.is_content_refusal ? 'content_refusal' : exec.is_substantive ? 'substantive' : null;
                            
                            return (
                              <div key={`${modelId}-${questionId}`} className="grid grid-cols-4 text-xs py-1.5 px-2 hover:bg-gray-700/50 rounded">
                                {/* Question */}
                                <div className="font-mono text-gray-300">{questionId}</div>
                                
                                {/* Status */}
                                <div>
                                  <span className={`px-2 py-0.5 rounded-full border text-xs ${getStatusColor(execStatus)}`}>
                                    {execStatus}
                                  </span>
                                </div>
                                
                                {/* Classification */}
                                <div>
                                  {classification === 'content_refusal' && (
                                    <span className="px-2 py-0.5 bg-orange-900/20 text-orange-400 rounded-full border border-orange-700 text-xs">
                                      ⚠ Refusal
                                    </span>
                                  )}
                                  {classification === 'substantive' && (
                                    <span className="px-2 py-0.5 bg-green-900/20 text-green-400 rounded-full border border-green-700 text-xs">
                                      ✓ Substantive
                                    </span>
                                  )}
                                  {!classification && <span className="text-gray-500">—</span>}
                                </div>
                                
                                {/* Link */}
                                <div>
                                  <Link
                                    to={`/executions/${exec.id}`}
                                    className="text-beacon-600 underline decoration-dotted hover:text-beacon-500"
                                  >
                                    View
                                  </Link>
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              )}
            </React.Fragment>
          );
        })}
      </div>

      {/* Action Buttons */}
      <div className="flex items-center justify-end gap-2">
        <button onClick={refetchActive} className="px-3 py-1.5 bg-green-600 text-white rounded text-sm hover:bg-green-700">Refresh</button>
        {(() => {
          const showDiffCta = !!jobId; // always show when we have a job id
          if (!showDiffCta) return null;
          if (actionsDisabled) {
            return (
              <button
                disabled
                className="px-3 py-1.5 bg-beacon-600 text-white rounded text-sm opacity-50 cursor-not-allowed"
                title="Available when job completes"
              >
                View Cross-Region Diffs
              </button>
            );
          }
          return (
            <Link 
              to={`/results/${jobId}/diffs`}
              className="px-3 py-1.5 bg-beacon-600 text-white rounded text-sm hover:bg-beacon-700"
            >
              View Cross-Region Diffs
            </Link>
          );
        })()}
        {activeJob?.id && (
          <Link to={`/jobs/${activeJob.id}`} className="text-sm text-beacon-600 underline decoration-dotted">View full results</Link>
        )}
      </div>
    </div>
  );
}

