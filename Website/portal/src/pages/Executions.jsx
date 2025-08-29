import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useQuery } from '../state/useQuery.js';
import { getExecutions } from '../lib/api.js';
import CopyButton from '../components/CopyButton.jsx';

function StatusPill({ value }) {
  const val = typeof value === 'string' ? value.toLowerCase() : value;
  const ok = val === true || val === 'ok' || val === 'healthy' || val === 'up' || val === 'ready' || val === 'running' || val === 'success' || val === 'completed';
  const warn = val === 'degraded' || val === 'warning' || val === 'partial' || val === 'pending';
  const bad = val === false || val === 'down' || val === 'error' || val === 'failed' || val === 'unhealthy';
  const cls = ok
    ? 'bg-green-100 text-green-700'
    : warn
    ? 'bg-amber-100 text-amber-800'
    : bad
    ? 'bg-red-100 text-red-700'
    : 'bg-slate-100 text-slate-700';
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
        <div className="text-sm text-slate-500">
          {loading ? 'Refreshing…' : error ? 'Backend unavailable' : `${filtered.length}${(jobFilter || regionFilter) ? ` of ${executions.length}` : ''} shown`}
        </div>
      </div>

      {(jobFilter || regionFilter) && (
        <div className="flex items-center gap-2 text-xs">
          <span className="px-2 py-0.5 rounded bg-slate-100 text-slate-700">job: <code className="font-mono">{jobFilter || '—'}</code></span>
          <span className="px-2 py-0.5 rounded bg-slate-100 text-slate-700">region: <code className="font-mono">{regionFilter || '—'}</code></span>
          <Link to="/executions" className="text-beacon-600 underline decoration-dotted">Clear filters</Link>
        </div>
      )}

      {loading ? (
        <div className="bg-white border rounded overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-slate-600">
              <tr>
                <th className="text-left px-3 py-2">ID</th>
                <th className="text-left px-3 py-2">Job</th>
                <th className="text-left px-3 py-2">Status</th>
                <th className="text-left px-3 py-2">Started</th>
                <th className="text-left px-3 py-2">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {Array.from({ length: 10 }).map((_, i) => (
                <tr key={i} className="animate-pulse">
                  <td className="px-3 py-2"><div className="h-3 w-28 bg-slate-200 rounded" /></td>
                  <td className="px-3 py-2"><div className="h-3 w-24 bg-slate-200 rounded" /></td>
                  <td className="px-3 py-2"><div className="h-5 w-16 bg-slate-200 rounded-full" /></td>
                  <td className="px-3 py-2"><div className="h-3 w-20 bg-slate-200 rounded" /></td>
                  <td className="px-3 py-2"><div className="h-3 w-32 bg-slate-200 rounded" /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : error ? (
        <div className="bg-white border rounded p-3 text-sm text-red-600">Backend unavailable - Executions service offline</div>
      ) : executions.length === 0 ? (
        <div className="bg-white border rounded p-3 text-sm text-slate-500">No executions yet.</div>
      ) : filtered.length === 0 ? (
        <div className="bg-white border rounded p-3 text-sm text-slate-500">No executions match current filters.</div>
      ) : (
        <div className="bg-white border rounded overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-slate-600">
              <tr>
                <th className="text-left px-3 py-2">ID</th>
                <th className="text-left px-3 py-2">Job</th>
                <th className="text-left px-3 py-2">Status</th>
                <th className="text-left px-3 py-2">Started</th>
                <th className="text-left px-3 py-2">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {filtered.map((e) => {
                const id = e?.id || e?.execution_id || '—';
                const jobId = e?.job_id || e?.jobId || e?.job?.id;
                const status = e?.status || e?.state || 'unknown';
                const created = e?.created_at || e?.started_at || e?.createdAt || e?.startedAt;
                return (
                  <tr key={id} className="hover:bg-slate-50">
                    <td className="px-3 py-2 font-mono whitespace-nowrap" title={id}>{truncateMiddle(String(id))}</td>
                    <td className="px-3 py-2 font-mono whitespace-nowrap" title={jobId || ''}>
                      {jobId ? (
                        <Link className="text-beacon-600 underline decoration-dotted" to={`/jobs/${encodeURIComponent(jobId)}`}>{truncateMiddle(String(jobId))}</Link>
                      ) : '—'}
                    </td>
                    <td className="px-3 py-2"><StatusPill value={status} /></td>
                    <td className="px-3 py-2 text-xs" title={formatDate(created)}>{timeAgo(created)}</td>
                    <td className="px-3 py-2 flex items-center gap-2">
                      {/* Execution detail route is not implemented yet; omit link to avoid 404 */}
                      {id && <CopyButton text={String(id)} label="Copy ID" />}
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
