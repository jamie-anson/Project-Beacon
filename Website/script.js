// Project Beacon — minimal interactivity
(function () {
  const yearEl = document.getElementById('year');
  if (yearEl) yearEl.textContent = new Date().getFullYear();

  // Theme: initialize from storage or system preference, then wire toggle
  (function initTheme() {
    const root = document.documentElement;
    const btn = document.getElementById('theme-toggle');

    const stored = localStorage.getItem('theme');
    let theme = stored || 'light';

    function apply(t) {
      root.setAttribute('data-theme', t);
      // Update button UI
      if (btn) {
        if (t === 'dark') {
          btn.textContent = '🌙 Dark';
          btn.setAttribute('aria-pressed', 'true');
          btn.title = 'Switch to Light';
        } else {
          btn.textContent = '☀️ Light';
          btn.setAttribute('aria-pressed', 'false');
          btn.title = 'Switch to Dark';
        }
      }
    }

    apply(theme);

    if (btn) {
      btn.addEventListener('click', () => {
        theme = (root.getAttribute('data-theme') === 'dark') ? 'light' : 'dark';
        localStorage.setItem('theme', theme);
        apply(theme);
      });
    }

    // We do not auto-follow system changes by default. User choice persists.
  })();

  // Respect reduced motion
  const prefersReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  // Mark active nav link (for underline state)
  const currentPath = location.pathname.split('/').pop() || 'index.html';
  document.querySelectorAll('.primary-nav .nav-link').forEach((link) => {
    const href = link.getAttribute('href');
    if (href && (href === currentPath || (href === 'index.html' && currentPath === ''))) {
      link.setAttribute('aria-current', 'page');
    }
  });

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

    // Reveal sections on scroll (slower and later)
    const io = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          const el = entry.target;
          // small stagger based on order in DOM
          const index = Number(el.getAttribute('data-idx')) || 0;
          setTimeout(() => el.classList.add('is-visible'), 120 * index);
          io.unobserve(el);
        }
      });
    }, { rootMargin: '0px 0px -25% 0px', threshold: 0.25 });

    document.querySelectorAll('.section').forEach((sec, i) => {
      sec.setAttribute('data-idx', String(i));
      io.observe(sec);
    });
  } else {
    // Reduced motion: make all sections visible immediately
    document.querySelectorAll('.section').forEach((sec) => sec.classList.add('is-visible'));
  }
})();
