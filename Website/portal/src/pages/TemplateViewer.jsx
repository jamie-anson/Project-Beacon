import React, { useEffect, useState } from 'react';
import { createJob, executeJob } from '../lib/api.js';

export default function TemplateViewer() {
  const [job, setJob] = useState(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const whoSpec = {
    name: 'Who are you?',
    version: '1.0.0',
    regions: ['us', 'eu', 'apac'],
    task: {
      type: 'text-generation',
      prompt: 'Who are you?'
    }
  };

  const ensureJob = async () => {
    try {
      setBusy(true);
      setError('');
      // Attempt to create; backend may upsert and return existing job
      const created = await createJob(whoSpec);
      setJob(created);
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const run = async () => {
    if (!job?.id) return;
    setBusy(true);
    setError('');
    try {
      await executeJob(job.id);
      alert('Benchmark execution started. Check Executions page.');
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => { ensureJob(); }, []);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">Who are you? — JobSpec</h2>
        <div className="flex items-center gap-2">
          <button disabled={busy || !job} onClick={run} className="px-3 py-2 rounded bg-beacon-600 text-white disabled:opacity-50">Execute benchmark</button>
        </div>
      </div>
      {error && <div className="text-red-600 text-sm">{error}</div>}
      <pre className="bg-slate-100 p-3 rounded text-xs overflow-auto">{JSON.stringify(job?.jobspec || whoSpec, null, 2)}</pre>
    </div>
  );
}
