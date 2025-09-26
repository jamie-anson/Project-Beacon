import React from 'react';

export default function ModelSelector({ models, selectedModel, onSelectModel }) {
  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
      <h3 className="text-lg font-semibold text-gray-100 mb-3">Select Model for Comparison</h3>
      <div className="flex flex-wrap gap-3">
        {models.map((model) => (
          <button
            key={model.id}
            onClick={() => onSelectModel(model.id)}
            className={`px-4 py-2 rounded-lg border transition-all ${
              selectedModel === model.id
                ? 'border-blue-400 bg-blue-900/20 text-blue-300'
                : 'border-gray-700 hover:border-gray-500 text-gray-200'
            }`}
          >
            <div className="font-medium">{model.name}</div>
            <div className="text-xs text-gray-400">{model.provider}</div>
          </button>
        ))}
      </div>
    </div>
  );
}
