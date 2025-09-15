import React from 'react';
import { Link } from 'react-router-dom';

export default function LiveProgressTable({ 
  activeJob, 
  selectedRegions, 
  loadingActive, 
  refetchActive,
  activeJobId 
}) {
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

  const truncateMiddle = (str, head = 6, tail = 4) => {
    if (!str || typeof str !== 'string') return '—';
    if (str.length <= head + tail + 1) return str;
    return `${str.slice(0, head)}…${str.slice(-tail)}`;
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed': return 'bg-green-100 text-green-800';
      case 'running': return 'bg-yellow-100 text-yellow-800';
      case 'failed': return 'bg-red-100 text-red-800';
      case 'stalled': return 'bg-orange-100 text-orange-800';
      default: return 'bg-gray-100 text-gray-800';
    }
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
      : 'bg-gray-600 text-gray-200';
    return <span className={`text-xs px-2 py-0.5 rounded-full ${cls}`}>{label}</span>;
  };

  // Overall progress calculation
  const execs = activeJob?.executions || [];
  const completed = execs.filter((e) => (e?.status || e?.state) === 'completed').length;
  const running = execs.filter((e) => (e?.status || e?.state) === 'running').length;
  const failed = execs.filter((e) => (e?.status || e?.state) === 'failed').length;
  const total = selectedRegions.length;
  const pct = Math.round((completed / total) * 100);

  return (
    <div className="p-4 space-y-3">
      {/* Overall Progress */}
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

      {/* Detailed Progress Table */}
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
          const status = e?.status || e?.state || 'pending';
          const started = e?.started_at || e?.created_at;
          const provider = e?.provider_id || e?.provider;
          const retries = e?.retries;
          const eta = e?.eta;

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

      {/* Action Buttons */}
      <div className="flex items-center justify-end gap-2">
        <button onClick={refetchActive} className="px-3 py-1.5 bg-green-600 text-white rounded text-sm hover:bg-green-700">Refresh</button>
        {(() => {
          const execs = activeJob?.executions || [];
          const completedRegions = execs.filter(e => (e?.status || e?.state) === 'completed').length;
          const totalRegions = selectedRegions.length;
          const hasMultiRegionResults = totalRegions >= 2 && completedRegions >= 2;
          
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
  );
}
