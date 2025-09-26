import React from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '../state/useQuery.js';
import { listJobs } from '../lib/api/runner/jobs.js';

export default function Jobs() {
  const [limit, setLimit] = React.useState(50);
  const { data, error, loading, refetch } = useQuery(
    ['jobs', limit],
    () => listJobs({ limit }),
    { interval: 30000 }
  );

  const jobs = data?.jobs || [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">Jobs</h2>
        <div className="flex items-center gap-2 text-sm">
          <Link to="/jobs/new" className="px-2 py-1 rounded bg-beacon-600 text-white">New Job</Link>
          <label className="text-slate-600">Limit</label>
          <select
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            className="border rounded px-2 py-1"
          >
            {[10,25,50,100,200].map(v => <option key={v} value={v}>{v}</option>)}
          </select>
          <button onClick={refetch} className="px-3 py-1.5 bg-green-600 text-white rounded hover:bg-green-700">Refresh</button>
        </div>
      </div>

      {loading && <div className="text-sm text-slate-500">Loading jobs…</div>}
      {error && <div className="text-sm text-red-600">{String(error.message || error)}</div>}

      {!loading && !error && jobs.length === 0 && (
        <div className="text-sm text-slate-500">No jobs yet.</div>
      )}

      {!loading && !error && jobs.length > 0 && (
        <div className="overflow-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="text-left text-slate-600">
                <th className="px-2 py-2">ID</th>
                <th className="px-2 py-2">Status</th>
                <th className="px-2 py-2">Created At</th>
              </tr>
            </thead>
            <tbody>
              {jobs.map((j) => (
                <tr key={j.id} className="border-t hover:bg-slate-50">
                  <td className="px-2 py-2 font-mono">
                    <Link className="text-beacon-600 underline decoration-dotted" to={`/jobs/${encodeURIComponent(j.id)}`}>{j.id}</Link>
                  </td>
                  <td className="px-2 py-2">
                    <span className="inline-block px-2 py-0.5 rounded bg-slate-100 text-slate-700">{j.status}</span>
                  </td>
                  <td className="px-2 py-2">{j.created_at}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
