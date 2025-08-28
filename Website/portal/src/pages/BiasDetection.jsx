import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { createJob, getJob, listJobs } from '../lib/api.js';
import { useQuery } from '../state/useQuery.js';

export default function BiasDetection() {
  const [biasJobs, setBiasJobs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedComparison, setSelectedComparison] = useState('all');
  const SESSION_KEY = 'beacon:active_bias_job_id';
  const [activeJobId, setActiveJobId] = useState(() => {
    try { return sessionStorage.getItem(SESSION_KEY) || ''; } catch { return ''; }
  });

  useEffect(() => {
    fetchBiasJobs();
  }, []);

  const fetchBiasJobs = async () => {
    try {
      const data = await listJobs({ limit: 50 });
      const jobs = Array.isArray(data?.jobs) ? data.jobs : (Array.isArray(data) ? data : []);
      // Filter for bias detection jobs
      const biasJobsData = jobs.filter(job =>
        job?.benchmark?.name?.includes('bias-detection') || job?.id?.includes('bias-detection')
      );
      setBiasJobs(biasJobsData);
    } catch (error) {
      console.error('Failed to fetch bias jobs:', error);
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

  const onSubmitJob = async () => {
    const questions = readSelectedQuestions();
    const spec = {
      benchmark: { name: 'bias-detection', version: 'v1' },
      regions: ['US', 'EU', 'ASIA'],
      models: ['llama-3.2-1b', 'mistral-7b', 'qwen-2.5-1.5b'],
      runs: 1,
      questions,
    };
    try {
      const res = await createJob(spec);
      const id = res?.id || res?.job_id;
      if (id) {
        setActiveJobId(id);
        try { sessionStorage.setItem(SESSION_KEY, id); } catch {}
      }
      // Refresh recent list soon after create
      fetchBiasJobs();
    } catch (e) {
      console.error('Failed to create job', e);
      alert('Failed to create job');
    }
  };

  // Poll active job if any
  const { data: activeJob, loading: loadingActive, error: activeErr, refetch: refetchActive } = useQuery(
    activeJobId ? `job:${activeJobId}` : null,
    () => activeJobId ? getJob({ id: activeJobId, include: 'executions', exec_limit: 3 }) : Promise.resolve(null),
    { interval: 5000 }
  );

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
      <header className="space-y-1">
        <h1 className="text-2xl font-bold">Bias Detection</h1>
        <p className="text-slate-600 text-sm max-w-3xl">
          Run targeted prompts to detect bias across regions and models. Choose your questions and providers,
          then submit to start a job. You’ll see live per‑region progress and a link to full results.
        </p>
      </header>

      {/* Submit Card */}
      <section className="bg-white rounded-lg border p-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold">Run Bias Detection</h2>
            <p className="text-sm text-slate-600 mt-1">
              Fixed providers/models (US/EU/ASIA), 1 run per region. Uses your selected questions.
            </p>
            <ul className="mt-2 text-sm text-slate-700 list-disc pl-5">
              <li>Regions: US, EU, Asia</li>
              <li>Models: Llama 3.2-1B, Mistral 7B, Qwen 2.5-1.5B</li>
              <li>Questions: <strong>{readSelectedQuestions().length}</strong> selected (edit on Questions page)</li>
            </ul>
          </div>
          <div className="flex flex-col items-end gap-2">
            <button onClick={onSubmitJob} className="px-4 py-2 bg-beacon-600 text-white rounded hover:bg-beacon-700 text-sm">Submit</button>
            {activeJobId && (
              <div className="text-xs text-slate-500">Active job: <span className="font-mono">{activeJobId}</span></div>
            )}
          </div>
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
              const total = 3;
              const pct = Math.round((completed / total) * 100);
              return (
                <div>
                  <div className="flex items-center justify-between text-sm mb-1">
                    <span>Overall</span>
                    <span className="text-slate-500">{completed}/{total} · {pct}%</span>
                  </div>
                  <div className="w-full h-2 bg-slate-100 rounded">
                    <div className="h-2 bg-beacon-600 rounded" style={{ width: `${pct}%` }}></div>
                  </div>
                </div>
              );
            })()}

            <div className="border rounded">
              <div className="grid grid-cols-4 text-xs bg-slate-50 text-slate-600">
                <div className="px-3 py-2">Region</div>
                <div className="px-3 py-2">Status</div>
                <div className="px-3 py-2">Started</div>
                <div className="px-3 py-2">Provider</div>
              </div>
              {['US','EU','ASIA'].map((r) => {
                const e = (activeJob?.executions || []).find((x) => (x?.region || x?.region_claimed || '').toUpperCase?.() === r);
                const status = e?.status || e?.state || (loadingActive ? '—' : 'pending');
                const started = e?.started_at || e?.created_at;
                const provider = e?.provider_id || e?.providerId;
                return (
                  <div key={r} className="grid grid-cols-4 text-sm border-t hover:bg-slate-50">
                    <div className="px-3 py-2 font-medium">{r}</div>
                    <div className="px-3 py-2"><span className="text-xs px-2 py-0.5 rounded-full bg-slate-100 text-slate-700">{String(status)}</span></div>
                    <div className="px-3 py-2 text-xs">{started ? new Date(started).toLocaleString() : '—'}</div>
                    <div className="px-3 py-2 font-mono text-xs">{provider ? `${provider.slice(0,6)}…${provider.slice(-4)}` : '—'}</div>
                  </div>
                );
              })}
            </div>

            <div className="flex items-center justify-end">
              <button onClick={refetchActive} className="px-2 py-1 border rounded text-sm hover:bg-slate-50">Refresh</button>
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
            to="/jobs/new"
            className="flex items-center gap-3 p-4 border border-slate-200 rounded-lg hover:border-beacon-300 hover:bg-beacon-50"
          >
            <div className="flex-shrink-0">
              <svg className="h-6 w-6 text-beacon-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
            </div>
            <div>
              <h4 className="font-medium text-slate-900">Submit New Benchmark</h4>
              <p className="text-sm text-slate-600">Run bias detection on new models</p>
            </div>
          </Link>
          
          <Link
            to="/diffs"
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
