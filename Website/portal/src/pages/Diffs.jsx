import React from 'react';
import { useQuery } from '../state/useQuery.js';
import { getDiffs } from '../lib/api.js';

export default function Diffs() {
  const { data } = useQuery('diffs', () => getDiffs({ limit: 50 }), { interval: 10000 });
  return (
    <div>
      <h2 className="text-xl font-semibold mb-3">Diffs</h2>
      <pre className="bg-slate-100 p-3 rounded text-xs overflow-auto">{JSON.stringify(data || [], null, 2)}</pre>
    </div>
  );
}
