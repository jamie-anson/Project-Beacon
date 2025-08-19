import React from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App.jsx';
import { ToastProvider } from './state/toast.jsx';
import Toasts from './components/Toasts.jsx';
import './index.css';

createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <ToastProvider>
      <BrowserRouter basename="/portal">
        <App />
        <Toasts />
      </BrowserRouter>
    </ToastProvider>
  </React.StrictMode>
);
