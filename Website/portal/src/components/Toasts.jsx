import React from 'react';
import { useToast } from '../state/toast.jsx';

export default function Toasts() {
  const { toasts, remove } = useToast();
  return (
    <div className="fixed top-4 right-4 z-50 space-y-2 w-80">
      {toasts.map(t => (
        <div key={t.id} className="bg-white border shadow rounded p-3 text-sm">
          <div className="flex items-start justify-between gap-2">
            <div>
              {t.title && <div className="font-medium mb-0.5">{t.title}</div>}
              <div className="text-slate-700">{t.message}</div>
            </div>
            <button className="text-slate-400 hover:text-slate-600" onClick={() => remove(t.id)}>✕</button>
          </div>
          {t.action}
        </div>
      ))}
    </div>
  );
}
