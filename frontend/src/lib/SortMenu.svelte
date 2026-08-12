<script lang="ts">
	// Sort picker for the collections and items lists. Sits beside the filter
	// box rather than in the page header, which keeps the header from wrapping
	// on a phone.
	import Icon from '$lib/Icon.svelte';
	import { SORT_OPTIONS, sortLabel, type SortOption } from '$lib/sort';

	let {
		value = $bindable('updated-desc' as SortOption),
		onchange
	}: {
		value?: SortOption;
		onchange?: (v: SortOption) => void;
	} = $props();

	let open = $state(false);
	let el = $state<HTMLDivElement | null>(null);

	function choose(v: SortOption) {
		value = v;
		onchange?.(v);
		open = false;
	}

	// Close on outside click or Escape, matching Dropdown's behaviour.
	$effect(() => {
		if (!open) return;
		function onDocClick(e: MouseEvent) {
			if (el && !el.contains(e.target as Node)) open = false;
		}
		function onKey(e: KeyboardEvent) {
			if (e.key === 'Escape') open = false;
		}
		document.addEventListener('click', onDocClick, true);
		document.addEventListener('keydown', onKey);
		return () => {
			document.removeEventListener('click', onDocClick, true);
			document.removeEventListener('keydown', onKey);
		};
	});
</script>

<div bind:this={el} class="relative shrink-0">
	<button
		type="button"
		onclick={() => (open = !open)}
		aria-haspopup="menu"
		aria-expanded={open}
		title={`Sort: ${sortLabel(value)}`}
		aria-label={`Sort: ${sortLabel(value)}`}
		class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-2 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
	>
		<Icon name="sort" class="h-4 w-4 shrink-0 text-slate-500" />
		<!-- The label would crowd the filter box on a phone; the icon carries it there. -->
		<span class="hidden max-w-40 truncate sm:inline">{sortLabel(value)}</span>
	</button>

	{#if open}
		<div
			role="menu"
			class="absolute right-0 z-30 mt-1 min-w-56 overflow-hidden rounded-md border border-slate-200 bg-white py-1 shadow-lg dark:border-slate-700 dark:bg-slate-900"
		>
			{#each SORT_OPTIONS as opt (opt.value)}
				{@const selected = opt.value === value}
				<button
					type="button"
					role="menuitemradio"
					aria-checked={selected}
					onclick={() => choose(opt.value)}
					class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-slate-100 dark:hover:bg-slate-800"
					class:font-semibold={selected}
					class:text-sky-600={selected}
					class:dark:text-sky-400={selected}
				>
					<Icon
						name="check"
						class={`h-4 w-4 shrink-0 ${selected ? '' : 'invisible'}`}
					/>
					{opt.label}
				</button>
			{/each}
		</div>
	{/if}
</div>
