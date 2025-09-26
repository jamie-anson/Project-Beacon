import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useQuery } from '../state/useQuery.js';
import { getExecutions } from '../lib/api/runner/executions.js';
import CopyButton from '../components/CopyButton.jsx';

function StatusPill({ value }) {
  const val = typeof value === 'string' ? value.toLowerCase() : value;
  const ok = val === true || val === 'ok' || val === 'healthy' || val === 'up' || val === 'ready' || val === 'running' || val === 'success' || val === 'completed';
  const warn = val === 'degraded' || val === 'warning' || val === 'partial' || val === 'pending';
  const bad = val === false || val === 'down' || val === 'error' || val === 'failed' || val === 'unhealthy';
  const cls = ok
    ? 'bg-green-900/20 text-green-400'
    : warn
    ? 'bg-yellow-900/20 text-yellow-400'
    : bad
    ? 'bg-red-900/20 text-red-400'
    : 'bg-gray-700 text-gray-300';
  const label = typeof value === 'boolean' ? (value ? 'ok' : 'down') : (String(value || '—'));
  return <span className={`text-xs px-2 py-0.5 rounded-full ${cls}`}>{label}</span>;
}

function truncateMiddle(str, head = 6, tail = 6) {
  if (!str || typeof str !== 'string') return '—';
  if (str.length <= head + tail + 1) return str;
  return `${str.slice(0, head)}…${str.slice(-tail)}`;
}

function formatDate(ts) {
  if (!ts) return '';
  try { return new Date(ts).toLocaleString(); } catch { return String(ts); }
}

function timeAgo(ts) {
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
}

export default function Executions() {
  const location = useLocation();
  const params = new URLSearchParams(location.search);
  const jobFilter = (params.get('job') || '').trim();
  const regionFilter = (params.get('region') || '').trim().toUpperCase();

  const { data, loading, error } = useQuery('executions', () => getExecutions({ limit: 100 }), { interval: 5000 });
  const executions = Array.isArray(data) ? data : [];

  const filtered = React.useMemo(() => {
    return executions.filter((e) => {
      const jobId = e?.job_id || e?.jobId || e?.job?.id;
      const region = (e?.region || e?.region_claimed || '').toUpperCase?.();
      const okJob = jobFilter ? String(jobId) === jobFilter : true;
      const okRegion = regionFilter ? region === regionFilter : true;
      return okJob && okRegion;
    });
  }, [executions, jobFilter, regionFilter]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">Executions</h2>
        <div className="text-sm text-gray-400">
          {loading ? 'Refreshing…' : error ? 'Backend unavailable' : `${filtered.length}${(jobFilter || regionFilter) ? ` of ${executions.length}` : ''} shown`}
        </div>
      </div>

      {(jobFilter || regionFilter) && (
        <div className="flex items-center gap-2 text-xs">
          <span className="px-2 py-0.5 rounded bg-gray-700 text-gray-300">job: <code className="font-mono">{jobFilter || '—'}</code></span>
          <span className="px-2 py-0.5 rounded bg-gray-700 text-gray-300">region: <code className="font-mono">{regionFilter || '—'}</code></span>
          <Link to="/executions" className="text-beacon-600 underline decoration-dotted">Clear filters</Link>
        </div>
      )}

      {loading ? (
        <div className="bg-gray-800 border border-gray-700 rounded overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-900 text-gray-300">
              <tr>
                <th className="text-left px-3 py-2">ID</th>
                <th className="text-left px-3 py-2">Job</th>
                <th className="text-left px-3 py-2">Status</th>
                <th className="text-left px-3 py-2">Started</th>
                <th className="text-left px-3 py-2">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-700">
              {Array.from({ length: 10 }).map((_, i) => (
                <tr key={i} className="animate-pulse">
                  <td className="px-3 py-2"><div className="h-3 w-28 bg-gray-700 rounded" /></td>
                  <td className="px-3 py-2"><div className="h-3 w-24 bg-gray-700 rounded" /></td>
                  <td className="px-3 py-2"><div className="h-5 w-16 bg-gray-700 rounded-full" /></td>
                  <td className="px-3 py-2"><div className="h-3 w-20 bg-gray-700 rounded" /></td>
                  <td className="px-3 py-2"><div className="h-3 w-32 bg-gray-700 rounded" /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : error ? (
        <div className="bg-red-900/20 border border-red-700 rounded p-3 text-sm text-red-400">Backend unavailable - Executions service offline</div>
      ) : executions.length === 0 ? (
        <div className="bg-gray-800 border border-gray-700 rounded p-3 text-sm text-gray-400">No executions yet.</div>
      ) : filtered.length === 0 ? (
        <div className="bg-gray-800 border border-gray-700 rounded p-3 text-sm text-gray-400">No executions match current filters.</div>
      ) : (
        <div className="bg-gray-800 border border-gray-700 rounded overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-900 text-gray-300">
              <tr>
                <th className="text-left px-3 py-2">ID</th>
                <th className="text-left px-3 py-2">Job</th>
                <th className="text-left px-3 py-2">Status</th>
                <th className="text-left px-3 py-2">Started</th>
                <th className="text-left px-3 py-2">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-700">
              {filtered.map((e) => {
                const id = e?.id || e?.execution_id || '—';
                const jobId = e?.job_id || e?.jobId || e?.job?.id;
                const status = e?.status || e?.state || 'unknown';
                const created = e?.created_at || e?.started_at || e?.createdAt || e?.startedAt;
                return (
                  <tr key={id} className="hover:bg-gray-700 text-gray-200">
                    <td className="px-3 py-2 font-mono whitespace-nowrap" title={id}>{truncateMiddle(String(id))}</td>
                    <td className="px-3 py-2 font-mono whitespace-nowrap" title={jobId || ''}>
                      {jobId ? (
                        <Link className="text-beacon-600 underline decoration-dotted" to={`/jobs/${encodeURIComponent(jobId)}`}>{truncateMiddle(String(jobId))}</Link>
                      ) : '—'}
                    </td>
                    <td className="px-3 py-2"><StatusPill value={status} /></td>
                    <td className="px-3 py-2 text-xs" title={formatDate(created)}>{timeAgo(created)}</td>
                    <td className="px-3 py-2 flex items-center gap-2">
                      {id && (
                        <>
                          <Link 
                            to={`/executions/${encodeURIComponent(id)}`}
                            className="text-beacon-600 hover:text-beacon-700 underline decoration-dotted text-xs"
                          >
                            View Receipt
                          </Link>
                          <CopyButton text={String(id)} label="Copy ID" />
                        </>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
