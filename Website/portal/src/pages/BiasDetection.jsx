import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { createJob, getJob, listJobs } from '../lib/api.js';
import { useQuery } from '../state/useQuery.js';
import { signJobSpecForAPI } from '../lib/crypto.js';
import WalletConnection from '../components/WalletConnection.jsx';
import { useToast } from '../state/toast.jsx';
import { createErrorToast, createSuccessToast, createWarningToast } from '../lib/errorUtils.js';
import { getWalletAuthStatus, isMetaMaskInstalled } from '../lib/wallet.js';
import ErrorMessage from '../components/ErrorMessage.jsx';
import InfrastructureStatus from '../components/InfrastructureStatus.jsx';

export default function BiasDetection() {
  const [biasJobs, setBiasJobs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedComparison, setSelectedComparison] = useState('all');
  const SESSION_KEY = 'beacon:active_bias_job_id';
  const [activeJobId, setActiveJobId] = useState(() => {
    try { return sessionStorage.getItem(SESSION_KEY) || ''; } catch { return ''; }
  });
  const { add: addToast } = useToast();
  
  // Multi-region state
  const [selectedRegions, setSelectedRegions] = useState(['US', 'EU', 'ASIA']);
  const [minRegions, setMinRegions] = useState(1);
  const [minSuccessRate, setMinSuccessRate] = useState(0.67);
  const [isMultiRegion, setIsMultiRegion] = useState(false);
  
  // Error handling state
  const [jobSubmissionError, setJobSubmissionError] = useState(null);
  const [jobListError, setJobListError] = useState(null);
  const [loadingActiveError, setLoadingActiveError] = useState(null);
  
  // Button state management
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [buttonClicked, setButtonClicked] = useState(false);
  
  const availableRegions = [
    { code: 'US', name: 'United States', model: 'Llama 3.2-1B', cost: 0.0003 },
    { code: 'EU', name: 'Europe', model: 'Mistral 7B', cost: 0.0004 },
    { code: 'ASIA', name: 'Asia Pacific', model: 'Qwen 2.5-1.5B', cost: 0.0005 }
  ];

  useEffect(() => {
    fetchBiasJobs();
  }, []);

  const fetchBiasJobs = async () => {
    try {
      setJobListError(null);
      const data = await listJobs({ limit: 50 });
      const jobs = Array.isArray(data?.jobs) ? data.jobs : (Array.isArray(data) ? data : []);
      // Filter for bias detection jobs
      const biasJobsData = jobs.filter(job =>
        job?.benchmark?.name?.includes('bias-detection') || job?.id?.includes('bias-detection')
      );
      setBiasJobs(biasJobsData);
    } catch (error) {
      console.error('Failed to fetch bias jobs:', error);
      setJobListError(error);
      addToast(createErrorToast(error));
    } finally {
      setLoading(false);
    }
  };

  // Read selected questions from localStorage written by Questions page
  const readSelectedQuestions = () => {
    try {
      const raw = localStorage.getItem('beacon:selected_questions');
      const arr = raw ? JSON.parse(raw) : [];
      return Array.isArray(arr) ? arr : [];
    } catch {
      return [];
    }
  };

  // Calculate estimated cost for multi-region execution
  const calculateEstimatedCost = () => {
    const questions = readSelectedQuestions();
    const questionCount = questions.length || 1;
    const regionCost = selectedRegions.reduce((total, regionCode) => {
      const region = availableRegions.find(r => r.code === regionCode);
      return total + (region?.cost || 0.0004);
    }, 0);
    return (regionCost * questionCount).toFixed(4);
  };

  // Handle region selection changes
  const handleRegionToggle = (regionCode) => {
    setSelectedRegions(prev => {
      if (prev.includes(regionCode)) {
        const newRegions = prev.filter(r => r !== regionCode);
        // Ensure at least one region is selected
        return newRegions.length > 0 ? newRegions : prev;
      } else {
        return [...prev, regionCode];
      }
    });
  };

  const onSubmitJob = async () => {
    if (isSubmitting) return; // Prevent double submission
    
    setIsSubmitting(true);
    setButtonClicked(true);
    
    const questions = readSelectedQuestions();
    
    // Validate questions selection
    if (questions.length === 0) {
      addToast(createWarningToast('Please select at least one question on the Questions page before submitting a job.'));
      setIsSubmitting(false);
      setButtonClicked(false);
      return;
    }

    // Check wallet authorization
    const walletStatus = getWalletAuthStatus();
    if (!walletStatus.isAuthorized) {
      addToast(createWarningToast('Please connect your wallet before submitting a job.'));
      setIsSubmitting(false);
      setButtonClicked(false);
      return;
    }

    const spec = {
      id: `bias-detection-${isMultiRegion ? 'multi' : 'single'}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
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
        regions: selectedRegions,
        min_regions: isMultiRegion ? minRegions : 1,
        min_success_rate: isMultiRegion ? minSuccessRate : undefined
      },
      metadata: {
        created_by: 'portal',
        wallet_address: walletStatus.address,
        execution_type: isMultiRegion ? 'cross-region' : 'single-region',
        estimated_cost: calculateEstimatedCost()
      },
      runs: 1,
      questions,
    };
    
    try {
      // Sign the JobSpec with Ed25519 and include wallet authentication
      const signedSpec = await signJobSpecForAPI(spec, { includeWalletAuth: true });
      const res = await createJob(signedSpec);
      const id = res?.id || res?.job_id;
      
      if (id) {
        setActiveJobId(id);
        try { sessionStorage.setItem(SESSION_KEY, id); } catch {}
        addToast(createSuccessToast(id, 'submitted'));
      } else {
        throw new Error('No job ID returned from server');
      }
      
      // Refresh recent list soon after create
      fetchBiasJobs();
    } catch (error) {
      console.error('Failed to create job', error);
      addToast(createErrorToast(error));
    } finally {
      setIsSubmitting(false);
      // Keep button clicked state briefly for visual feedback
      setTimeout(() => setButtonClicked(false), 200);
    }
  };

  // Poll active job if any
  const { data: activeJob, loading: loadingActive, error: activeErr, refetch: refetchActive } = useQuery(
    activeJobId ? `job:${activeJobId}` : null,
    () => activeJobId ? getJob({ id: activeJobId, include: 'executions', exec_limit: 3 }) : Promise.resolve(null),
    { interval: 5000 }
  );

  // Handle active job loading errors
  useEffect(() => {
    if (activeErr) {
      setLoadingActiveError(activeErr);
    } else {
      setLoadingActiveError(null);
    }
  }, [activeErr]);

  useEffect(() => {
    // Clear session if job completed
    const status = activeJob?.status;
    if (status && (status === 'completed' || status === 'failed' || status === 'cancelled')) {
      try { sessionStorage.removeItem(SESSION_KEY); } catch {}
    }
  }, [activeJob]);

  useEffect(() => {
    // If backend returns 404 for the active job (e.g., mock server restarted),
    // clear the active job to stop polling and let the user resubmit.
    if (activeErr && /404/.test(String(activeErr?.message || ''))) {
      setActiveJobId('');
      try { sessionStorage.removeItem(SESSION_KEY); } catch {}
    }
  }, [activeErr]);

  const getModelInfo = (job) => {
    const id = job.id || '';
    if (id.includes('llama')) return { name: 'Llama 3.2-1B', region: 'US', color: 'bg-blue-100 text-blue-800' };
    if (id.includes('qwen')) return { name: 'Qwen 2.5-1.5B', region: 'China', color: 'bg-red-100 text-red-800' };
    if (id.includes('mistral')) return { name: 'Mistral 7B', region: 'EU', color: 'bg-green-100 text-green-800' };
    return { name: 'Unknown', region: 'Unknown', color: 'bg-gray-100 text-gray-800' };
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed': return 'bg-green-100 text-green-800';
      case 'running': return 'bg-yellow-100 text-yellow-800';
      case 'failed': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const truncateMiddle = (str, head = 6, tail = 4) => {
    if (!str || typeof str !== 'string') return '—';
    if (str.length <= head + tail + 1) return str;
    return `${str.slice(0, head)}…${str.slice(-tail)}`;
  };

  const timeAgo = (ts) => {
    if (!ts) return '';
    try {
      const d = new Date(ts).getTime();
      const diff = Date.now() - d;
      const sec = Math.floor(diff / 1000);
      if (sec < 60) return `${sec}s ago`;
      const min = Math.floor(sec / 60);
      if (min < 60) return `${min}m ago`;
      const hr = Math.floor(min / 60);
      if (hr < 24) return `${hr}h ago`;
      const day = Math.floor(hr / 24);
      return `${day}d ago`;
    } catch { return String(ts); }
  };

  const VerifyBadge = ({ exec }) => {
    const verified = exec?.region_verified === true || String(exec?.verification_status || '').toLowerCase() === 'verified';
    const method = (exec?.verification_method || '').toLowerCase();
    const needsProbe = method === 'needs_probe' || (!verified && method === '');
    const label = verified ? (method === 'probe' ? 'probe-verified' : 'strict-verified') : (needsProbe ? 'needs-probe' : (method || 'unverified'));
    const cls = verified
      ? 'bg-green-100 text-green-800'
      : needsProbe
      ? 'bg-amber-100 text-amber-800'
      : 'bg-slate-100 text-slate-700';
    return <span className={`text-xs px-2 py-0.5 rounded-full ${cls}`}>{label}</span>;
  };

  const groupedJobs = biasJobs.reduce((acc, job) => {
    const model = getModelInfo(job);
    if (!acc[model.region]) acc[model.region] = [];
    acc[model.region].push({ ...job, modelInfo: model });
    return acc;
  }, {});

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-beacon-600"></div>
        <span className="ml-3 text-slate-600">Loading bias detection results...</span>
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
        
        {/* Job Submission Errors */}
        {jobSubmissionError && (
          <ErrorMessage 
            error={jobSubmissionError} 
            onRetry={() => {
              setJobSubmissionError(null);
              // Retry logic would go here
            }}
            retryAfter={jobSubmissionError.retry_after}
          />
        )}
        
        {/* Job List Errors */}
        {jobListError && (
          <ErrorMessage 
            error={jobListError} 
            onRetry={() => {
              setJobListError(null);
              fetchBiasJobs();
            }}
            retryAfter={jobListError.retry_after}
          />
        )}
        <p className="text-slate-600 text-sm max-w-3xl">
          Run targeted prompts to detect bias across regions and models. Choose your questions and providers,
          then submit to start a job. You’ll see live per‑region progress and a link to full results.
        </p>
      </div>

      {/* Wallet Authentication */}
      <WalletConnection />

      {/* Submit Card */}
      <section className="bg-white rounded-lg border p-6">
        <div className="space-y-6">
          {/* Header */}
          <div className="flex items-start justify-between">
            <div>
              <h2 className="text-lg font-semibold">Run Bias Detection</h2>
              <p className="text-sm text-slate-600 mt-1">
                Configure your bias detection job across multiple regions and models.
              </p>
            </div>
            <div className="flex items-center gap-2">
              <label className="text-sm font-medium text-slate-700">Multi-Region</label>
              <button
                onClick={() => setIsMultiRegion(!isMultiRegion)}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  isMultiRegion ? 'bg-beacon-600' : 'bg-slate-200'
                }`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    isMultiRegion ? 'translate-x-6' : 'translate-x-1'
                  }`}
                />
              </button>
            </div>
          </div>

          {/* Region Selection */}
          <div className="space-y-3">
            <h3 className="text-sm font-medium text-slate-900">Select Regions</h3>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
              {availableRegions.map((region) => (
                <div
                  key={region.code}
                  className={`relative border rounded-lg p-3 cursor-pointer transition-all ${
                    selectedRegions.includes(region.code)
                      ? 'border-beacon-300 bg-beacon-50'
                      : 'border-slate-200 hover:border-slate-300'
                  }`}
                  onClick={() => handleRegionToggle(region.code)}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          checked={selectedRegions.includes(region.code)}
                          onChange={() => handleRegionToggle(region.code)}
                          className="rounded border-slate-300 text-beacon-600 focus:ring-beacon-500"
                        />
                        <span className="font-medium text-slate-900">{region.code}</span>
                      </div>
                      <div className="mt-1 text-sm text-slate-600">{region.name}</div>
                      <div className="mt-1 text-xs text-slate-500">{region.model}</div>
                    </div>
                    <div className="text-right">
                      <div className="text-xs text-slate-500">Est. cost</div>
                      <div className="text-sm font-medium text-slate-900">${region.cost}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Multi-Region Configuration */}
          {isMultiRegion && (
            <div className="space-y-4 border-t pt-4">
              <h3 className="text-sm font-medium text-slate-900">Multi-Region Configuration</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Minimum Regions Required
                  </label>
                  <select
                    value={minRegions}
                    onChange={(e) => setMinRegions(parseInt(e.target.value))}
                    className="w-full border border-slate-300 rounded-md px-3 py-2 text-sm focus:ring-beacon-500 focus:border-beacon-500"
                  >
                    {Array.from({ length: selectedRegions.length }, (_, i) => i + 1).map(num => (
                      <option key={num} value={num}>{num}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Minimum Success Rate
                  </label>
                  <select
                    value={minSuccessRate}
                    onChange={(e) => setMinSuccessRate(parseFloat(e.target.value))}
                    className="w-full border border-slate-300 rounded-md px-3 py-2 text-sm focus:ring-beacon-500 focus:border-beacon-500"
                  >
                    <option value={0.33}>33% (1/3 regions)</option>
                    <option value={0.67}>67% (2/3 regions)</option>
                    <option value={1.0}>100% (all regions)</option>
                  </select>
                </div>
              </div>
            </div>
          )}

          {/* Job Summary */}
          <div className="bg-slate-50 rounded-lg p-4 space-y-2">
            <h3 className="text-sm font-medium text-slate-900">Job Summary</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <span className="text-slate-600">Questions:</span>
                <span className="ml-1 font-medium">{readSelectedQuestions().length}</span>
              </div>
              <div>
                <span className="text-slate-600">Regions:</span>
                <span className="ml-1 font-medium">{selectedRegions.length}</span>
              </div>
              <div>
                <span className="text-slate-600">Type:</span>
                <span className="ml-1 font-medium">{isMultiRegion ? 'Multi-Region' : 'Single-Region'}</span>
              </div>
              <div>
                <span className="text-slate-600">Est. Cost:</span>
                <span className="ml-1 font-medium">${calculateEstimatedCost()}</span>
              </div>
            </div>
          </div>

          {/* Submit Button */}
          <div className="flex items-center justify-between">
            <div className="text-xs text-slate-600">
              {readSelectedQuestions().length === 0 && (
                <span className="text-amber-600">⚠ Select questions on the Questions page first</span>
              )}
            </div>
            <div className="flex items-center gap-3">
              {activeJobId && (
                <div className="text-xs text-slate-500">
                  Active: <span className="font-mono">{activeJobId.slice(0, 8)}...</span>
                </div>
              )}
              {(() => {
                const hasWallet = isMetaMaskInstalled();
                const walletStatus = getWalletAuthStatus();
                const disabled = !hasWallet || !walletStatus.isAuthorized || readSelectedQuestions().length === 0 || isSubmitting;
                
                // Show "Refresh" if there's an active job, otherwise show submit button
                const showRefresh = activeJobId && !isSubmitting;
                
                return (
                  <button
                    onClick={showRefresh ? () => window.location.reload() : onSubmitJob}
                    disabled={disabled && !showRefresh}
                    className={`px-6 py-2 rounded-md text-sm font-medium transition-all duration-150 ${
                      buttonClicked && !showRefresh
                        ? 'bg-beacon-800 text-white transform scale-95'
                        : disabled && !showRefresh
                        ? 'bg-slate-300 text-slate-600 cursor-not-allowed' 
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
              <div className="text-xs text-slate-600 space-y-2">
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
        <section className="bg-white rounded-lg border">
          <div className="px-4 py-3 border-b">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold">Live Progress</h2>
              <div className="text-xs text-slate-500">{loadingActive ? 'Refreshing…' : activeJob?.status || '—'}</div>
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
                      <span className="text-slate-500">{completed}/{total} regions · {pct}%</span>
                    </div>
                    <div className="w-full h-3 bg-slate-100 rounded overflow-hidden">
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
                      <span className="text-slate-600">Completed: {completed}</span>
                    </div>
                    <div className="flex items-center gap-1">
                      <div className="w-2 h-2 bg-yellow-500 rounded"></div>
                      <span className="text-slate-600">Running: {running}</span>
                    </div>
                    <div className="flex items-center gap-1">
                      <div className="w-2 h-2 bg-red-500 rounded"></div>
                      <span className="text-slate-600">Failed: {failed}</span>
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
                      className="text-sm px-2 py-1 border border-amber-300 rounded bg-white hover:bg-amber-100"
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

            <div className="border rounded">
              <div className="grid grid-cols-7 text-xs bg-slate-50 text-slate-600">
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
                  <div key={r} className="grid grid-cols-7 text-sm border-t hover:bg-slate-50">
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

            <div className="flex items-center justify-end">
              <button onClick={refetchActive} className="px-3 py-1.5 bg-green-600 text-white rounded text-sm hover:bg-green-700">Refresh</button>
              {activeJob?.id && (
                <Link to={`/jobs/${activeJob.id}`} className="ml-2 text-sm text-beacon-600 underline decoration-dotted">View full results</Link>
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
              <div key={region} className="bg-white rounded-lg border p-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-medium text-slate-900">{region} Models</h3>
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
        <div className="bg-white rounded-lg border">
          <div className="px-6 py-4 border-b">
            <h2 className="text-lg font-medium text-slate-900">Bias Detection Jobs</h2>
          </div>
          <div className="divide-y">
            {biasJobs.map(job => {
              const modelInfo = getModelInfo(job);
              return (
                <div key={job.id} className="px-6 py-4 hover:bg-slate-50">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${modelInfo.color}`}>
                        {modelInfo.name}
                      </span>
                      <span className="text-sm text-slate-600">{modelInfo.region}</span>
                      <Link
                        to={`/jobs/${job.id}`}
                        className="font-medium text-slate-900 hover:text-beacon-600"
                      >
                        {job.id}
                      </Link>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(job.status)}`}>
                        {job.status}
                      </span>
                      <span className="text-sm text-slate-500">
                        {job.created_at ? new Date(job.created_at).toLocaleDateString() : 'Unknown'}
                      </span>
                    </div>
                  </div>
                  {job.benchmark?.description && (
                    <p className="text-sm text-slate-600 mt-1">{job.benchmark.description}</p>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Quick Actions */}
      <div className="bg-white rounded-lg border p-6">
        <h3 className="text-lg font-medium text-slate-900 mb-4">Quick Actions</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Link
            to="/demo-results"
            className="flex items-center gap-3 p-4 border border-slate-200 rounded-lg hover:border-beacon-300 hover:bg-beacon-50"
          >
            <div className="flex-shrink-0">
              <svg className="h-6 w-6 text-beacon-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
            </div>
            <div>
              <h4 className="font-medium text-slate-900">Compare Results</h4>
              <p className="text-sm text-slate-600">Analyze bias differences between models</p>
            </div>
          </Link>
          
          <div className="flex items-center gap-3 p-4 border border-slate-200 rounded-lg opacity-50 cursor-not-allowed">
            <div className="flex-shrink-0">
              <svg className="h-6 w-6 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
            </div>
            <div>
              <h4 className="font-medium text-slate-400">Submit New Benchmark</h4>
              <p className="text-sm text-slate-500">Run bias detection on new models (v1 Feature)</p>
            </div>
          </div>
          
          <div className="flex items-center gap-3 p-4 border border-slate-200 rounded-lg opacity-50">
            <div className="flex-shrink-0">
              <svg className="h-6 w-6 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <div>
              <h4 className="font-medium text-slate-400">Export Results</h4>
              <p className="text-sm text-slate-500">Download bias analysis data (Coming Soon)</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
