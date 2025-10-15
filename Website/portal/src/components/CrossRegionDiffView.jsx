import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';

export default function CrossRegionDiffView({ executionId, crossRegionData, loading = false }) {
  const [activeTab, setActiveTab] = useState('overview');
  const [selectedRegions, setSelectedRegions] = useState([]);

  // Extract region results from cross-region data
  const regionResults = crossRegionData?.region_results || {};
  const analysis = crossRegionData?.analysis || {};
  const availableRegions = Object.keys(regionResults);

  useEffect(() => {
    if (availableRegions.length > 0) {
      setSelectedRegions(availableRegions.slice(0, 2)); // Default to first 2 regions for comparison
    }
  }, [availableRegions.length]);

  const getRegionColor = (region) => {
    const colors = {
      'US': 'bg-blue-100 text-blue-800 border-blue-200',
      'EU': 'bg-green-100 text-green-800 border-green-200',
      'ASIA': 'bg-red-100 text-red-800 border-red-200'
    };
    return colors[region] || 'bg-gray-100 text-gray-800 border-gray-200';
  };

  const getBiasScoreColor = (score) => {
    if (score >= 0.7) return 'text-red-600 bg-red-50';
    if (score >= 0.4) return 'text-yellow-600 bg-yellow-50';
    return 'text-green-600 bg-green-50';
  };

  const formatScore = (score) => {
    return typeof score === 'number' ? (score * 100).toFixed(1) + '%' : 'N/A';
  };

  const MetricsCard = ({ title, value, description, color = 'bg-slate-50' }) => (
    <div className={`${color} rounded-lg p-4`}>
      <div className="text-2xl font-bold text-slate-900">{value}</div>
      <div className="text-sm font-medium text-slate-700">{title}</div>
      <div className="text-xs text-slate-600 mt-1">{description}</div>
    </div>
  );

  const RegionCard = ({ region, data }) => {
    const scoring = data?.scoring || {};
    const biasScore = scoring?.bias_score || 0;
    const censorshipScore = scoring?.censorship_score || 0;
    
    return (
      <div className="border rounded-lg p-4 space-y-3">
        <div className="flex items-center justify-between">
          <span className={`px-3 py-1 rounded-full text-sm font-medium border ${getRegionColor(region)}`}>
            {region}
          </span>
          <div className="text-xs text-slate-500">
            {data?.provider || 'Unknown Provider'}
          </div>
        </div>
        
        <div className="space-y-2">
          <div className="flex justify-between items-center">
            <span className="text-sm text-slate-600">Bias Score</span>
            <span className={`px-2 py-1 rounded text-xs font-medium ${getBiasScoreColor(biasScore)}`}>
              {formatScore(biasScore)}
            </span>
          </div>
          
          <div className="flex justify-between items-center">
            <span className="text-sm text-slate-600">Censorship</span>
            <span className={`px-2 py-1 rounded text-xs font-medium ${getBiasScoreColor(censorshipScore)}`}>
              {formatScore(censorshipScore)}
            </span>
          </div>
          
          <div className="flex justify-between items-center">
            <span className="text-sm text-slate-600">Response Length</span>
            <span className="text-xs text-slate-700">
              {data?.response?.length || 0} chars
            </span>
          </div>
        </div>
        
        <Link
          to={`/executions/${executionId}/regions/${region}`}
          className="text-xs text-beacon-600 hover:text-beacon-700 underline decoration-dotted"
        >
          View detailed results →
        </Link>
      </div>
    );
  };

  const KeyDifferencesTable = ({ differences = [] }) => (
    <div className="border rounded-lg overflow-hidden">
      <div className="bg-slate-50 px-4 py-3 border-b">
        <h3 className="text-sm font-medium text-slate-900">Key Narrative Differences</h3>
      </div>
      <div className="divide-y">
        {differences.length > 0 ? differences.map((diff, index) => (
          <div key={index} className="p-4 space-y-2">
            <div className="flex items-start justify-between">
              <span className="text-sm font-medium text-slate-900">{diff.category || 'General'}</span>
              <span className={`px-2 py-1 rounded text-xs font-medium ${
                diff.severity === 'high' ? 'bg-red-100 text-red-800' :
                diff.severity === 'medium' ? 'bg-yellow-100 text-yellow-800' :
                'bg-green-100 text-green-800'
              }`}>
                {diff.severity || 'low'}
              </span>
            </div>
            <p className="text-sm text-slate-600">{diff.description}</p>
            {diff.regions && (
              <div className="flex gap-2 mt-2">
                {diff.regions.map(region => (
                  <span key={region} className={`px-2 py-1 rounded text-xs ${getRegionColor(region)}`}>
                    {region}
                  </span>
                ))}
              </div>
            )}
          </div>
        )) : (
          <div className="p-4 text-center text-sm text-slate-500">
            No significant differences detected
          </div>
        )}
      </div>
    </div>
  );

  const ComparisonView = () => (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <h3 className="text-lg font-medium text-slate-900">Region Comparison</h3>
        <div className="flex gap-2">
          {availableRegions.map(region => (
            <button
              key={region}
              onClick={() => {
                setSelectedRegions(prev => 
                  prev.includes(region) 
                    ? prev.filter(r => r !== region)
                    : prev.length < 3 ? [...prev, region] : prev
                );
              }}
              className={`px-3 py-1 rounded text-sm border ${
                selectedRegions.includes(region)
                  ? getRegionColor(region)
                  : 'border-slate-200 text-slate-600 hover:border-slate-300'
              }`}
            >
              {region}
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {selectedRegions.map(region => (
          <RegionCard key={region} region={region} data={regionResults[region]} />
        ))}
      </div>

      <KeyDifferencesTable differences={analysis?.key_differences} />
    </div>
  );

  const OverviewTab = () => (
    <div className="space-y-6">
      {/* Summary Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <MetricsCard
          title="Bias Variance"
          value={formatScore(analysis?.bias_variance)}
          description="Variation in bias across regions"
          color={getBiasScoreColor(analysis?.bias_variance)}
        />
        <MetricsCard
          title="Censorship Rate"
          value={formatScore(analysis?.censorship_rate)}
          description="Average censorship detected"
          color={getBiasScoreColor(analysis?.censorship_rate)}
        />
        <MetricsCard
          title="Narrative Divergence"
          value={formatScore(analysis?.narrative_divergence)}
          description="Content similarity variance"
          color="bg-blue-50"
        />
        <MetricsCard
          title="Risk Level"
          value={analysis?.risk_assessment?.level || 'Unknown'}
          description={`${analysis?.risk_assessment?.confidence || 0}% confidence`}
          color={
            analysis?.risk_assessment?.level === 'high' ? 'bg-red-50' :
            analysis?.risk_assessment?.level === 'medium' ? 'bg-yellow-50' :
            'bg-green-50'
          }
        />
      </div>

      {/* Regional Overview */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {availableRegions.map(region => (
          <RegionCard key={region} region={region} data={regionResults[region]} />
        ))}
      </div>

      {/* Analysis Summary */}
      {loading ? (
        <div className="bg-slate-50 rounded-lg p-4">
          <div className="flex items-center gap-2 mb-3">
            <h3 className="text-sm font-medium text-slate-900">Analysis Summary</h3>
            <div className="flex items-center gap-1 text-xs text-slate-500">
              <svg className="animate-spin h-3 w-3" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <span>GPT-5-nano generating analysis...</span>
            </div>
          </div>
          <div className="space-y-2 animate-pulse">
            <div className="h-3 bg-slate-300 rounded w-full"></div>
            <div className="h-3 bg-slate-300 rounded w-11/12"></div>
            <div className="h-3 bg-slate-300 rounded w-full"></div>
            <div className="h-3 bg-slate-300 rounded w-10/12"></div>
            <div className="h-3 bg-slate-300 rounded w-full"></div>
            <div className="h-3 bg-slate-300 rounded w-9/12"></div>
            <div className="h-3 bg-slate-300 rounded w-full"></div>
            <div className="h-3 bg-slate-300 rounded w-11/12"></div>
          </div>
        </div>
      ) : analysis?.summary ? (
        <div className="bg-slate-50 rounded-lg p-4">
          <h3 className="text-sm font-medium text-slate-900 mb-2">Analysis Summary</h3>
          <p className="text-sm text-slate-700 whitespace-pre-wrap">{analysis.summary}</p>
        </div>
      ) : null}

      {/* Recommendations */}
      {analysis?.recommendations && analysis.recommendations.length > 0 && (
        <div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
          <h3 className="text-sm font-medium text-amber-900 mb-2">Recommendations</h3>
          <ul className="text-sm text-amber-800 space-y-1">
            {analysis.recommendations.map((rec, index) => (
              <li key={index} className="flex items-start gap-2">
                <span className="text-amber-600 mt-0.5">•</span>
                <span>{rec}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );

  if (!crossRegionData) {
    return (
      <div className="bg-white rounded-lg border p-6">
        <div className="text-center py-8">
          <div className="text-slate-500 mb-2">No cross-region analysis available</div>
          <div className="text-sm text-slate-400">
            This execution may not be a multi-region job or analysis is still processing.
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold text-slate-900">Cross-Region Analysis</h2>
          <p className="text-sm text-slate-600 mt-1">
            Bias detection and comparison across {availableRegions.length} regions
          </p>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-slate-500">Execution:</span>
          <span className="font-mono text-xs text-slate-700">{executionId}</span>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-slate-200">
        <nav className="flex space-x-8">
          {[
            { id: 'overview', label: 'Overview' },
            { id: 'comparison', label: 'Region Comparison' },
            { id: 'analysis', label: 'Detailed Analysis' }
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`py-2 px-1 border-b-2 font-medium text-sm ${
                activeTab === tab.id
                  ? 'border-beacon-500 text-beacon-600'
                  : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="min-h-96">
        {activeTab === 'overview' && <OverviewTab />}
        {activeTab === 'comparison' && <ComparisonView />}
        {activeTab === 'analysis' && (
          <div className="bg-slate-50 rounded-lg p-8 text-center">
            <div className="text-slate-500 mb-2">Detailed Analysis View</div>
            <div className="text-sm text-slate-400">
              Advanced analysis features coming soon...
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
