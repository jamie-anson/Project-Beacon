/**
 * Utilities for encoding/decoding question text to/from URL-friendly IDs
 * Uses hyphens instead of URL encoding for cleaner URLs
 */

/**
 * Encodes a question text into a URL-friendly ID
 * Replaces spaces with hyphens and removes special characters
 * @param {string} questionText - The question text
 * @returns {string} URL-friendly question ID
 */
export function encodeQuestionId(questionText) {
  if (!questionText) return '';
  
  return questionText
    .toLowerCase()
    .trim()
    // Replace underscores with hyphens first (for database IDs like "identity_basic")
    .replace(/_/g, '-')
    // Replace spaces with hyphens
    .replace(/\s+/g, '-')
    // Remove special characters except hyphens
    .replace(/[^a-z0-9-]/g, '')
    // Remove multiple consecutive hyphens
    .replace(/-+/g, '-')
    // Remove leading/trailing hyphens
    .replace(/^-+|-+$/g, '');
}

/**
 * Decodes a question ID back to the original text
 * Note: This is lossy - we can't perfectly reconstruct the original
 * So we'll need to match against available questions
 * @param {string} questionId - The URL-friendly question ID
 * @returns {string} Decoded question text (with hyphens as spaces)
 */
export function decodeQuestionId(questionId) {
  if (!questionId) return '';
  
  // For now, just replace hyphens with spaces and capitalize
  // In production, you'd match against actual question list
  return questionId
    .replace(/-/g, ' ')
    .split(' ')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

/**
 * Finds the best matching question from a list based on the question ID
 * @param {string} questionId - The URL-friendly question ID
 * @param {Array<string>} availableQuestions - List of available question texts
 * @returns {string|null} The matching question text or null
 */
export function matchQuestionFromId(questionId, availableQuestions) {
  if (!questionId || !availableQuestions?.length) return null;
  
  // Normalize the question ID
  const normalizedId = questionId.toLowerCase().replace(/-/g, ' ');
  
  // Find exact match first
  const exactMatch = availableQuestions.find(q => 
    q.toLowerCase().replace(/[^a-z0-9\s]/g, '').replace(/\s+/g, ' ').trim() === normalizedId
  );
  
  if (exactMatch) return exactMatch;
  
  // Find partial match (contains all words)
  const words = normalizedId.split(' ').filter(w => w.length > 2);
  const partialMatch = availableQuestions.find(q => {
    const qLower = q.toLowerCase();
    return words.every(word => qLower.includes(word));
  });
  
  return partialMatch || null;
}
