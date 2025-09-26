import React from 'react';
import { getTransparencyProof } from '../lib/api/runner/transparency.js';

export default function ProofViewer({ cid, executionId }) {
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState(null);
  const [proof, setProof] = React.useState(null);

  const load = React.useCallback(async () => {
    if (!cid && !executionId) return;
    setLoading(true);
    setError(null);
    try {
      const p = await getTransparencyProof({ ipfs_cid: cid, execution_id: executionId });
      setProof(p);
    } catch (e) {
      setError(e);
    } finally {
      setLoading(false);
    }
  }, [cid, executionId]);

  React.useEffect(() => { load(); }, [load]);

  return (
    <div className="space-y-3 text-sm">
      <div className="flex items-center justify-between">
        <div className="text-slate-600">
          {cid && (<span>CID: <span className="font-mono">{cid}</span></span>)}
          {executionId && (<span className="ml-3">Exec: <span className="font-mono">{executionId}</span></span>)}
        </div>
        <div className="flex items-center gap-2">
          <button onClick={load} className="text-xs px-2 py-1 border rounded">Refresh</button>
        </div>
      </div>

      {loading && (
        <div className="animate-pulse">
          <div className="h-4 bg-slate-200 rounded w-1/2"></div>
          <div className="h-4 bg-slate-100 rounded w-5/6 mt-2"></div>
        </div>
      )}

      {error && (
        <div className="text-red-600">{String(error?.message || error)}</div>
      )}

      {proof && (
        <div className="space-y-2">
          <div>
            <span className="text-slate-600">Merkle root:</span> <span className="font-mono break-all">{proof.merkle_root || proof.root || '—'}</span>
          </div>
          {proof.sequence != null && (
            <div className="text-xs text-slate-500">Seq #{proof.sequence}</div>
          )}
          <pre className="bg-slate-100 p-3 rounded text-xs overflow-auto">{JSON.stringify(proof, null, 2)}</pre>
        </div>
      )}

      {!loading && !error && !proof && (
        <div className="text-slate-500">No proof found.</div>
      )}
    </div>
  );
}
