import React from 'react';
import { Link } from 'react-router-dom';

export default function Home() {
  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold">Welcome to Project Beacon</h1>
        <p className="text-slate-600 text-sm max-w-3xl">
          This portal helps you run bias detection experiments across models and regions, track progress, and review results.
          Follow the steps below to get started.
        </p>
      </header>

      <section className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="bg-white border rounded-lg p-4">
          <h2 className="text-lg font-semibold">1. Pick your questions</h2>
          <p className="text-sm text-slate-600 mt-1">
            Curate or edit the prompts used for evaluation.
          </p>
          <div className="mt-3">
            <Link to="/questions" className="inline-flex items-center px-3 py-2 bg-beacon-600 text-white rounded-md text-sm hover:bg-beacon-700">
              Go to Questions
            </Link>
          </div>
        </div>

        <div className="bg-white border rounded-lg p-4">
          <h2 className="text-lg font-semibold">2. Select models</h2>
          <p className="text-sm text-slate-600 mt-1">
            Review available providers and models included in the benchmark.
          </p>
          <div className="mt-3">
            <Link to="/ais" className="inline-flex items-center px-3 py-2 bg-beacon-600 text-white rounded-md text-sm hover:bg-beacon-700">
              Browse Models
            </Link>
          </div>
        </div>

        <div className="bg-white border rounded-lg p-4">
          <h2 className="text-lg font-semibold">3. Run bias detection</h2>
          <p className="text-sm text-slate-600 mt-1">
            Submit a job to evaluate responses across regions. Watch live progress.
          </p>
          <div className="mt-3">
            <Link to="/bias-detection" className="inline-flex items-center px-3 py-2 bg-beacon-600 text-white rounded-md text-sm hover:bg-beacon-700">
              Open Bias Detection
            </Link>
          </div>
        </div>

        <div className="bg-white border rounded-lg p-4">
          <h2 className="text-lg font-semibold">4. Track and review</h2>
          <p className="text-sm text-slate-600 mt-1">
            Check overall system status and recent activity on the dashboard, and compare outputs in results.
          </p>
          <div className="mt-3 flex gap-3">
            <Link to="/dashboard" className="inline-flex items-center px-3 py-2 bg-slate-900 text-white rounded-md text-sm hover:bg-black">
              View Dashboard
            </Link>
            <Link to="/results" className="inline-flex items-center px-3 py-2 bg-slate-200 text-slate-900 rounded-md text-sm hover:bg-slate-300">
              View Results
            </Link>
          </div>
        </div>
      </section>

      <section className="bg-white border rounded-lg p-4">
        <h3 className="text-sm font-semibold text-slate-900">Tips</h3>
        <ul className="mt-2 list-disc pl-5 text-sm text-slate-700 space-y-1">
          <li>Keep the Bias Detection tab open to see live progress during a run.</li>
          <li>You can refresh results any time from the page actions.</li>
          <li>Settings let you adjust environment details when using a real backend.</li>
        </ul>
      </section>
    </div>
  );
}
