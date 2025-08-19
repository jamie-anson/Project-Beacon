import React from 'react';
import { getIpfsGateway } from '../lib/api.js';
import { useToast } from '../state/toast.jsx';

export default function Settings() {
  const { add: addToast } = useToast();
  const [gateway, setGateway] = React.useState(() => getIpfsGateway() || '');
  const [envGateway, setEnvGateway] = React.useState(() => (import.meta?.env?.VITE_IPFS_GATEWAY || '') );

  const save = (e) => {
    e.preventDefault();
    try {
      const trimmed = (gateway || '').trim();
      if (trimmed) localStorage.setItem('beacon:ipfs_gateway', trimmed.replace(/\/$/, ''));
      else localStorage.removeItem('beacon:ipfs_gateway');
      addToast({ title: 'Settings saved', message: trimmed ? `Gateway set to ${trimmed}` : 'Gateway reset to default' });
    } catch (err) {
      addToast({ title: 'Save failed', message: String(err) });
    }
  };

  const reset = () => {
    try {
      localStorage.removeItem('beacon:ipfs_gateway');
      const effective = getIpfsGateway() || '';
      setGateway(effective);
      addToast({ title: 'Gateway reset', message: effective ? `Using ${effective}` : 'Using local API gateway' });
    } catch (err) {
      addToast({ title: 'Reset failed', message: String(err) });
    }
  };

  return (
    <div className="space-y-6">
      <section>
        <h2 className="text-xl font-semibold">Settings</h2>
        <p className="text-sm text-slate-600">Configure runtime options for the Portal.</p>
      </section>

      <section className="bg-white border rounded p-4 space-y-3">
        <h3 className="font-medium">IPFS Gateway</h3>
        <form onSubmit={save} className="space-y-3">
          <div className="text-sm text-slate-600">
            Effective gateway used for links to CIDs. If empty, the app will use <code className="font-mono">VITE_IPFS_GATEWAY</code> if set, otherwise fallback to the local API gateway.
          </div>
          <div className="flex items-center gap-2">
            <input
              className="border rounded px-2 py-1 text-sm w-full"
              type="url"
              placeholder="https://ipfs.io"
              value={gateway}
              onChange={(e) => setGateway(e.target.value)}
            />
            <button type="submit" className="text-sm px-3 py-1.5 border rounded bg-beacon-600 text-white">Save</button>
            <button type="button" className="text-sm px-3 py-1.5 border rounded" onClick={reset}>Reset</button>
          </div>
          <div className="text-xs text-slate-500">
            Env default: {envGateway ? <code className="font-mono">{envGateway}</code> : 'not set'}
          </div>
        </form>
      </section>
    </div>
  );
}
