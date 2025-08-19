import React from 'react';

export default function Modal({ open, onClose, title, children, maxWidth = 'max-w-2xl' }) {
  React.useEffect(() => {
    if (!open) return;
    const onKey = (e) => { if (e.key === 'Escape') onClose?.(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} />
      <div className="absolute inset-0 flex items-center justify-center p-4">
        <div className={`w-full ${maxWidth} bg-white border rounded shadow-lg`}
          role="dialog" aria-modal="true">
          <div className="flex items-center justify-between px-4 py-3 border-b">
            <h3 className="font-semibold">{title}</h3>
            <button className="text-slate-500 hover:text-slate-900" onClick={onClose} aria-label="Close">✕</button>
          </div>
          <div className="p-4 max-h-[70vh] overflow-auto">
            {children}
          </div>
        </div>
      </div>
    </div>
  );
}
