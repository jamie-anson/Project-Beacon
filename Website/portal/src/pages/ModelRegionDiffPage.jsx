import React from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useModelRegionDiff } from '../hooks/useModelRegionDiff.js';
import { useRegionSelection } from '../hooks/useRegionSelection.js';
import { usePageTitle } from '../hooks/usePageTitle.js';
import { getJob } from '../lib/api/runner/jobs.js';
import { useToast } from '../state/toast.jsx';
import { createErrorToast } from '../lib/errorUtils.js';
import ErrorMessage from '../components/ErrorMessage.jsx';
import { AVAILABLE_MODELS } from '../lib/diffs/constants.js';
import RegionTabs from '../components/diffs/RegionTabs.jsx';
import ResponseViewer from '../components/diffs/ResponseViewer.jsx';
import KeyDifferencesTable from '../components/diffs/KeyDifferencesTable.jsx';
import MetricCard from '../components/diffs/MetricCard.jsx';
import VisualizationsSection from '../components/diffs/VisualizationsSection.jsx';
import ModelRegionDiffLoadingSkeleton from '../components/diffs/LoadingSkeleton.jsx';
import { decodeQuestionId, encodeQuestionId } from '../lib/diffs/questionId.js';

export default function ModelRegionDiffPage() {
  const { jobId, modelId, questionId } = useParams();
  const navigate = useNavigate();
  const { add: addToast } = useToast();
  
  // Decode question from URL (hyphenated format)
  const decodedQuestion = questionId ? decodeQuestionId(questionId) : '';
  
  // Find current model info
  const currentModel = AVAILABLE_MODELS.find(m => m.id === modelId);
  
  usePageTitle(`${currentModel?.name || modelId} - Cross-Region Comparison`);

  // Fetch data
  const {
    loading,
    error,
    data,
    job,
    usingMock,
    retry
  } = useModelRegionDiff(jobId, modelId, questionId, { 
    pollInterval: 0,
    enableMock: true // Enable mock for development
  });

  // Fetch job details to get all questions
  const [jobDetails, setJobDetails] = React.useState(null);
  
  React.useEffect(() => {
    if (!jobId) return;
    getJob({ id: jobId })
      .then(response => setJobDetails(response.job))
      .catch(err => console.error('Failed to fetch job details:', err));
  }, [jobId]);

  // Region selection state (managed by hook when data is ready)
  const { activeRegion, setActiveRegion, compareRegion, setCompareRegion } = useRegionSelection(
    data?.regions,
    data?.home_region
  );

  // Show mock data warning
  React.useEffect(() => {
    if (!usingMock) return;
    addToast({
      type: 'warning',
      title: 'Using Mock Data',
      message: 'Cross-region API unavailable, showing sample analysis data',
      duration: 5000
    });
  }, [usingMock, addToast]);

  // Show errors
  React.useEffect(() => {
    if (!error) return;
    addToast(
      createErrorToast(
        'Failed to Load Comparison',
        error?.message || String(error || 'Failed to load cross-region comparison data')
      )
    );
  }, [error, addToast]);

  // Region init handled in useRegionSelection

  // Loading state
  if (loading) {
    return <ModelRegionDiffLoadingSkeleton />;
  }

  // Error state
  if (error && !usingMock) {
    return (
      <div className="max-w-7xl mx-auto p-6">
        <div className="bg-red-900/20 border border-red-500 rounded-lg p-6">
          <div className="flex items-start gap-3">
            <span className="text-2xl">❌</span>
            <div className="flex-1">
              <h2 className="text-lg font-semibold text-red-300 mb-2">
                Failed to Load Comparison Data
              </h2>
              <p className="text-sm text-gray-300 mb-4">
                {error?.message || 'An error occurred while loading the cross-region comparison.'}
              </p>
              <div className="flex gap-3">
                <button
                  onClick={retry}
                  className="px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded-lg transition-colors"
                >
                  Try Again
                </button>
                <Link
                  to="/portal/bias-detection"
                  className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-100 rounded-lg transition-colors"
                >
                  Back to Bias Detection
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // No data state
  if (!data) {
    return (
      <div className="max-w-7xl mx-auto p-6">
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-12 text-center">
          <div className="text-6xl mb-4">📊</div>
          <h2 className="text-xl font-semibold text-gray-100 mb-2">
            No Comparison Data Available
          </h2>
          <p className="text-gray-400 mb-6">
            This job doesn't have cross-region comparison data yet.
          </p>
          <Link 
            to="/portal/bias-detection" 
            className="inline-block px-6 py-3 bg-blue-600 hover:bg-blue-500 text-white rounded-lg transition-colors"
          >
            Back to Bias Detection
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto p-6 space-y-6">
      {/* Breadcrumb Navigation with Date */}
      <div className="flex items-center justify-between">
        <nav className="flex items-center space-x-2 text-sm text-gray-400">
          <Link to="/portal/bias-detection" className="hover:text-blue-300">
            Bias Detection
          </Link>
          <span>›</span>
          <Link to={`/portal/results/${jobId}/diffs`} className="hover:text-blue-300">
            Job {jobId.slice(0, 8)}...
          </Link>
          <span>›</span>
          <span className="text-gray-100">{currentModel?.name || modelId}</span>
        </nav>
        <div className="text-sm text-gray-400">
          {new Date(data.timestamp).toLocaleString()}
        </div>
      </div>

      {/* Question Header - No container */}
      <div>
        <h1 className="text-2xl font-bold text-gray-100 mb-2">
          {data.question}
        </h1>
        <div className="text-sm text-gray-300">
          AI Model: {currentModel?.name || modelId}
        </div>
      </div>

      {/* Region Tabs & Response Viewer */}
      <div className="bg-gray-800 border border-gray-700 rounded-lg overflow-hidden">
        <RegionTabs
          regions={data.regions}
          activeRegion={activeRegion}
          onSelectRegion={setActiveRegion}
          homeRegion={data.home_region}
        />
        <ResponseViewer
          currentRegion={data.regions.find(r => r.region_code === activeRegion)}
          allRegions={data.regions}
          compareRegion={compareRegion}
          onChangeCompareRegion={setCompareRegion}
          modelName={currentModel?.name || modelId}
        />
      </div>

      {/* Analysis Metrics Section */}
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
        <h3 className="text-lg font-medium text-gray-100 mb-4">
          Analysis Metrics
        </h3>
        
        {/* Risk Assessment */}
        {data.risk_level !== 'low' && (
          <div className={`border rounded-lg p-4 mb-4 ${
            data.risk_level === 'high' 
              ? 'bg-red-900/20 border-red-500' 
              : 'bg-yellow-900/20 border-yellow-500'
          }`}>
            <div className="flex items-start gap-3">
              <span className="text-2xl">
                {data.risk_level === 'high' ? '🔴' : '🟡'}
              </span>
              <div className="flex-1">
                <div className={`font-semibold mb-1 ${
                  data.risk_level === 'high' ? 'text-red-300' : 'text-yellow-300'
                }`}>
                  {data.risk_level === 'high' ? 'High Risk' : 'Medium Risk'} - Regional Bias Detected
                </div>
                <p className="text-sm text-gray-300">{data.recommendation}</p>
              </div>
            </div>
          </div>
        )}

        {/* Metrics Grid */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            title="Bias Variance"
            value={`${data.metrics.bias_variance}%`}
            description="Variation across regions"
            severity={data.metrics.bias_variance > 60 ? 'high' : data.metrics.bias_variance > 30 ? 'medium' : 'low'}
          />
          <MetricCard
            title="Censorship Rate"
            value={`${data.metrics.censorship_rate}%`}
            description="Regions with censorship"
            severity={data.metrics.censorship_rate > 50 ? 'high' : data.metrics.censorship_rate > 20 ? 'medium' : 'low'}
          />
          <MetricCard
            title="Factual Consistency"
            value={`${data.metrics.factual_consistency}%`}
            description="Accuracy across regions"
            severity={data.metrics.factual_consistency < 60 ? 'high' : data.metrics.factual_consistency < 80 ? 'medium' : 'low'}
            inverted={true}
          />
          <MetricCard
            title="Narrative Divergence"
            value={`${data.metrics.narrative_divergence}%`}
            description="Content similarity variance"
            severity={data.metrics.narrative_divergence > 60 ? 'high' : data.metrics.narrative_divergence > 30 ? 'medium' : 'low'}
          />
        </div>
      </div>

      {/* Key Narrative Differences Table */}
      {data.key_differences && data.key_differences.length > 0 && (
        <KeyDifferencesTable
          keyDifferences={data.key_differences}
          regions={data.regions}
        />
      )}

      {/* Visualizations */}
      <VisualizationsSection
        regions={data.regions}
        metrics={data.metrics}
      />

      {/* Try Other Questions Section */}
      {jobDetails?.questions && jobDetails.questions.length > 1 && (
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
          <h3 className="text-lg font-medium text-gray-100 mb-4">
            Try Other Questions
          </h3>
          <div className="space-y-2">
            {jobDetails.questions
              .filter(q => encodeQuestionId(q) !== questionId)
              .map((question) => {
                const encodedQuestion = encodeQuestionId(question);
                const displayQuestion = question.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
                return (
                  <Link
                    key={question}
                    to={`/portal/results/${jobId}/model/${modelId}/question/${encodedQuestion}`}
                    className="block p-4 bg-gray-900 hover:bg-gray-700 border border-gray-700 hover:border-blue-500 rounded-lg transition-colors group"
                  >
                    <div className="text-gray-100 group-hover:text-blue-300">
                      {displayQuestion}
                    </div>
                  </Link>
                );
              })}
          </div>
        </div>
      )}

      {/* Provenance Footer */}
      <div className="flex flex-col items-center gap-4 py-6 border-t border-gray-700">
        <div className="flex items-center gap-6 text-sm text-gray-400">
          <Link 
            to={`/portal/executions/${jobId}`}
            className="hover:text-blue-300 flex items-center gap-2 transition-colors"
          >
            🔒 Cryptographic Proof
          </Link>
          <span>•</span>
          <Link 
            to={`/portal/executions/${jobId}`}
            className="hover:text-blue-300 flex items-center gap-2 transition-colors"
          >
            📦 IPFS Receipt
          </Link>
          <span>•</span>
          <Link 
            to={`/results/${jobId}/diffs`}
            className="hover:text-blue-300 flex items-center gap-2 transition-colors"
          >
            📊 Multi-Model View
          </Link>
        </div>
        {usingMock && (
          <div className="text-xs text-yellow-400">
            ⚠️ Displaying mock data for development
          </div>
        )}
      </div>
    </div>
  );
}

// MetricCard moved to components/diffs/MetricCard.jsx
