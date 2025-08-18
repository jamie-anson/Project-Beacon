import { useEffect, useRef, useState } from 'react';

export default function useWs(path = '/ws') {
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState(null);
  const wsRef = useRef(null);

  useEffect(() => {
    const proto = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const url = `${proto}://${window.location.host}${path}`;

    let ws;
    try {
      ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => setConnected(true);
      ws.onclose = () => setConnected(false);
      ws.onerror = (e) => setError(e);
    } catch (e) {
      setError(e);
    }

    return () => {
      try { ws && ws.close(); } catch {}
    };
  }, [path]);

  return { connected, error, socket: wsRef.current };
}
