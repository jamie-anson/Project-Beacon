/**
 * LiveProgressTable Component (Refactored)
 * Orchestrates job progress display using modular hooks and components
 */

import React from 'react';
import { useJobProgress } from '../../hooks/useJobProgress';
import { useRetryExecution } from '../../hooks/useRetryExecution';
import { useRegionExpansion } from '../../hooks/useRegionExpansion';
import { useCountdownTimer } from '../../hooks/useCountdownTimer';
import { filterVisibleRegions, regionCodeFromExec, mapRegionToDatabase } from '../../lib/utils/regionUtils';
import FailureAlert from './progress/FailureAlert';
import ProgressHeader from './progress/ProgressHeader';
import ProgressBreakdown from './progress/ProgressBreakdown';
import RegionRow from './progress/RegionRow';
import ExecutionDetails from './progress/ExecutionDetails';
import ProgressActions from './progress/ProgressActions';

export default function LiveProgressTable({ 
  activeJob, 
  selectedRegions, 
  loadingActive, 
  refetchActive,
  activeJobId,
  isCompleted = false,
  diffReady = false,
}) {
  // Custom hooks for state management
  const progress = useJobProgress(activeJob, selectedRegions, isCompleted);
  const { handleRetryQuestion, isRetrying } = useRetryExecution(refetchActive);
  const { toggleRegion, isExpanded } = useRegionExpansion();
  const { timeRemaining } = useCountdownTimer(
    !progress.overallCompleted && !progress.overallFailed,
    progress.jobCompleted,
    progress.jobFailed,
    progress.jobStartTime
  );
  
  // Filter visible regions
  const visibleRegions = filterVisibleRegions(
    ['US', 'EU'], // ASIA temporarily disabled
    progress.executions,
    selectedRegions
  );
  
  return (
    <div className="p-4 space-y-3">
      {/* Failure Alert */}
      <FailureAlert failureInfo={progress.failureInfo} />

      {/* Progress Header */}
      <ProgressHeader
        stage={progress.stage}
        timeRemaining={timeRemaining}
        completed={progress.completed}
        running={progress.running}
        failed={progress.failed}
        pending={progress.pending}
        total={progress.total}
        percentage={progress.percentage}
        showShimmer={progress.showShimmer}
        overallCompleted={progress.overallCompleted}
        overallFailed={progress.overallFailed}
        hasQuestions={progress.hasQuestions}
        specQuestions={progress.specQuestions}
        displayQuestions={progress.displayQuestions}
        specModels={progress.specModels}
        uniqueModels={progress.uniqueModels}
        selectedRegions={selectedRegions}
      />
      
      {/* Progress Breakdown */}
      <ProgressBreakdown
        completed={progress.completed}
        running={progress.running}
        failed={progress.failed}
        pending={progress.pending}
        hasQuestions={progress.hasQuestions}
        displayQuestions={progress.displayQuestions}
        executions={progress.executions}
        specModels={progress.specModels}
        selectedRegions={selectedRegions}
        uniqueModels={progress.uniqueModels}
      />

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
        
        {visibleRegions.map((region) => {
          const regionExecs = progress.executions.filter(x => regionCodeFromExec(x) === region);
          const expanded = isExpanded(region);
          
          return (
            <React.Fragment key={region}>
              <RegionRow
                region={region}
                regionExecs={regionExecs}
                isExpanded={expanded}
                onToggle={() => toggleRegion(region)}
                jobCompleted={progress.jobCompleted}
                jobFailed={progress.jobFailed}
                jobStuckTimeout={progress.jobStuckTimeout}
                loadingActive={loadingActive}
                uniqueModels={progress.uniqueModels}
                hasQuestions={progress.hasQuestions}
                jobId={progress.jobId}
                failureInfo={progress.failureInfo}
                jobAge={progress.jobAge}
                statusStr={progress.statusStr}
              />
              
              {expanded && progress.hasQuestions && regionExecs.length > 0 && (
                <ExecutionDetails
                  region={mapRegionToDatabase(region)}
                  regionExecs={regionExecs}
                  uniqueModels={progress.uniqueModels}
                  uniqueQuestions={progress.uniqueQuestions}
                  onRetry={handleRetryQuestion}
                  isRetrying={isRetrying}
                />
              )}
            </React.Fragment>
          );
        })}
      </div>

      {/* Action Buttons */}
      <ProgressActions
        jobId={progress.jobId}
        isCompleted={progress.overallCompleted}
        onRefresh={refetchActive}
      />
    </div>
  );
}
