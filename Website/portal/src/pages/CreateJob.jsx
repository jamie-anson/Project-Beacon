import React from 'react';
import { useNavigate } from 'react-router-dom';
import { createJob } from '../lib/api.js';

export default function CreateJob() {
  const [text, setText] = React.useState('');
  const [error, setError] = React.useState(null);
  const [submitting, setSubmitting] = React.useState(false);
  const navigate = useNavigate();

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
        <div className="flex items-center gap-2 text-sm">
          <input type="file" accept="application/json,.json" onChange={onFile} />
          <button type="button" className="px-2 py-1 border rounded" onClick={() => setText('')}>Clear</button>
        </div>
        <textarea
          className="w-full h-64 border rounded p-2 font-mono text-xs"
          placeholder={placeholderSample}
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
        {error && <div className="text-sm text-red-600">{error}</div>}
        <div className="flex items-center gap-2">
          <button type="submit" className="px-3 py-1.5 rounded bg-beacon-600 text-white disabled:opacity-50" disabled={submitting}>Submit</button>
          {submitting && <span className="text-sm text-slate-500">Submitting…</span>}
        </div>
      </form>
    </div>
  );
}
