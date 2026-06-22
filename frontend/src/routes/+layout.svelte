<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { applyTheme, getInitialTheme, type Theme } from '$lib/theme';

	let { children } = $props();
	let theme = $state<Theme>('light');

	onMount(() => {
		theme = getInitialTheme();
		applyTheme(theme);
	});

	function toggleTheme() {
		theme = theme === 'dark' ? 'light' : 'dark';
		applyTheme(theme);
	}
</script>

<div class="flex min-h-full flex-col bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
	<header class="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-slate-800">
		<span class="text-lg font-semibold">Capital&#8209;Hub</span>
		<button
			type="button"
			onclick={toggleTheme}
			class="rounded-md border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
		>
			{theme === 'dark' ? 'Light' : 'Dark'} mode
		</button>
	</header>

	<main class="flex-1 p-4">
		{@render children()}
	</main>
</div>
