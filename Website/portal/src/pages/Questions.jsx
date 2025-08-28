import React from 'react';
import { getQuestions } from '../lib/api.js';
import { useQuery } from '../state/useQuery.js';

export default function Questions() {
  const { data, loading, error, refetch } = useQuery('questions:list', getQuestions, { interval: 30000 });
  const categories = data?.categories || {};
  const catNames = Object.keys(categories).sort();

  // Persist selection in localStorage
  const STORAGE_KEY = 'beacon:selected_questions';
  const [selected, setSelected] = React.useState(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const arr = JSON.parse(raw);
        if (Array.isArray(arr)) return new Set(arr);
      }
    } catch {}
    return new Set();
  });

  // Initialize default selection (all) when data first arrives and nothing selected yet
  React.useEffect(() => {
    if (!loading && !error) {
      const allIds = [];
      for (const cat of Object.keys(categories)) {
        for (const q of categories[cat]) allIds.push(q.question_id);
      }
      if (selected.size === 0 && allIds.length > 0) {
        setSelected(new Set(allIds));
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loading, error, data]);

  // Persist on change
  React.useEffect(() => {
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(Array.from(selected))); } catch {}
  }, [selected]);

  const totalCount = React.useMemo(() => {
    let c = 0; for (const cat of Object.keys(categories)) c += categories[cat].length; return c;
  }, [categories]);
  const selectedCount = selected.size;

  const toggleOne = (id) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const setAll = (on) => {
    if (on) {
      const all = new Set();
      for (const cat of Object.keys(categories)) for (const q of categories[cat]) all.add(q.question_id);
      setSelected(all);
    } else {
      setSelected(new Set());
    }
  };

  const categoryAllChecked = (cat) => {
    const items = categories[cat] || [];
    if (items.length === 0) return false;
    return items.every((q) => selected.has(q.question_id));
  };
  const categorySomeChecked = (cat) => {
    const items = categories[cat] || [];
    return items.some((q) => selected.has(q.question_id));
  };

  const toggleCategory = (cat) => {
    const items = categories[cat] || [];
    const allOn = items.every((q) => selected.has(q.question_id));
    setSelected((prev) => {
      const next = new Set(prev);
      if (allOn) {
        for (const q of items) next.delete(q.question_id);
      } else {
        for (const q of items) next.add(q.question_id);
      }
      return next;
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">Questions</h2>
        <div className="flex items-center gap-2 text-sm">
          <button onClick={refetch} className="px-2 py-1 border rounded hover:bg-slate-50">Refresh</button>
        </div>
      </div>

      {/* Intro copy */}
      <div className="bg-white border rounded p-3 text-sm text-slate-700">
        <p className="mb-2">
          All questions are selected by default. Unchecking questions reduces run time and cost.
          Start with a smaller set for a quick signal, then re-run with the full set for comprehensive coverage.
        </p>
        <div className="flex items-center gap-2">
          <button onClick={() => setAll(true)} className="px-2 py-1 border rounded hover:bg-slate-50">Select all</button>
          <button onClick={() => setAll(false)} className="px-2 py-1 border rounded hover:bg-slate-50">Clear all</button>
          <span className="text-xs text-slate-500">{selectedCount}/{totalCount} selected</span>
        </div>
      </div>

      {loading && <div className="text-sm text-slate-500">Loading questions…</div>}
      {error && <div className="text-sm text-red-600">Failed to load questions.</div>}

      {!loading && !error && catNames.length === 0 && (
        <div className="text-sm text-slate-500">No questions found.</div>
      )}

      {!loading && !error && catNames.map((cat) => (
        <section key={cat} className="bg-white border rounded">
          <div className="px-3 py-2 border-b flex items-center justify-between">
            <div className="flex items-center gap-2">
              <input
                id={`cat-${cat}`}
                type="checkbox"
                checked={categoryAllChecked(cat)}
                ref={(el) => { if (el) el.indeterminate = !categoryAllChecked(cat) && categorySomeChecked(cat); }}
                onChange={() => toggleCategory(cat)}
              />
              <label htmlFor={`cat-${cat}`} className="font-medium capitalize cursor-pointer">{cat.replaceAll('_', ' ')}</label>
            </div>
            <span className="text-xs text-slate-500">{categories[cat].filter((q) => selected.has(q.question_id)).length}/{categories[cat].length} selected</span>
          </div>
          <ul className="divide-y">
            {categories[cat].map((q) => (
              <li key={q.question_id} className="px-3 py-2 text-sm flex items-start gap-3">
                <input
                  id={`q-${q.question_id}`}
                  type="checkbox"
                  className="mt-0.5"
                  checked={selected.has(q.question_id)}
                  onChange={() => toggleOne(q.question_id)}
                />
                <label htmlFor={`q-${q.question_id}`} className="flex-1 cursor-pointer" title={q.question_id}>
                  <span className="flex-1">{q.question}</span>
                </label>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
