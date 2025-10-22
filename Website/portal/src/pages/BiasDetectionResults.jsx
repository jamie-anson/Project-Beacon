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
  const [pollCount, setPollCount] = useState(0);
  const [isPolling, setIsPolling] = useState(false);

  useEffect(() => {
    if (jobId) {
      loadAnalysis();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [jobId]);

  // Polling effect - check every 5 seconds if analysis is still loading
  useEffect(() => {
    if (!isPolling || analysis || error) {
      return;
    }

    // Stop polling after 24 attempts (2 minutes)
    if (pollCount >= 24) {
      setError('Analysis is taking longer than expected. Please check back later or view the comparison results.');
      setIsPolling(false);
      setLoading(false);
      return;
    }

    const pollTimer = setTimeout(() => {
      setPollCount(prev => prev + 1);
      loadAnalysis(true); // Silent reload
    }, 5000);

    return () => clearTimeout(pollTimer);
  }, [isPolling, pollCount, analysis, error]);

  async function loadAnalysis(silent = false) {
    try {
      if (!silent) {
        setLoading(true);
        setError(null);
      }
      const data = await getBiasAnalysis(jobId);
      setAnalysis(data);
      setIsPolling(false); // Stop polling once we have data
    } catch (err) {
      console.error('Failed to load bias analysis:', err);
      
      // Check if this is a 404 (analysis not ready yet)
      const isNotFound = err.message?.includes('404') || err.message?.includes('not found');
      
      if (isNotFound && !silent) {
        // Start polling if this is the first load and analysis isn't ready
        setIsPolling(true);
        setPollCount(0);
      } else if (!isNotFound) {
        // Real error, not just "not ready yet"
        setError(err.message || 'Failed to load bias analysis');
        setIsPolling(false);
      }
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  }

  if (loading || isPolling) {
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

        {/* Loading Summary with polling indicator */}
        <SummaryCard loading={true} />

        {isPolling && (
          <div className="bg-blue-900/20 border border-blue-500/50 rounded-lg p-4 text-center">
            <p className="text-blue-300 text-sm">
              Analysis is being generated... Checking again in a few seconds. 
              {pollCount > 0 && ` (Attempt ${pollCount + 1}/24)`}
            </p>
          </div>
        )}

        {/* Skeleton for other sections */}
        <div className="animate-pulse space-y-8">
          <div className="h-64 bg-gray-700 rounded"></div>
          <div className="h-48 bg-gray-700 rounded"></div>
        </div>
      </div>
    );
  }

  if (error) {
    // Check if this is a "not found" error (analysis not generated yet)
    const isNotFound = error.includes('404') || error.includes('not found');
    
    return (
      <div className="max-w-7xl mx-auto p-6">
        <div className={`border rounded-lg p-6 ${
          isNotFound 
            ? 'bg-yellow-900/20 border-yellow-500/50' 
            : 'bg-red-900/20 border-red-500/50'
        }`}>
          <h2 className={`text-xl font-semibold mb-2 ${
            isNotFound ? 'text-yellow-300' : 'text-red-300'
          }`}>
            {isNotFound ? 'Analysis Not Available' : 'Error Loading Analysis'}
          </h2>
          <p className={`mb-4 ${isNotFound ? 'text-yellow-200' : 'text-red-200'}`}>
            {isNotFound ? (
              <>
                Bias analysis generation is not yet available for this job. 
                The job executed successfully, but the analysis feature is still in development.
                <br /><br />
                <strong>You can still view:</strong>
                <ul className="list-disc list-inside mt-2 space-y-1">
                  <li>Individual execution results in the Executions page</li>
                  <li>Cross-region comparison using the "Compare" button</li>
                  <li>Raw execution data and receipts</li>
                </ul>
              </>
            ) : (
              error
            )}
          </p>
          <div className="flex gap-4">
            {!isNotFound && (
              <button
                onClick={loadAnalysis}
                className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded transition"
              >
                Retry
              </button>
            )}
            <Link
              to="/portal/executions"
              className="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded transition"
            >
              Back to Executions
            </Link>
            <Link
              to={`/results/${jobId}/diffs`}
              className="px-4 py-2 bg-beacon-600 hover:bg-beacon-700 text-white rounded transition"
            >
              View Comparison
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
      {loading ? (
        <SummaryCard loading={true} />
      ) : analysis.analysis && (
        <SummaryCard
          summary={analysis.analysis.summary}
          recommendation={analysis.analysis.recommendation}
          summarySource={analysis.analysis.summary_source}
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
