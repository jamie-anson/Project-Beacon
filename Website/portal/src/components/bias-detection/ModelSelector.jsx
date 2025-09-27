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

export default function ModelSelector({ 
  selectedModel, 
  selectedModels, 
  onModelChange, 
  className = '',
  multiSelect = false 
}) {
  // Defensive programming - ensure we have safe values
  const safeSelectedModel = typeof selectedModel === 'string' ? selectedModel : 'qwen2.5-1.5b';
  const safeSelectedModels = Array.isArray(selectedModels) ? selectedModels : [safeSelectedModel];
  const safeOnModelChange = typeof onModelChange === 'function' ? onModelChange : () => {};
  
  // Use multi-select mode if selectedModels is provided, otherwise single-select
  const isMultiSelect = multiSelect || (selectedModels && Array.isArray(selectedModels));
  
  const selectedModelInfo = AVAILABLE_MODELS.find(m => m.id === safeSelectedModel);
  const selectedModelInfos = AVAILABLE_MODELS.filter(m => safeSelectedModels.includes(m.id));

  return (
    <div className={`space-y-4 ${className}`}>
      <div>
        <h3 className="text-lg font-medium text-gray-100 mb-3">Model Selection</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {AVAILABLE_MODELS.map((model) => (
            <div
              key={model.id}
              className={`border rounded-lg p-4 cursor-pointer transition-all ${
                isMultiSelect 
                  ? (safeSelectedModels.includes(model.id)
                      ? 'border-orange-500 bg-orange-50 bg-opacity-10'
                      : 'border-gray-600 hover:border-gray-500')
                  : (safeSelectedModel === model.id
                      ? 'border-orange-500 bg-orange-50 bg-opacity-10'
                      : 'border-gray-600 hover:border-gray-500')
              }`}
              onClick={() => {
                try {
                  if (isMultiSelect) {
                    const newSelection = safeSelectedModels.includes(model.id)
                      ? safeSelectedModels.filter(id => id !== model.id)
                      : [...safeSelectedModels, model.id];
                    // Ensure at least one model is selected
                    const finalSelection = newSelection.length > 0 ? newSelection : [model.id];
                    safeOnModelChange(finalSelection);
                  } else {
                    safeOnModelChange(model.id);
                  }
                } catch (error) {
                  console.warn('Error in model selection:', error);
                }
              }}
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <input
                    type={isMultiSelect ? "checkbox" : "radio"}
                    name={isMultiSelect ? undefined : "model"}
                    value={model.id}
                    checked={isMultiSelect 
                      ? safeSelectedModels.includes(model.id)
                      : safeSelectedModel === model.id
                    }
                    onChange={(e) => {
                      try {
                        if (isMultiSelect) {
                          const newSelection = e.target.checked
                            ? [...safeSelectedModels, model.id]
                            : safeSelectedModels.filter(id => id !== model.id);
                          // Ensure at least one model is selected
                          const finalSelection = newSelection.length > 0 ? newSelection : [model.id];
                          safeOnModelChange(finalSelection);
                        } else {
                          safeOnModelChange(model.id);
                        }
                      } catch (error) {
                        console.warn('Error in input change:', error);
                      }
                    }}
                    className="rounded border-gray-600 bg-gray-700 text-orange-500 focus:ring-orange-500"
                  />
                  <span className="font-medium text-gray-100">{model.name}</span>
                </div>
                <span className="text-xs text-gray-300">Est. cost</span>
              </div>
              <div className="text-sm text-gray-300">{model.description}</div>
              <div className="text-xs text-gray-400 mt-1">Speed: {model.speed} • Quality: {model.quality}</div>
              <div className="text-sm font-medium text-orange-400 mt-2">
                ${model.cost.toFixed(4)}/sec
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export { AVAILABLE_MODELS };
