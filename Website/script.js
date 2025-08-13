// Project Beacon — minimal interactivity
(function () {
  const yearEl = document.getElementById('year');
  if (yearEl) yearEl.textContent = new Date().getFullYear();

  // Respect reduced motion
  const prefersReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  if (!prefersReduced) {
    // Smooth scroll for same-page links (fallback for browsers ignoring CSS smooth-behavior)
    document.querySelectorAll('a[href^="#"]').forEach((a) => {
      a.addEventListener('click', (e) => {
        const targetId = a.getAttribute('href');
        if (targetId.length > 1) {
          const el = document.querySelector(targetId);
          if (el) {
            e.preventDefault();
            el.scrollIntoView({ behavior: 'smooth', block: 'start' });
            history.pushState(null, '', targetId);
          }
        }
      });
    });
  }
})();
