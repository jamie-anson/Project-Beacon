import React from 'react';
import { useQuery } from '../state/useQuery.js';
import { getHealth, getExecutions, getDiffs, getTransparencyRoot, listJobs, getHybridHealth, getHybridProviders } from '../lib/api.js';
import { Link } from 'react-router-dom';
import useWs from '../state/useWs.js';
import { useToast } from '../state/toast.jsx';
import ActivityFeed from '../components/ActivityFeed.jsx';
import CopyButton from '../components/CopyButton.jsx';
import InfrastructureStatus from '../components/InfrastructureStatus.jsx';

// Small inline UI helpers to avoid new files for now
function StatusPill({ value }) {
  const val = typeof value === 'string' ? value.toLowerCase() : value;
  const ok = val === true || val === 'ok' || val === 'healthy' || val === 'up' || val === 'ready' || val === 'running' || val === 'success';
  const warn = val === 'degraded' || val === 'warning' || val === 'partial';
  const bad = val === false || val === 'down' || val === 'error' || val === 'failed' || val === 'unhealthy';
  const cls = ok
    ? 'bg-green-100 text-green-700'
    : warn
    ? 'bg-amber-100 text-amber-800'
    : bad
    ? 'bg-red-100 text-red-700'
    : 'bg-gray-600 text-gray-200';
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

export default function Dashboard() {
  const { data: health, loading: loadingHealth, error: healthError } = useQuery('health', getHealth, { interval: 30000 });
  const { data: executions, loading: loadingExecs, error: execsError } = useQuery('executions:latest', () => getExecutions({ limit: 5 }), { interval: 15000 });
  const { data: diffs, loading: loadingDiffs, error: diffsError } = useQuery('diffs:latest', () => getDiffs({ limit: 5 }), { interval: 20000 });
  // Hybrid router (Railway) live data via Netlify proxy
  const { data: hybridHealth, error: hybridErr } = useQuery('hybrid:health', getHybridHealth, { interval: 30000 });
  const { data: hybridProviders, error: providersErr } = useQuery('hybrid:providers', getHybridProviders, { interval: 30000 });

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
  const { data: tRoot, loading: loadingRoot, error: rootError } = useQuery('transparency:root', getTransparencyRoot, { interval: 15000 });
  const { data: recentJobs, loading: loadingJobs, error: jobsError } = useQuery('jobs:recent', () => listJobs({ limit: 5 }), { interval: 20000 });
  const recentJobsArr = React.useMemo(() => {
    if (Array.isArray(recentJobs)) return recentJobs;
    if (recentJobs && Array.isArray(recentJobs.jobs)) return recentJobs.jobs;
    return [];
  }, [recentJobs]);

  return (
    <div className="space-y-6">
      <section>
        <h2 className="text-xl font-semibold">Live activity</h2>
        <ActivityFeed events={events} />
      </section>
      <section>
        <h2 className="text-xl font-semibold">Transparency root</h2>
        {loadingRoot ? (
          <div className="bg-gray-800 border border-gray-700 rounded p-3 animate-pulse">
            <div className="h-4 bg-gray-600 rounded w-2/3"></div>
            <div className="h-3 bg-gray-700 rounded w-1/3 mt-2"></div>
          </div>
        ) : rootError ? (
          <div className="bg-gray-800 border border-gray-700 rounded p-3 text-sm">
            <div className="text-red-600">Backend unavailable</div>
            <div className="text-xs text-gray-400 mt-1">Transparency service offline</div>
          </div>
        ) : tRoot ? (
          <div className="bg-gray-800 border border-gray-700 rounded p-3 text-sm">
            <div className="flex items-center gap-2">
              <div>Root: <span className="font-mono break-all">{tRoot.root || tRoot.merkle_root || '—'}</span></div>
              {(tRoot.root || tRoot.merkle_root) && (
                <CopyButton text={tRoot.root || tRoot.merkle_root} label="Copy root" />
              )}
            </div>
            {tRoot.sequence != null && (
              <div className="text-xs text-gray-400 mt-1">Seq #{tRoot.sequence}{tRoot.updated_at ? ` · ${tRoot.updated_at}` : ''}</div>
            )}
          </div>
        ) : (
          <div className="text-sm text-gray-400">Loading…</div>
        )}
      </section>
      <section>
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold">Recent jobs</h2>
          <Link to="/jobs" className="text-sm text-orange-400 underline decoration-dotted">See all Jobs</Link>
        </div>
        <div className="bg-gray-800 border border-gray-700 rounded divide-y">
          {loadingJobs ? (
            Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="p-3 animate-pulse">
                <div className="h-4 bg-gray-600 rounded w-1/2"></div>
                <div className="h-3 bg-gray-700 rounded w-1/3 mt-2"></div>
              </div>
            ))
          ) : jobsError ? (
            <div className="p-3 text-sm text-red-600">Backend unavailable - Jobs service offline</div>
          ) : recentJobsArr.slice(0, 5).map((j) => (
            <div key={j.id} className="p-3 text-sm flex items-center justify-between">
              <div className="min-w-0">
                <div className="font-mono truncate">{j.id}</div>
                <div className="text-xs text-gray-400">{j.created_at}</div>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-xs px-2 py-0.5 rounded-full bg-gray-600 text-gray-200">{j.status}</span>
                <Link className="text-orange-400 text-xs underline decoration-dotted" to={`/jobs/${encodeURIComponent(j.id)}`}>View</Link>
                <CopyButton text={j.id} label="Copy ID" />
              </div>
            </div>
          ))}
          {(!jobsError && recentJobsArr.length === 0) && (
            <div className="p-3 text-sm text-gray-400">No jobs yet.</div>
          )}
        </div>
      </section>
      <section>
        <h2 className="text-xl font-semibold">Infrastructure Status</h2>
        <InfrastructureStatus />
      </section>
      
      <section>
        <h2 className="text-xl font-semibold">Legacy Server Status</h2>
        <div className="bg-gray-800 border border-gray-700 rounded p-4">
          <div className="space-y-3">
            {/* Main Runner API (Fly) */}
            <div className="flex items-center justify-between py-2 border-b border-gray-700 last:border-b-0">
              <div className="flex items-center gap-3">
                <div className={`w-3 h-3 rounded-full ${healthError ? 'bg-red-500' : 'bg-green-500'}`}></div>
                <div>
                  <div className="font-medium text-sm">Runner API</div>
                  <div className="text-xs text-gray-400">Fly.io</div>
                </div>
              </div>
              <div className="text-right">
                <StatusPill value={healthError ? 'down' : 'healthy'} />
                <div className="text-xs text-gray-400 mt-1">beacon-runner-change-me</div>
              </div>
            </div>

            {/* Hybrid Router (Railway) */}
            <div className="flex items-center justify-between py-2 border-b border-gray-700 last:border-b-0">
              <div className="flex items-center gap-3">
                <div className={`w-3 h-3 rounded-full ${hybridErr ? 'bg-red-500' : 'bg-green-500'}`}></div>
                <div>
                  <div className="font-medium text-sm">Hybrid Router</div>
                  <div className="text-xs text-gray-400">Railway</div>
                </div>
              </div>
              <div className="text-right">
                <StatusPill value={hybridHealth?.status || (hybridErr ? 'down' : 'healthy')} />
                <div className="text-xs text-gray-400 mt-1">project-beacon-production.up.railway.app</div>
              </div>
            </div>

            {/* Golem Provider (EU) */}
            <div className="flex items-center justify-between py-2">
              <div className="flex items-center gap-3">
                {(() => {
                  const eu = Array.isArray(hybridProviders) ? hybridProviders.find(p => (p.type === 'golem') && (p.region === 'eu-west')) : null;
                  const ok = !!eu?.healthy;
                  return <div className={`w-3 h-3 rounded-full ${ok ? 'bg-green-500' : 'bg-amber-400'}`}></div>;
                })()}
                <div>
                  <div className="font-medium text-sm">Golem Provider (EU)</div>
                  <div className="text-xs text-gray-400">Fly.io</div>
                </div>
              </div>
              <div className="text-right">
                {(() => {
                  const eu = Array.isArray(hybridProviders) ? hybridProviders.find(p => (p.type === 'golem') && (p.region === 'eu-west')) : null;
                  return <StatusPill value={eu ? (eu.healthy ? 'healthy' : 'degraded') : 'unknown'} />;
                })()}
                <div className="text-xs text-gray-400 mt-1">beacon-golem-simple</div>
              </div>
            </div>

            {/* Monitoring */}
            <div className="flex items-center justify-between py-2 border-b border-gray-700 last:border-b-0">
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 rounded-full bg-slate-400"></div>
                <div>
                  <div className="font-medium text-sm">beacon-prom-agent</div>
                  <div className="text-xs text-gray-400">Prometheus Agent</div>
                </div>
              </div>
              <div className="text-right">
                <StatusPill value="suspended" />
                <div className="text-xs text-gray-400 mt-1">fly.io</div>
              </div>
            </div>

            {/* Portal */}
            <div className="flex items-center justify-between py-2">
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 rounded-full bg-green-500"></div>
                <div>
                  <div className="font-medium text-sm">projectbeacon.netlify.app</div>
                  <div className="text-xs text-gray-400">Portal & Docs</div>
                </div>
              </div>
              <div className="text-right">
                <StatusPill value="deployed" />
                <div className="text-xs text-gray-400 mt-1">netlify</div>
              </div>
            </div>
          </div>
        </div>
      </section>
      <section>
        <h2 className="text-xl font-semibold">GPU Status</h2>
        <div className="bg-gray-800 border border-gray-700 rounded p-4">
          <div className="space-y-3">
            {/* Regional GPU Services */}
            <div className="flex items-center justify-between py-2 border-b border-gray-700 last:border-b-0">
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 rounded-full bg-green-500"></div>
                <div>
                  <div className="font-medium text-sm">US</div>
                  <div className="text-xs text-gray-400">T4/A10 • US-East/Central/West</div>
                </div>
              </div>
              <div className="text-right">
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  healthy
                </span>
                <div className="text-xs text-gray-400 mt-1">modal.com</div>
              </div>
            </div>

            <div className="flex items-center justify-between py-2 border-b border-gray-700 last:border-b-0">
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 rounded-full bg-green-500"></div>
                <div>
                  <div className="font-medium text-sm">EU</div>
                  <div className="text-xs text-gray-400">T4/A10 • EU-West/North</div>
                </div>
              </div>
              <div className="text-right">
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  healthy
                </span>
                <div className="text-xs text-gray-400 mt-1">modal.com</div>
              </div>
            </div>

            <div className="flex items-center justify-between py-2">
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 rounded-full bg-green-500"></div>
                <div>
                  <div className="font-medium text-sm">APAC</div>
                  <div className="text-xs text-gray-400">T4/A10 • AP-Southeast/Northeast</div>
                </div>
              </div>
              <div className="text-right">
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  healthy
                </span>
                <div className="text-xs text-gray-400 mt-1">modal.com</div>
              </div>
            </div>
          </div>
        </div>
      </section>
      <section>
        <h2 className="text-xl font-semibold">API Services</h2>
        <div className="bg-gray-800 border border-gray-700 rounded p-4">
          <div className="space-y-3">
            {/* Railway Service */}
            <div className="flex items-center justify-between py-2 border-b border-gray-700">
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 rounded-full bg-blue-500"></div>
                <div>
                  <div className="font-medium text-sm">Railway Router</div>
                  <div className="text-xs text-gray-400">Hybrid API • Multi-region</div>
                </div>
              </div>
              <div className="text-right">
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  healthy
                </span>
                <div className="text-xs text-gray-400 mt-1">railway.app</div>
              </div>
            </div>

            {/* Fly.io Service */}
            <div className="flex items-center justify-between py-2">
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 rounded-full bg-red-500"></div>
                <div>
                  <div className="font-medium text-sm">Fly.io Router</div>
                  <div className="text-xs text-gray-400">Legacy API • Deprecated</div>
                </div>
              </div>
              <div className="text-right">
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                  suspended
                </span>
                <div className="text-xs text-gray-400 mt-1">fly.io</div>
              </div>
            </div>
          </div>
        </div>
      </section>
      <section>
        <h2 className="text-xl font-semibold">System status</h2>
        {loadingHealth ? (
          <div className="bg-gray-800 border border-gray-700 rounded p-3 animate-pulse">
            <div className="h-4 bg-gray-600 rounded w-2/3"></div>
            <div className="h-4 bg-gray-700 rounded w-full mt-2"></div>
            <div className="h-4 bg-gray-700 rounded w-5/6 mt-2"></div>
          </div>
        ) : healthError ? (
          <div className="bg-gray-800 border border-gray-700 rounded p-3">
            <div className="text-sm text-red-600">Backend unavailable - Health service offline</div>
          </div>
        ) : (
          <div className="bg-gray-800 border border-gray-700 rounded p-3">
            {!health || Object.keys(health || {}).length === 0 ? (
              <div className="text-sm text-gray-400">No status available.</div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                {/* Core Services */}
                {['database', 'redis', 'ipfs'].map(service => health[service] && (
                  <div key={service} className="border rounded p-3">
                    <div className="text-xs uppercase tracking-wide text-gray-400">{service}</div>
                    <div className="mt-1 text-sm flex items-center gap-2">
                      <StatusPill value={health[service]} />
                    </div>
                  </div>
                ))}
                
                {/* Golem Network */}
                {health.yagna && (
                  <div className="border rounded p-3">
                    <div className="text-xs uppercase tracking-wide text-gray-400">Golem Network</div>
                    <div className="mt-1 text-sm flex items-center gap-2">
                      <StatusPill value={health.yagna} />
                    </div>
                  </div>
                )}
                
                
                {/* Any other services */}
                {Object.entries(health).filter(([k]) => !['database', 'redis', 'ipfs', 'yagna', 'overall'].includes(k)).map(([k, v]) => (
                  <div key={k} className="border rounded p-3">
                    <div className="text-xs uppercase tracking-wide text-gray-400">{k}</div>
                    <div className="mt-1 text-sm flex items-center gap-2">
                      {typeof v === 'object' && v !== null ? (
                        <span className="font-mono text-xs bg-gray-700 px-2 py-1 rounded overflow-hidden text-ellipsis">
                          {truncateMiddle(JSON.stringify(v))}
                        </span>
                      ) : (
                        <StatusPill value={v} />
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </section>
      <section>
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold">Recent executions</h2>
          <Link to="/executions" className="text-sm text-orange-400 underline decoration-dotted">See all Executions</Link>
        </div>
        {loadingExecs ? (
          <div className="bg-gray-800 border border-gray-700 rounded overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-700 text-gray-300">
                <tr>
                  <th className="text-left px-3 py-2">ID</th>
                  <th className="text-left px-3 py-2">Job</th>
                  <th className="text-left px-3 py-2">Status</th>
                  <th className="text-left px-3 py-2">Started</th>
                  <th className="text-left px-3 py-2">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {Array.from({ length: 5 }).map((_, i) => (
                  <tr key={i} className="animate-pulse">
                    <td className="px-3 py-2">
                      <div className="h-3 w-24 bg-gray-600 rounded" />
                    </td>
                    <td className="px-3 py-2">
                      <div className="h-3 w-20 bg-gray-600 rounded" />
                    </td>
                    <td className="px-3 py-2">
                      <div className="h-5 w-14 bg-gray-600 rounded-full" />
                    </td>
                    <td className="px-3 py-2">
                      <div className="h-3 w-16 bg-gray-600 rounded" />
                    </td>
                    <td className="px-3 py-2">
                      <div className="h-3 w-28 bg-gray-600 rounded" />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="bg-gray-800 border border-gray-700 rounded overflow-hidden">
            {execsError ? (
              <div className="p-3 text-sm text-red-600">Backend unavailable - Executions service offline</div>
            ) : (!executions || executions.length === 0) ? (
              <div className="p-3 text-sm text-gray-400">No executions yet.</div>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-gray-700 text-gray-300">
                  <tr>
                    <th className="text-left px-3 py-2">ID</th>
                    <th className="text-left px-3 py-2">Job</th>
                    <th className="text-left px-3 py-2">Status</th>
                    <th className="text-left px-3 py-2">Started</th>
                    <th className="text-left px-3 py-2">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {(executions || []).slice(0, 5).map((e) => {
                    const id = e?.id || e?.execution_id || '—';
                    const jobId = e?.job_id || e?.jobId || e?.job?.id;
                    const status = e?.status || e?.state || 'unknown';
                    const created = e?.created_at || e?.started_at || e?.createdAt || e?.startedAt;
                    return (
                      <tr key={id} className="hover:bg-gray-700">
                        <td className="px-3 py-2 font-mono whitespace-nowrap" title={id}>{truncateMiddle(String(id))}</td>
                        <td className="px-3 py-2 font-mono whitespace-nowrap" title={jobId || ''}>{truncateMiddle(String(jobId || '—'))}</td>
                        <td className="px-3 py-2"><StatusPill value={status} /></td>
                        <td className="px-3 py-2 text-xs" title={formatDate(created)}>{timeAgo(created)}</td>
                        <td className="px-3 py-2 flex items-center gap-2">
                          {id && <Link className="text-orange-400 text-xs underline decoration-dotted" to={`/executions/${encodeURIComponent(id)}`}>View</Link>}
                          {id && <CopyButton text={String(id)} label="Copy ID" />}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            )}
          </div>
        )}
      </section>
      <section>
        <h2 className="text-xl font-semibold">Recent diffs</h2>
        {loadingDiffs ? (
          <div className="bg-gray-800 border border-gray-700 rounded overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-700 text-gray-300">
                <tr>
                  <th className="text-left px-3 py-2">ID</th>
                  <th className="text-left px-3 py-2">Type</th>
                  <th className="text-left px-3 py-2">Job</th>
                  <th className="text-left px-3 py-2">Created</th>
                  <th className="text-left px-3 py-2">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {Array.from({ length: 5 }).map((_, i) => (
                  <tr key={i} className="animate-pulse">
                    <td className="px-3 py-2"><div className="h-3 w-24 bg-gray-600 rounded" /></td>
                    <td className="px-3 py-2"><div className="h-3 w-16 bg-gray-600 rounded" /></td>
                    <td className="px-3 py-2"><div className="h-3 w-20 bg-gray-600 rounded" /></td>
                    <td className="px-3 py-2"><div className="h-3 w-16 bg-gray-600 rounded" /></td>
                    <td className="px-3 py-2"><div className="h-3 w-28 bg-gray-600 rounded" /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="bg-gray-800 border border-gray-700 rounded overflow-hidden">
            {diffsError ? (
              <div className="p-3 text-sm text-red-600">Backend unavailable - Diffs service offline</div>
            ) : (!diffs || diffs.length === 0) ? (
              <div className="p-3 text-sm text-gray-400">No diffs yet.</div>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-gray-700 text-gray-300">
                  <tr>
                    <th className="text-left px-3 py-2">ID</th>
                    <th className="text-left px-3 py-2">Type</th>
                    <th className="text-left px-3 py-2">Job</th>
                    <th className="text-left px-3 py-2">Created</th>
                    <th className="text-left px-3 py-2">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {(diffs || []).slice(0, 5).map((d) => {
                    const id = d?.id || d?.diff_id || '—';
                    const type = d?.type || d?.category || '—';
                    const jobId = d?.job_id || d?.jobId || d?.job?.id;
                    const created = d?.created_at || d?.createdAt;
                    return (
                      <tr key={id} className="hover:bg-gray-700">
                        <td className="px-3 py-2 font-mono whitespace-nowrap" title={id}>{truncateMiddle(String(id))}</td>
                        <td className="px-3 py-2 text-xs">{String(type)}</td>
                        <td className="px-3 py-2 font-mono whitespace-nowrap" title={jobId || ''}>{truncateMiddle(String(jobId || '—'))}</td>
                        <td className="px-3 py-2 text-xs" title={formatDate(created)}>{timeAgo(created)}</td>
                        <td className="px-3 py-2 flex items-center gap-2">
                          {id && <Link className="text-orange-400 text-xs underline decoration-dotted" to={`/diffs/${encodeURIComponent(id)}`}>View</Link>}
                          {id && <CopyButton text={String(id)} label="Copy ID" />}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            )}
          </div>
        )}
      </section>
    </div>
  );
}
