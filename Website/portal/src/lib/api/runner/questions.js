import { runnerFetch } from '../http.js';

export async function getQuestions() {
  const data = await runnerFetch('/questions');
  if (data && data.categories) return data;
  if (Array.isArray(data)) {
    const grouped = {};
    for (const q of data) {
      const cat = q.category || 'uncategorized';
      if (!grouped[cat]) grouped[cat] = [];
      grouped[cat].push({ question_id: q.question_id, question: q.question });
    }
    return { categories: grouped };
  }
  return { categories: {} };
}
