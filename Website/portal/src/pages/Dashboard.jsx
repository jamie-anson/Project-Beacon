import React from 'react';
import { useQuery } from '../state/useQuery.js';
import { getHealth, getExecutions, getDiffs } from '../lib/api.js';

export default function Dashboard() {
  const { data: health } = useQuery('health', getHealth, { interval: 30000 });
  const { data: executions } = useQuery('executions:latest', () => getExecutions({ limit: 5 }), { interval: 15000 });
  const { data: diffs } = useQuery('diffs:latest', () => getDiffs({ limit: 5 }), { interval: 20000 });

  return (
    <div className="space-y-6">
      <section>
        <h2 className="text-xl font-semibold">System status</h2>
        <pre className="bg-slate-100 p-3 rounded text-xs overflow-auto">{JSON.stringify(health || {}, null, 2)}</pre>
      </section>
      <section>
        <h2 className="text-xl font-semibold">Recent executions</h2>
        <pre className="bg-slate-100 p-3 rounded text-xs overflow-auto">{JSON.stringify(executions || [], null, 2)}</pre>
      </section>
      <section>
        <h2 className="text-xl font-semibold">Recent diffs</h2>
        <pre className="bg-slate-100 p-3 rounded text-xs overflow-auto">{JSON.stringify(diffs || [], null, 2)}</pre>
      </section>
    </div>
  );
}
