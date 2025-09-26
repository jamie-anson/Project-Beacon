import React from 'react';

function getBiasClasses(score = 0) {
  if (score < 30) return 'bg-green-900/20 border-green-500 text-green-300';
  if (score < 70) return 'bg-yellow-900/20 border-yellow-500 text-yellow-300';
  return 'bg-red-900/20 border-red-500 text-red-300';
}

export default function RegionalBreakdown({ modelName, regions }) {
  if (!regions?.length) return null;

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
      {regions.map((region) => {
        const biasClass = getBiasClasses(region.bias_score);
        return (
          <div key={region.region_code} className="bg-gray-800 border border-gray-700 rounded-lg">
            <div
              className={`px-6 py-4 border-b border-gray-700 ${
                region.bias_score < 30
                  ? 'bg-green-900/20'
                  : region.bias_score < 70
                  ? 'bg-yellow-900/20'
                  : 'bg-red-900/20'
              }`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className="text-lg">{region.flag}</span>
                  <h3 className="font-semibold text-gray-100">{region.region_name}</h3>
                </div>
                <span
                  className={`px-2 py-1 rounded-full text-xs font-medium ${
                    region.bias_score < 30
                      ? 'bg-green-900/30 text-green-300'
                      : region.bias_score < 70
                      ? 'bg-yellow-900/30 text-yellow-300'
                      : 'bg-red-900/30 text-red-300'
                  }`}
                >
                  Bias: {region.bias_score}%
                </span>
              </div>
              <div className="mt-2 text-sm text-gray-300">
                <div>
                  <strong>Provider:</strong> {region.provider_id}
                </div>
                <div>
                  <strong>Model:</strong> {modelName} • <strong>Status:</strong>{' '}
                  <span
                    className={`ml-1 font-medium ${
                      region.censorship_level === 'low' ? 'text-green-400' : 'text-red-400'
                    }`}
                  >
                    {region.censorship_level === 'low' ? 'Uncensored' : 'Censored'}
                  </span>
                </div>
              </div>
            </div>
            <div className="p-6">
              <div
                className={`bg-gray-900 border-l-4 p-4 rounded-r ${
                  region.bias_score < 30
                    ? 'border-green-500'
                    : region.bias_score < 70
                    ? 'border-yellow-500'
                    : 'border-red-500'
                }`}
              >
                <p className="text-sm text-gray-200 italic">"{region.response}"</p>
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
                  {(region.keywords || []).map((keyword, index) => (
                    <span
                      key={`${region.region_code}-${index}`}
                      className={`px-2 py-1 rounded-full text-xs ${
                        region.bias_score < 30
                          ? 'bg-green-900/30 text-green-300'
                          : region.bias_score < 70
                          ? 'bg-yellow-900/30 text-yellow-300'
                          : 'bg-red-900/30 text-red-300'
                      }`}
                    >
                      {keyword}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
