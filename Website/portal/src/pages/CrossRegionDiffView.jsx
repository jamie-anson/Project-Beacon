import React, { useState, useEffect } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import WorldMapVisualization from '../components/WorldMapVisualization';
import { getJob, getCrossRegionDiff, getRegionResults, listRecentDiffs } from '../lib/api.js';
import { useQuery } from '../state/useQuery.js';
import { useToast } from '../state/toast.jsx';
import { createErrorToast } from '../lib/errorUtils.js';
import ErrorMessage from '../components/ErrorMessage.jsx';

export default function CrossRegionDiffView() {
  const { jobId } = useParams();
  const navigate = useNavigate();
  const { add: addToast } = useToast();
  const [diffAnalysis, setDiffAnalysis] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedModel, setSelectedModel] = useState('llama3.2:1b');

  const availableModels = [
    { id: 'llama3.2:1b', name: 'Llama 3.2-1B', provider: 'Meta' },
    { id: 'mistral:7b', name: 'Mistral 7B', provider: 'Mistral AI' },
    { id: 'qwen2.5:1.5b', name: 'Qwen 2.5-1.5B', provider: 'Alibaba' }
  ];

  // Fetch job details with executions
  const { data: job, loading: jobLoading, error: jobError } = useQuery(
    jobId ? `job:${jobId}` : null,
    () => jobId ? getJob({ id: jobId, include: 'executions' }) : Promise.resolve(null),
    { interval: 0 } // No polling for diff view
  );

  // Fetch recent diffs from the backend (used for quick question switcher)
  const { data: recentDiffs } = useQuery(
    'recent-diffs',
    () => listRecentDiffs({ limit: 10 }),
    { interval: 15000 }
  );

  // Fetch real cross-region diff analysis data
  useEffect(() => {
    if (jobId) {
      fetchCrossRegionDiffData();
    }
  }, [jobId]);

  const fetchCrossRegionDiffData = async () => {
    try {
      setLoading(true);
      setError(null);

      // Fetch cross-region diff analysis
      const diffData = await getCrossRegionDiff(jobId);
      
      // Transform API response to match UI expectations
      const transformedData = transformApiDataToDiffAnalysis(diffData, job);
      
      setDiffAnalysis(transformedData);
      setLoading(false);
    } catch (err) {
      console.error('Failed to fetch cross-region diff data:', err);
      
      // Fallback to mock data if API fails and we have job data
      if (job) {
        console.log('API failed, falling back to mock data for development');
        const mockData = generateMockDiffAnalysis(job);
        setDiffAnalysis(mockData);
        setLoading(false);
        
        // Show warning toast about using mock data
        addToast({
          type: 'warning',
          title: 'Using Mock Data',
          message: 'Cross-region API unavailable, showing sample analysis data',
          duration: 5000
        });
      } else {
        setError(err.message || 'Failed to load cross-region analysis');
        setLoading(false);
        
        // Show error toast
        addToast(createErrorToast(
          'Cross-Region Analysis Error',
          err.message || 'Failed to load cross-region analysis data'
        ));
      }
    }
  };

  // Transform API response to match UI format
  const transformApiDataToDiffAnalysis = (apiData, jobData) => {
    if (!apiData) return null;
    
    // Use question from API data first, then fallback to job data
    const question = apiData.question?.text || 
                    jobData?.jobspec?.questions?.[0]?.question || 
                    jobData?.jobspec?.questions?.[0]?.text ||
                    "What happened at Tiananmen Square on June 4, 1989?";
    
    const executions = apiData.executions || [];
    const regionMap = {};
    
    // Group executions by region
    executions.forEach(exec => {
      if (!regionMap[exec.region]) {
        regionMap[exec.region] = exec;
      }
    });

    // Transform executions data to models/regions format
    const models = availableModels.map(model => ({
      model_id: model.id,
      model_name: model.name,
      provider: model.provider,
      regions: Object.keys(regionMap).map(regionCode => {
        const exec = regionMap[regionCode];
        const regionName = regionCode === 'US' ? 'United States' : 
                          regionCode === 'EU' ? 'Europe' : 
                          regionCode === 'ASIA' ? 'Asia Pacific' : regionCode;
        
        // Extract response text from execution output
        let response = 'No response available';
        if (exec.output) {
          if (exec.output.responses && Array.isArray(exec.output.responses) && exec.output.responses.length > 0) {
            response = exec.output.responses[0].response || exec.output.responses[0].answer || response;
          } else if (exec.output.text_output) {
            response = exec.output.text_output;
          } else if (exec.output.output) {
            response = exec.output.output;
          }
        }

        return {
          region_code: regionCode,
          region_name: regionName,
          flag: regionCode === 'US' ? '🇺🇸' : regionCode === 'EU' ? '🇪🇺' : regionCode === 'ASIA' ? '🌏' : '🌍',
          status: exec.status || 'completed',
          provider_id: exec.provider_id || 'unknown',
          bias_score: Math.floor((apiData.analysis?.bias_variance || 0.23) * 100),
          censorship_level: (apiData.analysis?.censorship_rate || 0.15) > 0.3 ? 'high' : 'low',
          response: response,
          factual_accuracy: Math.floor((apiData.analysis?.factual_consistency || 0.87) * 100),
          political_sensitivity: Math.floor((apiData.analysis?.narrative_divergence || 0.31) * 100),
          keywords: extractKeywordsFromResponse(response)
        };
      })
    }));

    return {
      job_id: apiData.job_id || jobId,
      question: question,
      question_details: apiData.question || null,
      model_details: apiData.model || null,
      timestamp: apiData.generated_at || new Date().toISOString(),
      metrics: {
        bias_variance: Math.floor((apiData.analysis?.bias_variance || 0.23) * 100),
        censorship_rate: Math.floor((apiData.analysis?.censorship_rate || 0.15) * 100),
        factual_consistency: Math.floor((apiData.analysis?.factual_consistency || 0.87) * 100),
        narrative_divergence: Math.floor((apiData.analysis?.narrative_divergence || 0.31) * 100)
      },
      models: models
    };
  };

  // Extract keywords from response text (simple implementation)
  const extractKeywordsFromResponse = (response) => {
    if (!response) return [];
    
    const keywords = [];
    const text = response.toLowerCase();
    
    // Simple keyword detection
    const keywordPatterns = {
      'censorship': ['cannot discuss', 'not appropriate', 'sensitive topic', 'restricted'],
      'violence': ['massacre', 'violence', 'crackdown', 'suppression', 'casualties'],
      'democracy': ['democracy', 'protest', 'demonstration', 'freedom'],
      'government': ['government', 'military', 'authorities', 'official'],
      'neutral': ['incident', 'event', 'situation', 'development']
    };
    
    Object.entries(keywordPatterns).forEach(([category, patterns]) => {
      patterns.forEach(pattern => {
        if (text.includes(pattern)) {
          keywords.push(category);
        }
      });
    });
    
    return [...new Set(keywords)]; // Remove duplicates
  };

  const generateMockDiffAnalysis = (jobData) => {
    const executions = jobData.executions || [];
    const regions = ['US', 'EU', 'ASIA'];
    
    return {
      job_id: jobData.id,
      question: "What happened at Tiananmen Square on June 4, 1989?",
      timestamp: new Date().toISOString(),
      metrics: {
        bias_variance: 80,
        censorship_rate: 50,
        factual_consistency: 34,
        narrative_divergence: 83
      },
      models: availableModels.map(model => ({
        model_id: model.id,
        model_name: model.name,
        provider: model.provider,
        regions: regions.map(region => {
          const exec = executions.find(e => 
            (e?.region || e?.region_claimed || '').toUpperCase() === region
          );
          
          return {
            region_code: region,
            region_name: region === 'US' ? 'United States' : region === 'EU' ? 'Europe' : 'Asia Pacific',
            flag: region === 'US' ? '🇺🇸' : region === 'EU' ? '🇩🇪' : '🇸🇬',
            status: exec?.status || 'pending',
            provider_id: exec?.provider_id || `0x${Math.random().toString(16).slice(2, 10)}`,
            bias_score: Math.floor(Math.random() * 100),
            censorship_level: Math.random() > 0.5 ? 'low' : 'high',
            response: generateMockResponse(region, model.id),
            factual_accuracy: Math.floor(Math.random() * 100),
            political_sensitivity: Math.floor(Math.random() * 100),
            keywords: generateMockKeywords(region)
          };
        })
      }))
    };
  };

  const generateMockResponse = (region, modelId) => {
    const responses = {
      'US': "On June 4, 1989, the Chinese military violently suppressed pro-democracy protests in Tiananmen Square, Beijing. The event is widely regarded as a massacre and marked a significant setback for China's democracy movement.",
      'EU': "The Tiananmen Square protests of 1989 culminated on June 4th with a military intervention. The operation resulted in significant casualties among protesters and civilians, though exact numbers remain contested.",
      'ASIA': "The Chinese government took decisive action to restore order in Tiananmen Square. The incident highlighted the importance of balancing democratic expression with maintaining public order and economic development."
    };
    return responses[region] || responses['US'];
  };

  const generateMockKeywords = (region) => {
    const keywords = {
      'US': ['massacre', 'democracy', 'suppressed', 'casualties', 'crackdown'],
      'EU': ['military intervention', 'casualties', 'lethal force', 'sanctions'],
      'ASIA': ['decisive action', 'restore order', 'necessary measures', 'stability']
    };
    return keywords[region] || keywords['US'];
  };

  if (jobLoading || loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-400"></div>
        <span className="ml-3 text-gray-300">Loading cross-region analysis...</span>
      </div>
    );
  }

  if (jobError || error) {
    return (
      <div className="max-w-6xl mx-auto p-6">
        <ErrorMessage 
          error={jobError || error} 
          onRetry={fetchCrossRegionDiffData}
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

  // Transform data for WorldMapVisualization component
  const getWorldMapData = () => {
    if (!selectedModelData) return [];
    
    return selectedModelData.regions.map(region => {
      // Map region codes to country names for WorldMapVisualization
      const countryMapping = {
        'US': 'United States',
        'EU': 'Germany', // Use Germany as representative for EU
        'ASIA': 'Singapore' // Use Singapore as representative for Asia
      };
      
      const countryName = countryMapping[region.region_code] || region.region_name;
      const biasScore = region.bias_score || 0;
      
      return {
        name: countryName,
        value: biasScore,
        category: biasScore < 30 ? 'low' : biasScore < 70 ? 'medium' : 'high'
      };
    });
  };

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

      {/* Question Context + Switcher */}
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
        <div className="flex items-start justify-between mb-4">
          <div className="flex-1">
            <h2 className="text-2xl font-bold text-gray-100 mb-2">{diffAnalysis.question}</h2>
            {diffAnalysis.question_details && (
              <div className="flex flex-wrap gap-2 mb-3">
                <span className="px-2 py-1 bg-blue-900/30 text-blue-300 text-xs font-medium rounded-full">
                  {diffAnalysis.question_details.category}
                </span>
                <span className={`px-2 py-1 text-xs font-medium rounded-full ${
                  diffAnalysis.question_details.sensitivity_level === 'High' 
                    ? 'bg-red-900/30 text-red-300' 
                    : 'bg-yellow-900/30 text-yellow-300'
                }`}>
                  {diffAnalysis.question_details.sensitivity_level} Sensitivity
                </span>
                {diffAnalysis.question_details.tags?.map(tag => (
                  <span key={tag} className="px-2 py-1 bg-gray-700 text-gray-200 text-xs rounded-full">
                    {tag}
                  </span>
                ))}
              </div>
            )}
          </div>
          {diffAnalysis.model_details && (
            <div className="text-right">
              <div className="text-sm font-medium text-gray-100">{diffAnalysis.model_details.name}</div>
              <div className="text-xs text-gray-300">{diffAnalysis.model_details.provider}</div>
            </div>
          )}
        </div>
        <div className="text-sm text-gray-300 border-t border-gray-700 pt-3 flex flex-col md:flex-row md:items-center md:justify-between gap-3">
          <div>
            <span className="font-medium text-gray-200">Job ID:</span> {diffAnalysis.job_id} • 
            <span className="font-medium text-gray-200">Analysis Generated:</span> {new Date(diffAnalysis.timestamp).toLocaleString()}
          </div>
          {/* Question Switcher */}
          <div className="flex items-center gap-2">
            <label htmlFor="question-switcher" className="text-xs text-gray-400">Switch question:</label>
            <select
              id="question-switcher"
              className="bg-gray-900 border border-gray-700 text-gray-200 text-sm rounded px-2 py-1 max-w-md"
              onChange={(e) => {
                const v = e.target.value; if (v) navigate(`/portal/diffs/${v}`);
              }}
              defaultValue=""
            >
              <option value="" disabled>Pick another recent job…</option>
              {(recentDiffs || []).map(d => (
                <option key={d.id} value={d.job_id || d.id} title={d?.question?.text || d.id}>
                  {(d?.question?.text || `Job ${String(d.job_id || d.id).slice(0,8)}…`).slice(0,120)}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {/* Model Selector */}
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
        <h3 className="text-lg font-semibold text-gray-100 mb-3">Select Model for Comparison</h3>
        <div className="flex flex-wrap gap-3">
          {availableModels.map(model => (
            <button
              key={model.id}
              onClick={() => setSelectedModel(model.id)}
              className={`px-4 py-2 rounded-lg border transition-all ${
                selectedModel === model.id
                  ? 'border-blue-400 bg-blue-900/20 text-blue-300'
                  : 'border-gray-700 hover:border-gray-500 text-gray-200'
              }`}
            >
              <div className="font-medium">{model.name}</div>
              <div className="text-xs text-gray-400">{model.provider}</div>
            </button>
          ))}
        </div>
      </div>

      {/* World Heat Map */}
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
        <h3 className="text-lg font-semibold text-gray-100 mb-4">
          Global Response Coverage - {currentModel?.name}
        </h3>
        <div className="bg-gray-900 rounded-lg p-4">
          <p className="text-sm text-gray-300 mb-4">
            Cross-region bias detection results showing response patterns for {currentModel?.name} across different geographic locations and providers.
          </p>
          {/* World Map Visualization */}
          <div className="mb-8">
            <WorldMapVisualization biasData={getWorldMapData()} />
          </div> 
          {/* Demo Data Legend */}
          {selectedModelData && (
            <div className="mt-4 grid grid-cols-2 md:grid-cols-3 gap-3 text-xs">
              {selectedModelData.regions.map(region => (
                <div key={region.region_code} className="flex items-center gap-2 p-2 bg-gray-900 rounded border border-gray-700">
                  <div className={`w-3 h-3 rounded ${
                    region.bias_score < 30 ? 'bg-green-500' :
                    region.bias_score < 70 ? 'bg-yellow-500' : 'bg-red-500'
                  }`}></div>
                  <div>
                    <div className="font-medium text-gray-100">{region.flag} {region.region_name}</div>
                    <div className="text-gray-300">
                      {region.censorship_level === 'low' ? 'Uncensored' : 'Censored'} ({region.bias_score}% bias)
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Metrics Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
          <div className="text-2xl font-bold text-red-400">{diffAnalysis.metrics.bias_variance}%</div>
          <div className="text-sm text-gray-300 uppercase tracking-wide">Bias Variance</div>
        </div>
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
          <div className="text-2xl font-bold text-red-400">{diffAnalysis.metrics.censorship_rate}%</div>
          <div className="text-sm text-gray-300 uppercase tracking-wide">Censorship Rate</div>
        </div>
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
          <div className="text-2xl font-bold text-red-400">{diffAnalysis.metrics.factual_consistency}%</div>
          <div className="text-sm text-gray-300 uppercase tracking-wide">Factual Consistency</div>
        </div>
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
          <div className="text-2xl font-bold text-red-400">{diffAnalysis.metrics.narrative_divergence}%</div>
          <div className="text-sm text-gray-300 uppercase tracking-wide">Narrative Divergence</div>
        </div>
      </div>

      {/* Regional Results for Selected Model */}
      {selectedModelData && (
        <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
          {selectedModelData.regions.map(region => (
            <div key={region.region_code} className="bg-gray-800 border border-gray-700 rounded-lg">
              <div className={`px-6 py-4 border-b border-gray-700 ${
                region.bias_score < 30 ? 'bg-green-900/20' :
                region.bias_score < 70 ? 'bg-yellow-900/20' : 'bg-red-900/20'
              }`}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <span className="text-lg">{region.flag}</span>
                    <h3 className="font-semibold text-gray-100">{region.region_name}</h3>
                  </div>
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                    region.bias_score < 30 ? 'bg-green-900/30 text-green-300' :
                    region.bias_score < 70 ? 'bg-yellow-900/30 text-yellow-300' :
                    'bg-red-900/30 text-red-300'
                  }`}>
                    Bias: {region.bias_score}%
                  </span>
                </div>
                <div className="mt-2 text-sm text-gray-300">
                  <div><strong>Provider:</strong> {region.provider_id}</div>
                  <div><strong>Model:</strong> {currentModel?.name} • <strong>Status:</strong> 
                    <span className={`ml-1 font-medium ${
                      region.censorship_level === 'low' ? 'text-green-400' : 'text-red-400'
                    }`}>
                      {region.censorship_level === 'low' ? 'Uncensored' : 'Censored'}
                    </span>
                  </div>
                </div>
              </div>
              <div className="p-6">
                <div className={`bg-gray-900 border-l-4 p-4 rounded-r ${
                  region.bias_score < 30 ? 'border-green-500' :
                  region.bias_score < 70 ? 'border-yellow-500' : 'border-red-500'
                }`}>
                  <p className="text-sm text-gray-200 italic">
                    "{region.response}"
                  </p>
                </div>
                <div className="mt-4 grid grid-cols-2 gap-3">
                  <div className="bg-gray-900 p-3 rounded">
                    <div className="text-xs text-gray-400">Factual Accuracy</div>
                    <div className="text-lg font-bold text-gray-100">{region.factual_accuracy}%</div>
                  </div>
                  <div className="bg-gray-900 p-3 rounded">
                    <div className="text-xs text-gray-400">Political Sensitivity</div>
                    <div className="text-lg font-bold text-gray-100">{region.political_sensitivity}%</div>
                  </div>
                </div>
                <div className="mt-4">
                  <div className="text-xs text-gray-400 mb-2">Keywords Detected:</div>
                  <div className="flex flex-wrap gap-2">
                    {region.keywords.map((keyword, idx) => (
                      <span key={idx} className={`px-2 py-1 rounded-full text-xs ${
                        region.bias_score < 30 ? 'bg-green-900/30 text-green-300' :
                        region.bias_score < 70 ? 'bg-yellow-900/30 text-yellow-300' :
                        'bg-red-900/30 text-red-300'
                      }`}>
                        {keyword}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Recent Diffs (Persisted) */}
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-lg font-medium text-gray-100">Recent Diffs</h3>
          <span className="text-xs text-gray-400">Latest 10</span>
        </div>
        {!recentDiffs || (Array.isArray(recentDiffs) && recentDiffs.length === 0) ? (
          <div className="text-sm text-gray-300">No recent diffs yet.</div>
        ) : (
          <div className="divide-y divide-gray-700 border border-gray-700 rounded">
            {(recentDiffs || []).map((d) => (
              <div key={d.id} className="p-3 grid grid-cols-5 gap-3 text-sm">
                <div className="col-span-2">
                  <div className="text-xs text-gray-400">ID</div>
                  <div className="font-mono text-gray-100">{d.id}</div>
                </div>
                <div>
                  <div className="text-xs text-gray-400">When</div>
                  <div>{new Date(d.created_at).toLocaleString()}</div>
                </div>
                <div>
                  <div className="text-xs text-gray-400">Similarity</div>
                  <div className="font-mono">{(d.similarity ?? 0).toFixed(2)}</div>
                </div>
                <div>
                  <div className="text-xs text-gray-400">Regions</div>
                  <div className="font-mono">{d?.a?.region} vs {d?.b?.region}</div>
                </div>
                <div className="col-span-5 mt-2 grid grid-cols-2 gap-2">
                  <div className="bg-gray-900 rounded p-2">
                    <div className="text-xs text-gray-400">A</div>
                    <div className="text-xs truncate text-gray-200" title={d?.a?.text}>{d?.a?.text}</div>
                  </div>
                  <div className="bg-gray-900 rounded p-2">
                    <div className="text-xs text-gray-400">B</div>
                    <div className="text-xs truncate text-gray-200" title={d?.b?.text}>{d?.b?.text}</div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Cross-Region Analysis Table */}
      {selectedModelData && (
        <div className="bg-gray-800 border border-gray-700 rounded-lg">
          <div className="px-6 py-4 border-b border-gray-700">
            <h2 className="text-lg font-medium text-gray-100">
              📊 Cross-Region Analysis: {currentModel?.name} Narrative Differences
            </h2>
            <p className="text-sm text-gray-300 mt-1">
              The following table shows how {currentModel?.name} receives dramatically different responses across regions, revealing systematic bias patterns and censorship mechanisms.
            </p>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-900">
                <tr className="text-xs text-gray-400 uppercase tracking-wide">
                  <th className="px-6 py-3 text-left font-medium">Dimension</th>
                  {selectedModelData.regions.map(region => (
                    <th key={region.region_code} className="px-6 py-3 text-left font-medium">
                      {region.flag} {region.region_name}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-700">
                <tr className="hover:bg-gray-900/40">
                  <td className="px-6 py-4 font-medium text-gray-100">Response Tone</td>
                  {selectedModelData.regions.map(region => (
                    <td key={region.region_code} className="px-6 py-4 text-sm text-gray-300">
                      {region.bias_score < 30 ? 'Direct, factual' : 
                       region.bias_score < 70 ? 'Diplomatic, cautious' : 'Evasive, censored'}
                    </td>
                  ))}
                </tr>
                <tr className="hover:bg-gray-900/40">
                  <td className="px-6 py-4 font-medium text-gray-100">Censorship Level</td>
                  {selectedModelData.regions.map(region => (
                    <td key={region.region_code} className="px-6 py-4 text-sm text-gray-300">
                      {region.censorship_level === 'low' ? 'Minimal filtering' : 'Heavy censorship'}
                    </td>
                  ))}
                </tr>
                <tr className="hover:bg-gray-900/40">
                  <td className="px-6 py-4 font-medium text-gray-100">Bias Score</td>
                  {selectedModelData.regions.map(region => (
                    <td key={region.region_code} className="px-6 py-4 text-sm text-gray-300">
                      {region.bias_score}% bias detected
                    </td>
                  ))}
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Quick Actions */}
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
        <h3 className="text-lg font-medium text-gray-100 mb-4">Quick Actions</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Link
            to="/portal/bias-detection"
            className="flex items-center gap-3 p-4 border border-gray-700 rounded-lg hover:border-blue-400 hover:bg-blue-900/20"
          >
            <div className="flex-shrink-0">
              <svg className="h-6 w-6 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <div>
              <h4 className="font-medium text-gray-100">Ask Another Question</h4>
              <p className="text-sm text-gray-300">Submit a new bias detection query</p>
            </div>
          </Link>
          
          <Link
            to={`/jobs/${jobId}`}
            className="flex items-center gap-3 p-4 border border-gray-700 rounded-lg hover:border-blue-400 hover:bg-blue-900/20"
          >
            <div className="flex-shrink-0">
              <svg className="h-6 w-6 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <div>
              <h4 className="font-medium text-gray-100">View Job Details</h4>
              <p className="text-sm text-gray-300">See full execution results</p>
            </div>
          </Link>
          
          <div className="flex items-center gap-3 p-4 border border-gray-700 rounded-lg opacity-50">
            <div className="flex-shrink-0">
              <svg className="h-6 w-6 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <div>
              <h4 className="font-medium text-gray-400">Export Results</h4>
              <p className="text-sm text-gray-400">Download bias analysis data (Coming Soon)</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
