import React, { useState } from 'react';
import { Link } from 'react-router-dom';
// Force rebuild to clear cache issues

export default function LiveProgressTable({ 
  activeJob, 
  selectedRegions, 
  loadingActive, 
  refetchActive,
  activeJobId,
  isCompleted = false,
  diffReady = false,
}) {
  const timeAgo = (ts) => {
    if (!ts) return '';
    try {
      const d = new Date(ts).getTime();
      const diff = Date.now() - d;
      const sec = Math.floor(diff / 1000);
      if (sec < 60) return `${sec}s ago`;
      const min = Math.floor(sec / 60);
      const hr = Math.floor(min / 60);
      if (hr < 24) return `${hr}h ago`;
      const day = Math.floor(hr / 24);
      return `${day}d ago`;
    } catch { return String(ts); }
  };

  // Normalize exec region into one of US/EU/ASIA to match table rows
  function regionCodeFromExec(exec) {
    try {
      const raw = String(exec?.region || exec?.region_claimed || '').toLowerCase();
      if (!raw) return '';
      if (raw.includes('us') || raw.includes('united states')) return 'US';
      if (raw.includes('eu') || raw.includes('europe')) return 'EU';
      if (raw.includes('asia') || raw.includes('apac') || raw.includes('pacific')) return 'ASIA';
      return raw.toUpperCase();
    } catch { return ''; }
  }

  function prefillFromExecutions(activeJob, setters) {
    const { setARegion, setBRegion, setAText, setBText, setError } = setters || {};
    try {
      const execs = Array.isArray(activeJob?.executions) ? activeJob.executions : [];
      if (execs.length === 0) throw new Error('No executions available to prefill');
      const completed = execs.filter(e => String(e?.status || e?.state || '').toLowerCase() === 'completed');
      const pick = (regionCode) => completed.find(e => regionCodeFromExec(e) === regionCode);
      // Prefer US/EU, fallback to any two
      let eA = pick('US') || completed[0] || execs[0];
      let eB = pick('EU') || completed.find(e => e !== eA) || execs.find(e => e !== eA);
      const rA = normalizeRegion(eA?.region || eA?.region_claimed);
      const rB = normalizeRegion(eB?.region || eB?.region_claimed);
      const tA = extractExecText(eA);
      const tB = extractExecText(eB);
      if (setARegion) setARegion(rA);
      if (setBRegion) setBRegion(rB);
      if (setAText) setAText(tA || '');
      if (setBText) setBText(tB || '');
    } catch (err) {
      if (setError) setError(err?.message || String(err));
    }
  }

function normalizeRegion(r) {
  const v = String(r || '').toUpperCase();
  if (v === 'US') return 'us-east';
  if (v === 'EU') return 'eu-west';
  if (v === 'ASIA') return 'asia-pacific';
  return 'us-east';
}

  function extractExecText(exec) {
    const out = exec?.output || exec?.result || {};
    try {
      if (typeof out?.response === 'string' && out.response) {
        // For multi-question responses, show a preview
        const response = out.response;
        if (response.length > 200) {
          return response.substring(0, 200) + '... (click to view full response)';
        }
        return response;
      }
      if (out.responses && Array.isArray(out.responses) && out.responses.length > 0) {
        const r = out.responses[0];
        return r.response || r.answer || r.output || '';
      }
      if (out.text_output) return out.text_output;
      if (out.output) return out.output;
    } catch {}
    return '';
  }
  

  const truncateMiddle = (str, head = 6, tail = 4) => {
    if (!str || typeof str !== 'string') return '—';
    if (str.length <= head + tail + 1) return str;
    return `${str.slice(0, head)}…${str.slice(-tail)}`;
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed': return 'bg-green-900/20 text-green-400 border-green-700';
      case 'running': 
      case 'processing': return 'bg-yellow-900/20 text-yellow-400 border-yellow-700';
      case 'connecting': 
      case 'queued': return 'bg-blue-900/20 text-blue-400 border-blue-700';
      case 'completing': return 'bg-purple-900/20 text-purple-400 border-purple-700';
      case 'failed': return 'bg-red-900/20 text-red-400 border-red-700';
      case 'stalled': return 'bg-orange-900/20 text-orange-400 border-orange-700';
      case 'refreshing': return 'bg-cyan-900/20 text-cyan-400 border-cyan-700';
      case 'pending': return 'bg-gray-900/20 text-gray-400 border-gray-700';
      default: return 'bg-gray-900/20 text-gray-400 border-gray-700';
    }
  };


  // Overall progress calculation
  const execs = activeJob?.executions || [];
  const total = selectedRegions.length;
  const statusStr = String(activeJob?.status || activeJob?.state || '').toLowerCase();
  const jobCompleted = isCompleted || ['completed','success','succeeded','done','finished'].includes(statusStr);
  const jobId = activeJob?.id || activeJob?.job?.id;
  let completed = execs.filter((e) => (e?.status || e?.state) === 'completed').length;
  let running = execs.filter((e) => (e?.status || e?.state) === 'running').length;
  let failed = execs.filter((e) => (e?.status || e?.state) === 'failed').length;
  // Only override execution counts for successful jobs, not failed ones
  if (jobCompleted) {
    // If the job is successfully complete but we might not have full per-region execution info,
    // present a simple, clear UX: mark progress as fully completed.
    completed = total;
    running = 0;
    failed = 0;
  }
  const pct = Math.round((completed / Math.max(total, 1)) * 100);
  const overallCompleted = jobCompleted || (total > 0 && completed >= total);
  const actionsDisabled = !overallCompleted;
  const showShimmer = !overallCompleted && running > 0;

  return (
    <div className="p-4 space-y-3">
      {/* Overall Progress */}
      <div className="space-y-3">
        <div>
          <div className="flex items-center justify-between text-sm mb-1">
            <span>Overall Progress</span>
            <span className="text-gray-400">{completed}/{total} regions · {pct}%</span>
          </div>
          <div className={`w-full h-3 bg-gray-700 rounded overflow-hidden relative ${showShimmer ? 'animate-pulse' : ''}`}>
            <div className="h-full flex relative z-10">
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
            {showShimmer && (
              <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/10 to-transparent"></div>
            )}
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
        <div className="grid grid-cols-5 text-xs bg-gray-700 text-gray-300">
          <div className="px-3 py-2">Region</div>
          <div className="px-3 py-2">Status</div>
          <div className="px-3 py-2">Started</div>
          <div className="px-3 py-2">Provider</div>
          <div className="px-3 py-2">Answer</div>
        </div>
        {['US','EU','ASIA'].map((r) => {
          const e = (activeJob?.executions || []).find((x) => regionCodeFromExec(x) === r);
          const status = jobCompleted ? 'completed' : (e?.status || e?.state || 'pending');
          const started = e?.started_at || e?.created_at;
          const provider = e?.provider_id || e?.provider;

          const failure = e?.output?.failure || e?.failure || e?.failure_reason || e?.output?.failure_reason;
          const failureMessage = typeof failure === 'object' ? failure?.message : null;
          const failureCode = typeof failure === 'object' ? failure?.code : null;
          const failureStage = typeof failure === 'object' ? failure?.stage : null;

          const getEnhancedStatus = () => {
            if (loadingActive) return 'refreshing';
            if (!e) return jobCompleted ? 'completed' : 'pending';
            
            // Check for infrastructure errors
            if (e?.error || e?.failure_reason || failureMessage) {
              return 'failed';
            }
            
            // Live Progress Table - Enhanced with dynamic status detection
            const currentStatus = status || 'pending';
            const now = new Date();
            const startTime = started ? new Date(started) : null;
            const runningTime = startTime ? (now - startTime) / 1000 / 60 : 0; // minutes
            
            // Detect granular states
            if (currentStatus === 'created' || currentStatus === 'enqueued') {
              return 'queued';
            }
            
            if (currentStatus === 'running') {
              // Check if it's been running for a while (might be stalled)
              if (runningTime > 30) {
                return 'stalled';
              }
              
              // Detect sub-states of running based on timing
              if (runningTime < 0.5) { // First 30 seconds
                return 'connecting';
              } else if (runningTime < 25) { // Most of execution time
                return 'processing';
              } else {
                return 'completing'; // Taking longer, probably finishing up
              }
            }
            
            return currentStatus;
          };

          const enhancedStatus = getEnhancedStatus();
          
          return (
            <div key={r} className="grid grid-cols-5 text-sm border-t border-gray-600 hover:bg-gray-700">
              <div className="px-3 py-2 font-medium">
                <span>{r}</span>
              </div>
              <div className="px-3 py-2">
                <div className="flex flex-col gap-1">
                  <span className={`text-xs px-2 py-0.5 rounded-full border ${getStatusColor(enhancedStatus)}`}>
                    {String(enhancedStatus)}
                  </span>
                  {(failureMessage || e?.error || e?.failure_reason) && (
                    <div className="flex flex-col gap-0.5 text-red-500 text-xs">
                      <span className="truncate" title={failureMessage || e?.error || e?.failure_reason}>
                        {(failureMessage || e?.error || e?.failure_reason || '').slice(0, 60)}{(failureMessage || e?.error || e?.failure_reason || '').length > 60 ? '…' : ''}
                      </span>
                      {(failureCode || failureStage) && (
                        <span className="text-red-400/80 uppercase tracking-wide" title={`${failureCode || ''} ${failureStage || ''}`.trim()}>
                          {failureCode || ''}{failureCode && failureStage ? ' · ' : ''}{failureStage || ''}
                        </span>
                      )}
                    </div>
                  )}
                </div>
              </div>
              <div className="px-3 py-2 text-xs" title={started ? new Date(started).toLocaleString() : ''}>{started ? timeAgo(started) : '—'}</div>
              <div className="px-3 py-2 font-mono text-xs" title={provider}>{provider ? truncateMiddle(provider, 6, 4) : '—'}</div>
              <div className="px-3 py-2">
                {e?.id ? (
                  <Link
                    to={`/executions?job=${encodeURIComponent(activeJob?.id || activeJob?.job?.id || '')}&region=${encodeURIComponent(r)}`}
                    className="text-xs text-beacon-600 underline decoration-dotted"
                  >Answer</Link>
                ) : (
                  <span className="text-xs text-gray-500">—</span>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Action Buttons */}
      <div className="flex items-center justify-end gap-2">
        <button onClick={refetchActive} className="px-3 py-1.5 bg-green-600 text-white rounded text-sm hover:bg-green-700">Refresh</button>
        {(() => {
          const showDiffCta = !!jobId; // always show when we have a job id
          if (!showDiffCta) return null;
          if (actionsDisabled) {
            return (
              <button
                disabled
                className="px-3 py-1.5 bg-beacon-600 text-white rounded text-sm opacity-50 cursor-not-allowed"
                title="Available when job completes"
              >
                View Cross-Region Diffs
              </button>
            );
          }
          return (
            <Link 
              to={`/results/${jobId}/diffs`}
              className="px-3 py-1.5 bg-beacon-600 text-white rounded text-sm hover:bg-beacon-700"
            >
              View Cross-Region Diffs
            </Link>
          );
        })()}
        {activeJob?.id && (
          <Link to={`/jobs/${activeJob.id}`} className="text-sm text-beacon-600 underline decoration-dotted">View full results</Link>
        )}
      </div>
    </div>
  );
}

