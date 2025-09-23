import React from 'react';

const AVAILABLE_MODELS = [
  {
    id: 'llama3.2-1b',
    name: 'Llama 3.2-1B',
    description: 'Fast 1B parameter model for quick inference',
    contextLength: 128000,
    speed: 'Fast',
    quality: 'Good',
    cost: 0.0003
  },
  {
    id: 'mistral-7b',
    name: 'Mistral 7B Instruct',
    description: 'Strong 7B parameter general-purpose model',
    contextLength: 32768,
    speed: 'Medium',
    quality: 'Excellent',
    cost: 0.0004
  },
  {
    id: 'qwen2.5-1.5b',
    name: 'Qwen 2.5-1.5B Instruct',
    description: 'Efficient 1.5B parameter model',
    contextLength: 32768,
    speed: 'Fast',
    quality: 'Very Good',
    cost: 0.00035
  }
];

export default function ModelSelector({ selectedModels = [], onModelChange, className = '' }) {
  const selectedModelInfos = AVAILABLE_MODELS.filter(m => selectedModels.includes(m.id));

  return (
    <div className={`space-y-4 ${className}`}>
      <div>
        <label className="block text-sm font-medium text-gray-300 mb-2">
          Model Selection
        </label>
        <p className="text-xs text-gray-400 mb-3">
          Choose one or more language models for bias detection across all regions
        </p>
      </div>

      <div className="grid gap-3">
        {AVAILABLE_MODELS.map((model) => (
          <div
            key={model.id}
            className={`relative rounded-lg border p-4 cursor-pointer transition-all ${
              selectedModels.includes(model.id)
                ? 'border-orange-500 bg-orange-500/10 ring-1 ring-orange-500'
                : 'border-gray-600 bg-gray-700/50 hover:border-gray-500 hover:bg-gray-700'
            }`}
            onClick={() => {
              const newSelection = selectedModels.includes(model.id)
                ? selectedModels.filter(id => id !== model.id)
                : [...selectedModels, model.id];
              onModelChange(newSelection);
            }}
          >
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    value={model.id}
                    checked={selectedModels.includes(model.id)}
                    onChange={(e) => {
                      const newSelection = e.target.checked
                        ? [...selectedModels, model.id]
                        : selectedModels.filter(id => id !== model.id);
                      onModelChange(newSelection);
                    }}
                    className="text-orange-500 focus:ring-orange-500 border-gray-600 bg-gray-700 rounded"
                  />
                  <h3 className="text-sm font-medium text-gray-100">
                    {model.name}
                  </h3>
                </div>
                <p className="text-xs text-gray-400 mt-1 ml-6">
                  {model.description}
                </p>
                <div className="flex items-center gap-4 mt-2 ml-6 text-xs text-gray-500">
                  <span>Speed: {model.speed}</span>
                  <span>Quality: {model.quality}</span>
                  <span>Context: {model.contextLength.toLocaleString()} tokens</span>
                  <span>Cost: ${model.cost}/sec</span>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {selectedModelInfos.length > 0 && (
        <div className="bg-gray-700/50 rounded-lg p-3 border border-gray-600">
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-300">Selected Models:</span>
            <span className="text-orange-400 font-medium">
              {selectedModelInfos.map(m => m.name).join(', ')}
            </span>
          </div>
          <div className="flex items-center justify-between text-xs text-gray-400 mt-1">
            <span>Available in all regions (US, EU, APAC)</span>
            <span>
              Estimated cost: ${Math.min(...selectedModelInfos.map(m => m.cost))}-${Math.max(...selectedModelInfos.map(m => m.cost))}/sec per region
            </span>
          </div>
        </div>
      )}
    </div>
  );
}

export { AVAILABLE_MODELS };
