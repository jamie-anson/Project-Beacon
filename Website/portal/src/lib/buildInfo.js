export const BUILD_CID = import.meta.env.VITE_DOCS_CID || 'unknown';
export const BUILD_COMMIT = import.meta.env.VITE_BUILD_COMMIT || 'unknown';

export function shortCommit(commit = BUILD_COMMIT) {
  return commit && commit.length > 8 ? commit.slice(0, 8) : commit;
}
