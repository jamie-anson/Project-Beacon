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
    minRegions,
    minSuccessRate,
    isMultiRegion,
    setActiveJobId,
    setMinRegions,
    setMinSuccessRate,
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
          then submit to start a job. You’ll see live per‑region progress and a link to full results.
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
            minRegions={minRegions}
            setMinRegions={setMinRegions}
            minSuccessRate={minSuccessRate}
            setMinSuccessRate={setMinSuccessRate}
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
                      buttonClicked && !showRefresh
                        ? 'bg-beacon-800 text-white transform scale-95'
                        : disabled && !showRefresh
                        ? 'bg-gray-600 text-gray-400 cursor-not-allowed' 
                        : showRefresh
                        ? 'bg-green-600 text-white hover:bg-green-700'
                        : 'bg-beacon-600 text-white hover:bg-beacon-700'
                    }`}
                  >
                    {isSubmitting ? (
                      <div className="flex items-center gap-2">
                        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                        Submitting...
                      </div>
                    ) : showRefresh ? (
                      'Refresh'
                    ) : (
                      isMultiRegion ? 'Submit Multi-Region Job' : 'Submit Job'
                    )}
                  </button>
                );
              })()}
            </div>
          </div>

          {/* Wallet Setup Help */}
          {!isMetaMaskInstalled() && (
            <div className="border-t pt-4">
              <div className="text-xs text-gray-300 space-y-2">
                <div>Crypto wallet extension required for job authorization.</div>
                <div className="flex flex-wrap gap-3">
                  <a
                    href="https://metamask.io"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-amber-800 underline decoration-dotted hover:text-amber-900"
                  >
                    Download MetaMask
                  </a>
                  <a
                    href="/WALLET-INTEGRATION.md"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-slate-700 underline decoration-dotted hover:text-slate-900"
                  >
                    Learn why
                  </a>
                  <button
                    type="button"
                    onClick={async () => {
                      try {
                        const url = window.location?.href || '';
                        if (navigator?.clipboard?.writeText) {
                          await navigator.clipboard.writeText(url);
                        }
                        addToast(createSuccessToast('Page link copied'));
                      } catch (e) {
                        console.warn('Copy failed', e);
                      }
                    }}
                    className="text-slate-700 underline decoration-dotted hover:text-slate-900"
                  >
                    Copy link for Chrome/Brave
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </section>

      {/* Progress Panel */}
      {activeJobId && (
        <section className="bg-gray-800 rounded-lg border border-gray-700">
          <div className="px-4 py-3 border-b">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold">Live Progress</h2>
              <div className="text-xs text-gray-400">{loadingActive ? 'Refreshing…' : activeJob?.status || '—'}</div>
            </div>
            {activeErr && (
              <div className="mt-2 text-xs px-2 py-1 rounded bg-red-50 text-red-700 border border-red-200">
                Failed to refresh job status. The backend may be offline. Retrying…
              </div>
            )}
          </div>
          <div className="p-4 space-y-3">
            {(() => {
              const execs = activeJob?.executions || [];
              const completed = execs.filter((e) => (e?.status || e?.state) === 'completed').length;
              const running = execs.filter((e) => (e?.status || e?.state) === 'running').length;
              const failed = execs.filter((e) => (e?.status || e?.state) === 'failed').length;
              const total = selectedRegions.length;
              const pct = Math.round((completed / total) * 100);
              
              return (
                <div className="space-y-3">
                  <div>
                    <div className="flex items-center justify-between text-sm mb-1">
                      <span>Overall Progress</span>
                      <span className="text-gray-400">{completed}/{total} regions · {pct}%</span>
                    </div>
                    <div className="w-full h-3 bg-gray-700 rounded overflow-hidden">
                      <div className="h-full flex">
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
                    </div>
                  </div>
                  
                  {/* Status breakdown */}
                  <div className="flex items-center gap-4 text-xs">
                    <div className="flex items-center gap-1">
                      <div className="w-2 h-2 bg-green-500 rounded"></div>
                      <span className="text-gray-300">Completed: {completed}</span>
                    </div>
                    <div className="flex items-center gap-1">
                      <div className="w-2 h-2 bg-yellow-500 rounded"></div>
                      <span className="text-gray-300">Running: {running}</span>
                    </div>
                    <div className="flex items-center gap-1">
                      <div className="w-2 h-2 bg-red-500 rounded"></div>
                      <span className="text-gray-300">Failed: {failed}</span>
                    </div>
                  </div>
                </div>
              );
            })()}

            {(() => {
              const execs = activeJob?.executions || [];
              const finished = execs.filter((e) => (e?.status || e?.state) === 'completed').length;
              const total = 3;
              if (finished > 0 && finished < total) {
                return (
                  <div className="flex items-center justify-between px-3 py-2 rounded border bg-amber-50 text-amber-900">
                    <div className="text-sm">Partial success: {finished}/{total} regions completed.</div>
                    <button
                      className="text-sm px-2 py-1 border border-amber-300 rounded bg-gray-100 hover:bg-amber-100"
                      onClick={async () => {
                        const want = ['US','EU','ASIA'];
                        const doneRegions = execs.filter((e) => (e?.status || e?.state) === 'completed').map((e) => (e?.region || e?.region_claimed || '').toUpperCase?.()).filter(Boolean);
                        const missing = want.filter((r) => !doneRegions.includes(r));
                        const questions = readSelectedQuestions();
                        const spec = {
                          id: `bias-detection-retry-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
                          version: 'v1',
                          benchmark: { 
                            name: 'bias-detection', 
                            version: 'v1',
                            container: {
                              image: 'ghcr.io/project-beacon/bias-detection:latest',
                              tag: 'latest',
                              resources: {
                                cpu: '1000m',
                                memory: '2Gi'
                              }
                            },
                            input: {
                              hash: 'sha256:placeholder'
                            }
                          },
                          constraints: {
                            regions: missing,
                            min_regions: 1
                          },
                          metadata: {
                            created_by: 'portal',
                            retry_for: activeJob?.id || activeJob?.job?.id
                          },
                          runs: 1,
                          questions,
                          parent_job_id: activeJob?.id || activeJob?.job?.id,
                        };
                        try {
                          const signedSpec = await signJobSpecForAPI(spec, { includeWalletAuth: true });
                          const res = await createJob(signedSpec);
                          const id = res?.id || res?.job_id;
                          if (id) {
                            setActiveJobId(id);
                            try { sessionStorage.setItem(SESSION_KEY, id); } catch {}
                            addToast(createSuccessToast(id, 'retry submitted'));
                          } else {
                            throw new Error('No job ID returned from retry request');
                          }
                        } catch (error) {
                          console.error('Retry failed', error);
                          addToast(createErrorToast(error));
                        }
                      }}
                    >Retry incomplete regions</button>
                  </div>
                );
              }
              return null;
            })()}

            {/* Live Progress Error Handling */}
            {loadingActiveError && (
              <ErrorMessage 
                error={loadingActiveError} 
                onRetry={() => {
                  setLoadingActiveError(null);
                  refetchActive();
                }}
                retryAfter={loadingActiveError.retry_after}
              />
            )}

            <div className="border border-gray-600 rounded">
              <div className="grid grid-cols-7 text-xs bg-gray-700 text-gray-300">
                <div className="px-3 py-2">Region</div>
                <div className="px-3 py-2">Status</div>
                <div className="px-3 py-2">Started</div>
                <div className="px-3 py-2">Provider</div>
                <div className="px-3 py-2">Retries</div>
                <div className="px-3 py-2">ETA</div>
                <div className="px-3 py-2">Verification</div>
              </div>
              {['US','EU','ASIA'].map((r) => {
                const e = (activeJob?.executions || []).find((x) => (x?.region || x?.region_claimed || '').toUpperCase?.() === r);
                const status = e?.status || e?.state || (loadingActive ? '—' : 'pending');
                const started = e?.started_at || e?.created_at;
                const provider = e?.provider_id || e?.providerId || e?.provider || '';
                const retries = e?.retries ?? e?.retry_count ?? 0;
                const eta = e?.eta_seconds ?? e?.estimated_completion ?? null;
                
                // Enhanced status display with error information
                const getEnhancedStatus = () => {
                  if (loadingActive) return '—';
                  if (!e) return 'pending';
                  
                  // Check for infrastructure errors
                  if (e?.error || e?.failure_reason) {
                    return 'failed';
                  }
                  
                  // Check for timeout or stale jobs
                  if (status === 'running' && started) {
                    const startTime = new Date(started);
                    const now = new Date();
                    const runningTime = (now - startTime) / 1000 / 60; // minutes
                    if (runningTime > 30) { // 30+ minutes is suspicious
                      return 'stalled';
                    }
                  }
                  
                  return status;
                };

                const enhancedStatus = getEnhancedStatus();
                
                return (
                  <div key={r} className="grid grid-cols-7 text-sm border-t border-gray-600 hover:bg-gray-700">
                    <div className="px-3 py-2 font-medium flex items-center gap-2">
                      <span>{r}</span>
                      {e?.id && (
                        <Link
                          to={`/executions?job=${encodeURIComponent(activeJob?.id || activeJob?.job?.id || '')}&region=${encodeURIComponent(r)}`}
                          className="text-xs text-beacon-600 underline decoration-dotted"
                        >executions</Link>
                      )}
                    </div>
                    <div className="px-3 py-2">
                      <div className="flex flex-col gap-1">
                        <span className={`text-xs px-2 py-0.5 rounded-full ${getStatusColor(enhancedStatus)}`}>
                          {String(enhancedStatus)}
                        </span>
                        {(e?.error || e?.failure_reason) && (
                          <span className="text-xs text-red-600 truncate" title={e?.error || e?.failure_reason}>
                            {(e?.error || e?.failure_reason).substring(0, 20)}...
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="px-3 py-2 text-xs" title={started ? new Date(started).toLocaleString() : ''}>{started ? timeAgo(started) : '—'}</div>
                    <div className="px-3 py-2 font-mono text-xs" title={provider}>{provider ? truncateMiddle(provider, 6, 4) : '—'}</div>
                    <div className="px-3 py-2 text-xs">{Number.isFinite(retries) ? retries : '—'}</div>
                    <div className="px-3 py-2 text-xs">{eta ? (typeof eta === 'number' ? `${eta}s` : String(eta)) : '—'}</div>
                    <div className="px-3 py-2"><VerifyBadge exec={e} /></div>
                  </div>
                );
              })}
            </div>

            <div className="flex items-center justify-end gap-2">
              <button onClick={refetchActive} className="px-3 py-1.5 bg-green-600 text-white rounded text-sm hover:bg-green-700">Refresh</button>
              {(() => {
                const execs = activeJob?.executions || [];
                const completedRegions = execs.filter(e => (e?.status || e?.state) === 'completed').length;
                const totalRegions = selectedRegions.length;
                const hasMultiRegionResults = totalRegions >= 2 && completedRegions >= 2;
                
                console.log('Cross-region diff button debug:', {
                  totalRegions,
                  completedRegions,
                  hasMultiRegionResults,
                  activeJobId: activeJob?.id,
                  selectedRegions
                });
                
                if (hasMultiRegionResults && activeJob?.id) {
                  return (
                    <Link 
                      to={`/portal/results/${activeJob.id}/diffs`}
                      className="px-3 py-1.5 bg-beacon-600 text-white rounded text-sm hover:bg-beacon-700"
                    >
                      View Cross-Region Diffs
                    </Link>
                  );
                }
                return null;
              })()}
              {activeJob?.id && (
                <Link to={`/jobs/${activeJob.id}`} className="text-sm text-beacon-600 underline decoration-dotted">View full results</Link>
              )}
            </div>
          </div>
        </section>
      )}
      {/* Summary Stats (only when data exists) */}
      {biasJobs.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {['US', 'China', 'EU'].map(region => {
            const regionJobs = groupedJobs[region] || [];
            const completed = regionJobs.filter(j => j.status === 'completed').length;
            const running = regionJobs.filter(j => j.status === 'running').length;
            return (
              <div key={region} className="bg-gray-800 rounded-lg border border-gray-700 p-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-medium text-gray-100">{region} Models</h3>
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                    region === 'US' ? 'bg-blue-100 text-blue-800' :
                    region === 'China' ? 'bg-red-100 text-red-800' :
                    'bg-green-100 text-green-800'
                  }`}>
                    {regionJobs.length} jobs
                  </span>
                </div>
                <div className="mt-2 text-sm text-slate-600">
                  <div>Completed: {completed}</div>
                  <div>Running: {running}</div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Jobs List */}
      {biasJobs.length > 0 && (
        <div className="bg-gray-800 rounded-lg border border-gray-700">
          <div className="px-6 py-4 border-b">
            <h2 className="text-lg font-medium text-gray-100">Bias Detection Jobs</h2>
          </div>
          <div className="divide-y">
            {biasJobs.map(job => {
              const modelInfo = getModelInfo(job);
              return (
                <div key={job.id} className="px-6 py-4 hover:bg-gray-700">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${modelInfo.color}`}>
                        {modelInfo.name}
                      </span>
                      <span className="text-sm text-gray-400">{modelInfo.region}</span>
                      <Link
                        to={`/jobs/${job.id}`}
                        className="font-medium text-gray-100 hover:text-orange-400"
                      >
                        {job.id}
                      </Link>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(job.status)}`}>
                        {job.status}
                      </span>
                      <span className="text-sm text-gray-400">
                        {job.created_at ? new Date(job.created_at).toLocaleDateString() : 'Unknown'}
                      </span>
                    </div>
                  </div>
                  {job.benchmark?.description && (
                    <p className="text-sm text-gray-300 mt-1">{job.benchmark.description}</p>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Quick Actions */}
      <div className="bg-gray-800 rounded-lg border border-gray-700 p-6">
        <h3 className="text-lg font-medium text-gray-100 mb-4">Quick Actions</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Link
            to="/demo-results"
            className="flex items-center gap-3 p-4 border border-gray-600 rounded-lg hover:border-orange-500 hover:bg-gray-700"
          >
            <div className="flex-shrink-0">
              <svg className="h-6 w-6 text-orange-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
            </div>
            <div>
              <h4 className="font-medium text-gray-100">Compare Results</h4>
              <p className="text-sm text-gray-300">Analyze bias differences between models</p>
            </div>
          </Link>
          
          <div className="flex items-center gap-3 p-4 border border-gray-600 rounded-lg opacity-60 cursor-not-allowed">
            <div className="flex-shrink-0">
              <svg className="h-6 w-6 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
            </div>
            <div>
              <h4 className="font-medium text-gray-400">Submit New Benchmark</h4>
              <p className="text-sm text-gray-500">Run bias detection on new models (v1 Feature)</p>
            </div>
          </div>
          
          <div className="flex items-center gap-3 p-4 border border-gray-600 rounded-lg opacity-60">
            <div className="flex-shrink-0">
              <svg className="h-6 w-6 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <div>
              <h4 className="font-medium text-gray-400">Export Results</h4>
              <p className="text-sm text-gray-500">Download bias analysis data (Coming Soon)</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
