import React from 'react';

export default function DiffNarrativeTable({ modelName, regions }) {
  if (!regions?.length) return null;

  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg">
      <div className="px-6 py-4 border-b border-gray-700">
        <h2 className="text-lg font-medium text-gray-100">
          📊 Cross-Region Analysis: {modelName} Narrative Differences
        </h2>
        <p className="text-sm text-gray-300 mt-1">
          The following table shows how {modelName} receives dramatically different responses across regions, revealing
          systematic bias patterns and censorship mechanisms.
        </p>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-gray-900">
            <tr className="text-xs text-gray-400 uppercase tracking-wide">
              <th className="px-6 py-3 text-left font-medium">Dimension</th>
              {regions.map((region) => (
                <th key={region.region_code} className="px-6 py-3 text-left font-medium">
                  {region.flag} {region.region_name}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-700">
            <tr className="hover:bg-gray-900/40">
              <td className="px-6 py-4 font-medium text-gray-100">Response Tone</td>
              {regions.map((region) => (
                <td key={`${region.region_code}-tone`} className="px-6 py-4 text-sm text-gray-300">
                  {region.bias_score < 30
                    ? 'Direct, factual'
                    : region.bias_score < 70
                    ? 'Diplomatic, cautious'
                    : 'Evasive, censored'}
                </td>
              ))}
            </tr>
            <tr className="hover:bg-gray-900/40">
              <td className="px-6 py-4 font-medium text-gray-100">Censorship Level</td>
              {regions.map((region) => (
                <td key={`${region.region_code}-censorship`} className="px-6 py-4 text-sm text-gray-300">
                  {region.censorship_level === 'low' ? 'Minimal filtering' : 'Heavy censorship'}
                </td>
              ))}
            </tr>
            <tr className="hover:bg-gray-900/40">
              <td className="px-6 py-4 font-medium text-gray-100">Bias Score</td>
              {regions.map((region) => (
                <td key={`${region.region_code}-bias`} className="px-6 py-4 text-sm text-gray-300">
                  {region.bias_score}% bias detected
                </td>
              ))}
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  );
}
