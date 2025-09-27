import { useEffect } from 'react';

/**
 * Custom hook to set the page title dynamically
 * @param {string} title - The title to set (without "Project Beacon • Portal" suffix)
 */
export function usePageTitle(title) {
  useEffect(() => {
    if (title) {
      document.title = `${title} • Project Beacon • Portal`;
    } else {
      document.title = 'Project Beacon • Portal';
    }

    // Cleanup function to reset title when component unmounts
    return () => {
      document.title = 'Project Beacon • Portal';
    };
  }, [title]);
}

export default usePageTitle;
