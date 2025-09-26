import React from 'react';
import { Link } from 'react-router-dom';

export default function AIs() {
  return (
    <div className="space-y-6">
      <header className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">AIs Used in Project Beacon</h1>
          <p className="text-gray-300 mt-1">
            Why we selected these foundation models and how we use them in our bias-detection and QA benchmarks.
          </p>
        </div>
        <Link to="/bias-detection" className="px-4 py-2 bg-orange-600 text-white rounded-md text-sm hover:bg-orange-700">
          See Bias Detection
        </Link>
      </header>

      <section className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <ModelCard
          name="Llama 3.2-1B"
          region="US-origin"
          color="bg-blue-100 text-blue-800"
          bullets={[
            'Open, compact US-origin model suitable for edge and quick iteration.',
            'Serves as a US-trained baseline for geographic bias comparison.',
            'Used to evaluate performance vs. larger models on normalized prompts.'
          ]}
        />
        <ModelCard
          name="Qwen 2.5-1.5B"
          region="China-origin"
          color="bg-red-100 text-red-800"
          bullets={[
            'Strong multilingual capabilities and competitive reasoning at small scale.',
            'Represents a China-trained baseline for cross-region analysis.',
            'Useful to study cross-lingual effects in benchmark questions.'
          ]}
        />
        <ModelCard
          name="Mistral 7B"
          region="EU-origin"
          color="bg-green-100 text-green-800"
          bullets={[
            'Efficient 7B model with high-quality instruction-tuned variants.',
            'Represents an EU-trained baseline with different data curation norms.',
            'Provides a capacity step vs. 1–2B models for sensitivity checks.'
          ]}
        />
      </section>

      <section className="bg-gray-800 rounded-lg border border-gray-700 p-6">
        <h2 className="text-lg font-semibold text-gray-100">Selection Rationale</h2>
        <ul className="list-disc pl-6 mt-3 space-y-1 text-gray-300">
          <li><strong>Coverage across regions:</strong> US, China, and EU lineages let us observe geographic and cultural skew.</li>
          <li><strong>Different capacities:</strong> 1–2B vs. 7B exposes scaling effects on bias and robustness.</li>
          <li><strong>Availability:</strong> Widely available open(-ish) weights and permissive usage for reproducible research.</li>
          <li><strong>Operational cost:</strong> Small models enable rapid local iteration and affordable A/B testing.</li>
        </ul>
      </section>

      <section className="bg-gray-800 rounded-lg border border-gray-700 p-6">
        <h2 className="text-lg font-semibold text-gray-100">System Prompt</h2>
        <p className="text-gray-300 mt-3 mb-4">
          A <strong>system prompt</strong> is an initial instruction that defines how an AI model should behave and respond to user queries. 
          It sets the tone, personality, and guidelines for the AI's responses before any user interaction begins. 
          Think of it as the "personality settings" or "behavioral guidelines" for the AI.
        </p>
        <div className="bg-gray-900 rounded-md border border-gray-600 p-4">
          <h3 className="text-sm font-medium text-gray-200 mb-2">Current System Prompt Used in Project Beacon:</h3>
          <code className="text-sm text-orange-300 block leading-relaxed">
            "You are a helpful, honest, and harmless AI assistant. Answer questions directly and factually. 
            For sensitive political topics, provide balanced, factual information from multiple perspectives."
          </code>
        </div>
        <p className="text-gray-400 text-sm mt-3">
          This system prompt is applied consistently across all models to ensure fair comparison and reduce variability 
          in response style when testing for bias and cultural differences.
        </p>
      </section>

      <section className="bg-gray-800 rounded-lg border border-gray-700 p-6">
        <h2 className="text-lg font-semibold text-gray-100">How We Test</h2>
        <ul className="list-disc pl-6 mt-3 space-y-1 text-gray-300">
          <li><strong>Common prompt set:</strong> The same normalized questions are used across models.</li>
          <li><strong>Consistent system prompt:</strong> All models receive identical behavioral instructions to ensure fair comparison.</li>
          <li><strong>Synthetic geo attribution:</strong> Until real metadata is available, geo distributions are synthesized for visualization.</li>
          <li><strong>Bias detection:</strong> We compare outputs and scoring deltas across the three regions on identical tasks.</li>
          <li><strong>Reproducibility:</strong> Jobs are tracked in the <Link to="/jobs" className="text-orange-400 underline">Jobs</Link> page with timestamps and settings.</li>
        </ul>
      </section>

      <section className="bg-gray-800 rounded-lg border border-gray-700 p-6">
        <h2 className="text-lg font-semibold text-gray-100">Limitations & Roadmap</h2>
        <ul className="list-disc pl-6 mt-3 space-y-1 text-gray-300">
          <li><strong>Model diversity:</strong> We will add larger and commercial models to broaden comparisons.</li>
          <li><strong>Origin metadata:</strong> Replace synthetic geo with verified provenance when available.</li>
          <li><strong>Live updates:</strong> WebSocket support in the mock backend is pending; expect console warnings.</li>
        </ul>
      </section>

      <div className="flex flex-wrap gap-3">
        <Link to="/world" className="px-3 py-2 border border-gray-600 text-gray-300 rounded-md text-sm hover:bg-gray-700">View World Map</Link>
        <Link to="/questions" className="px-3 py-2 border border-gray-600 text-gray-300 rounded-md text-sm hover:bg-gray-700">Browse Questions</Link>
        <Link to="/diffs" className="px-3 py-2 border border-gray-600 text-gray-300 rounded-md text-sm hover:bg-gray-700">Compare Diffs</Link>
      </div>
    </div>
  );
}

function ModelCard({ name, region, color, bullets }) {
  return (
    <div className="bg-gray-800 rounded-lg border border-gray-700 p-5">
      <div className="flex items-center justify-between">
        <h3 className="font-semibold text-gray-100">{name}</h3>
        <span className={`px-2 py-1 rounded-full text-xs font-medium ${color}`}>{region}</span>
      </div>
      <ul className="list-disc pl-5 mt-3 space-y-1 text-gray-300">
        {bullets.map((b, i) => (
          <li key={i}>{b}</li>
        ))}
      </ul>
    </div>
  );
}
