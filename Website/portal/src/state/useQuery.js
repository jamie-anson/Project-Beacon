import { useEffect, useRef, useState } from 'react';

export function useQuery(key, fn, { interval } = {}) {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const timer = useRef(null);

  const load = async () => {
    try {
      setError(null);
      const d = await fn();
      setData(d);
    } catch (e) {
      setError(e);
    }
  };

  useEffect(() => {
    load();
    if (interval) {
      timer.current = setInterval(load, interval);
      return () => clearInterval(timer.current);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, interval]);

  return { data, error, reload: load };
}
