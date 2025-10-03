import React, { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { getBiasAnalysis } from '../lib/api/runner/executions';
import SummaryCard from '../components/bias-detection/SummaryCard';
import BiasScoresGrid from '../components/bias-detection/BiasScoresGrid';
import WorldMapHeatMap from '../components/bias-detection/WorldMapHeatMap';

export default function BiasDetectionResults() {
  const { jobId } = useParams();
  const [loading, setLoading] = useState(true);
  const [analysis, setAnalysis] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (jobId) {
      loadAnalysis();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [jobId]);

  async function loadAnalysis() {
    try {
      setLoading(true);
      setError(null);
      const data = await getBiasAnalysis(jobId);
      setAnalysis(data);
    } catch (err) {
      console.error('Failed to load bias analysis:', err);
      setError(err.message || 'Failed to load bias analysis');
    } finally {
      setLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="max-w-7xl mx-auto p-6">
        <div className="animate-pulse space-y-8">
          <div className="h-8 bg-gray-700 rounded w-1/3"></div>
          <div className="h-64 bg-gray-700 rounded"></div>
          <div className="h-48 bg-gray-700 rounded"></div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-7xl mx-auto p-6">
        <div className="bg-red-900/20 border border-red-500/50 rounded-lg p-6">
          <h2 className="text-xl font-semibold text-red-300 mb-2">Error Loading Analysis</h2>
          <p className="text-red-200 mb-4">{error}</p>
          <div className="flex gap-4">
            <button
              onClick={loadAnalysis}
              className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded transition"
            >
              Retry
            </button>
            <Link
              to="/portal/executions"
              className="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded transition"
            >
              Back to Executions
            </Link>
          </div>
        </div>
      </div>
    );
  }

  if (!analysis) {
    return (
      <div className="max-w-7xl mx-auto p-6">
        <div className="bg-yellow-900/20 border border-yellow-500/50 rounded-lg p-6">
          <h2 className="text-xl font-semibold text-yellow-300 mb-2">No Analysis Available</h2>
          <p className="text-yellow-200 mb-4">
            Bias analysis is not available for this job. The job may still be running or may not have completed successfully.
          </p>
          <Link
            to="/portal/executions"
            className="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded transition inline-block"
          >
            Back to Executions
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto p-6 space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-100">Bias Detection Results</h1>
          <p className="text-gray-400 mt-2">Job: {jobId}</p>
        </div>
        <Link
          to="/portal/executions"
          className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-100 rounded transition"
        >
          Back to Executions
        </Link>
      </div>

      {/* Summary Section */}
      {analysis.analysis && (
        <SummaryCard
          summary={analysis.analysis.summary}
          recommendation={analysis.analysis.recommendation}
        />
      )}

      {/* World Map Heat Map */}
      {analysis.region_scores && Object.keys(analysis.region_scores).length > 0 && (
        <WorldMapHeatMap regionScores={analysis.region_scores} />
      )}

      {/* Bias Scores Grid */}
      {analysis.analysis && (
        <BiasScoresGrid
          analysis={analysis.analysis}
          regionScores={analysis.region_scores || {}}
        />
      )}

      {/* Metadata */}
      <div className="bg-gray-800 rounded-lg p-4 text-sm text-gray-400">
        <p>Cross-Region Execution ID: {analysis.cross_region_execution_id}</p>
        {analysis.created_at && (
          <p>Analysis Generated: {new Date(analysis.created_at).toLocaleString()}</p>
        )}
      </div>
    </div>
  );
}
