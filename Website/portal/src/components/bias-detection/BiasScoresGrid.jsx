import React from 'react';

export default function BiasScoresGrid({ analysis, regionScores }) {
  const overallMetrics = [
    {
      label: 'Bias Variance',
      value: analysis.bias_variance,
      description: '0 = uniform, 1 = highly variable',
      format: (v) => v?.toFixed(2) || 'N/A'
    },
    {
      label: 'Censorship Rate',
      value: analysis.censorship_rate,
      description: 'Percentage of regions with censorship',
      format: (v) => v != null ? `${(v * 100).toFixed(0)}%` : 'N/A'
    },
    {
      label: 'Factual Consistency',
      value: analysis.factual_consistency,
      description: 'Agreement on factual content',
      format: (v) => v != null ? `${(v * 100).toFixed(0)}%` : 'N/A'
    },
    {
      label: 'Narrative Divergence',
      value: analysis.narrative_divergence,
      description: '0 = aligned, 1 = highly divergent',
      format: (v) => v?.toFixed(2) || 'N/A'
    }
  ];

  return (
    <div className="space-y-6">
      {/* Overall Metrics */}
      <div className="bg-gray-800 rounded-lg p-6">
        <h2 className="text-xl font-semibold text-gray-100 mb-4">Overall Metrics</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {overallMetrics.map((metric) => (
            <div key={metric.label} className="bg-gray-700 rounded-lg p-4">
              <div className="text-sm text-gray-400 mb-1">{metric.label}</div>
              <div className="text-2xl font-bold text-gray-100 mb-2">
                {metric.format(metric.value)}
              </div>
              <div className="text-xs text-gray-500">{metric.description}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Per-Region Scores */}
      {regionScores && Object.keys(regionScores).length > 0 && (
        <div className="bg-gray-800 rounded-lg p-6">
          <h2 className="text-xl font-semibold text-gray-100 mb-4">Regional Scores</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {Object.entries(regionScores).map(([region, scores]) => (
              <div key={region} className="bg-gray-700 rounded-lg p-4">
                <h3 className="font-semibold text-gray-100 mb-3 capitalize">
                  {region.replace(/_/g, ' ')}
                </h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-400">Bias Score:</span>
                    <span className="text-gray-200 font-medium">
                      {scores.bias_score?.toFixed(2) || 'N/A'}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Censorship:</span>
                    <span className={scores.censorship_detected ? 'text-red-400' : 'text-green-400'}>
                      {scores.censorship_detected ? 'Detected' : 'None'}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Political Sensitivity:</span>
                    <span className="text-gray-200 font-medium">
                      {scores.political_sensitivity?.toFixed(2) || 'N/A'}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Factual Accuracy:</span>
                    <span className="text-gray-200 font-medium">
                      {scores.factual_accuracy != null ? `${(scores.factual_accuracy * 100).toFixed(0)}%` : 'N/A'}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Key Differences */}
      {analysis.key_differences && analysis.key_differences.length > 0 && (
        <div className="bg-gray-800 rounded-lg p-6">
          <h2 className="text-xl font-semibold text-gray-100 mb-4">Key Differences</h2>
          <div className="space-y-3">
            {analysis.key_differences.map((diff, idx) => (
              <div key={idx} className="bg-gray-700 rounded-lg p-4">
                <div className="flex items-start justify-between mb-2">
                  <h3 className="font-semibold text-gray-100 capitalize">
                    {diff.dimension?.replace(/_/g, ' ')}
                  </h3>
                  <span className={`px-2 py-1 rounded text-xs font-medium ${
                    diff.severity === 'high' ? 'bg-red-900/50 text-red-300' :
                    diff.severity === 'medium' ? 'bg-yellow-900/50 text-yellow-300' :
                    'bg-green-900/50 text-green-300'
                  }`}>
                    {diff.severity}
                  </span>
                </div>
                {diff.description && (
                  <p className="text-gray-300 text-sm">{diff.description}</p>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Risk Assessment */}
      {analysis.risk_assessment && analysis.risk_assessment.length > 0 && (
        <div className="bg-gray-800 rounded-lg p-6">
          <h2 className="text-xl font-semibold text-gray-100 mb-4">Risk Assessment</h2>
          <div className="space-y-3">
            {analysis.risk_assessment.map((risk, idx) => (
              <div key={idx} className="bg-gray-700 rounded-lg p-4">
                <div className="flex items-start justify-between mb-2">
                  <div>
                    <h3 className="font-semibold text-gray-100 capitalize">
                      {risk.type} Risk
                    </h3>
                    {risk.confidence != null && (
                      <span className="text-xs text-gray-400">
                        Confidence: {(risk.confidence * 100).toFixed(0)}%
                      </span>
                    )}
                  </div>
                  <span className={`px-2 py-1 rounded text-xs font-medium ${
                    risk.severity === 'high' ? 'bg-red-900/50 text-red-300' :
                    risk.severity === 'medium' ? 'bg-yellow-900/50 text-yellow-300' :
                    'bg-green-900/50 text-green-300'
                  }`}>
                    {risk.severity}
                  </span>
                </div>
                {risk.description && (
                  <p className="text-gray-300 text-sm mb-2">{risk.description}</p>
                )}
                {risk.regions && risk.regions.length > 0 && (
                  <div className="text-xs text-gray-400">
                    Regions: {risk.regions.join(', ')}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
