import React from 'react';
import { useToast } from '../state/toast.jsx';

export default function Toasts() {
  const { toasts, remove } = useToast();
  
  const getToastStyles = (type) => {
    switch (type) {
      case 'error':
        return 'bg-red-900/20 border-red-700 text-red-400';
      case 'success':
        return 'bg-green-900/20 border-green-700 text-green-400';
      case 'warning':
        return 'bg-yellow-900/20 border-yellow-700 text-yellow-400';
      default:
        return 'bg-gray-800 border-gray-700 text-gray-300';
    }
  };

  const getIconForType = (type) => {
    switch (type) {
      case 'error':
        return (
          <svg className="h-5 w-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        );
      case 'success':
        return (
          <svg className="h-5 w-5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        );
      case 'warning':
        return (
          <svg className="h-5 w-5 text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
          </svg>
        );
      default:
        return (
          <svg className="h-5 w-5 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        );
    }
  };

  return (
    <div className="fixed top-4 right-4 z-50 space-y-2 w-96">
      {toasts.map(t => (
        <div key={t.id} className={`border shadow-lg rounded-lg p-4 text-sm ${getToastStyles(t.type)}`}>
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0 mt-0.5">
              {getIconForType(t.type)}
            </div>
            <div className="flex-1 min-w-0">
              {t.title && <div className="font-semibold mb-1">{t.title}</div>}
              <div className="break-words">{t.message}</div>
              {t.details && (
                <div className="mt-2 text-xs opacity-75 break-words">{t.details}</div>
              )}
            </div>
            <button 
              className="flex-shrink-0 ml-2 text-current opacity-50 hover:opacity-75 transition-opacity" 
              onClick={() => remove(t.id)}
            >
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          {t.action && (
            <div className="mt-3 pt-2 border-t border-current border-opacity-20">
              {t.action}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
