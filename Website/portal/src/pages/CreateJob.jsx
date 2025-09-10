import React from 'react';
import Ajv from 'ajv';
import { useNavigate } from 'react-router-dom';
import { createJob } from '../lib/api.js';

export default function CreateJob() {
  const [text, setText] = React.useState('');
  const [error, setError] = React.useState(null);
  const [submitting, setSubmitting] = React.useState(false);
  const [validationErrors, setValidationErrors] = React.useState([]);
  const navigate = useNavigate();

  // Initialize AJV validator
  const ajv = React.useMemo(() => new Ajv({ allErrors: true, strict: false }), []);
  const validateAjv = React.useMemo(() => ajv.compile(jobSpecSchema), [ajv]);

  const placeholderSample = `{
  "job": { ... },
  "public_key": "...",
  "signature": "..."
}`;

  const onFile = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const content = await file.text();
    setText(content);
  };

  const onSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    let spec;
    try {
      spec = JSON.parse(text);
    } catch (err) {
      setError('Invalid JSON: ' + err.message);
      return;
    }
    // Run client-side validation before submit
    const errs = validateSpec(spec, validateAjv);
    setValidationErrors(errs);
    if (errs.length > 0) {
      setError('Please fix validation errors before submitting.');
      return;
    }
    setSubmitting(true);
    try {
      const res = await createJob(spec);
      // Expect: { id: "<jobspec_id>", status: "enqueued" }
      if (res?.id) {
        navigate(`/jobs/${encodeURIComponent(res.id)}`);
      } else {
        setError('Unexpected response from server.');
      }
    } catch (err) {
      setError(String(err.message || err));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-semibold">Create Job</h2>
      <p className="text-sm text-slate-600">Paste a signed JobSpec JSON (must include public_key and signature).</p>

      <form onSubmit={onSubmit} className="space-y-3">
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <input type="file" accept="application/json,.json" onChange={onFile} />
          <button type="button" className="px-2 py-1 border rounded" onClick={() => setText('')}>Clear</button>
          <span className="text-slate-500">Templates:</span>
          <button type="button" className="px-2 py-1 border rounded" onClick={() => setText(getTemplate('llama'))}>Llama 3.2-1B (US)</button>
          <button type="button" className="px-2 py-1 border rounded" onClick={() => setText(getTemplate('qwen'))}>Qwen 2.5-1.5B (CN)</button>
          <button type="button" className="px-2 py-1 border rounded" onClick={() => setText(getTemplate('mistral'))}>Mistral 7B (EU)</button>
        </div>
        <textarea
          className="w-full h-64 border rounded p-2 font-mono text-xs"
          placeholder={placeholderSample}
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
        {validationErrors.length > 0 && (
          <div className="text-sm text-red-700 bg-red-50 border border-red-200 rounded p-2">
            <div className="font-medium mb-1">Validation errors:</div>
            <ul className="list-disc pl-5 space-y-0.5">
              {validationErrors.map((e, i) => (<li key={i}>{e}</li>))}
            </ul>
          </div>
        )}
        {error && <div className="text-sm text-red-600">{error}</div>}
        <div className="flex items-center gap-2">
          <button type="submit" className="px-3 py-1.5 rounded bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50" disabled={submitting || validationErrors.length > 0}>Submit</button>
          {submitting && <span className="text-sm text-slate-500">Submitting…</span>}
        </div>
      </form>
    </div>
  );
}

// --- Helpers: templates and validation ---

function getTemplate(kind) {
  const base = {
    job: {
      id: "job-<unique-id>",
      version: "v0",
      benchmark: { name: "llm-benchmark", params: { prompt_set: "neutral_v1" } },
      container: { image: "", command: ["python", "benchmark.py"] },
      input: { hash: "<sha256-of-inputs>" },
      scoring: { mode: "standard" },
      constraints: { timeout_sec: 600, max_memory_mb: 4096 },
      regions: []
    },
    public_key: "<base58-or-hex-ed25519-pubkey>",
    signature: "<base64-ed25519-signature>"
  };
  if (kind === 'llama') {
    base.job.benchmark.name = "llama3.2-1b";
    base.job.container.image = "beacon/llm-benchmark-llama:latest";
    base.job.regions = ["US"];
  } else if (kind === 'qwen') {
    base.job.benchmark.name = "qwen2.5-1.5b";
    base.job.container.image = "beacon/llm-benchmark-qwen:latest";
    base.job.regions = ["CN"];
  } else if (kind === 'mistral') {
    base.job.benchmark.name = "mistral-7b";
    base.job.container.image = "beacon/llm-benchmark-mistral:latest";
    base.job.regions = ["EU"];
  }
  return JSON.stringify(base, null, 2);
}

function validateSpec(spec, validateAjv) {
  if (!spec || typeof spec !== 'object') return ['Spec must be a JSON object.'];
  const ok = validateAjv(spec);
  if (ok) return [];
  const errors = validateAjv.errors || [];
  return errors.map(e => {
    const path = (e.instancePath || e.schemaPath || '').replace(/^#\//, '') || 'root';
    const msg = e.message || 'is invalid';
    // Include expected/received values where helpful
    if (e.keyword === 'required' && e.params?.missingProperty) {
      return `${path}: missing required property '${e.params.missingProperty}'`;
    }
    if (e.keyword === 'type') {
      return `${path}: ${msg}`;
    }
    if (e.keyword === 'enum') {
      return `${path}: must be one of ${JSON.stringify(e.params.allowedValues)}`;
    }
    if (e.keyword === 'pattern') {
      return `${path}: ${msg}`;
    }
    return `${path}: ${msg}`;
  });
}

// JSON Schema for signed JobSpec (v0)
const jobSpecSchema = {
  type: 'object',
  required: ['job', 'public_key', 'signature'],
  additionalProperties: true,
  properties: {
    public_key: { type: 'string', minLength: 8 },
    signature: { type: 'string', minLength: 16 },
    job: {
      type: 'object',
      required: ['id', 'version', 'benchmark', 'container', 'regions', 'input'],
      additionalProperties: true,
      properties: {
        id: { type: 'string', minLength: 3 },
        version: { type: 'string', enum: ['v0'] },
        benchmark: {
          type: 'object',
          required: ['name'],
          properties: {
            name: { type: 'string', minLength: 1 },
            params: { type: 'object', additionalProperties: true }
          }
        },
        container: {
          type: 'object',
          required: ['image'],
          properties: {
            image: { type: 'string', minLength: 1 },
            command: {
              type: 'array', items: { type: 'string' }, minItems: 1
            }
          }
        },
        input: {
          type: 'object',
          required: ['hash'],
          properties: {
            hash: { type: 'string', minLength: 8 }
          }
        },
        scoring: {
          type: 'object',
          additionalProperties: true,
          properties: {
            mode: { type: 'string' }
          }
        },
        constraints: {
          type: 'object',
          additionalProperties: true,
          properties: {
            timeout_sec: { type: 'number' },
            max_memory_mb: { type: 'number' }
          }
        },
        regions: { type: 'array', items: { type: 'string' }, minItems: 1 }
      }
    }
  }
};
