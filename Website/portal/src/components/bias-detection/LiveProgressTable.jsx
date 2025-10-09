/**
 * LiveProgressTable Component (Refactored)
 * Orchestrates job progress display using modular hooks and components
 * 
 * Structure: Question → Model → Region hierarchy
 */

import React from 'react';
import { useJobProgress } from '../../hooks/useJobProgress';
import { useRetryExecution } from '../../hooks/useRetryExecution';
import { useCountdownTimer } from '../../hooks/useCountdownTimer';
import { transformExecutionsToQuestions } from './liveProgressHelpers';
import FailureAlert from './progress/FailureAlert';
import ProgressHeader from './progress/ProgressHeader';
import ProgressBreakdown from './progress/ProgressBreakdown';
import QuestionRow from './QuestionRow';
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
  const { timeRemaining } = useCountdownTimer(
    !progress.overallCompleted && !progress.overallFailed,
    progress.jobCompleted,
    progress.jobFailed,
    progress.jobStartTime
  );
  
  // Transform executions into question-centric hierarchy
  const questionData = transformExecutionsToQuestions(activeJob, selectedRegions);
  
  return (
    <div className="p-4 space-y-3">
      {/* Failure Alert */}
      <FailureAlert failureInfo={progress.failureInfo} />

      {/* Progress Header with animated stage indicators */}
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
      
      {/* Progress Breakdown with status counts */}
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

      {/* Question-Centric Progress Display: Question → Model → Region */}
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
      <ProgressActions
        jobId={progress.jobId}
        isCompleted={progress.overallCompleted}
        isFailed={progress.overallFailed || progress.jobStuckTimeout}
        onRefresh={refetchActive}
        onRetryJob={() => window.location.reload()}
      />
    </div>
  );
}
