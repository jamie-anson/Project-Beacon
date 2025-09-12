import React from 'react';
import { useParams, Link } from 'react-router-dom';
import { useQuery } from '../state/useQuery.js';
import { getExecution, getExecutionReceipt } from '../lib/api.js';
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

function formatDate(ts) {
  if (!ts) return 'N/A';
  try { return new Date(ts).toLocaleString(); } catch { return String(ts); }
}

function formatDuration(nanoseconds) {
  if (!nanoseconds) return 'N/A';
  const seconds = nanoseconds / 1000000000;
  return `${seconds.toFixed(2)}s`;
}

function truncateMiddle(str, maxLength = 40) {
  if (!str || str.length <= maxLength) return str || 'N/A';
  const start = Math.floor(maxLength / 2) - 2;
  const end = Math.floor(maxLength / 2) - 2;
  return `${str.slice(0, start)}...${str.slice(-end)}`;
}

export default function ExecutionDetail() {
  const { id } = useParams();
  
  const { data: execution, loading: executionLoading, error: executionError } = useQuery(
    `execution-${id}`, 
    () => getExecution(id), 
    { interval: 10000 }
  );

  const { data: receipt, loading: receiptLoading, error: receiptError } = useQuery(
    `receipt-${id}`, 
    () => getExecutionReceipt(id), 
    { interval: 30000 }
  );

  if (executionLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-2">
          <Link to="/executions" className="text-beacon-600 hover:text-beacon-700">← Back to Executions</Link>
        </div>
        <div className="bg-white border rounded-lg p-6">
          <div className="animate-pulse space-y-4">
            <div className="h-6 bg-slate-200 rounded w-1/3"></div>
            <div className="h-4 bg-slate-200 rounded w-1/2"></div>
            <div className="space-y-2">
              <div className="h-3 bg-slate-200 rounded"></div>
              <div className="h-3 bg-slate-200 rounded w-3/4"></div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (executionError || !execution) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-2">
          <Link to="/executions" className="text-beacon-600 hover:text-beacon-700">← Back to Executions</Link>
        </div>
        <div className="bg-red-50 border border-red-200 rounded-lg p-6">
          <div className="flex items-center gap-2 text-red-800">
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd"></path>
            </svg>
            <span className="font-medium">Execution not found</span>
          </div>
          <p className="mt-1 text-red-700 text-sm">
            {executionError?.message || `Execution ${id} could not be loaded.`}
          </p>
        </div>
      </div>
    );
  }

  const hasReceiptData = receipt && typeof receipt === 'object';

  return (
    <div className="space-y-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm">
        <Link to="/executions" className="text-beacon-600 hover:text-beacon-700">← Back to Executions</Link>
      </div>

      {/* Execution Header */}
      <div className="bg-white border rounded-lg overflow-hidden">
        <div className="border-b bg-slate-50 px-6 py-4">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-xl font-semibold text-slate-900">Execution {execution.id}</h1>
              <p className="text-sm text-slate-600 mt-1">
                Job ID: {execution.job_id ? (
                  <Link to={`/jobs/${execution.job_id}`} className="text-beacon-600 hover:text-beacon-700 underline decoration-dotted">
                    {execution.job_id}
                  </Link>
                ) : 'N/A'}
              </p>
            </div>
            <div className="flex items-center gap-3">
              <StatusPill value={execution.status} />
              <CopyButton text={String(execution.id)} label="Copy ID" />
            </div>
          </div>
        </div>

        <div className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 text-sm">
            <div className="space-y-3">
              <div className="flex justify-between">
                <span className="text-slate-600">Region:</span>
                <span className="font-semibold text-beacon-600">{execution.region || 'N/A'}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-600">Started:</span>
                <span>{formatDate(execution.started_at)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-600">Provider:</span>
                <code className="text-xs bg-slate-100 px-2 py-1 rounded">{execution.provider_id || 'N/A'}</code>
              </div>
            </div>
            <div className="space-y-3">
              <div className="flex justify-between">
                <span className="text-slate-600">Duration:</span>
                <span className="font-mono">{formatDuration(execution.duration)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-600">Completed:</span>
                <span>{formatDate(execution.completed_at)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-600">Exit Code:</span>
                <span className="font-mono">{execution.exit_code ?? 'N/A'}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Receipt Section */}
      <div className="bg-white border rounded-lg overflow-hidden">
        <div className="border-b bg-slate-50 px-6 py-4">
          <h2 className="font-semibold text-slate-900">Execution Receipt</h2>
          <p className="text-sm text-slate-600 mt-1">Cryptographic proof of execution with detailed provenance</p>
        </div>

        <div className="p-6">
          {receiptLoading ? (
            <div className="text-center py-8">
              <div className="animate-spin inline-block w-6 h-6 border-2 border-beacon-600 border-t-transparent rounded-full"></div>
              <p className="mt-2 text-slate-600">Loading receipt data...</p>
            </div>
          ) : receiptError ? (
            <div className="text-center py-8">
              <div className="text-slate-400 mb-2">📄</div>
              <p className="text-slate-500">Receipt data not available</p>
              <p className="text-sm text-slate-400 mt-1">{receiptError.message}</p>
            </div>
          ) : !hasReceiptData ? (
            <div className="text-center py-8">
              <div className="text-slate-400 mb-2">📄</div>
              <p className="text-slate-500">No receipt data available for this execution</p>
            </div>
          ) : (
            <div className="space-y-6">
              {/* AI Output Section */}
              {receipt.output && (
                <div className="space-y-3">
                  <h4 className="font-medium text-slate-900 flex items-center gap-2">
                    <span className="text-blue-500">🤖</span>
                    AI Response Output
                  </h4>
                  <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                    <div className="text-sm text-blue-900 font-medium mb-2">Generated Response:</div>
                    <div className="text-blue-800 italic">
                      "{receipt.output?.data?.text_output || receipt.output?.stdout || 'No output available'}"
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div className="flex justify-between">
                      <span className="text-slate-600">Tokens Generated:</span>
                      <span className="font-mono">{receipt.output?.data?.metadata?.tokens_generated || 'N/A'}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-600">Execution Time:</span>
                      <span className="font-mono">{receipt.output?.data?.metadata?.execution_time || 'N/A'}</span>
                    </div>
                  </div>
                  {receipt.output?.hash && (
                    <div className="text-xs">
                      <span className="text-slate-600">Output Hash:</span>
                      <code className="ml-2 bg-slate-100 px-2 py-1 rounded text-slate-800">{truncateMiddle(receipt.output.hash)}</code>
                      <CopyButton text={receipt.output.hash} label="Copy" className="ml-2" />
                    </div>
                  )}
                </div>
              )}

              {/* Provider Information */}
              {receipt.provenance?.provider_info && (
                <div className="space-y-3">
                  <h4 className="font-medium text-slate-900 flex items-center gap-2">
                    <span className="text-purple-500">🏢</span>
                    Provider Information
                  </h4>
                  <div className="bg-purple-50 border border-purple-200 rounded-lg p-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                      <div className="space-y-2">
                        <div className="flex justify-between">
                          <span className="text-slate-600">Name:</span>
                          <span className="font-medium">{receipt.provenance.provider_info.name || 'N/A'}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-slate-600">Score:</span>
                          <span className="font-mono">{receipt.provenance.provider_info.score || 'N/A'}</span>
                        </div>
                      </div>
                      <div className="space-y-2">
                        <div className="flex justify-between">
                          <span className="text-slate-600">CPU:</span>
                          <span className="font-mono">{receipt.provenance.provider_info.resources?.cpu || 'N/A'}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-slate-600">Memory:</span>
                          <span className="font-mono">{receipt.provenance.provider_info.resources?.memory || 'N/A'}MB</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* Cryptographic Verification */}
              <div className="space-y-3">
                <h4 className="font-medium text-slate-900 flex items-center gap-2">
                  <span className="text-amber-500">🔐</span>
                  Cryptographic Verification
                </h4>
                <div className="space-y-3">
                  {receipt.jobspec_id && (
                    <div className="text-sm">
                      <span className="text-slate-600">JobSpec ID:</span>
                      <code className="ml-2 bg-slate-100 px-2 py-1 rounded text-slate-800">{truncateMiddle(receipt.jobspec_id)}</code>
                      <CopyButton text={receipt.jobspec_id} label="Copy" className="ml-2" />
                    </div>
                  )}
                  {receipt.public_key && (
                    <div className="text-sm">
                      <span className="text-slate-600">Public Key:</span>
                      <code className="ml-2 bg-slate-100 px-2 py-1 rounded text-slate-800">{truncateMiddle(receipt.public_key)}</code>
                      <CopyButton text={receipt.public_key} label="Copy" className="ml-2" />
                    </div>
                  )}
                  {receipt.signature && (
                    <div className="bg-green-50 border border-green-200 rounded-lg p-3">
                      <div className="text-sm font-medium text-green-800 mb-2 flex items-center justify-between">
                        <span>Digital Signature:</span>
                        <CopyButton text={receipt.signature} label="Copy Signature" />
                      </div>
                      <code className="text-xs text-green-700 break-all">{receipt.signature}</code>
                    </div>
                  )}
                  {receipt.provenance?.benchmark_hash && (
                    <div className="text-sm">
                      <span className="text-slate-600">Benchmark Hash:</span>
                      <code className="ml-2 bg-slate-100 px-2 py-1 rounded text-slate-800">{truncateMiddle(receipt.provenance.benchmark_hash)}</code>
                      <CopyButton text={receipt.provenance.benchmark_hash} label="Copy" className="ml-2" />
                    </div>
                  )}
                  {receipt.provenance?.execution_env?.container_image && (
                    <div className="text-sm">
                      <span className="text-slate-600">Container Image:</span>
                      <code className="ml-2 bg-slate-100 px-2 py-1 rounded text-slate-800">{receipt.provenance.execution_env.container_image}</code>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
