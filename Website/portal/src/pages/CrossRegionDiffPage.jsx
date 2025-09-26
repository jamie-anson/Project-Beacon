import React, { useState, useEffect } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import DiffHeader from '../components/diffs/DiffHeader.jsx';
import ModelSelector from '../components/diffs/ModelSelector.jsx';
import DiffMapSection from '../components/diffs/DiffMapSection.jsx';
import MetricsGrid from '../components/diffs/MetricsGrid.jsx';
import RegionalBreakdown from '../components/diffs/RegionalBreakdown.jsx';
import RecentDiffsList from '../components/diffs/RecentDiffsList.jsx';
import DiffNarrativeTable from '../components/diffs/DiffNarrativeTable.jsx';
import QuickActions from '../components/diffs/QuickActions.jsx';
import { useToast } from '../state/toast.jsx';
import { createErrorToast } from '../lib/errorUtils.js';
import ErrorMessage from '../components/ErrorMessage.jsx';
import { useCrossRegionDiff } from '../hooks/useCrossRegionDiff.js';
import { useRecentDiffs } from '../hooks/useRecentDiffs.js';
import { AVAILABLE_MODELS } from '../lib/diffs/constants.js';
import { useQuery } from '../state/useQuery.js';
import { runnerFetch } from '../lib/api/http.js';

export default function CrossRegionDiffPage() {
  const { jobId } = useParams();
  const navigate = useNavigate();
  const { add: addToast } = useToast();
  const [selectedModel, setSelectedModel] = useState('llama3.2:1b');

  const availableModels = AVAILABLE_MODELS;

  const {
    loading,
    error,
    data: diffAnalysis,
    job,
    usingMock,
    retry: retryDiff
  } = useCrossRegionDiff(jobId, { pollInterval: 0 });

  const { data: recentDiffs } = useRecentDiffs({ limit: 10, pollInterval: 15000 });
  
  // Fetch available questions
  const { data: questionsData } = useQuery('questions', () => runnerFetch('/questions'), { interval: 0 });
  const availableQuestions = questionsData ? [
    ...(questionsData.categories?.control_questions || []),
    ...(questionsData.categories?.bias_detection || []),
    ...(questionsData.categories?.cultural_perspective || [])
  ] : [];

  const handleQuestionSelect = (questionId) => {
    // For now, just show a toast - in the future this could submit a new job with the selected question
    addToast({
      type: 'info',
      title: 'Question Selected',
      message: `Selected question: ${questionId}. Feature to submit new job with this question coming soon!`,
      duration: 3000
    });
  };

  useEffect(() => {
    if (!usingMock) return;
    addToast({
      type: 'warning',
      title: 'Using Mock Data',
      message: 'Cross-region API unavailable, showing sample analysis data',
      duration: 5000
    });
  }, [usingMock, addToast]);

  useEffect(() => {
    if (!error) return;
    addToast(
      createErrorToast(
        'Cross-Region Analysis Error',
        error?.message || String(error || 'Failed to load cross-region analysis data')
      )
    );
  }, [error, addToast]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-400"></div>
        <span className="ml-3 text-gray-300">Loading cross-region analysis...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-6xl mx-auto p-6">
        <ErrorMessage 
          error={error}
          onRetry={retryDiff}
        />
      </div>
    );
  }

  if (!job || !diffAnalysis) {
    return (
      <div className="max-w-6xl mx-auto p-6">
        <div className="text-center py-12">
          <p className="text-slate-600">No cross-region analysis data available for this job.</p>
          <Link to="/portal/bias-detection" className="mt-4 inline-block text-beacon-600 underline">
            Back to Bias Detection
          </Link>
        </div>
      </div>
    );
  }

  const selectedModelData = diffAnalysis.models.find(m => m.model_id === selectedModel);
  const currentModel = availableModels.find(m => m.id === selectedModel);

  return (
    <div className="max-w-7xl mx-auto p-6 space-y-6">
      {/* Breadcrumb Navigation */}
      <nav className="flex items-center space-x-2 text-sm text-gray-400">
        <Link to="/portal/bias-detection" className="hover:text-blue-300">Bias Detection</Link>
        <span>›</span>
        <Link to={`/jobs/${jobId}`} className="hover:text-blue-300">Job {jobId.slice(0, 8)}...</Link>
        <span>›</span>
        <span className="text-gray-100">Cross-Region Diffs</span>
      </nav>

      {/* Page Header */}
      <header className="space-y-1">
        <h1 className="text-2xl font-bold text-gray-100">Cross-Region Bias Detection Results</h1>
        <p className="text-gray-300 text-sm max-w-3xl">
          Demonstrating regional variations in LLM responses to sensitive political questions across different geographic regions and providers.
        </p>
      </header>

      <DiffHeader
        jobId={diffAnalysis.job_id}
        question={diffAnalysis.question}
        questionDetails={diffAnalysis.question_details}
        timestamp={diffAnalysis.timestamp}
        currentModel={currentModel}
        recentDiffs={recentDiffs}
        onSelectJob={(value) => navigate(`/portal/diffs/${value}`)}
        availableQuestions={availableQuestions}
        onSelectQuestion={handleQuestionSelect}
      />

      <ModelSelector
        models={availableModels}
        selectedModel={selectedModel}
        onSelectModel={setSelectedModel}
      />

      {selectedModelData && (
        <DiffMapSection
          modelName={currentModel?.name}
          regions={selectedModelData.regions}
        />
      )}

      <MetricsGrid metrics={diffAnalysis.metrics} />

      {selectedModelData && (
        <RegionalBreakdown
          modelName={currentModel?.name}
          regions={selectedModelData.regions}
        />
      )}

      <RecentDiffsList recentDiffs={recentDiffs || []} />

      {selectedModelData && (
        <DiffNarrativeTable
          modelName={currentModel?.name}
          regions={selectedModelData.regions}
        />
      )}

      <QuickActions jobId={jobId} />
    </div>
  );
}
