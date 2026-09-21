const toggle = document.querySelector('.theme-toggle');

function setTheme(theme) {
  document.documentElement.dataset.theme = theme;
  localStorage.theme = theme;
}

toggle?.addEventListener('click', () => {
  const currentTheme = document.documentElement.dataset.theme;

  if (currentTheme === 'dark') {
    setTheme('light');
  } else {
    setTheme('dark');
  }
});