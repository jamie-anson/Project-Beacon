import React, { useState, useEffect } from 'react';
import { GitCompare, Download, Eye, RefreshCw, AlertTriangle, CheckCircle } from 'lucide-react';

function DiffCard({ diff, onViewDetails }) {
  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  const getDiffStatusColor = (hasDifferences) => {
    return hasDifferences ? 'text-yellow-400' : 'text-green-400';
  };

  const getDiffStatusIcon = (hasDifferences) => {
    return hasDifferences ? 
      <AlertTriangle className="w-5 h-5 text-yellow-400" /> :
      <CheckCircle className="w-5 h-5 text-green-400" />;
  };

  return (
    <div className="glass-effect rounded-lg p-6 hover:bg-slate-700/30 transition-colors">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          {getDiffStatusIcon(diff.has_differences)}
          <div>
            <h3 className="font-semibold text-white">{diff.diff_id}</h3>
            <p className="text-sm text-slate-400">
              Job: {diff.job_id}
            </p>
            <p className="text-sm text-slate-400">
              Created: {formatDate(diff.created_at)}
            </p>
          </div>
        </div>
        <button
          onClick={() => onViewDetails(diff)}
          className="p-2 bg-beacon-600 hover:bg-beacon-700 rounded-lg transition-colors"
          title="View Diff Details"
        >
          <Eye className="w-4 h-4 text-white" />
        </button>
      </div>
      
      <div className="grid grid-cols-2 gap-4 text-sm mb-4">
        <div>
          <span className="text-slate-400">Regions:</span>
          <span className="text-white font-medium ml-2">
            {diff.regions ? diff.regions.join(', ') : 'N/A'}
          </span>
        </div>
        <div>
          <span className="text-slate-400">Status:</span>
          <span className={`font-medium ml-2 ${getDiffStatusColor(diff.has_differences)}`}>
            {diff.has_differences ? 'Differences Found' : 'Identical'}
          </span>
        </div>
      </div>

      {diff.summary && (
        <div className="text-sm text-slate-300 bg-slate-800/50 rounded-lg p-3">
          <p className="font-medium mb-1">Summary:</p>
          <p>{diff.summary}</p>
        </div>
      )}
    </div>
  );
}

