import React from 'react';

export default function MetricsGrid({ metrics }) {
  if (!metrics) return null;

  const items = [
    { label: 'Bias Variance', value: metrics.bias_variance },
    { label: 'Censorship Rate', value: metrics.censorship_rate },
    { label: 'Factual Consistency', value: metrics.factual_consistency },
    { label: 'Narrative Divergence', value: metrics.narrative_divergence }
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
      {items.map((item) => (
        <div key={item.label} className="bg-gray-800 border border-gray-700 rounded-lg p-4">
          <div className="text-2xl font-bold text-red-400">{item.value}%</div>
          <div className="text-sm text-gray-300 uppercase tracking-wide">{item.label}</div>
        </div>
      ))}
    </div>
  );
}
