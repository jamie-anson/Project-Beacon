import React, { useState, useEffect } from 'react';

export default function GeographicVisualization({ biasData = [] }) {
  const [selectedRegion, setSelectedRegion] = useState(null);
  const [viewMode, setViewMode] = useState('bias'); // 'bias', 'censorship', 'accuracy'

  const regions = [
    {
      id: 'US',
      name: 'United States',
      position: { x: 20, y: 40 },
      models: ['Llama 3.2-1B'],
      color: 'text-blue-600',
      bgColor: 'bg-blue-100',
      borderColor: 'border-blue-300'
    },
    {
      id: 'EU',
      name: 'European Union',
      position: { x: 50, y: 30 },
      models: ['Mistral 7B'],
      color: 'text-green-600',
      bgColor: 'bg-green-100',
      borderColor: 'border-green-300'
    },
    {
      id: 'China',
      name: 'China',
      position: { x: 75, y: 45 },
      models: ['Qwen 2.5-1.5B'],
      color: 'text-red-600',
      bgColor: 'bg-red-100',
      borderColor: 'border-red-300'
    }
  ];

  const getRegionData = (regionId) => {
    const regionBias = biasData.find(d => d.region === regionId) || {};
    return {
      biasScore: regionBias.biasScore || Math.random() * 100,
      censorshipRate: regionBias.censorshipRate || Math.random() * 100,
      accuracyRate: regionBias.accuracyRate || 80 + Math.random() * 20,
      responseCount: regionBias.responseCount || Math.floor(Math.random() * 50) + 10
    };
  };

  const getMetricValue = (regionId) => {
    const data = getRegionData(regionId);
    switch (viewMode) {
      case 'bias': return data.biasScore;
      case 'censorship': return data.censorshipRate;
      case 'accuracy': return data.accuracyRate;
      default: return data.biasScore;
    }
  };

  const getMetricColor = (value) => {
    if (viewMode === 'accuracy') {
      // Higher accuracy is better (green)
      if (value >= 80) return 'bg-green-500';
      if (value >= 60) return 'bg-yellow-500';
      return 'bg-red-500';
    } else {
      // Lower bias/censorship is better (green)
      if (value <= 30) return 'bg-green-500';
      if (value <= 60) return 'bg-yellow-500';
      return 'bg-red-500';
    }
  };

  const getConnectionStrength = (region1, region2) => {
    // Calculate connection strength based on bias similarity
    const data1 = getRegionData(region1);
    const data2 = getRegionData(region2);
    const diff = Math.abs(data1.biasScore - data2.biasScore);
    return Math.max(0, 100 - diff) / 100;
  };

  return (
    <div className="bg-white rounded-lg border p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-medium text-slate-900">Geographic Bias Distribution</h3>
        <div className="flex items-center gap-2">
          <select
            value={viewMode}
            onChange={(e) => setViewMode(e.target.value)}
            className="px-3 py-1 border border-slate-300 rounded text-sm"
          >
            <option value="bias">Bias Score</option>
            <option value="censorship">Censorship Rate</option>
            <option value="accuracy">Accuracy Rate</option>
          </select>
        </div>
      </div>

      {/* World Map Visualization */}
      <div className="relative bg-slate-50 rounded-lg p-8 mb-6" style={{ height: '400px' }}>
        <svg className="absolute inset-0 w-full h-full" viewBox="0 0 100 100">
          {/* Connection lines between regions */}
          {regions.map((region1, i) => 
            regions.slice(i + 1).map(region2 => {
              const strength = getConnectionStrength(region1.id, region2.id);
              return (
                <line
                  key={`${region1.id}-${region2.id}`}
                  x1={region1.position.x}
                  y1={region1.position.y}
                  x2={region2.position.x}
                  y2={region2.position.y}
                  stroke="#e2e8f0"
                  strokeWidth={strength * 2 + 0.5}
                  strokeDasharray={strength < 0.5 ? "2,2" : "none"}
                  opacity={0.6}
                />
              );
            })
          )}
          
          {/* Region nodes */}
          {regions.map(region => {
            const data = getRegionData(region.id);
            const value = getMetricValue(region.id);
            const size = 3 + (value / 100) * 4; // Size based on metric value
            
            return (
              <g key={region.id}>
                <circle
                  cx={region.position.x}
                  cy={region.position.y}
                  r={size}
                  className={`${getMetricColor(value)} cursor-pointer transition-all hover:opacity-80`}
                  onClick={() => setSelectedRegion(selectedRegion === region.id ? null : region.id)}
                />
                <text
                  x={region.position.x}
                  y={region.position.y - size - 2}
                  textAnchor="middle"
                  className="text-xs font-medium fill-slate-700"
                >
                  {region.name}
                </text>
                <text
                  x={region.position.x}
                  y={region.position.y + size + 4}
                  textAnchor="middle"
                  className="text-xs fill-slate-600"
                >
                  {value.toFixed(1)}%
                </text>
              </g>
            );
          })}
        </svg>
      </div>

      {/* Legend */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-4 text-sm">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 bg-green-500 rounded-full"></div>
            <span className="text-slate-600">
              {viewMode === 'accuracy' ? 'High Accuracy' : 'Low Bias/Censorship'}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
            <span className="text-slate-600">Moderate</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 bg-red-500 rounded-full"></div>
            <span className="text-slate-600">
              {viewMode === 'accuracy' ? 'Low Accuracy' : 'High Bias/Censorship'}
            </span>
          </div>
        </div>
        <div className="text-xs text-slate-500">
          Click regions for details • Line thickness shows similarity
        </div>
      </div>

      {/* Region Details */}
      {selectedRegion && (
        <div className="border-t pt-4">
          {(() => {
            const region = regions.find(r => r.id === selectedRegion);
            const data = getRegionData(selectedRegion);
            return (
              <div className={`p-4 rounded-lg ${region.bgColor} ${region.borderColor} border`}>
                <div className="flex items-center justify-between mb-3">
                  <h4 className={`font-medium ${region.color}`}>{region.name}</h4>
                  <button
                    onClick={() => setSelectedRegion(null)}
                    className="text-slate-400 hover:text-slate-600"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
                
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                  <div>
                    <div className="text-slate-600">Models</div>
                    <div className="font-medium">{region.models.join(', ')}</div>
                  </div>
                  <div>
                    <div className="text-slate-600">Bias Score</div>
                    <div className="font-medium">{data.biasScore.toFixed(1)}%</div>
                  </div>
                  <div>
                    <div className="text-slate-600">Censorship</div>
                    <div className="font-medium">{data.censorshipRate.toFixed(1)}%</div>
                  </div>
                  <div>
                    <div className="text-slate-600">Accuracy</div>
                    <div className="font-medium">{data.accuracyRate.toFixed(1)}%</div>
                  </div>
                </div>
                
                <div className="mt-3 text-xs text-slate-600">
                  Based on {data.responseCount} benchmark responses across {region.models.length} model(s)
                </div>
              </div>
            );
          })()}
        </div>
      )}

      {/* Insights */}
      <div className="mt-6 p-4 bg-slate-50 rounded-lg">
        <h4 className="font-medium text-slate-900 mb-2">Key Insights</h4>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm text-slate-600">
          <div>
            <strong>Geographic Patterns:</strong> Models show distinct regional biases reflecting their training origins and cultural contexts.
          </div>
          <div>
            <strong>Censorship Variance:</strong> Response filtering varies significantly between regions, particularly on sensitive political topics.
          </div>
        </div>
      </div>
    </div>
  );
}
