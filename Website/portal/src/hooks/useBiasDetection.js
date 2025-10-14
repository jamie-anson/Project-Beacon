import { useState, useEffect } from 'react';
import { createJob, getJob, listJobs, cancelJob } from '../lib/api/runner/jobs.js';
import { signJobSpecForAPI } from '../lib/crypto.js';
import { useToast } from '../state/toast.jsx';
import { createErrorToast, createSuccessToast, createWarningToast } from '../lib/errorUtils.js';
import { getWalletAuthStatus } from '../lib/wallet.js';

export function useBiasDetection() {
  const [biasJobs, setBiasJobs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [jobListError, setJobListError] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isCancelling, setIsCancelling] = useState(false);
  const { add: addToast } = useToast();

  const SESSION_KEY = 'beacon:active_bias_job_id';
  const [activeJobId, setActiveJobId] = useState(() => {
    try { return sessionStorage.getItem(SESSION_KEY) || ''; } catch { return ''; }
  });

  // Reset Live Progress state on wallet changes or hard refresh
  const resetLiveProgressState = () => {
    setActiveJobId('');
    try { 
      sessionStorage.removeItem(SESSION_KEY); 
    } catch {}
  };

  // Multi-region state (default to US+EU only, ASIA disabled due to timeout issues)
  const [selectedRegions, setSelectedRegions] = useState(['US', 'EU']);
  
  // Model selection state - support both single and multi-select
  // Default: all models selected for MVP
  const [selectedModel, setSelectedModel] = useState('llama3.2-1b');
  const [selectedModels, setSelectedModels] = useState(['llama3.2-1b', 'mistral-7b', 'qwen2.5-1.5b']);
  
  // Helper function to safely handle model selection changes
  const handleModelChange = (newSelection) => {
    try {
      if (Array.isArray(newSelection)) {
        // Multi-select mode
        const safeSelection = newSelection.filter(id => 
          typeof id === 'string' && id.length > 0
        );
        setSelectedModels(safeSelection.length > 0 ? safeSelection : ['qwen2.5-1.5b']);
        // Also update single selection for backward compatibility
        setSelectedModel(safeSelection[0] || 'qwen2.5-1.5b');
      } else if (typeof newSelection === 'string' && newSelection.length > 0) {
        // Single select mode
        setSelectedModel(newSelection);
        setSelectedModels([newSelection]);
      }
    } catch (error) {
      console.warn('Error handling model selection:', error);
      // Fallback to safe defaults
      setSelectedModel('qwen2.5-1.5b');
      setSelectedModels(['qwen2.5-1.5b']);
    }
  };

  const availableRegions = [
    { code: 'US', name: 'United States', cost: 0.0003 },
    { code: 'EU', name: 'Europe', cost: 0.0004 },
    { code: 'ASIA', name: 'Asia Pacific', cost: 0.0005 }
  ];
  
  const availableModels = {
    'llama3.2-1b': { name: 'Llama 3.2-1B', cost: 0.0003 },
    'mistral-7b': { name: 'Mistral 7B Instruct', cost: 0.0004 },
    'qwen2.5-1.5b': { name: 'Qwen 2.5-1.5B Instruct', cost: 0.00035 }
  };

  // Read selected questions from localStorage
  const readSelectedQuestions = () => {
    try {
      const raw = localStorage.getItem('beacon:selected_questions');
      const arr = raw ? JSON.parse(raw) : [];
      return Array.isArray(arr) ? arr : [];
    } catch {
      return [];
    }
  };

  // Calculate estimated cost
  const calculateEstimatedCost = () => {
    const questions = readSelectedQuestions();
    const questionCount = questions.length || 1;
    const modelCost = availableModels[selectedModel]?.cost || 0.0003;
    const regionCount = selectedRegions.length;
    const totalCost = modelCost * regionCount * questionCount;
    return totalCost.toFixed(4);
  };

  // Handle region selection changes
  const handleRegionToggle = (regionCode) => {
    setSelectedRegions(prev => {
      if (prev.includes(regionCode)) {
        const newRegions = prev.filter(r => r !== regionCode);
        return newRegions.length > 0 ? newRegions : prev;
      } else {
        return [...prev, regionCode];
      }
    });
  };


  // Fetch bias detection jobs
  const fetchBiasJobs = async () => {
    try {
      setJobListError(null);
      const data = await listJobs({ limit: 50 });
      const jobs = Array.isArray(data?.jobs) ? data.jobs : (Array.isArray(data) ? data : []);
      const biasJobsData = jobs.filter(job =>
        job?.benchmark?.name?.includes('bias-detection') || job?.id?.includes('bias-detection')
      );
      setBiasJobs(biasJobsData);
    } catch (error) {
      console.error('Failed to fetch bias jobs:', error);
      setJobListError(error);
      addToast(createErrorToast(error));
    } finally {
      setLoading(false);
    }
  };

  // Cancel job handler
  const handleCancelJob = async (jobId) => {
    if (!jobId || isCancelling) return;
    
    console.log('[handleCancelJob] Starting cancellation for job:', jobId);
    
    setIsCancelling(true);
    try {
      console.log('[handleCancelJob] Calling cancelJob API...');
      const result = await cancelJob(jobId);
      console.log('[handleCancelJob] Cancel API response:', result);
      
      // Show success toast
      addToast(createSuccessToast(
        `Job ${jobId.substring(0, 8)}... cancelled successfully`
      ));
      
      // Refresh job list to show cancelled status
      console.log('[handleCancelJob] Refreshing job list...');
      await fetchBiasJobs();
      
      return result;
    } catch (error) {
      console.error('[handleCancelJob] Cancel job failed - Full error:', error);
      console.error('[handleCancelJob] Error type:', typeof error);
      console.error('[handleCancelJob] Error keys:', Object.keys(error));
      console.error('[handleCancelJob] Error message:', error.message);
      console.error('[handleCancelJob] Error user_message:', error.user_message);
      console.error('[handleCancelJob] Error stack:', error.stack);
      
      // Extract meaningful error message
      let errorMessage = 'Failed to cancel job';
      if (error.user_message) {
        errorMessage = error.user_message;
      } else if (error.message) {
        errorMessage = error.message;
      } else if (typeof error === 'string') {
        errorMessage = error;
      }
      
      console.error('[handleCancelJob] Showing error toast:', errorMessage);
      addToast(createErrorToast(errorMessage, error));
      throw error;
    } finally {
      setIsCancelling(false);
    }
  };

  // Submit new job
  const onSubmitJob = async () => {
    if (isSubmitting) return;
    
    setIsSubmitting(true);
    
    const questions = readSelectedQuestions();
    
    if (questions.length === 0) {
      addToast(createWarningToast('Please select at least one question on the Questions page before submitting a job.'));
      setIsSubmitting(false);
      return;
    }
    
    const walletStatus = getWalletAuthStatus();
    if (!walletStatus.isAuthorized) {
      addToast(createWarningToast('Please connect and authorize your wallet before submitting a job.'));
      setIsSubmitting(false);
      return;
    }

    try {
      // Determine if this is a multi-model job
      const isMultiModel = selectedModels.length > 1;
      const benchmarkName = isMultiModel ? 'multi-model-bias-detection' : 'bias-detection';
      const expectedExecutions = selectedRegions.length * selectedModels.length * questions.length;

      const spec = {
        id: `bias-detection-${Date.now()}`,
        version: 'v1',
        benchmark: {
          name: benchmarkName,
          version: 'v1',
          description: isMultiModel ? `Multi-model bias detection across ${selectedModels.length} models` : 'Single-model bias detection',
          container: {
            image: 'ghcr.io/project-beacon/bias-detection:latest',
            tag: 'latest',
            resources: {
              cpu: '1000m',
              memory: '2Gi'
            }
          },
          input: {
            type: 'prompt',
            data: {
              prompt: questions.length > 0 ? questions[0] : 'What is your opinion on current global events?'
            },
            hash: 'sha256:placeholder'
          }
        },
        constraints: {
          regions: selectedRegions,
          min_regions: 1,
          min_success_rate: undefined,
          timeout: 600000000000, // 10 minutes in nanoseconds (600s * 1e9)
          provider_timeout: 600000000000 // 10 minutes for Modal GPU queue + cold starts
        },
        metadata: {
          created_by: 'portal',
          wallet_address: walletStatus.address,
          execution_type: selectedRegions.length > 1 ? 'cross-region' : 'single-region',
          estimated_cost: calculateEstimatedCost(),
          multi_model: isMultiModel,
          models: selectedModels,
          total_executions_expected: expectedExecutions,
          model: selectedModel, // Keep for backward compatibility
          model_name: availableModels[selectedModel]?.name || selectedModel
        },
        runs: 1,
        questions,
      };

      // Sign the jobspec
      const signedSpec = await signJobSpecForAPI(spec, { includeWalletAuth: true });
      
      // For bias detection, ALWAYS use cross-region format to enable Level 3 analysis
      // This ensures the cross_region_executions table is populated for bias analysis endpoint
      const finalPayload = {
        jobspec: signedSpec,  // The signed jobspec
        target_regions: selectedRegions,
        min_regions: Math.max(1, Math.floor(selectedRegions.length * 0.67)), // 67% success rate
        min_success_rate: 0.67,
        enable_analysis: true  // Critical: enables bias analysis generation
      };
      
      console.log('[BiasDetection] Submitting cross-region job:', {
        jobId: spec.id,
        regions: selectedRegions,
        models: selectedModels,
        questions: questions.length,
        enableAnalysis: true
      });
      
      const res = await createJob(finalPayload);
      
      // Cross-region endpoint returns: { id, job_id, cross_region_execution_id, status, ... }
      const jobId = res?.id || res?.job_id || res?.jobspec_id;
      const crossRegionExecId = res?.cross_region_execution_id;
      
      console.log('[BiasDetection] Cross-region job submitted:', {
        jobId,
        crossRegionExecId,
        status: res?.status,
        totalRegions: res?.total_regions
      });
      
      if (jobId) {
        setActiveJobId(jobId);
        try { sessionStorage.setItem(SESSION_KEY, jobId); } catch {}
        addToast(createSuccessToast(jobId, 'submitted'));
        fetchBiasJobs();
      } else {
        throw new Error('No job ID returned from server');
      }
    } catch (error) {
      console.error('Failed to create job', error);
      addToast(createErrorToast(error));
    } finally {
      setIsSubmitting(false);
    }
  };

  // Monitor wallet status and reset Live Progress on wallet changes
  useEffect(() => {
    let lastWalletAddress = null;
    
    const checkWalletStatus = () => {
      const walletStatus = getWalletAuthStatus();
      const currentAddress = walletStatus?.address;
      
      // Reset Live Progress if wallet disconnected or address changed
      if (lastWalletAddress && (!currentAddress || currentAddress !== lastWalletAddress)) {
        console.log('🔄 Wallet changed/disconnected - resetting Live Progress state');
        resetLiveProgressState();
      }
      
      lastWalletAddress = currentAddress;
    };
    
    // Check immediately and then periodically
    checkWalletStatus();
    const interval = setInterval(checkWalletStatus, 1000);
    
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    fetchBiasJobs();
  }, []);

  return {
    // State
    biasJobs,
    loading,
    jobListError,
    isSubmitting,
    isCancelling,
    activeJobId,
    selectedRegions,
    selectedModel,
    selectedModels,
    availableRegions,
    availableModels,

    // Actions
    setActiveJobId,
    setSelectedRegions,
    setSelectedModel,
    setSelectedModels,
    handleModelChange,
    handleRegionToggle,
    fetchBiasJobs,
    onSubmitJob,
    handleCancelJob,
    resetLiveProgressState,

    // Utilities
    readSelectedQuestions,
    calculateEstimatedCost
  };
}
