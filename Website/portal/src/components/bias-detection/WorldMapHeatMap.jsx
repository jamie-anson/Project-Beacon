import React from 'react';
import BiasHeatMap from '../BiasHeatMap';

export default function WorldMapHeatMap({ regionScores }) {
  // Transform regionScores to match BiasHeatMap's expected format
  const transformedData = {};
  
  Object.entries(regionScores).forEach(([regionKey, scores]) => {
    // Map region keys to BiasHeatMap's expected format (US, EU, ASIA)
    let mappedRegion = null;
    if (regionKey.includes('us') || regionKey.includes('america')) {
      mappedRegion = 'US';
    } else if (regionKey.includes('eu') || regionKey.includes('europe')) {
      mappedRegion = 'EU';
    } else if (regionKey.includes('asia') || regionKey.includes('pacific')) {
      mappedRegion = 'ASIA';
    }
    
    if (mappedRegion) {
      transformedData[mappedRegion] = {
        scoring: {
          bias_score: scores.bias_score || 0,
          censorship_score: scores.censorship_detected ? 0.8 : 0.2,
        },
        provider: regionKey.replace(/_/g, ' ')
      };
    }
  });

  return (
    <div className="bg-gray-800 rounded-lg p-6">
      <h2 className="text-xl font-semibold text-gray-100 mb-4">Regional Bias Heat Map</h2>
      <div className="h-96">
        <BiasHeatMap regionData={transformedData} className="h-full" />
      </div>
    </div>
  );
}
