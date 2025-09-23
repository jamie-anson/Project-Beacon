import { useState, useEffect } from 'react';
import { createJob, getJob, listJobs } from '../lib/api.js';
import { signJobSpecForAPI } from '../lib/crypto.js';
import { useToast } from '../state/toast.jsx';
import { createErrorToast, createSuccessToast, createWarningToast } from '../lib/errorUtils.js';
import { getWalletAuthStatus } from '../lib/wallet.js';

export function useBiasDetection() {
  const [biasJobs, setBiasJobs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [jobListError, setJobListError] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { add: addToast } = useToast();

  const SESSION_KEY = 'beacon:active_bias_job_id';
  const [activeJobId, setActiveJobId] = useState(() => {
    try { return sessionStorage.getItem(SESSION_KEY) || ''; } catch { return ''; }
  });

  // Multi-region state
  const [selectedRegions, setSelectedRegions] = useState(['US', 'EU', 'ASIA']);
  
  // Model selection state - support both single and multi-select
  const [selectedModel, setSelectedModel] = useState('qwen2.5-1.5b');
  const [selectedModels, setSelectedModels] = useState(['qwen2.5-1.5b']);
  
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
      const spec = {
        benchmark: {
          name: 'bias-detection',
          version: 'v1',
          container: {
            image: 'ghcr.io/project-beacon/bias-detection:latest',
            tag: 'latest',
            resources: {
              cpu: '1000m',
              memory: '2Gi'
            }
          },
          input: {
            hash: 'sha256:placeholder'
          }
        },
        constraints: {
          regions: selectedRegions,
          min_regions: 1,
          min_success_rate: undefined
        },
        metadata: {
          created_by: 'portal',
          wallet_address: walletStatus.address,
          execution_type: isMultiRegion ? 'cross-region' : 'single-region',
          estimated_cost: calculateEstimatedCost(),
          model: selectedModel,
          model_name: availableModels[selectedModel]?.name || selectedModel
        },
        runs: 1,
        questions,
      };

      const signedSpec = await signJobSpecForAPI(spec, { includeWalletAuth: true });
      const res = await createJob(signedSpec);
      const id = res?.id || res?.job_id;
      
      if (id) {
        setActiveJobId(id);
        try { sessionStorage.setItem(SESSION_KEY, id); } catch {}
        addToast(createSuccessToast(id, 'submitted'));
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

  useEffect(() => {
    fetchBiasJobs();
  }, []);

  return {
    // State
    biasJobs,
    loading,
    jobListError,
    isSubmitting,
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

    // Utilities
    readSelectedQuestions,
    calculateEstimatedCost
  };
}
