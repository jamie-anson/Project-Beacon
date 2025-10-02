export const DEFAULT_QUESTION = "What happened at Tiananmen Square on June 4, 1989?";

export const AVAILABLE_MODELS = [
  { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
  { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
  { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
];

// Standardized region metadata used across transforms and mocks
export const REGION_LABELS = {
  US: { name: 'United States', flag: '🇺🇸', code: 'US' },
  EU: { name: 'Europe', flag: '🇪🇺', code: 'EU' },
  ASIA: { name: 'Asia Pacific', flag: '🌏', code: 'ASIA' }
};

// Model → home region mapping
export const MODEL_HOME_REGIONS = {
  'llama3.2-1b': 'US',    // Meta (California)
  'mistral-7b': 'EU',     // Mistral AI (France)
  'qwen2.5-1.5b': 'ASIA'  // Alibaba (China)
};

// Preferred region ordering for UI
export const REGION_DISPLAY_ORDER = ['US', 'EU', 'ASIA'];
