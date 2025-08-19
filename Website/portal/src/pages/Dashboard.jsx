import React from 'react';
import { useQuery } from '../state/useQuery.js';
import { getHealth, getExecutions, getDiffs, getTransparencyRoot, listJobs } from '../lib/api.js';
import { Link } from 'react-router-dom';
import useWs from '../state/useWs.js';
import { useToast } from '../state/toast.js';
import ActivityFeed from '../components/ActivityFeed.jsx';
import CopyButton from '../components/CopyButton.jsx';

export default function Dashboard() {
  const { data: health, loading: loadingHealth } = useQuery('health', getHealth, { interval: 30000 });
  const { data: executions, loading: loadingExecs } = useQuery('executions:latest', () => getExecutions({ limit: 5 }), { interval: 15000 });
  const { data: diffs, loading: loadingDiffs } = useQuery('diffs:latest', () => getDiffs({ limit: 5 }), { interval: 20000 });

  const { add: addToast } = useToast();
  const [events, setEvents] = React.useState(() => {
    try {
      const raw = sessionStorage.getItem('beacon:activity');
      return raw ? JSON.parse(raw) : [];
    } catch {
      return [];
    }
  });

  const onWsMessage = React.useCallback((msg) => {
    const type = msg?.event || msg?.type;
    if (type !== 'transparency.entry_appended') return;
    const ev = {
      timestamp: msg?.timestamp || new Date().toISOString(),
      merkle_root: msg?.merkle_root || msg?.root,
      ipfs_cid: msg?.ipfs_cid || msg?.cid,
      execution_id: msg?.execution_id || msg?.executionId,
    };
    setEvents((prev) => [ev, ...prev].slice(0, 20));
    addToast({
      title: 'Transparency log updated',
      message: ev.execution_id ? `Exec ${ev.execution_id} anchored` : `Root ${ev.merkle_root?.slice?.(0, 8)}…`,
    });
  }, [addToast]);

  useWs('/ws', { onMessage: onWsMessage });

  // Persist events
  React.useEffect(() => {
    try { sessionStorage.setItem('beacon:activity', JSON.stringify(events)); } catch {}
  }, [events]);

  // Transparency root and recent jobs
  const { data: tRoot, loading: loadingRoot } = useQuery('transparency:root', getTransparencyRoot, { interval: 15000 });
  const { data: recentJobs, loading: loadingJobs } = useQuery('jobs:recent', () => listJobs({ limit: 5 }), { interval: 20000 });

  return (
    <div className="space-y-6">
      <section>
        <h2 className="text-xl font-semibold">Live activity</h2>
        <ActivityFeed events={events} />
      </section>
      <section>
        <h2 className="text-xl font-semibold">Transparency root</h2>
        {loadingRoot ? (
          <div className="bg-white border rounded p-3 animate-pulse">
            <div className="h-4 bg-slate-200 rounded w-2/3"></div>
            <div className="h-3 bg-slate-100 rounded w-1/3 mt-2"></div>
          </div>
        ) : tRoot ? (
          <div className="bg-white border rounded p-3 text-sm">
            <div className="flex items-center gap-2">
              <div>Root: <span className="font-mono break-all">{tRoot.root || tRoot.merkle_root || '—'}</span></div>
              {(tRoot.root || tRoot.merkle_root) && (
                <CopyButton text={tRoot.root || tRoot.merkle_root} label="Copy root" />
              )}
            </div>
            {tRoot.sequence != null && (
              <div className="text-xs text-slate-500 mt-1">Seq #{tRoot.sequence}{tRoot.updated_at ? ` · ${tRoot.updated_at}` : ''}</div>
            )}
          </div>
        ) : (
          <div className="text-sm text-slate-500">Loading…</div>
        )}
      </section>
      <section>
        <h2 className="text-xl font-semibold">Recent jobs</h2>
        <div className="bg-white border rounded divide-y">
          {loadingJobs ? (
            Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="p-3 animate-pulse">
                <div className="h-4 bg-slate-200 rounded w-1/2"></div>
                <div className="h-3 bg-slate-100 rounded w-1/3 mt-2"></div>
              </div>
            ))
          ) : (recentJobs || []).slice(0, 5).map((j) => (
            <div key={j.id} className="p-3 text-sm flex items-center justify-between">
              <div className="min-w-0">
                <div className="font-mono truncate">{j.id}</div>
                <div className="text-xs text-slate-500">{j.created_at}</div>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-xs px-2 py-0.5 rounded-full bg-slate-100 text-slate-700">{j.status}</span>
                <Link className="text-beacon-600 text-xs underline decoration-dotted" to={`/jobs/${encodeURIComponent(j.id)}`}>View</Link>
                <CopyButton text={j.id} label="Copy ID" />
              </div>
            </div>
          ))}
          {(!recentJobs || recentJobs.length === 0) && (
            <div className="p-3 text-sm text-slate-500">No jobs yet.</div>
          )}
        </div>
      </section>
      <section>
        <h2 className="text-xl font-semibold">System status</h2>
        {loadingHealth ? (
          <div className="bg-white border rounded p-3 animate-pulse">
            <div className="h-4 bg-slate-200 rounded w-2/3"></div>
            <div className="h-4 bg-slate-100 rounded w-full mt-2"></div>
            <div className="h-4 bg-slate-100 rounded w-5/6 mt-2"></div>
          </div>
        ) : (
          <pre className="bg-slate-100 p-3 rounded text-xs overflow-auto">{JSON.stringify(health || {}, null, 2)}</pre>
        )}
      </section>
      <section>
        <h2 className="text-xl font-semibold">Recent executions</h2>
        {loadingExecs ? (
          <div className="bg-white border rounded p-3 animate-pulse">
            <div className="h-4 bg-slate-200 rounded w-1/2"></div>
            <div className="h-4 bg-slate-100 rounded w-5/6 mt-2"></div>
          </div>
        ) : (
          <pre className="bg-slate-100 p-3 rounded text-xs overflow-auto">{JSON.stringify(executions || [], null, 2)}</pre>
        )}
      </section>
      <section>
        <h2 className="text-xl font-semibold">Recent diffs</h2>
        {loadingDiffs ? (
          <div className="bg-white border rounded p-3 animate-pulse">
            <div className="h-4 bg-slate-200 rounded w-1/2"></div>
            <div className="h-4 bg-slate-100 rounded w-5/6 mt-2"></div>
          </div>
        ) : (
          <pre className="bg-slate-100 p-3 rounded text-xs overflow-auto">{JSON.stringify(diffs || [], null, 2)}</pre>
        )}
      </section>
    </div>
  );
}
