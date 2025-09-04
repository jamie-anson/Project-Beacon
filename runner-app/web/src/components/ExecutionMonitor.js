import React, { useState, useEffect } from 'react';
import { RefreshCw, Clock, CheckCircle, XCircle, Eye, Download, Globe } from 'lucide-react';

function ExecutionCard({ execution, onViewDetails }) {
  const getStatusIcon = (status) => {
    switch (status?.toLowerCase()) {
      case 'completed':
        return <CheckCircle className="w-5 h-5 text-green-400" />;
      case 'failed':
        return <XCircle className="w-5 h-5 text-red-400" />;
      case 'running':
        return <RefreshCw className="w-5 h-5 text-blue-400 animate-spin" />;
      default:
        return <Clock className="w-5 h-5 text-yellow-400" />;
    }
  };

  const formatDuration = (startTime, endTime) => {
    if (!startTime) return 'N/A';
    const start = new Date(startTime);
    const end = endTime ? new Date(endTime) : new Date();
    const duration = Math.round((end - start) / 1000);
    return `${duration}s`;
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <div className="glass-effect rounded-lg p-6 hover:bg-slate-700/30 transition-colors">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          {getStatusIcon(execution.status)}
          <div>
            <h3 className="font-semibold text-white">{execution.execution_id}</h3>
            <p className="text-sm text-slate-400">
              Job: {execution.job_id}
            </p>
            <p className="text-sm text-slate-400">
              Started: {formatDate(execution.started_at)}
            </p>
          </div>
        </div>
        <button
          onClick={() => onViewDetails(execution)}
          className="p-2 bg-beacon-600 hover:bg-beacon-700 rounded-lg transition-colors"
          title="View Details"
        >
          <Eye className="w-4 h-4 text-white" />
        </button>
      </div>
      
      <div className="grid grid-cols-2 gap-4 text-sm">
        <div>
          <span className="text-slate-400">Region:</span>
          <span className="text-white font-medium ml-2">{execution.region}</span>
        </div>
        <div>
          <span className="text-slate-400">Duration:</span>
          <span className="text-white font-medium ml-2">
            {formatDuration(execution.started_at, execution.completed_at)}
          </span>
        </div>
        <div>
          <span className="text-slate-400">Status:</span>
          <span className={`font-medium ml-2 ${
            execution.status === 'completed' ? 'text-green-400' :
            execution.status === 'failed' ? 'text-red-400' :
            execution.status === 'running' ? 'text-blue-400' : 'text-yellow-400'
          }`}>
            {execution.status || 'pending'}
          </span>
        </div>
        <div>
          <span className="text-slate-400">Provider:</span>
          <span className="text-white font-medium ml-2">
            {execution.provider_id ? execution.provider_id.substring(0, 8) + '...' : 'N/A'}
          </span>
        </div>
      </div>

      {execution.error && (
        <div className="mt-4 p-3 bg-red-900/20 border border-red-700/50 rounded-lg">
          <p className="text-red-400 text-sm">{execution.error}</p>
        </div>
      )}
    </div>
  );
}

