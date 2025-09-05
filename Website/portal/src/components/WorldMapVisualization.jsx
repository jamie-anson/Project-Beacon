import React from 'react';

const WorldMapVisualization = ({ biasData = [] }) => {
  // Default demo data if none provided
  const defaultBiasData = [
    { name: 'United States', value: 15, category: 'low' },
    { name: 'Germany', value: 18, category: 'low' },
    { name: 'France', value: 18, category: 'low' },
    { name: 'United Kingdom', value: 18, category: 'low' },
    { name: 'China', value: 95, category: 'high' },
    { name: 'Singapore', value: 45, category: 'medium' },
    { name: 'Thailand', value: 45, category: 'medium' },
    { name: 'Malaysia', value: 45, category: 'medium' },
    { name: 'Indonesia', value: 45, category: 'medium' }
  ];

  const data = biasData.length > 0 ? biasData : defaultBiasData;

  // Simple SVG world map representation
  const regions = [
    { name: 'United States', x: 120, y: 180, width: 120, height: 80 },
    { name: 'China', x: 520, y: 200, width: 100, height: 70 },
    { name: 'Germany', x: 380, y: 140, width: 40, height: 30 },
    { name: 'France', x: 360, y: 150, width: 35, height: 30 },
    { name: 'United Kingdom', x: 340, y: 130, width: 30, height: 25 },
    { name: 'Singapore', x: 580, y: 280, width: 15, height: 15 },
    { name: 'Thailand', x: 560, y: 260, width: 25, height: 30 },
    { name: 'Malaysia', x: 570, y: 290, width: 30, height: 20 },
    { name: 'Indonesia', x: 590, y: 310, width: 50, height: 25 }
  ];

  const getColor = (regionName) => {
    const regionData = data.find(d => d.name === regionName);
    if (!regionData) return '#e5e7eb';
    
    switch (regionData.category) {
      case 'high': return '#ef4444';
      case 'medium': return '#f59e0b';
      case 'low': return '#10b981';
      default: return '#e5e7eb';
    }
  };

  const getBiasValue = (regionName) => {
    const regionData = data.find(d => d.name === regionName);
    return regionData ? regionData.value : 0;
  };

  return (
    <div className="w-full">
      <div className="bg-white rounded-lg border p-6">
        <h2 className="text-2xl font-bold text-center mb-6 text-slate-900">
          Global Response Coverage
        </h2>
        
        {/* Simple SVG World Map */}
        <div className="flex justify-center mb-6">
          <svg width="800" height="400" viewBox="0 0 800 400" className="border rounded">
            {/* Background */}
            <rect width="800" height="400" fill="#f8fafc" />
            
            {/* Continents outline */}
            <rect x="80" y="120" width="200" height="150" fill="#e2e8f0" stroke="#94a3b8" strokeWidth="1" rx="10" />
            <text x="180" y="200" textAnchor="middle" className="text-sm font-medium fill-slate-600">North America</text>
            
            <rect x="320" y="100" width="150" height="120" fill="#e2e8f0" stroke="#94a3b8" strokeWidth="1" rx="10" />
            <text x="395" y="165" textAnchor="middle" className="text-sm font-medium fill-slate-600">Europe</text>
            
            <rect x="480" y="140" width="200" height="180" fill="#e2e8f0" stroke="#94a3b8" strokeWidth="1" rx="10" />
            <text x="580" y="235" textAnchor="middle" className="text-sm font-medium fill-slate-600">Asia</text>
            
            {/* Country regions */}
            {regions.map((region) => (
              <g key={region.name}>
                <rect
                  x={region.x}
                  y={region.y}
                  width={region.width}
                  height={region.height}
                  fill={getColor(region.name)}
                  stroke="#374151"
                  strokeWidth="1"
                  rx="3"
                  className="hover:opacity-80 cursor-pointer"
                />
                <text
                  x={region.x + region.width / 2}
                  y={region.y + region.height / 2 + 4}
                  textAnchor="middle"
                  className="text-xs font-medium fill-white pointer-events-none"
                >
                  {getBiasValue(region.name)}%
                </text>
              </g>
            ))}
          </svg>
        </div>

        {/* Legend */}
        <div className="flex justify-center items-center gap-6 mb-4">
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-green-500 rounded"></div>
            <span className="text-sm text-slate-600">Low Bias (0-30%)</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-yellow-500 rounded"></div>
            <span className="text-sm text-slate-600">Medium Bias (30-70%)</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-red-500 rounded"></div>
            <span className="text-sm text-slate-600">High Bias (70-100%)</span>
          </div>
        </div>

        <p className="text-center text-slate-600 text-sm">
          Cross-region bias detection results showing response patterns across different geographic locations and providers.
        </p>
      </div>
    </div>
  );
};

export default WorldMapVisualization;
