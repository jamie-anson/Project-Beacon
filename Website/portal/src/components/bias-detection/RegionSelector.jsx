import React from 'react';

export default function RegionSelector({ 
  selectedRegions, 
  onRegionToggle, 
  calculateEstimatedCost,
  readSelectedQuestions
}) {
  const availableRegions = [
    { code: 'US', name: 'United States', model: 'Llama 3.2-1B', cost: 0.0003 },
    { code: 'EU', name: 'Europe', model: 'Mistral 7B', cost: 0.0004 },
    { code: 'ASIA', name: 'Asia Pacific', model: 'Qwen 2.5-1.5B', cost: 0.0005 }
  ];

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-medium text-gray-100 mb-3">Select Regions</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {availableRegions.map((region) => (
            <div
              key={region.code}
              className={`border rounded-lg p-4 cursor-pointer transition-all ${
                selectedRegions.includes(region.code)
                  ? 'border-orange-500 bg-orange-50 bg-opacity-10'
                  : 'border-gray-600 hover:border-gray-500'
              }`}
              onClick={() => onRegionToggle(region.code)}
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={selectedRegions.includes(region.code)}
                    onChange={() => onRegionToggle(region.code)}
                    className="rounded border-gray-600 bg-gray-700 text-orange-500 focus:ring-orange-500"
                  />
                  <span className="font-medium text-gray-100">{region.code}</span>
                </div>
                <span className="text-xs text-gray-300">Est. cost</span>
              </div>
              <div className="text-sm text-gray-300">{region.name}</div>
              <div className="text-xs text-gray-400 mt-1">{region.model}</div>
              <div className="text-sm font-medium text-orange-400 mt-2">
                ${region.cost.toFixed(4)}
              </div>
            </div>
          ))}
        </div>
      </div>


      {/* Job Summary */}
      <div className="bg-gray-700 rounded-lg p-4 space-y-2">
        <h3 className="text-sm font-medium text-gray-100">Job Summary</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <div>
            <span className="text-gray-300">Questions:</span>
            <span className="ml-1 font-medium">{readSelectedQuestions().length}</span>
          </div>
          <div>
            <span className="text-gray-300">Regions:</span>
            <span className="ml-1 font-medium">{selectedRegions.length}</span>
          </div>
          <div>
            <span className="text-gray-300">Type:</span>
            <span className="ml-1 font-medium">{selectedRegions.length > 1 ? 'Multi-Region' : 'Single-Region'}</span>
          </div>
          <div>
            <span className="text-gray-300">Est. Cost:</span>
            <span className="ml-1 font-medium">${calculateEstimatedCost()}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
