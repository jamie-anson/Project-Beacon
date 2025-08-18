import React from 'react';
import { useQuery } from '../state/useQuery.js';
import { getExecutions } from '../lib/api.js';

export default function Executions() {
  const { data } = useQuery('executions', () => getExecutions({ limit: 50 }), { interval: 5000 });
  return (
    <div>
      <h2 className="text-xl font-semibold mb-3">Executions</h2>
      <pre className="bg-slate-100 p-3 rounded text-xs overflow-auto">{JSON.stringify(data || [], null, 2)}</pre>
    </div>
  );
}
