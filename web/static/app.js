// Global JS — component-scoped interactions use Alpine.js directly in templates.

function toggleTheme() {
	const isDark = document.documentElement.classList.toggle('dark');
	localStorage.setItem('theme', isDark ? 'dark' : 'light');
}
