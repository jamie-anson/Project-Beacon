import React, { useState, useEffect } from 'react';
import { useQuery } from '../state/useQuery.js';
import { getJob } from '../lib/api.js';
import WalletConnection from '../components/WalletConnection.jsx';
import { isMetaMaskInstalled } from '../lib/wallet.js';
import ErrorMessage from '../components/ErrorMessage.jsx';
import InfrastructureStatus from '../components/InfrastructureStatus.jsx';
import { useBiasDetection } from '../hooks/useBiasDetection.js';
import RegionSelector from '../components/bias-detection/RegionSelector.jsx';
import LiveProgressTable from '../components/bias-detection/LiveProgressTable.jsx';
import JobSummaryCards from '../components/bias-detection/JobSummaryCards.jsx';
import QuickActions from '../components/bias-detection/QuickActions.jsx';

export default function BiasDetection() {
  const [selectedComparison, setSelectedComparison] = useState('all');
  const [buttonClicked, setButtonClicked] = useState(false);
  
  const {
    biasJobs,
    loading,
    jobListError,
    isSubmitting,
    activeJobId,
    selectedRegions,
    isMultiRegion,
    setActiveJobId,
    setIsMultiRegion,
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

  // Poll active job if any
  const { data: activeJob, loading: loadingActive, error: activeErr, refetch: refetchActive } = useQuery(
    activeJobId ? `job:${activeJobId}` : null,
    () => activeJobId ? getJob({ id: activeJobId, include: 'executions', exec_limit: 3 }) : Promise.resolve(null),
    { interval: 5000 }
  );

  useEffect(() => {
    // Clear session if job completed or failed
    const status = activeJob?.status;
    if (status && (status === 'completed' || status === 'failed' || status === 'cancelled')) {
      setActiveJobId('');
      try { sessionStorage.removeItem('beacon:active_bias_job_id'); } catch {}
    }
  }, [activeJob, setActiveJobId]);

  useEffect(() => {
    // If backend returns 404 for the active job, clear it
    if (activeErr && /404/.test(String(activeErr?.message || ''))) {
      setActiveJobId('');
      try { sessionStorage.removeItem('beacon:active_bias_job_id'); } catch {}
    }
  }, [activeErr, setActiveJobId]);

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
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={isMultiRegion}
                  onChange={(e) => setIsMultiRegion(e.target.checked)}
                  className="rounded border-gray-600 bg-gray-700 text-orange-500 focus:ring-orange-500"
                />
                <span className="text-gray-300">Multi-Region Analysis</span>
              </label>
            </div>
          </div>

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
              className={`px-6 py-3 rounded-lg font-medium transition-all ${
                buttonClicked
                  ? 'bg-green-600 text-white scale-95'
                  : isSubmitting
                  ? 'bg-gray-600 text-gray-300 cursor-not-allowed'
                  : 'bg-beacon-600 hover:bg-beacon-700 text-white'
              }`}
            >
              {isSubmitting ? 'Submitting...' : 'Submit Bias Detection Job'}
            </button>
          </div>
        </div>
      </section>

      {/* Live Progress Section */}
      {activeJobId && (
        <section className="bg-gray-800 rounded-lg border border-gray-700">
          <div className="px-6 py-4 border-b border-gray-700">
            <h3 className="text-lg font-medium text-gray-100">Live Progress</h3>
            <p className="text-sm text-gray-400 mt-1">Real-time execution status across regions</p>
          </div>
          <LiveProgressTable
            activeJob={activeJob}
            selectedRegions={selectedRegions}
            loadingActive={loadingActive}
            refetchActive={refetchActive}
            activeJobId={activeJobId}
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
