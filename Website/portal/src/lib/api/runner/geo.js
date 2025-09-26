import { runnerFetch } from '../http.js';

export async function getGeo() {
  const data = await runnerFetch('/geo');
  if (data && data.countries) return data;
  return { countries: {} };
}
