import React, { useState, useEffect, useMemo } from 'react';
import QuestionRow from './QuestionRow';
import { transformExecutionsToQuestions } from './liveProgressHelpers';

/**
 * LiveProgressTableV2 - Redesigned question-centric live progress display
 * 
 * Structure:
 * - Progress Summary (existing, kept from V1)
 * - Question List (new, question-centric)
 *   - Question → Model → Region hierarchy
 *   - View Diffs at question and model levels
 *   - Answer links at region level
 */
export default function LiveProgressTableV2({ 
  activeJob, 
  selectedRegions, 
  loadingActive, 
  refetchActive,
  activeJobId,
  isCompleted = false,
  diffReady = false,
}) {
  // State for countdown timer (triggers re-render every second)
  const [tick, setTick] = useState(0);
  
  // State to track when we first saw this job (for countdown)
  const [jobStartTime, setJobStartTime] = useState(null);
  
  // Extract primitive values for stable dependencies
  const jobStatusValue = activeJob?.status || '';
  const hasActiveJob = !!activeJob;
  const currentJobId = activeJob?.id;
  
  // Reset start time when job ID changes (start countdown from now)
  useEffect(() => {
    if (currentJobId && currentJobId !== jobStartTime?.jobId) {
      setJobStartTime({
        jobId: currentJobId,
        startTime: Date.now()
      });
    }
  }, [currentJobId, jobStartTime?.jobId]);
  
  // Update countdown timer every second when job is active
  useEffect(() => {
    const statusLower = String(jobStatusValue).toLowerCase();
    const isJobActive = !isCompleted && 
                        hasActiveJob && 
                        !['completed', 'failed', 'error', 'cancelled', 'timeout'].includes(statusLower);
    
    if (isJobActive) {
      const interval = setInterval(() => {
        setTick(t => t + 1);
      }, 1000);
      return () => clearInterval(interval);
    } else {
      // Reset tick when job becomes inactive to prevent stale state
      setTick(0);
    }
  }, [isCompleted, hasActiveJob, jobStatusValue]);
  
  // Transform executions into question-centric data (memoized to prevent unnecessary recalculations)
  const questionData = useMemo(() => {
    return transformExecutionsToQuestions(activeJob, selectedRegions);
  }, [activeJob?.executions, selectedRegions, activeJob?.job?.questions, activeJob?.job?.models]);
  
  // Calculate overall progress for summary
  const execs = activeJob?.executions || [];
  const jobSpec = activeJob?.job || activeJob;
  const specQuestions = jobSpec?.questions || [];
  // Models can be in jobSpec.models OR jobSpec.metadata.models (for cross-region jobs)
  const specModels = jobSpec?.models || jobSpec?.metadata?.models || [];
  
  // Calculate expected total: questions × models × selected regions
  const expectedTotal = specQuestions.length * specModels.length * selectedRegions.length;
  const completed = execs.filter(e => e?.status === 'completed').length;
  const running = execs.filter(e => e?.status === 'running' || e?.status === 'processing').length;
  const failed = execs.filter(e => e?.status === 'failed').length;
  const pending = Math.max(0, expectedTotal - completed - running - failed);
  
  const statusStr = String(activeJob?.status || activeJob?.state || '').toLowerCase();
  const jobCompleted = isCompleted || ['completed','success','succeeded','done','finished'].includes(statusStr);
  const jobFailed = ['failed', 'error', 'cancelled', 'timeout'].includes(statusStr);
  
  // Calculate time remaining (10 minute countdown from when we first saw the job)
  const calculateTimeRemaining = () => {
    if (jobCompleted || jobFailed) return null;
    if (!jobStartTime) return null;
    
    const estimatedDuration = 10 * 60; // 10 minutes in seconds
    const now = Date.now();
    const elapsedMs = now - jobStartTime.startTime;
    const elapsedSeconds = Math.floor(elapsedMs / 1000);
    const remainingSeconds = Math.max(0, estimatedDuration - elapsedSeconds);
    
    if (remainingSeconds <= 0) return null;
    
    const remainingMinutes = Math.floor(remainingSeconds / 60);
    const remainingSecsDisplay = remainingSeconds % 60;
    
    return `${remainingMinutes}:${remainingSecsDisplay.toString().padStart(2, '0')}`;
  };
  
  const _ = tick; // Force dependency on tick
  const timeRemaining = calculateTimeRemaining();
  
  return (
    <div className="p-4 space-y-4">
      {/* Progress Summary */}
      <div className="bg-gray-800 border border-gray-600 rounded-lg p-4">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-gray-300">
            {specQuestions.length} questions × {specModels.length} models × {selectedRegions.length} regions
          </span>
          <span className="text-sm text-gray-400">
            {timeRemaining ? `Time remaining: ~${timeRemaining}` : `${completed}/${expectedTotal} executions`}
          </span>
        </div>
        
        {/* Progress Bar */}
        <div className="w-full h-3 bg-gray-700 rounded overflow-hidden mb-2">
          <div className="h-full flex">
            <div 
              className="h-full bg-green-500" 
              style={{ width: `${(completed / expectedTotal) * 100}%` }}
            />
            <div 
              className="h-full bg-yellow-500" 
              style={{ width: `${(running / expectedTotal) * 100}%` }}
            />
            <div 
              className="h-full bg-red-500" 
              style={{ width: `${(failed / expectedTotal) * 100}%` }}
            />
          </div>
        </div>
        
        {/* Status Breakdown */}
        <div className="flex items-center gap-4 text-xs">
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 bg-green-500 rounded" />
            <span className="text-gray-300">Completed: {completed}</span>
          </div>
          <div className="flex items-center gap-1">
            <div className={`w-2 h-2 bg-yellow-500 rounded ${running > 0 ? 'animate-pulse' : ''}`} />
            <span className="text-gray-300">Running: {running}</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 bg-red-500 rounded" />
            <span className="text-gray-300">Failed: {failed}</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 bg-gray-500 rounded" />
            <span className="text-gray-300">Pending: {pending}</span>
          </div>
        </div>
      </div>
      
      {/* Question List */}
      <div className="space-y-4">
        {questionData.map(question => (
          <QuestionRow
            key={question.questionId}
            questionData={question}
            jobId={activeJobId}
            selectedRegions={selectedRegions}
          />
        ))}
      </div>
      
      {/* Action Buttons */}
      <div className="flex items-center justify-end gap-2">
        <button 
          onClick={refetchActive} 
          className="px-3 py-1.5 bg-green-600 text-white rounded text-sm hover:bg-green-700"
        >
          Refresh
        </button>
      </div>
    </div>
  );
}
