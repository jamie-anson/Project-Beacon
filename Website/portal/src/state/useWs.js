import { useCallback, useEffect, useRef, useState, useMemo } from 'react';

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

  function isTruthy(v) {
    try { return /^(1|true|yes|on)$/i.test(String(v || '')); } catch { return false; }
  }

  // Memoize wsEnabled to prevent infinite loops
  const wsEnabled = useMemo(() => {
    // Explicit opt-in required to prevent runaway connection attempts
    try {
      const lsVal = localStorage.getItem('beacon:enable_ws');
      if (lsVal != null) return isTruthy(lsVal);
    } catch {}
    try {
      const envVal = import.meta?.env?.VITE_ENABLE_WS;
      if (envVal != null) return isTruthy(envVal);
    } catch {}
    // Default: disabled to prevent console spam and connection issues
    return false;
  }, []); // Empty deps - only calculate once

  const connect = useCallback(() => {
    // Feature flag: allow enabling via env or localStorage
    if (!wsEnabled) {
      try {
        const envVal = import.meta?.env?.VITE_ENABLE_WS;
        const lsVal = (()=>{ try{ return localStorage.getItem('beacon:enable_ws'); }catch{return null;} })();
        console.log('WebSocket disabled by config (set VITE_ENABLE_WS=1 or localStorage beacon:enable_ws=true to enable)', { envVal, lsVal });
      } catch {
        console.log('WebSocket disabled by config (set VITE_ENABLE_WS=1 or localStorage beacon:enable_ws=true to enable)');
      }
      setConnected(false);
      setError(new Error('WebSocket disabled by config'));
      return;
    }
    
    // Use environment variable for WebSocket base, allow runtime override, fallback to same-origin
    let wsBase = import.meta.env?.VITE_WS_BASE;
    try {
      const overrideBase = localStorage.getItem('beacon:ws_base');
      if (overrideBase && overrideBase.trim()) {
        wsBase = overrideBase.trim();
      }
    } catch {}
    // Normalize scheme if missing or using http(s)
    if (wsBase && !/^wss?:\/\//i.test(wsBase)) {
      if (/^https?:\/\//i.test(wsBase)) {
        wsBase = wsBase.replace(/^http/i, 'ws');
      } else if (/^[a-z0-9.-]+(:\d+)?$/i.test(wsBase)) {
        // host[:port] only
        wsBase = (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + wsBase;
      }
    }
    if (!wsBase || wsBase.trim() === '') {
      // Use same-origin WebSocket (Netlify proxies to Railway hybrid router)
      wsBase = window.location.protocol === 'https:' ? 'wss://' + window.location.host : 'ws://' + window.location.host;
    }
    try { console.info('[Beacon] Using WebSocket base:', wsBase); } catch {}
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
        if (!closedRef.current && retryRef.current < 5) {
          // Cap retries at 5 attempts to prevent runaway reconnections
          const delay = Math.min(30000, 1000 * Math.pow(2, retryRef.current++));
          setRetries(retryRef.current);
          setNextDelayMs(delay);
          setTimeout(connect, delay);
        } else if (retryRef.current >= 5) {
          console.warn('WebSocket max retries reached (5), stopping reconnection attempts');
          setError(new Error('WebSocket connection failed after 5 retries'));
        }
      };
      ws.onerror = (e) => {
        if (retryRef.current === 0) {
          console.warn('WebSocket connection failed - backend may be offline or WebSocket not supported');
        }
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
  }, [path, onMessage, wsEnabled]);

  useEffect(() => {
    if (!wsEnabled) {
      setConnected(false);
      setError(new Error('WebSocket disabled by config'));
      return;
    }
    connect();
    return () => {
      closedRef.current = true;
      try { wsRef.current && wsRef.current.close(); } catch {}
    };
  }, [connect, wsEnabled]);

  return { connected, error, socket: wsRef.current, retries, nextDelayMs };
}
