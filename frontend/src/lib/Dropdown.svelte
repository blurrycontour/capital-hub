<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon from '$lib/Icon.svelte';

	let {
		label = 'Options',
		children
	}: { label?: string; children: Snippet } = $props();

	let open = $state(false);
	let el = $state<HTMLDivElement | null>(null);

	function toggle() {
		open = !open;
	}

	function close() {
		open = false;
	}

	// Close on outside click or Escape while the menu is open.
	$effect(() => {
		if (!open) return;
		function onDocClick(e: MouseEvent) {
			if (el && !el.contains(e.target as Node)) close();
		}
		function onKey(e: KeyboardEvent) {
			if (e.key === 'Escape') close();
		}
		document.addEventListener('click', onDocClick, true);
		document.addEventListener('keydown', onKey);
		return () => {
			document.removeEventListener('click', onDocClick, true);
			document.removeEventListener('keydown', onKey);
		};
	});
</script>

<div bind:this={el} class="relative">
	<button
		type="button"
		onclick={toggle}
		aria-haspopup="menu"
		aria-expanded={open}
		class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
	>
		<Icon name="ellipsis" class="h-4 w-4" />
		<span class="hidden sm:inline">{label}</span>
	</button>

	{#if open}
		<!-- Clicking any menu item bubbles up here and closes the menu. -->
		<div
			role="menu"
			tabindex="-1"
			onclick={close}
			onkeydown={() => {}}
			class="absolute right-0 z-30 mt-1 min-w-44 overflow-hidden rounded-md border border-slate-200 bg-white py-1 shadow-lg dark:border-slate-700 dark:bg-slate-900"
		>
			{@render children()}
		</div>
	{/if}
</div>
