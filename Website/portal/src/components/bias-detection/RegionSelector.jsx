import React from 'react';

export default function RegionSelector({ 
  selectedRegions, 
  onRegionToggle, 
  calculateEstimatedCost,
  readSelectedQuestions,
  selectedModels = []
}) {
  const availableRegions = [
    { code: 'US', name: 'United States', disabled: false },
    { code: 'EU', name: 'Europe', disabled: false },
    { 
      code: 'ASIA', 
      name: 'Asia Pacific', 
      disabled: true,
      disabledReason: 'Temporarily unavailable due to infrastructure optimization'
    }
  ];

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-medium text-gray-100 mb-3">Select Regions</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {availableRegions.map((region) => (
            <div
              key={region.code}
              className={`border rounded-lg p-4 transition-all ${
                region.disabled
                  ? 'border-gray-700 bg-gray-800 opacity-50 cursor-not-allowed'
                  : selectedRegions.includes(region.code)
                  ? 'border-orange-500 bg-orange-50 bg-opacity-10 cursor-pointer'
                  : 'border-gray-600 hover:border-gray-500 cursor-pointer'
              }`}
              onClick={() => !region.disabled && onRegionToggle(region.code)}
              title={region.disabled ? region.disabledReason : ''}
            >
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={selectedRegions.includes(region.code)}
                  onChange={() => onRegionToggle(region.code)}
                  disabled={region.disabled}
                  className="rounded border-gray-600 bg-gray-700 text-orange-500 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed"
                />
                <span className={`font-medium ${region.disabled ? 'text-gray-500' : 'text-gray-100'}`}>
                  {region.name}
                  {region.disabled && (
                    <span className="ml-2 text-xs text-gray-500">(Disabled)</span>
                  )}
                </span>
              </div>
              {region.disabled && region.disabledReason && (
                <p className="mt-2 text-xs text-gray-500">{region.disabledReason}</p>
              )}
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
