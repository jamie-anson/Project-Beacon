/**
 * useJobProgress Hook
 * Manages job progress state and calculations
 */

import { useState, useEffect, useMemo } from 'react';
import {
  calculateExpectedTotal,
  calculateProgress,
  calculateJobAge,
  isJobStuck,
  getUniqueModels,
  getUniqueQuestions
} from '../lib/utils/progressUtils';
import { getJobStage, getFailureMessage } from '../lib/utils/jobStatusUtils';

/**
 * Custom hook for tracking job progress
 * @param {Object} activeJob - The active job object
 * @param {Array} selectedRegions - Array of selected region codes
 * @param {boolean} isCompleted - Whether job is marked as completed
 * @returns {Object} Progress state and metrics
 */
export function useJobProgress(activeJob, selectedRegions = [], isCompleted = false) {
  const [jobStartTime, setJobStartTime] = useState(null);
  
  // Extract primitive values for stable dependencies
  const jobStatusValue = activeJob?.status || '';
  const currentJobId = activeJob?.id;
  
  // Reset start time when job ID changes
  useEffect(() => {
    if (currentJobId && currentJobId !== jobStartTime?.jobId) {
      setJobStartTime({
        jobId: currentJobId,
        startTime: Date.now()
      });
    }
  }, [currentJobId, jobStartTime?.jobId]);
  
  // Calculate all progress metrics
  const metrics = useMemo(() => {
    const execs = activeJob?.executions || [];
    const statusStr = String(activeJob?.status || activeJob?.state || '').toLowerCase();
    
    // Job completion states
    const jobCompleted = isCompleted || ['completed','success','succeeded','done','finished'].includes(statusStr);
    const jobFailed = ['failed', 'error', 'cancelled', 'timeout'].includes(statusStr);
    
    // Calculate job age and stuck status
    const jobAge = calculateJobAge(jobStartTime);
    const jobStuckTimeout = isJobStuck(jobAge, execs, jobCompleted, jobFailed);
    
    // Calculate expected total and progress
    const expectedTotal = calculateExpectedTotal(activeJob, selectedRegions);
    const total = expectedTotal > 0 ? expectedTotal : Math.max(execs.length, selectedRegions.length);
    const progress = calculateProgress(execs, total);
    
    // Handle special cases
    let { completed, running, failed, pending } = progress;
    
    if (jobCompleted && execs.length === 0) {
      // Job completed but no execution records
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
    
    // Recalculate pending
    pending = total - completed - running - failed;
    const percentage = Math.round((completed / Math.max(total, 1)) * 100);
    
    // Determine job stage
    const stage = getJobStage(activeJob, execs, jobCompleted, jobFailed, jobStuckTimeout);
    
    // Get failure info
    const failureInfo = getFailureMessage(activeJob, jobAge, jobFailed, jobStuckTimeout);
    
    // Get unique models and questions
    const uniqueModels = getUniqueModels(execs);
    const uniqueQuestions = getUniqueQuestions(execs);
    
    // Get spec data
    const jobSpec = activeJob?.job || activeJob;
    const specQuestions = jobSpec?.questions || [];
    const specModels = jobSpec?.models || [];
    const hasQuestions = specQuestions.length > 0 || uniqueQuestions.length > 0;
    const displayQuestions = specQuestions.length > 0 ? specQuestions : uniqueQuestions;
    
    // Overall completion states
    const overallCompleted = jobCompleted || (total > 0 && completed >= total);
    const overallFailed = jobFailed || jobStuckTimeout || (total > 0 && failed >= total);
    const showShimmer = !overallCompleted && !overallFailed && (running > 0 || stage === 'spawning');
    
    return {
      // Counts
      completed,
      running,
      failed,
      pending,
      total,
      percentage,
      
      // States
      jobCompleted,
      jobFailed,
      jobStuckTimeout,
      overallCompleted,
      overallFailed,
      showShimmer,
      
      // Stage and failure
      stage,
      failureInfo,
      jobAge,
      
      // Models and questions
      uniqueModels,
      uniqueQuestions,
      hasQuestions,
      displayQuestions,
      specQuestions,
      specModels,
      
      // Job data
      jobId: activeJob?.id || activeJob?.job?.id,
      jobSpec,
      executions: execs,
      statusStr
    };
  }, [activeJob, selectedRegions, isCompleted, jobStartTime]);
  
  return {
    ...metrics,
    jobStartTime
  };
}
