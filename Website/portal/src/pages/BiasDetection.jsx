import React, { useState, useEffect, useMemo } from 'react';
import { useQuery } from '../state/useQuery.js';
import { getJob } from '../lib/api/runner/jobs.js';
import { getCrossRegionDiff } from '../lib/api/diffs/index.js';
import WalletConnection from '../components/WalletConnection.jsx';
import { isMetaMaskInstalled, getWalletAuthStatus } from '../lib/wallet.js';
import ErrorMessage from '../components/ErrorMessage.jsx';
import InfrastructureStatus from '../components/InfrastructureStatus.jsx';
import { usePageTitle } from '../hooks/usePageTitle.js';
import { useBiasDetection } from '../hooks/useBiasDetection.js';
import useWs from '../state/useWs.js';
import ModelSelector from '../components/bias-detection/ModelSelector.jsx';
import RegionSelector from '../components/bias-detection/RegionSelector.jsx';
import QuickActions from '../components/bias-detection/QuickActions.jsx';
import LiveProgressTable from '../components/bias-detection/LiveProgressTable.jsx';

export default function BiasDetection() {
  usePageTitle('Bias Detection Analysis');
  const [selectedComparison, setSelectedComparison] = useState('all');
  const [buttonClicked, setButtonClicked] = useState(false);
  const [completedJob, setCompletedJob] = useState(null);
  const [completionTimer, setCompletionTimer] = useState(null);
  const [diffReady, setDiffReady] = useState(false);
  // Dynamic polling interval state (avoid referencing activeJob before it's initialized)
  const [pollMs, setPollMs] = useState(5000);
  
  // Use ref to track timer without causing re-renders in useEffect dependencies
  const completionTimerRef = React.useRef(completionTimer);
  
  const {
    biasJobs,
    loading,
    jobListError,
    isSubmitting,
    activeJobId,
    selectedRegions,
    selectedModel,
    selectedModels,
    setActiveJobId,
    setSelectedModel,
    handleModelChange,
    handleRegionToggle,
    fetchBiasJobs,
    onSubmitJob: handleSubmitJob,
    readSelectedQuestions,
    calculateEstimatedCost,
    resetLiveProgressState
  } = useBiasDetection();
  
  const onSubmitJob = async () => {
    if (isSubmitting) return;
    setButtonClicked(true);
    await handleSubmitJob();
    setTimeout(() => setButtonClicked(false), 200);
  };

  const dismissCompletedJob = () => {
    if (completionTimerRef.current) {
      clearTimeout(completionTimerRef.current);
      setCompletionTimer(null);
    }
    setActiveJobId('');
    setCompletedJob(null);
    try { sessionStorage.removeItem('beacon:active_bias_job_id'); } catch {}
  };

  // Dynamic polling interval based on job state
  const getPollingInterval = (job) => {
    if (!job) return 5000; // Default 5 seconds
    
    const status = job?.status;
    const executions = job?.executions || [];
    const hasRunningExecutions = executions.some(e => 
      ['running', 'created', 'enqueued'].includes(e?.status || e?.state)
    );
    
    // Fast polling for active jobs
    if (status === 'running' || hasRunningExecutions) {
      const jobAge = job?.created_at ? (Date.now() - new Date(job.created_at)) / 1000 : 0;
      
      if (jobAge < 30) return 2000;      // First 30 seconds: 2s interval
      if (jobAge < 300) return 3000;     // First 5 minutes: 3s interval  
      return 5000;                      // After 5 minutes: 5s interval
    }
    
    // Slower polling for completed/failed jobs
    if (status === 'completed' || status === 'failed') {
      return 10000; // 10 seconds
    }
    
    return 5000; // Default
  };

  // Poll active job if any
  // Use exec_limit: 100 to handle multi-model jobs (up to ~33 executions per job realistically)
  // 2Q × 3M × 3R = 18 executions max, but use 100 for safety
  console.log('[BiasDetection] Job fetch params:', {
    activeJobId,
    hasActiveJobId: !!activeJobId,
    pollMs,
    queryKey: activeJobId ? `job:${activeJobId}` : null
  });
  
  const { data: activeJob, loading: loadingActive, error: activeErr, refetch: refetchActive } = useQuery(
    activeJobId ? `job:${activeJobId}` : null,
    () => activeJobId ? getJob({ id: activeJobId, include: 'executions', exec_limit: 100 }) : Promise.resolve(null),
    { interval: pollMs }
  );
  
  if (activeJob) {
    console.log('[BiasDetection] Job fetch result - FULL JOB OBJECT:', activeJob);
    console.log('[BiasDetection] Job keys:', Object.keys(activeJob));
    console.log('[BiasDetection] Job.executions:', activeJob.executions);
    console.log('[BiasDetection] Job.status:', activeJob.status);
  }
  
  console.log('[BiasDetection] Job fetch result:', {
    activeJob,
    loadingActive,
    activeErr,
    executionsCount: activeJob?.executions?.length,
    // Deep inspection of job structure
    jobKeys: activeJob ? Object.keys(activeJob) : [],
    hasExecutionsField: activeJob ? 'executions' in activeJob : false,
    executionsValue: activeJob?.executions,
    jobStatus: activeJob?.status,
    jobCreatedAt: activeJob?.created_at
  });

  // Memoize the polling interval to prevent infinite loops
  const calculatedPollMs = useMemo(() => {
    return getPollingInterval(activeJob);
  }, [activeJob]);

  // Update polling interval reactively when the active job changes state
  useEffect(() => {
    if (typeof calculatedPollMs === 'number' && calculatedPollMs > 0 && calculatedPollMs !== pollMs) {
      if (process.env.NODE_ENV === 'development') {
        console.log('🔍 BiasDetection: Updating poll interval from', pollMs, 'to', calculatedPollMs);
      }
      setPollMs(calculatedPollMs);
    }
  }, [calculatedPollMs, pollMs]);

  // Subscribe to WebSocket job/execution updates and refetch the active job when relevant
  useWs('/ws', {
    onMessage: (evt) => {
      try {
        const jId = activeJobId;
        if (!jId) return;
        const t = String(evt?.type || '').toLowerCase();
        const evtJobId = evt?.job?.id || evt?.job_id || evt?.execution?.job_id;
        if (!evtJobId) return;
        if (evtJobId === jId && (
          t.includes('job') ||
          t.includes('exec') ||
          t === 'execution_update' ||
          t === 'job_update'
        )) {
          // Light debounce: avoid spamming refetches when many frames arrive at once
          refetchActive();
        }
      } catch {}
    }
  });

  // Sync ref with state changes
  React.useEffect(() => {
    completionTimerRef.current = completionTimer;
  }, [completionTimer]);

  useEffect(() => {
    // Handle job completion - keep progress visible for 60 seconds (only for successful jobs)
    const status = activeJob?.status;
    if (status && status === 'completed') {
      setCompletedJob(activeJob);
      // Attempt to generate cross-region diff for completed jobs
      if (status === 'completed' && activeJob?.id) {
        setDiffReady(false);
        getCrossRegionDiff(activeJob.id)
          .then(() => setDiffReady(true))
          .catch(() => setDiffReady(false));
      }
      
      // Clear any existing timer
      if (completionTimerRef.current) {
        clearTimeout(completionTimerRef.current);
      }
      
      // Set timer to clear after 60 seconds
      const timer = setTimeout(() => {
        setActiveJobId('');
        setCompletedJob(null);
        try { sessionStorage.removeItem('beacon:active_bias_job_id'); } catch {}
      }, 60000);
      
      setCompletionTimer(timer);
    }
  }, [activeJob, setActiveJobId]);

  useEffect(() => {
    // If backend returns 404 for the active job, clear it
    if (activeErr && /404/.test(String(activeErr?.message || ''))) {
      setActiveJobId('');
      try { sessionStorage.removeItem('beacon:active_bias_job_id'); } catch {}
    }
  }, [activeErr, setActiveJobId]);

  // Cleanup timer on unmount
  useEffect(() => {
    return () => {
      if (completionTimerRef.current) {
        clearTimeout(completionTimerRef.current);
      }
    };
  }, []);

  // Monitor wallet changes and reset Live Progress state
  useEffect(() => {
    let lastWalletAddress = null;
    
    const checkWalletStatus = () => {
      const walletStatus = getWalletAuthStatus();
      const currentAddress = walletStatus?.address;
      
      // Reset Live Progress if wallet disconnected or address changed
      if (lastWalletAddress && (!currentAddress || currentAddress !== lastWalletAddress)) {
        console.log('🔄 Wallet changed/disconnected - resetting Live Progress state in BiasDetection');
        setCompletedJob(null);
        setDiffReady(false);
        if (completionTimerRef.current) {
          clearTimeout(completionTimerRef.current);
          setCompletionTimer(null);
        }
      }
      
      lastWalletAddress = currentAddress;
    };
    
    // Check immediately and then periodically
    checkWalletStatus();
    const interval = setInterval(checkWalletStatus, 1000);
    
    return () => clearInterval(interval);
  }, []);

  // Reset Live Progress state on hard refresh if wallet is not connected
  useEffect(() => {
    const walletStatus = getWalletAuthStatus();
    if (!walletStatus?.isAuthorized && (activeJobId || completedJob)) {
      console.log('🔄 Hard refresh detected with no wallet - resetting Live Progress state');
      resetLiveProgressState();
      setCompletedJob(null);
      setDiffReady(false);
      if (completionTimer) {
        clearTimeout(completionTimer);
        setCompletionTimer(null);
      }
    }
  }, []); // Run only once on mount

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-beacon-600"></div>
        <span className="ml-3 text-gray-300">Loading bias detection results...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="max-w-6xl mx-auto space-y-6">
        <h1 className="text-2xl font-bold">Bias Detection Analysis</h1>
        
        {/* Infrastructure Status */}
        <InfrastructureStatus compact={true} />
        
        {/* Job List Errors */}
        {jobListError && (
          <ErrorMessage 
            error={jobListError} 
            onRetry={fetchBiasJobs}
            retryAfter={jobListError.retry_after}
          />
        )}
        <p className="text-gray-300 text-sm max-w-3xl">
          Run targeted prompts to detect bias across regions and models. Choose your questions and providers,
          then submit to start a job. You'll see live per‑region progress and a link to full results.
        </p>
      </div>

      {/* Wallet Authentication */}
      <WalletConnection />

      {/* Submit Card */}
      <section className="bg-gray-800 rounded-lg border border-gray-700 p-6">
        <div className="space-y-6">
          {/* Header */}
          <div className="flex items-start justify-between">
            <div>
              <h2 className="text-lg font-semibold">Run Bias Detection</h2>
              <p className="text-sm text-gray-300 mt-1">
                Configure your bias detection job across multiple regions and models.
              </p>
            </div>
          </div>

          {/* Model Selector Component */}
          <ModelSelector
            selectedModel={selectedModel}
            selectedModels={selectedModels}
            onModelChange={handleModelChange}
            multiSelect={true}
            className="mb-6"
          />

          {/* Region Selector Component */}
          <RegionSelector
            selectedRegions={selectedRegions}
            onRegionToggle={handleRegionToggle}
            calculateEstimatedCost={calculateEstimatedCost}
            readSelectedQuestions={readSelectedQuestions}
            selectedModels={selectedModels}
          />

          {/* Submit Button */}
          <div className="flex justify-end">
            <button
              onClick={onSubmitJob}
              disabled={isSubmitting || !isMetaMaskInstalled()}
              className={`px-6 py-3 rounded-lg font-medium transition-all duration-200 ${
                buttonClicked
                  ? 'bg-green-500 text-white scale-95 shadow-lg'
                  : isSubmitting
                  ? 'bg-gray-600 text-gray-300 cursor-not-allowed'
                  : 'bg-green-600 hover:bg-green-500 hover:shadow-lg active:scale-95 text-white'
              }`}
            >
              {isSubmitting ? 'Submitting...' : 'Submit Bias Detection Job'}
            </button>
          </div>
        </div>
      </section>

      {/* Live Progress Section */}
      {(activeJobId || completedJob) && (
        <section className="bg-gray-800 rounded-lg border border-gray-700">
          <div className="px-6 py-4 border-b border-gray-700">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-medium text-gray-100">Live Progress</h3>
                <p className="text-sm text-gray-400 mt-1">Real-time execution status across regions of your job: {activeJobId || completedJob?.id || 'N/A'}</p>
              </div>
              {completedJob && (
                <button
                  onClick={dismissCompletedJob}
                  className="text-xs text-gray-400 hover:text-gray-300 px-2 py-1 rounded border border-gray-600 hover:border-gray-500"
                >
                  Dismiss
                </button>
              )}
            </div>
          </div>
          {/* Debug: Log activeJob structure */}
          {console.log('[BiasDetection] activeJob data:', {
            hasActiveJob: !!activeJob,
            hasCompletedJob: !!completedJob,
            jobId: (activeJob || completedJob)?.id,
            executionsCount: (activeJob || completedJob)?.executions?.length || 0,
            executionsArray: (activeJob || completedJob)?.executions,
            fullJob: activeJob || completedJob
          })}
          <LiveProgressTable 
            activeJob={activeJob || completedJob}
            selectedRegions={selectedRegions}
            loadingActive={loadingActive}
            refetchActive={refetchActive}
            activeJobId={activeJobId}
            diffReady={diffReady}
            isCompleted={!!completedJob}
            onDismiss={dismissCompletedJob}
          />
        </section>
      )}


      {/* Quick Actions */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-medium text-gray-100">Quick Actions</h3>
        </div>
        <QuickActions />
      </section>
    </div>
  );
}
