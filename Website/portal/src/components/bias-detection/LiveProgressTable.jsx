import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';

export default function LiveProgressTable({ 
  activeJob, 
  selectedRegions, 
  loadingActive, 
  refetchActive,
  activeJobId,
  isCompleted = false,
  onDismiss
}) {
  const [expandedRegions, setExpandedRegions] = useState(new Set());
  const [showPartialResults, setShowPartialResults] = useState(false);
  const [hasNotified, setHasNotified] = useState(false);
  
  const toggleRegionExpansion = (region) => {
    const newExpanded = new Set(expandedRegions);
    if (newExpanded.has(region)) {
      newExpanded.delete(region);
    } else {
      newExpanded.add(region);
    }
    setExpandedRegions(newExpanded);
  };

  // Browser notification on completion
  useEffect(() => {
    if (isCompleted && !hasNotified && 'Notification' in window) {
      if (Notification.permission === 'granted') {
        new Notification('Project Beacon - Job Completed!', {
          body: `Bias detection completed across regions`,
          icon: '/favicon.ico'
        });
        setHasNotified(true);
      } else if (Notification.permission !== 'denied') {
        Notification.requestPermission().then(permission => {
          if (permission === 'granted') {
            new Notification('Project Beacon - Job Completed!', {
              body: `Bias detection completed across regions`,
              icon: '/favicon.ico'
            });
          }
        });
        setHasNotified(true);
      }
    }
  }, [isCompleted, hasNotified]);
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
      case 'completed': return 'bg-green-900/20 text-green-400 border-green-700';
      case 'running': 
      case 'processing': return 'bg-yellow-900/20 text-yellow-400 border-yellow-700';
      case 'connecting': 
      case 'queued': return 'bg-blue-900/20 text-blue-400 border-blue-700';
      case 'completing': return 'bg-purple-900/20 text-purple-400 border-purple-700';
      case 'failed': return 'bg-red-900/20 text-red-400 border-red-700';
      case 'stalled': return 'bg-orange-900/20 text-orange-400 border-orange-700';
      case 'pending': return 'bg-gray-900/20 text-gray-400 border-gray-700';
      default: return 'bg-gray-900/20 text-gray-400 border-gray-700';
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
  
  // Calculate job completion time
  const jobStartTime = activeJob?.created_at;
  const jobEndTime = activeJob?.completed_at || activeJob?.updated_at;
  const executionTime = jobStartTime && jobEndTime ? 
    Math.round((new Date(jobEndTime) - new Date(jobStartTime)) / 1000) : null;
    
  const formatDuration = (seconds) => {
    if (!seconds) return '—';
    if (seconds < 60) return `${seconds}s`;
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return `${minutes}m ${remainingSeconds}s`;
  };

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
                className="h-full bg-green-500 transition-all duration-500 ease-out" 
                style={{ width: `${(completed / total) * 100}%` }}
              ></div>
              <div 
                className={`h-full bg-yellow-500 transition-all duration-500 ease-out ${
                  running > 0 ? 'animate-pulse' : ''
                }`}
                style={{ width: `${(running / total) * 100}%` }}
              ></div>
              <div 
                className="h-full bg-red-500 transition-all duration-500 ease-out" 
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

      {/* Completion Celebration Banner */}
      {isCompleted && (
        <div className="bg-green-900/20 border border-green-700 rounded-lg p-4 space-y-3">
          <div className="flex items-center gap-3">
            <div className="flex-shrink-0">
              <div className="w-8 h-8 bg-green-600 rounded-full flex items-center justify-center">
                <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
            </div>
            <div className="flex-1">
              <h4 className="text-green-400 font-medium">Job Completed Successfully!</h4>
              <p className="text-sm text-gray-300 mt-1">
                Executed across {completed} region{completed !== 1 ? 's' : ''} in {formatDuration(executionTime)}
                {failed > 0 && ` • ${failed} region${failed !== 1 ? 's' : ''} failed`}
              </p>
            </div>
          </div>
          
          {/* Action Buttons */}
          <div className="flex items-center gap-3 pt-2">
            {(() => {
              const hasMultiRegionResults = total >= 2 && completed >= 2;
              
              return (
                <>
                  {activeJob?.id && (
                    <Link 
                      to={`/jobs/${activeJob.id}`}
                      className="px-4 py-2 bg-beacon-600 text-white rounded-lg text-sm font-medium hover:bg-beacon-700 transition-colors"
                    >
                      View Full Results
                    </Link>
                  )}
                  {hasMultiRegionResults && activeJob?.id && (
                    <Link 
                      to={`/portal/results/${activeJob.id}/diffs`}
                      className="px-4 py-2 bg-orange-600 text-white rounded-lg text-sm font-medium hover:bg-orange-700 transition-colors"
                    >
                      View Cross-Region Diffs
                    </Link>
                  )}
                  <button
                    onClick={onDismiss}
                    className="px-3 py-2 text-gray-400 hover:text-gray-300 text-sm border border-gray-600 hover:border-gray-500 rounded-lg transition-colors"
                  >
                    Dismiss
                  </button>
                </>
              );
            })()}
          </div>
        </div>
      )}

      {/* Partial Results Toggle */}
      {!isCompleted && completed > 0 && (
        <div className="flex items-center justify-between p-3 bg-blue-900/10 border border-blue-700/30 rounded-lg">
          <div className="flex items-center gap-2">
            <svg className="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
            </svg>
            <span className="text-sm text-blue-400">
              {completed} region{completed !== 1 ? 's' : ''} completed - Preview available
            </span>
          </div>
          <button
            onClick={() => setShowPartialResults(!showPartialResults)}
            className="px-3 py-1 text-xs bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors"
          >
            {showPartialResults ? 'Hide Preview' : 'Quick Preview'}
          </button>
        </div>
      )}

      {/* Partial Results Preview */}
      {showPartialResults && completed > 0 && (
        <div className="bg-gray-800/50 border border-gray-600 rounded-lg p-4">
          <h4 className="text-sm font-medium text-gray-200 mb-3 flex items-center gap-2">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
            </svg>
            Partial Results ({completed} regions completed)
          </h4>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {['US','EU','ASIA'].map((r) => {
              const e = (activeJob?.executions || []).find((x) => (x?.region || x?.region_claimed || '').toUpperCase?.() === r);
              const status = e?.status || e?.state || 'pending';
              if (status !== 'completed') return null;
              
              return (
                <div key={r} className="bg-gray-900/50 rounded p-3">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium text-gray-200">{r}</span>
                    <span className="text-xs text-green-400">✓ Completed</span>
                  </div>
                  {e?.output && (
                    <div className="text-xs text-gray-400">
                      <div className="bg-black/20 rounded p-2 max-h-20 overflow-y-auto">
                        <pre className="whitespace-pre-wrap">
                          {typeof e.output === 'string' 
                            ? e.output.substring(0, 100) + (e.output.length > 100 ? '...' : '')
                            : JSON.stringify(e.output, null, 2).substring(0, 100) + '...'
                          }
                        </pre>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

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
          
          // Calculate estimated completion time
          const calculateETA = () => {
            if (enhancedStatus === 'completed' || enhancedStatus === 'failed') return '—';
            if (!started) return '—';
            
            const startTime = new Date(started);
            const now = new Date();
            const runningTime = (now - startTime) / 1000; // seconds
            
            // Estimate based on status and typical execution times
            const estimates = {
              'connecting': 30,    // 30 seconds to connect
              'processing': 120,   // 2 minutes to process
              'completing': 30,    // 30 seconds to complete
              'queued': 60        // 1 minute in queue
            };
            
            const estimatedTotal = estimates[enhancedStatus] || 120;
            const remaining = Math.max(0, estimatedTotal - runningTime);
            
            if (remaining < 60) return `${Math.round(remaining)}s`;
            return `${Math.round(remaining / 60)}m`;
          };
          
          // Provider health indicator
          const getProviderHealth = () => {
            if (!provider) return null;
            
            // Simple heuristic based on execution state
            if (enhancedStatus === 'failed') return 'unhealthy';
            if (enhancedStatus === 'stalled') return 'degraded';
            if (['connecting', 'processing', 'completing'].includes(enhancedStatus)) return 'healthy';
            return 'unknown';
          };
          
          const providerHealth = getProviderHealth();

          const getEnhancedStatus = () => {
            if (loadingActive) return 'refreshing';
            if (!e) return 'pending';
            
            // Check for infrastructure errors
            if (e?.error || e?.failure_reason) {
              return 'failed';
            }
            
            // Enhanced status detection based on execution state
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
              } else if (runningTime < 5) { // First 5 minutes
                return 'processing';
              } else {
                return 'completing'; // Taking longer, probably finishing up
              }
            }
            
            return currentStatus;
          };

          const enhancedStatus = getEnhancedStatus();
          const isExpanded = expandedRegions.has(r);
          
          return (
            <div 
              key={r}
              className="grid grid-cols-7 text-sm border-t border-gray-600 hover:bg-gray-700 cursor-pointer transition-colors"
              onClick={() => toggleRegionExpansion(r)}
            >
                <div className="px-3 py-2 font-medium flex items-center gap-2">
                  <button className="flex items-center gap-1 hover:text-blue-400">
                    <svg 
                      className={`w-3 h-3 transition-transform ${isExpanded ? 'rotate-90' : ''}`} 
                      fill="none" 
                      stroke="currentColor" 
                      viewBox="0 0 24 24"
                    >
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                    </svg>
                    <span>{r}</span>
                  </button>
                  {e?.id && (
                    <Link
                      to={`/executions?job=${encodeURIComponent(activeJob?.id || activeJob?.job?.id || '')}&region=${encodeURIComponent(r)}`}
                      className="text-xs text-beacon-600 underline decoration-dotted"
                      onClick={(ev) => ev.stopPropagation()}
                    >executions</Link>
                  )}
                </div>
              <div className="px-3 py-2">
                <div className="flex flex-col gap-1">
                  <div className="flex items-center gap-2">
                    <span className={`text-xs px-2 py-0.5 rounded-full border ${getStatusColor(enhancedStatus)} ${
                      ['connecting', 'processing', 'completing', 'queued'].includes(enhancedStatus) 
                        ? 'animate-pulse' 
                        : ''
                    }`}>
                      {/* Status icon */}
                      <span className="flex items-center gap-1">
                        {enhancedStatus === 'connecting' && (
                          <svg className="w-3 h-3 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                          </svg>
                        )}
                        {enhancedStatus === 'processing' && (
                          <svg className="w-3 h-3 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                          </svg>
                        )}
                        {enhancedStatus === 'completing' && (
                          <svg className="w-3 h-3 animate-bounce" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                          </svg>
                        )}
                        {enhancedStatus === 'completed' && (
                          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                          </svg>
                        )}
                        {enhancedStatus === 'failed' && (
                          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        )}
                        {enhancedStatus === 'queued' && (
                          <svg className="w-3 h-3 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                          </svg>
                        )}
                        {String(enhancedStatus)}
                      </span>
                    </span>
                  </div>
                  {(e?.error || e?.failure_reason) && (
                    <span className="text-xs text-red-400 truncate" title={e?.error || e?.failure_reason}>
                      {(e?.error || e?.failure_reason).substring(0, 20)}...
                    </span>
                  )}
                </div>
              </div>
              <div className="px-3 py-2 text-xs" title={started ? new Date(started).toLocaleString() : ''}>{started ? timeAgo(started) : '—'}</div>
              <div className="px-3 py-2 font-mono text-xs">
                <div className="flex flex-col gap-1">
                  <span title={provider}>{provider ? truncateMiddle(provider, 6, 4) : '—'}</span>
                  {providerHealth && (
                    <span className={`text-xs px-1 py-0.5 rounded ${
                      providerHealth === 'healthy' ? 'bg-green-900/20 text-green-400' :
                      providerHealth === 'degraded' ? 'bg-yellow-900/20 text-yellow-400' :
                      providerHealth === 'unhealthy' ? 'bg-red-900/20 text-red-400' :
                      'bg-gray-900/20 text-gray-400'
                    }`}>
                      {providerHealth}
                    </span>
                  )}
                </div>
              </div>
              <div className="px-3 py-2 text-xs">{Number.isFinite(retries) ? retries : '—'}</div>
              <div className="px-3 py-2 text-xs">
                <span className={`${
                  ['connecting', 'processing', 'completing'].includes(enhancedStatus) 
                    ? 'text-yellow-400 animate-pulse' 
                    : 'text-gray-400'
                }`}>
                  {calculateETA()}
                </span>
              </div>
              <div className="px-3 py-2"><VerifyBadge exec={e} /></div>
            </div>
          );
        })}
      </div>

      {/* Action Buttons */}
      {!isCompleted && (
        <div className="flex items-center justify-end gap-2">
          <button 
            onClick={refetchActive} 
            className="px-3 py-1.5 bg-green-600 text-white rounded text-sm hover:bg-green-700 transition-colors"
          >
            Refresh
          </button>
          {activeJob?.id && (
            <Link 
              to={`/jobs/${activeJob.id}`} 
              className="text-sm text-beacon-600 underline decoration-dotted hover:text-beacon-500"
            >
              View details
            </Link>
          )}
        </div>
      )}
    </div>
  );
}
