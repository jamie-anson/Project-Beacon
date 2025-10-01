import React from 'react';

export default function RegionSelector({ 
  selectedRegions, 
  onRegionToggle, 
  calculateEstimatedCost,
  readSelectedQuestions,
  selectedModels = []
}) {
  const availableRegions = [
    { code: 'US', name: 'United States' },
    { code: 'EU', name: 'Europe' },
    { code: 'ASIA', name: 'Asia Pacific' }
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
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={selectedRegions.includes(region.code)}
                  onChange={() => onRegionToggle(region.code)}
                  className="rounded border-gray-600 bg-gray-700 text-orange-500 focus:ring-orange-500"
                />
                <span className="font-medium text-gray-100">{region.name}</span>
              </div>
            </div>
          ))}
        </div>
      </div>


      {/* Job Summary */}
      <div className="bg-gray-700 rounded-lg p-4 space-y-2">
        <h3 className="text-sm font-medium text-gray-100">Job Summary</h3>
        <div className="grid grid-cols-3 gap-4 text-sm">
          <div>
            <span className="text-gray-300">Questions:</span>
            <span className="ml-1 font-medium">{readSelectedQuestions().length}</span>
          </div>
          <div>
            <span className="text-gray-300">Regions:</span>
            <span className="ml-1 font-medium">{selectedRegions.length}</span>
          </div>
          <div>
            <span className="text-gray-300">Models:</span>
            <span className="ml-1 font-medium">{selectedModels.length}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
