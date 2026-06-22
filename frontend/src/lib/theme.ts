/**
 * Theme management: persists the user's light/dark preference and applies it
 * to the document root. Defaults to the OS preference when unset.
 */
import { browser } from '$app/environment';

export type Theme = 'light' | 'dark';

const STORAGE_KEY = 'ch-theme';

export function getInitialTheme(): Theme {
	if (!browser) return 'light';
	const stored = localStorage.getItem(STORAGE_KEY) as Theme | null;
	if (stored === 'light' || stored === 'dark') return stored;
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function applyTheme(theme: Theme): void {
	if (!browser) return;
	document.documentElement.classList.toggle('dark', theme === 'dark');
	localStorage.setItem(STORAGE_KEY, theme);
}
