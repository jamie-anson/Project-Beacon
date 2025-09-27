import React, { useState, useEffect } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import DiffHeader from '../components/diffs/DiffHeader.jsx';
import ModelSelector from '../components/diffs/ModelSelector.jsx';
import MetricsGrid from '../components/diffs/MetricsGrid.jsx';
import RegionalBreakdown from '../components/diffs/RegionalBreakdown.jsx';
import RecentDiffsList from '../components/diffs/RecentDiffsList.jsx';
import DiffNarrativeTable from '../components/diffs/DiffNarrativeTable.jsx';
import QuickActions from '../components/diffs/QuickActions.jsx';
import DiffMapSection from '../components/diffs/DiffMapSection.jsx';
import { createErrorToast } from '../lib/errorUtils.js';
import ErrorMessage from '../components/ErrorMessage.jsx';
import { useToast } from '../state/toast.jsx';
import { usePageTitle } from '../hooks/usePageTitle.js';
import { useCrossRegionDiff } from '../hooks/useCrossRegionDiff.js';
import { useRecentDiffs } from '../hooks/useRecentDiffs.js';
import { AVAILABLE_MODELS } from '../lib/diffs/constants.js';
import { useQuery } from '../state/useQuery.js';
import { runnerFetch } from '../lib/api/http.js';

export default function CrossRegionDiffPage() {
  const { jobId } = useParams();
  const navigate = useNavigate();
  const { add: addToast } = useToast();
  const [selectedModel, setSelectedModel] = useState(null); // Will be set based on available data
  
  usePageTitle('Cross-Region Bias Detection Results');

  const availableModels = AVAILABLE_MODELS;

  const {
    loading,
    error,
    data: diffAnalysis,
    job,
    usingMock,
    retry: retryDiff
  } = useCrossRegionDiff(jobId, availableModels);

  // Auto-select first available model when data loads
  React.useEffect(() => {
    if (diffAnalysis?.models?.length > 0 && !selectedModel) {
      const firstAvailableModel = diffAnalysis.models[0].model_id;
      console.log('🎯 Auto-selecting first available model:', firstAvailableModel);
      setSelectedModel(firstAvailableModel);
    }
  }, [diffAnalysis, selectedModel]);

  const { data: recentDiffs } = useRecentDiffs({ limit: 10, pollInterval: 15000 });
  
  // Fetch available questions
  const { data: questionsData } = useQuery('questions', () => runnerFetch('/questions'), { interval: 0 });
  const availableQuestions = questionsData ? [
    ...(questionsData.categories?.control_questions || []),
    ...(questionsData.categories?.bias_detection || []),
    ...(questionsData.categories?.cultural_perspective || [])
  ] : [];

  const handleQuestionSelect = async (questionId) => {
    try {
      addToast({
        type: 'info',
        title: 'Submitting New Job',
        message: `Creating new analysis for question: ${questionId}...`,
        duration: 2000
      });

      // Create multi-model job specification for the selected question
      const jobData = {
        id: `multi-model-${questionId}-${Date.now()}`,
        version: "v1",
        benchmark: {
          name: "multi-model-bias-detection",
          version: "v1",
          description: `Multi-model question analysis for ${questionId}`,
          container: {
            image: "ghcr.io/project-beacon/bias-detection:latest",
            tag: "latest",
            resources: {
              cpu: "1000m",
              memory: "2Gi"
            }
          },
          input: {
            type: "",
            data: null,
            hash: "sha256:placeholder"
          },
          scoring: {
            method: "default",
            parameters: {}
          },
          metadata: {}
        },
        constraints: {
          regions: ["US", "EU", "ASIA"],
          min_regions: 3,
          min_success_rate: 0.67,
          timeout: 600000000000,
          provider_timeout: 120000000000
        },
        questions: [questionId],
        models: [
          {
            id: "llama3.2-1b",
            name: "Llama 3.2-1B Instruct",
            provider: "modal",
            container_image: "ghcr.io/jamie-anson/project-beacon/llama-3.2-1b:latest",
            regions: ["US", "EU", "ASIA"]
          },
          {
            id: "qwen2.5-1.5b",
            name: "Qwen 2.5-1.5B Instruct", 
            provider: "modal",
            container_image: "ghcr.io/jamie-anson/project-beacon/qwen-2.5-1.5b:latest",
            regions: ["ASIA", "EU", "US"]
          },
          {
            id: "mistral-7b",
            name: "Mistral 7B Instruct",
            provider: "modal", 
            container_image: "ghcr.io/jamie-anson/project-beacon/mistral-7b:latest",
            regions: ["EU", "US", "ASIA"]
          }
        ],
        metadata: {
          created_by: "multi-model-question-switcher",
          multi_model: true,
          total_executions_expected: 9,
          timestamp: new Date().toISOString(),
          wallet_address: "0x67f3d16a91991cf169920f1e79f78e66708da328"
        },
        created_at: new Date().toISOString()
      };

      // Submit the job
      const response = await runnerFetch('/jobs', {
        method: 'POST',
        body: JSON.stringify(jobData)
      });

      if (response?.id) {
        addToast({
          type: 'success',
          title: 'Job Submitted Successfully',
          message: `Job ${response.id} created. Redirecting to results...`,
          duration: 3000
        });

        // Wait a moment then navigate to the new job's results
        setTimeout(() => {
          navigate(`/portal/results/${response.id}/diffs`);
        }, 1500);
      } else {
        throw new Error('Job submission failed - no job ID returned');
      }

    } catch (error) {
      console.error('Question switch job submission failed:', error);
      addToast({
        type: 'error',
        title: 'Job Submission Failed',
        message: error.message || 'Failed to create new analysis job',
        duration: 5000
      });
    }
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
      <div className="space-y-6">
        <div className="animate-pulse">
          <div className="h-8 bg-gray-700 rounded w-1/2 mb-4"></div>
          <div className="h-4 bg-gray-700 rounded w-3/4 mb-8"></div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
            {[1, 2, 3].map(i => (
              <div key={i} className="h-32 bg-gray-700 rounded"></div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return <ErrorMessage error={error} />;
  }

  if (!diffAnalysis) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-400">No analysis data available for job {jobId}</p>
        <Link to="/portal/bias-detection" className="mt-4 inline-block text-beacon-600 underline">
          Back to Bias Detection
        </Link>
      </div>
    );
  }

  // Debug model selection
  console.log('🔍 Model Selection Debug:', {
    selectedModel,
    availableModels,
    diffAnalysisModels: diffAnalysis?.models,
    modelIds: diffAnalysis?.models?.map(m => m.model_id)
  });

  const selectedModelData = diffAnalysis.models.find(m => m.model_id === selectedModel);
  const currentModel = availableModels.find(m => m.id === selectedModel);

  console.log('🎯 Selected Model Data:', {
    selectedModelData,
    currentModel,
    hasRegions: selectedModelData?.regions?.length
  });

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
        onSelectJob={(value) => navigate(`/portal/results/${value}/diffs`)}
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