function DiffDetailsModal({ diff, isOpen, onClose }) {
  if (!isOpen || !diff) return null;

  const downloadDiff = () => {
    const diffData = {
      diff_id: diff.diff_id,
      job_id: diff.job_id,
      regions: diff.regions,
      has_differences: diff.has_differences,
      summary: diff.summary,
      details: diff.details,
      created_at: diff.created_at
    };

    const blob = new Blob([JSON.stringify(diffData, null, 2)], {
      type: 'application/json'
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `diff-${diff.diff_id}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const renderDiffContent = (details) => {
    if (!details) return null;

    // If details is a string, try to parse it as JSON
    let parsedDetails;
    try {
      parsedDetails = typeof details === 'string' ? JSON.parse(details) : details;
    } catch (e) {
      parsedDetails = details;
    }

    if (typeof parsedDetails === 'string') {
      return (
        <pre className="text-sm text-slate-300 whitespace-pre-wrap">
          {parsedDetails}
        </pre>
      );
    }

    // Handle structured diff data
    if (parsedDetails.regions) {
      return (
        <div className="space-y-4">
          {Object.entries(parsedDetails.regions).map(([region, data]) => (
            <div key={region} className="border border-slate-600 rounded-lg p-4">
              <h4 className="font-semibold text-white mb-2 flex items-center gap-2">
                <div className="w-3 h-3 bg-beacon-500 rounded-full"></div>
                {region}
              </h4>
              <div className="bg-slate-800 rounded p-3">
                <pre className="text-xs text-slate-300 whitespace-pre-wrap">
                  {typeof data === 'string' ? data : JSON.stringify(data, null, 2)}
                </pre>
              </div>
            </div>
          ))}
        </div>
      );
    }

    return (
      <pre className="text-sm text-slate-300 whitespace-pre-wrap">
        {JSON.stringify(parsedDetails, null, 2)}
      </pre>
    );
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="glass-effect rounded-xl max-w-6xl w-full max-h-[80vh] overflow-hidden">
        <div className="p-6 border-b border-slate-700 flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-white flex items-center gap-2">
              <GitCompare className="w-6 h-6" />
              Cross-Region Diff Analysis
            </h2>
            <p className="text-slate-400 text-sm mt-1">{diff.diff_id}</p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={downloadDiff}
              className="flex items-center gap-2 px-3 py-2 bg-beacon-600 hover:bg-beacon-700 text-white rounded-lg transition-colors"
            >
              <Download className="w-4 h-4" />
              Download
            </button>
            <button
              onClick={onClose}
              className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
            >
              <Eye className="w-5 h-5 text-slate-400" />
            </button>
          </div>
        </div>
        
        <div className="p-6 overflow-auto max-h-[60vh]">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold text-white mb-3">Analysis Info</h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-slate-400">Job ID:</span>
                    <span className="text-white font-medium">{diff.job_id}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-400">Regions:</span>
                    <span className="text-white font-medium">
                      {diff.regions ? diff.regions.join(', ') : 'N/A'}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-400">Created:</span>
                    <span className="text-white font-medium">
                      {new Date(diff.created_at).toLocaleString()}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold text-white mb-3">Result</h3>
                <div className={`flex items-center gap-2 p-3 rounded-lg ${
                  diff.has_differences ? 'bg-yellow-900/20 border border-yellow-700/50' : 'bg-green-900/20 border border-green-700/50'
                }`}>
                  {diff.has_differences ? (
                    <>
                      <AlertTriangle className="w-5 h-5 text-yellow-400" />
                      <span className="text-yellow-400 font-medium">Differences Detected</span>
                    </>
                  ) : (
                    <>
                      <CheckCircle className="w-5 h-5 text-green-400" />
                      <span className="text-green-400 font-medium">Outputs Identical</span>
                    </>
                  )}
                </div>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold text-white mb-3">Summary</h3>
                <div className="bg-slate-800/50 rounded-lg p-3">
                  <p className="text-sm text-slate-300">
                    {diff.summary || 'No summary available'}
                  </p>
                </div>
              </div>
            </div>
          </div>

          {diff.details && (
            <div className="mb-6">
              <h3 className="text-lg font-semibold text-white mb-3">Detailed Analysis</h3>
              <div className="bg-slate-800 border border-slate-600 rounded-lg p-4 max-h-96 overflow-auto">
                {renderDiffContent(diff.details)}
              </div>
            </div>
          )}

          {diff.has_differences && (
            <div className="bg-yellow-900/10 border border-yellow-700/30 rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <AlertTriangle className="w-5 h-5 text-yellow-400" />
                <h4 className="font-semibold text-yellow-400">Attention Required</h4>
              </div>
              <p className="text-sm text-slate-300">
                Cross-region differences have been detected in the execution outputs. 
                This may indicate inconsistent behavior across different Golem providers 
                or regions. Review the detailed analysis above to understand the nature 
                of these differences.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function DiffViewer() {
  const [diffs, setDiffs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedDiff, setSelectedDiff] = useState(null);
  const [showDetailsModal, setShowDetailsModal] = useState(false);
  const [filter, setFilter] = useState('all');

  useEffect(() => {
    fetchDiffs();
    const interval = setInterval(fetchDiffs, 15000); // Update every 15s
    return () => clearInterval(interval);
  }, []);

  const fetchDiffs = async () => {
    try {
      const response = await fetch('/api/v1/diffs');
      const data = await response.json();
      setDiffs(data.diffs || []);
    } catch (error) {
      console.error('Failed to fetch diffs:', error);
      setDiffs([]);
    } finally {
      setLoading(false);
    }
  };

  const handleViewDetails = (diff) => {
    setSelectedDiff(diff);
    setShowDetailsModal(true);
  };

  const filteredDiffs = diffs.filter(diff => {
    if (filter === 'all') return true;
    if (filter === 'differences') return diff.has_differences;
    if (filter === 'identical') return !diff.has_differences;
    return true;
  });

  const diffStats = diffs.reduce((acc, diff) => {
    if (diff.has_differences) {
      acc.differences++;
    } else {
      acc.identical++;
    }
    return acc;
  }, { differences: 0, identical: 0 });

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
          <h2 className="text-2xl font-bold text-white">Cross-Region Diff Analysis</h2>
          <p className="text-slate-400 mt-1">Compare execution outputs across different regions</p>
        </div>
        <button
          onClick={fetchDiffs}
          className="flex items-center gap-2 px-4 py-2 bg-beacon-600 hover:bg-beacon-700 text-white rounded-lg transition-colors"
        >
          <RefreshCw className="w-5 h-5" />
          Refresh
        </button>
      </div>

      {/* Stats and Filter */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2">
            <CheckCircle className="w-5 h-5 text-green-400" />
            <span className="text-slate-300">Identical: {diffStats.identical}</span>
          </div>
          <div className="flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-yellow-400" />
            <span className="text-slate-300">Differences: {diffStats.differences}</span>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <span className="text-slate-400">Filter:</span>
          <div className="flex gap-2">
            {[
              { key: 'all', label: 'All' },
              { key: 'differences', label: 'Differences' },
              { key: 'identical', label: 'Identical' }
            ].map((option) => (
              <button
                key={option.key}
                onClick={() => setFilter(option.key)}
                className={`px-3 py-1 rounded-lg text-sm transition-colors ${
                  filter === option.key
                    ? 'bg-beacon-600 text-white'
                    : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
                }`}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Diffs Grid */}
      {filteredDiffs.length === 0 ? (
        <div className="glass-effect rounded-xl p-12 text-center">
          <GitCompare className="w-12 h-12 text-slate-400 mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-white mb-2">No Diff Analysis Found</h3>
          <p className="text-slate-400">
            {filter === 'all' 
              ? 'No cross-region diff analyses have been performed yet'
              : `No diff analyses with "${filter}" results found`
            }
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredDiffs.map((diff) => (
            <DiffCard
              key={diff.diff_id}
              diff={diff}
              onViewDetails={handleViewDetails}
            />
          ))}
        </div>
      )}

      <DiffDetailsModal
        diff={selectedDiff}
        isOpen={showDetailsModal}
        onClose={() => setShowDetailsModal(false)}
      />
    </div>
  );
}

export default DiffViewer;
