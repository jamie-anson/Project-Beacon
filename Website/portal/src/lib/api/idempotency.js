function getTabStorage() {
  return sessionStorage;
}

export function getTabId() {
  try {
    const storage = getTabStorage();
    let id = storage.getItem('beacon:tab_id');
    if (!id) {
      id = `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
      storage.setItem('beacon:tab_id', id);
    }
    return id;
  } catch {
    return 'tab-unknown';
  }
}

function shortHash(str) {
  let h = 0;
  for (let i = 0; i < str.length; i++) {
    h = ((h << 5) - h) + str.charCodeAt(i) | 0;
  }
  return (h >>> 0).toString(36);
}

export function computeIdempotencyKey(jobspec, windowSeconds = 60) {
  const tab = getTabId();
  const bucket = Math.floor(Date.now() / 1000 / windowSeconds);
  let specStr = '';
  try {
    specStr = JSON.stringify(jobspec);
  } catch {
    specStr = String(jobspec || '');
  }
  const base = `${tab}:${bucket}:${specStr}`;
  return `beacon-${shortHash(base)}`;
}

function isTruthy(v) {
  try {
    return /^(1|true|yes|on)$/i.test(String(v || ''));
  } catch {
    return false;
  }
}

export function shouldSendIdempotency() {
  try {
    const envVal = import.meta?.env?.VITE_ENABLE_IDEMPOTENCY;
    if (envVal != null) return isTruthy(envVal);
  } catch {}

  try {
    const lsVal = localStorage.getItem('beacon:enable_idempotency');
    if (lsVal != null) return isTruthy(lsVal);
  } catch {}

  return false;
}
