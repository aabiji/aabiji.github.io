const toggle = document.querySelector('.theme-toggle');
const mq = matchMedia('(prefers-color-scheme: dark)');

function setTheme(theme, persist) {
  document.documentElement.dataset.theme = theme;
  if (persist) {
    try { localStorage.setItem('theme', theme); } catch (e) { }
  }
}

toggle?.addEventListener('click', () => {
  const next = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
  setTheme(next, true);
});

// Follow live OS changes only if the user hasn't picked a theme
mq.addEventListener('change', (e) => {
  let saved;
  try { saved = localStorage.getItem('theme'); } catch (err) { }
  if (!saved) setTheme(e.matches ? 'dark' : 'light', false);
});