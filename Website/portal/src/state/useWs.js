import { useCallback, useEffect, useRef, useState } from 'react';

export default function useWs(path = '/ws', opts = {}) {
  const onMessage = opts.onMessage; // function(eventObj)
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState(null);
  const [retries, setRetries] = useState(0);
  const [nextDelayMs, setNextDelayMs] = useState(0);
  const wsRef = useRef(null);
  const retryRef = useRef(0);
  const bufferRef = useRef('');
  const closedRef = useRef(false);

  const connect = useCallback(() => {
    // Temporarily disable WebSocket connections until backend support is ready
    console.log('WebSocket disabled - backend endpoint not available');
    setConnected(false);
    setError(new Error('WebSocket temporarily disabled'));
    return;
    
    // Use environment variable for WebSocket base, fallback to same-origin proxy
    let wsBase = import.meta.env?.VITE_WS_BASE;
    if (!wsBase || wsBase.trim() === '') {
      // Use same-origin WebSocket (Netlify proxies to Railway hybrid router)
      wsBase = window.location.protocol === 'https:' ? 'wss://' + window.location.host : 'ws://' + window.location.host;
    }
    const url = `${wsBase}${path.startsWith('/') ? path : '/' + path}`;
    let ws;
    try {
      ws = new WebSocket(url);
      wsRef.current = ws;
      closedRef.current = false;

      ws.onopen = () => {
        setConnected(true);
        retryRef.current = 0;
        setRetries(0);
        setNextDelayMs(0);
      };
      ws.onclose = () => {
        setConnected(false);
        if (!closedRef.current) {
          const delay = Math.min(30000, 1000 * Math.pow(2, retryRef.current++));
          setRetries(retryRef.current);
          setNextDelayMs(delay);
          setTimeout(connect, delay);
        }
      };
      ws.onerror = (e) => {
        console.warn('WebSocket connection failed - backend may be offline');
        setError(e);
      };
      ws.onmessage = (evt) => {
        const chunk = String(evt.data || '');
        // Handle newline-delimited JSON frames
        const combined = bufferRef.current + chunk;
        const parts = combined.split(/\n+/);
        bufferRef.current = parts.pop() || '';
        for (const p of parts) {
          const s = p.trim();
          if (!s) continue;
          try {
            const obj = JSON.parse(s);
            onMessage && onMessage(obj);
          } catch (_) {
            // ignore malformed frames
          }
        }
      };
    } catch (e) {
      setError(e);
    }
  }, [path, onMessage]);

  useEffect(() => {
    connect();
    return () => {
      closedRef.current = true;
      try { wsRef.current && wsRef.current.close(); } catch {}
    };
  }, [connect]);

  return { connected, error, socket: wsRef.current, retries, nextDelayMs };
}
