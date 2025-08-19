import React, { createContext, useContext, useMemo, useState, useCallback } from 'react';

const ToastCtx = createContext(null);

export function ToastProvider({ children }) {
  const [toasts, setToasts] = useState([]);

  const remove = useCallback((id) => setToasts((ts) => ts.filter((t) => t.id !== id)), []);

  const add = useCallback((toast) => {
    const id = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    const t = { id, timeout: 5000, ...toast };
    setToasts((ts) => [t, ...ts].slice(0, 5));
    if (t.timeout) setTimeout(() => remove(id), t.timeout);
    return id;
  }, [remove]);

  const value = useMemo(() => ({ toasts, add, remove }), [toasts, add, remove]);
  return (
    <ToastCtx.Provider value={value}>{children}</ToastCtx.Provider>
  );
}

export function useToast() {
  const ctx = useContext(ToastCtx);
  if (!ctx) throw new Error('useToast must be used within ToastProvider');
  return ctx;
}