function ExecutionDetailsModal({ execution, isOpen, onClose }) {
  if (!isOpen || !execution) return null;

  const downloadReceipt = () => {
    if (execution.receipt) {
      const blob = new Blob([JSON.stringify(execution.receipt, null, 2)], {
        type: 'application/json'
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `receipt-${execution.execution_id}.json`;
      a.click();
      URL.revokeObjectURL(url);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="glass-effect rounded-xl max-w-4xl w-full max-h-[80vh] overflow-hidden">
        <div className="p-6 border-b border-slate-700 flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-white">Execution Details</h2>
            <p className="text-slate-400 text-sm mt-1">{execution.execution_id}</p>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
          >
            <XCircle className="w-5 h-5 text-slate-400" />
          </button>
        </div>
        
        <div className="p-6 overflow-auto max-h-[60vh]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold text-white mb-3">Execution Info</h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-slate-400">Job ID:</span>
                    <span className="text-white font-medium">{execution.job_id}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-400">Region:</span>
                    <span className="text-white font-medium">{execution.region}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-400">Provider:</span>
                    <span className="text-white font-medium">
                      {execution.provider_id || 'N/A'}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-400">Status:</span>
                    <span className={`font-medium ${
                      execution.status === 'completed' ? 'text-green-400' :
                      execution.status === 'failed' ? 'text-red-400' :
                      execution.status === 'running' ? 'text-blue-400' : 'text-yellow-400'
                    }`}>
                      {execution.status}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold text-white mb-3">Timing</h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-slate-400">Started:</span>
                    <span className="text-white font-medium">
                      {new Date(execution.started_at).toLocaleString()}
                    </span>
                  </div>
                  {execution.completed_at && (
                    <div className="flex justify-between">
                      <span className="text-slate-400">Completed:</span>
                      <span className="text-white font-medium">
                        {new Date(execution.completed_at).toLocaleString()}
                      </span>
                    </div>
                  )}
                  <div className="flex justify-between">
                    <span className="text-slate-400">Duration:</span>
                    <span className="text-white font-medium">
                      {execution.started_at && execution.completed_at
                        ? `${Math.round((new Date(execution.completed_at) - new Date(execution.started_at)) / 1000)}s`
                        : 'N/A'}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {execution.output && (
            <div className="mb-6">
              <h3 className="text-lg font-semibold text-white mb-3">Output</h3>
              <div className="bg-slate-800 border border-slate-600 rounded-lg p-4">
                <pre className="text-sm text-slate-300 whitespace-pre-wrap">
                  {execution.output}
                </pre>
              </div>
            </div>
          )}

          {execution.error && (
            <div className="mb-6">
              <h3 className="text-lg font-semibold text-white mb-3">Error</h3>
              <div className="bg-red-900/20 border border-red-700/50 rounded-lg p-4">
                <pre className="text-sm text-red-400 whitespace-pre-wrap">
                  {execution.error}
                </pre>
              </div>
            </div>
          )}

          {execution.receipt && (
            <div className="mb-6">
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-lg font-semibold text-white">Cryptographic Receipt</h3>
                <button
                  onClick={downloadReceipt}
                  className="flex items-center gap-2 px-3 py-1 bg-beacon-600 hover:bg-beacon-700 text-white rounded-lg transition-colors text-sm"
                >
                  <Download className="w-4 h-4" />
                  Download
                </button>
              </div>
              <div className="bg-slate-800 border border-slate-600 rounded-lg p-4 max-h-64 overflow-auto">
                <pre className="text-xs text-slate-300">
                  {JSON.stringify(execution.receipt, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function ExecutionMonitor() {
  const [executions, setExecutions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedExecution, setSelectedExecution] = useState(null);
  const [showDetailsModal, setShowDetailsModal] = useState(false);
  const [filter, setFilter] = useState('all');

  useEffect(() => {
    fetchExecutions();
    const interval = setInterval(fetchExecutions, 10000); // Update every 10s
    return () => clearInterval(interval);
  }, []);

  const fetchExecutions = async () => {
    try {
      const response = await fetch('/api/v1/executions');
      const data = await response.json();
      setExecutions(data.executions || []);
    } catch (error) {
      console.error('Failed to fetch executions:', error);
      setExecutions([]);
    } finally {
      setLoading(false);
    }
  };

  const handleViewDetails = (execution) => {
    setSelectedExecution(execution);
    setShowDetailsModal(true);
  };

  const filteredExecutions = executions.filter(execution => {
    if (filter === 'all') return true;
    return execution.status === filter;
  });

  const statusCounts = executions.reduce((acc, execution) => {
    acc[execution.status] = (acc[execution.status] || 0) + 1;
    return acc;
  }, {});

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-8 h-8 border-4 border-beacon-400 border-t-transparent rounded-full animate-spin"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-white">Execution Monitor</h2>
          <p className="text-slate-400 mt-1">Real-time monitoring of Golem network executions</p>
        </div>
        <button
          onClick={fetchExecutions}
          className="flex items-center gap-2 px-4 py-2 bg-beacon-600 hover:bg-beacon-700 text-white rounded-lg transition-colors"
        >
          <RefreshCw className="w-5 h-5" />
          Refresh
        </button>
      </div>

      {/* Status Filter */}
      <div className="flex items-center gap-4">
        <span className="text-slate-400">Filter by status:</span>
        <div className="flex gap-2">
          {['all', 'running', 'completed', 'failed'].map((status) => (
            <button
              key={status}
              onClick={() => setFilter(status)}
              className={`px-3 py-1 rounded-lg text-sm transition-colors ${
                filter === status
                  ? 'bg-beacon-600 text-white'
                  : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
              }`}
            >
              {status.charAt(0).toUpperCase() + status.slice(1)}
              {status !== 'all' && statusCounts[status] && (
                <span className="ml-1 text-xs">({statusCounts[status]})</span>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* Executions Grid */}
      {filteredExecutions.length === 0 ? (
        <div className="glass-effect rounded-xl p-12 text-center">
          <Globe className="w-12 h-12 text-slate-400 mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-white mb-2">No Executions Found</h3>
          <p className="text-slate-400">
            {filter === 'all' 
              ? 'No executions have been started yet'
              : `No executions with status "${filter}" found`
            }
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredExecutions.map((execution) => (
            <ExecutionCard
              key={execution.execution_id}
              execution={execution}
              onViewDetails={handleViewDetails}
            />
          ))}
        </div>
      )}

      <ExecutionDetailsModal
        execution={selectedExecution}
        isOpen={showDetailsModal}
        onClose={() => setShowDetailsModal(false)}
      />
    </div>
  );
}

export default ExecutionMonitor;
