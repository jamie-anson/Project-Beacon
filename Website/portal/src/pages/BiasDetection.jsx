import React, { useState, useEffect } from 'react';
import { useQuery } from '../state/useQuery.js';
import useWs from '../state/useWs.js';
import { getJob, getCrossRegionDiff } from '../lib/api.js';
import WalletConnection from '../components/WalletConnection.jsx';
import { isMetaMaskInstalled } from '../lib/wallet.js';
import ErrorMessage from '../components/ErrorMessage.jsx';
import InfrastructureStatus from '../components/InfrastructureStatus.jsx';
import { useBiasDetection } from '../hooks/useBiasDetection.js';
import RegionSelector from '../components/bias-detection/RegionSelector.jsx';
import ModelSelector from '../components/bias-detection/ModelSelector.jsx';
import LiveProgressTable from '../components/bias-detection/LiveProgressTable.jsx';
import JobSummaryCards from '../components/bias-detection/JobSummaryCards.jsx';
import QuickActions from '../components/bias-detection/QuickActions.jsx';

export default function BiasDetection() {
  const [selectedComparison, setSelectedComparison] = useState('all');
  const [buttonClicked, setButtonClicked] = useState(false);
  const [completedJob, setCompletedJob] = useState(null);
  const [completionTimer, setCompletionTimer] = useState(null);
  const [diffReady, setDiffReady] = useState(false);
  // Dynamic polling interval state (avoid referencing activeJob before it's initialized)
  const [pollMs, setPollMs] = useState(5000);
  
  const {
    biasJobs,
    loading,
    jobListError,
    isSubmitting,
    activeJobId,
    selectedRegions,
    isMultiRegion,
    selectedModel,
    setActiveJobId,
    setIsMultiRegion,
    setSelectedModel,
    handleRegionToggle,
    fetchBiasJobs,
    onSubmitJob: handleSubmitJob,
    readSelectedQuestions,
    calculateEstimatedCost
  } = useBiasDetection();
  
  const onSubmitJob = async () => {
    if (isSubmitting) return;
    setButtonClicked(true);
    await handleSubmitJob();
    setTimeout(() => setButtonClicked(false), 200);
  };

  const dismissCompletedJob = () => {
    if (completionTimer) {
      clearTimeout(completionTimer);
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
  const { data: activeJob, loading: loadingActive, error: activeErr, refetch: refetchActive } = useQuery(
    activeJobId ? `job:${activeJobId}` : null,
    () => activeJobId ? getJob({ id: activeJobId, include: 'executions', exec_limit: 10 }) : Promise.resolve(null),
    { interval: pollMs }
  );

  // Update polling interval reactively when the active job changes state
  useEffect(() => {
    const next = getPollingInterval(activeJob);
    if (typeof next === 'number' && next > 0 && next !== pollMs) {
      setPollMs(next);
    }
  }, [activeJob, pollMs]);

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
      if (completionTimer) {
        clearTimeout(completionTimer);
      }
      
      // Set timer to clear after 60 seconds
      const timer = setTimeout(() => {
        setActiveJobId('');
        setCompletedJob(null);
        try { sessionStorage.removeItem('beacon:active_bias_job_id'); } catch {}
      }, 60000);
      
      setCompletionTimer(timer);
    }
  }, [activeJob, setActiveJobId, completionTimer]);

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
      if (completionTimer) {
        clearTimeout(completionTimer);
      }
    };
  }, [completionTimer]);

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
            <div className="flex items-center gap-2">
              <label className={`flex items-center gap-2 text-sm ${selectedRegions.length > 1 ? 'cursor-pointer' : 'cursor-not-allowed opacity-50'}`}>
                <input
                  type="checkbox"
                  checked={isMultiRegion}
                  onChange={(e) => setIsMultiRegion(e.target.checked)}
                  disabled={selectedRegions.length <= 1}
                  className="rounded border-gray-600 bg-gray-700 text-orange-500 focus:ring-orange-500 disabled:opacity-50"
                />
                <span className="text-gray-300">Multi-Region Analysis</span>
                {selectedRegions.length <= 1 && (
                  <span className="text-xs text-gray-500 ml-1">(Select multiple regions to enable)</span>
                )}
              </label>
            </div>
          </div>

          {/* Model Selector Component */}
          <ModelSelector
            selectedModel={selectedModel}
            onModelChange={setSelectedModel}
            className="mb-6"
          />

          {/* Region Selector Component */}
          <RegionSelector
            selectedRegions={selectedRegions}
            onRegionToggle={handleRegionToggle}
            calculateEstimatedCost={calculateEstimatedCost}
            readSelectedQuestions={readSelectedQuestions}
            isMultiRegion={isMultiRegion}
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
                <p className="text-sm text-gray-400 mt-1">Real-time execution status across regions</p>
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

      {/* Job Summary Cards */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-medium text-gray-100">Recent Jobs by Region</h3>
        </div>
        <JobSummaryCards biasJobs={biasJobs} loading={loading} />
      </section>

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
