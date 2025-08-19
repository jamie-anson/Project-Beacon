import React from 'react';
import Modal from './Modal.jsx';
import ProofViewer from './ProofViewer.jsx';
import { bundleUrl } from '../lib/api.js';
import CopyButton from './CopyButton.jsx';

export default function ActivityFeed({ events = [] }) {
  const [open, setOpen] = React.useState(false);
  const [sel, setSel] = React.useState({ cid: null, exec: null });
  const openProof = (cid, exec) => { setSel({ cid, exec }); setOpen(true); };
  const close = () => setOpen(false);

  if (!events.length) return (
    <div className="text-sm text-slate-500">No recent activity.</div>
  );

  return (
    <div className="bg-white border rounded divide-y">
      <ul className="space-y-2 text-sm">
        {events.map((ev, idx) => (
          <li key={idx} className="p-3 border rounded">
            <div className="flex items-center justify-between">
              <div className="font-medium">Transparency log updated</div>
              <div className="text-xs text-slate-500">{ev.timestamp || new Date().toISOString()}</div>
            </div>
            <div className="mt-1 grid grid-cols-1 md:grid-cols-2 gap-1 text-xs">
              {ev.execution_id && (
                <div>
                  Exec ID: <span className="font-mono">{ev.execution_id}</span>
                </div>
              )}
              {ev.ipfs_cid && (
                <div className="flex items-center gap-2">
                  <span>
                    CID: <a className="font-mono text-beacon-600 underline decoration-dotted" href={bundleUrl(ev.ipfs_cid)} target="_blank" rel="noreferrer">{ev.ipfs_cid}</a>
                  </span>
                  <a
                    className="text-xs px-2 py-0.5 border rounded"
                    href={bundleUrl(ev.ipfs_cid)}
                    target="_blank"
                    rel="noreferrer"
                    title="Open in gateway"
                  >Open</a>
                  <button
                    className="text-xs px-2 py-0.5 border rounded"
                    onClick={() => openProof(ev.ipfs_cid, ev.execution_id)}
                    title="View transparency proof"
                  >View proof</button>
                  <CopyButton text={ev.ipfs_cid} label="Copy CID" />
                </div>
              )}
              {ev.merkle_root && (
                <div className="md:col-span-2 flex items-center gap-2">
                  <span>Root: <span className="font-mono">{ev.merkle_root}</span></span>
                  <CopyButton text={ev.merkle_root} label="Copy root" />
                </div>
              )}
            </div>
          </li>
        ))}
      </ul>
      <Modal open={open} onClose={close} title="Transparency proof">
        <ProofViewer cid={sel.cid} executionId={sel.exec} />
      </Modal>
    </div>
  );
}
