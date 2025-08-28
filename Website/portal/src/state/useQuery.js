import { useEffect, useMemo, useRef, useState } from 'react';

export function useQuery(key, fn, { interval } = {}) {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const timer = useRef(null);

  // Stabilize the key across renders even if callers pass arrays/objects
  const stableKey = useMemo(() => {
    try {
      return typeof key === 'string' ? key : JSON.stringify(key);
    } catch {
      // Fallback to String() if non-serializable
      return String(key);
    }
  }, [key]);

  const load = async () => {
    try {
      setError(null);
      setLoading(true);
      const d = await fn();
      setData(d);
    } catch (e) {
      setError(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    if (interval) {
      timer.current = setInterval(load, interval);
      return () => clearInterval(timer.current);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stableKey, interval]);

  return { data, error, loading, reload: load, refetch: load };
}
