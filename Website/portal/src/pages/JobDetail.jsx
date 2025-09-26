import React from 'react';
import { useParams } from 'react-router-dom';
import { useQuery } from '../state/useQuery.js';
import { getJob } from '../lib/api/runner/jobs.js';
import { getTransparencyProof, getTransparencyRoot } from '../lib/api/runner/transparency.js';
import { bundleUrl } from '../lib/api/ipfs.js';

export default function JobDetail() {
  const { id } = useParams();

  const [include, setInclude] = React.useState('latest'); // 'latest' | 'executions'
  const [execLimit, setExecLimit] = React.useState(10);
  const [execOffset, setExecOffset] = React.useState(0);

  const { data, loading, error, refetch } = useQuery(
    ['job', id, include, execLimit, execOffset],
    () => getJob({ id, include, exec_limit: include === 'executions' ? execLimit : undefined, exec_offset: include === 'executions' ? execOffset : undefined }),
    { interval: 15000 }
  );

  const job = data?.job;
  const executions = data?.executions || [];

  const [proofOpen, setProofOpen] = React.useState(false);
  const [selectedExec, setSelectedExec] = React.useState(null);
  const [proof, setProof] = React.useState(null);
  const [proofErr, setProofErr] = React.useState(null);
  const [root, setRoot] = React.useState(null);
  const [loadingProof, setLoadingProof] = React.useState(false);

  const openProof = async (exec) => {
    setSelectedExec(exec);
    setProof(null);
    setProofErr(null);
    setLoadingProof(true);
    setProofOpen(true);
    try {
      const [p, r] = await Promise.all([
        getTransparencyProof({ execution_id: exec.id || exec.execution_id, ipfs_cid: exec.ipfs_cid }),
        getTransparencyRoot(),
      ]);
      setProof(p);
      setRoot(r);
    } catch (e) {
      setProofErr(e);
    } finally {
      setLoadingProof(false);
    }
  };

  const includeOptions = [
    { value: 'latest', label: 'Latest receipt' },
    { value: 'executions', label: 'All executions' },
  ];

  return (
    <div className="space-y-6">
      <header className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Job</h2>
          <div className="text-xs text-slate-500 mt-1">
            <div><span className="text-slate-600">ID:</span> <code className="font-mono">{id}</code></div>
            <div><span className="text-slate-600">Status:</span> <span className="inline-block px-2 py-0.5 rounded bg-slate-100 text-slate-700">{data?.status || '—'}</span></div>
          </div>
        </div>
        <div className="flex items-center gap-2 text-sm">
          <label className="text-slate-600">Include</label>
          <select
            value={include}
            onChange={(e) => { setInclude(e.target.value); setExecOffset(0); refetch(); }}
            className="border rounded px-2 py-1"
          >
            {includeOptions.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
          {include === 'executions' && (
            <>
              <label className="text-slate-600">Limit</label>
              <select value={execLimit} onChange={(e) => { setExecLimit(Number(e.target.value)); setExecOffset(0); }} className="border rounded px-2 py-1">
                {[10,20,50,100,200].map(v => <option key={v} value={v}>{v}</option>)}
              </select>
              <button onClick={() => setExecOffset(Math.max(0, execOffset - execLimit))} className="px-2 py-1 border rounded disabled:opacity-50" disabled={execOffset === 0}>Prev</button>
              <button onClick={() => setExecOffset(execOffset + execLimit)} className="px-2 py-1 border rounded">Next</button>
            </>
          )}
          <button onClick={refetch} className="px-3 py-1.5 bg-green-600 text-white rounded hover:bg-green-700">Refresh</button>
        </div>
      </header>

      {loading && <div className="text-sm text-slate-500">Loading…</div>}
      {error && <div className="text-sm text-red-600">{String(error.message || error)}</div>}

      {job && (
        <section>
          <h3 className="font-medium">JobSpec</h3>
          <pre className="bg-gray-800 text-gray-100 p-3 rounded text-xs overflow-auto border border-gray-600">{JSON.stringify(job, null, 2)}</pre>
        </section>
      )}

      <section>
        <h3 className="font-medium">{include === 'latest' ? 'Latest receipt' : 'Executions'}</h3>
        {executions.length === 0 ? (
          <div className="text-sm text-slate-500">No executions yet.</div>
        ) : (
          <div className="overflow-auto">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="text-left text-slate-600">
                  <th className="px-2 py-2">Execution ID</th>
                  <th className="px-2 py-2">IPFS CID</th>
                  <th className="px-2 py-2">Region</th>
                  <th className="px-2 py-2">Created</th>
                  <th className="px-2 py-2"></th>
                </tr>
              </thead>
              <tbody>
                {executions.map((e) => (
                  <tr key={(e.id || e.execution_id)} className="border-t">
                    <td className="px-2 py-2 font-mono">{e.id || e.execution_id}</td>
                    <td className="px-2 py-2 font-mono">
                      {e.ipfs_cid ? (
                        <a className="text-beacon-600 underline decoration-dotted" href={bundleUrl(e.ipfs_cid)} target="_blank" rel="noreferrer">{e.ipfs_cid}</a>
                      ) : '—'}
                    </td>
                    <td className="px-2 py-2">{e.region || '—'}</td>
                    <td className="px-2 py-2">{e.created_at || e.timestamp || '—'}</td>
                    <td className="px-2 py-2 text-right">
                      {e.ipfs_cid && (
                        <button onClick={() => openProof(e)} className="px-3 py-1.5 bg-blue-600 text-white rounded hover:bg-blue-700">View proof</button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {proofOpen && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center p-4 z-50" onClick={() => setProofOpen(false)}>
          <div className="bg-white rounded shadow-xl max-w-3xl w-full" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between border-b px-4 py-2">
              <div className="font-medium">Merkle Proof</div>
              <button className="px-2 py-1" onClick={() => setProofOpen(false)}>Close</button>
            </div>
            <div className="p-4 space-y-3">
              {loadingProof && <div className="text-sm text-slate-500">Fetching proof…</div>}
              {proofErr && <div className="text-sm text-red-600">{String(proofErr.message || proofErr)}</div>}
              {root && proof && (
                <div className="text-sm">
                  <div className="mb-2">
                    Transparency Root: <code className="font-mono">{root.root}</code>
                  </div>
                  <div className="mb-2">
                    Proof Merkle Root: <code className="font-mono">{proof.merkle_root || '(see payload)'}
                    </code>
                    {proof.merkle_root && root.root && (
                      <span className={`ml-2 inline-block px-2 py-0.5 rounded text-white ${proof.merkle_root === root.root ? 'bg-green-600' : 'bg-red-600'}`}>
                        {proof.merkle_root === root.root ? 'verified' : 'mismatch'}
                      </span>
                    )}
                  </div>
                </div>
              )}
              {proof && (
                <pre className="bg-slate-100 p-3 rounded text-xs overflow-auto max-h-80">{JSON.stringify(proof, null, 2)}</pre>
              )}
              {!loadingProof && !proof && !proofErr && (
                <div className="text-sm text-slate-500">No proof data.</div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
