import React from 'react';
import WorldMapVisualization from '../WorldMapVisualization';

const COUNTRY_MAPPING = {
  US: 'United States',
  EU: 'Germany',
  ASIA: 'Singapore'
};

function getBiasCategory(score = 0) {
  if (score < 30) return 'low';
  if (score < 70) return 'medium';
  return 'high';
}

function getBiasColorClass(score = 0) {
  const category = getBiasCategory(score);
  if (category === 'low') return 'bg-green-500';
  if (category === 'medium') return 'bg-yellow-500';
  return 'bg-red-500';
}

export default function DiffMapSection({ modelName, regions }) {
  const mapData = regions.map((region) => {
    const countryName = COUNTRY_MAPPING[region.region_code] || region.region_name;
    return {
      name: countryName,
      value: region.bias_score || 0,
      category: getBiasCategory(region.bias_score)
    };
  });

  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
      <h3 className="text-lg font-semibold text-gray-100 mb-4">
        Global Response Coverage - {modelName}
      </h3>
      <div className="bg-gray-900 rounded-lg p-4">
        <p className="text-sm text-gray-300 mb-4">
          Cross-region bias detection results showing response patterns for {modelName} across different geographic locations and providers.
        </p>
        <div className="mb-8">
          <WorldMapVisualization biasData={mapData} />
        </div>
        <div className="mt-4 grid grid-cols-2 md:grid-cols-3 gap-3 text-xs">
          {regions.map((region) => (
            <div
              key={region.region_code}
              className="flex items-center gap-2 p-2 bg-gray-900 rounded border border-gray-700"
            >
              <div className={`w-3 h-3 rounded ${getBiasColorClass(region.bias_score)}`} />
              <div>
                <div className="font-medium text-gray-100">
                  {region.flag} {region.region_name}
                </div>
                <div className="text-gray-300">
                  {region.censorship_level === 'low' ? 'Uncensored' : 'Censored'} ({region.bias_score}% bias)
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
