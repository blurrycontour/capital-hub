<script lang="ts" generics="T extends string">
	// Sort picker for the list pages. Sits beside the filter box rather than in
	// the page header, which keeps the header from wrapping on a phone. The
	// options are supplied by the caller, because search sorts on a different
	// set from the collection and item lists.
	import Icon from '$lib/Icon.svelte';
	import { SORT_OPTIONS } from '$lib/sort';

	let {
		value = $bindable(),
		options = SORT_OPTIONS as unknown as { value: T; label: string }[],
		onchange
	}: {
		value: T;
		options?: { value: T; label: string }[];
		onchange?: (v: T) => void;
	} = $props();

	const currentLabel = $derived(options.find((o) => o.value === value)?.label ?? '');

	let open = $state(false);
	let el = $state<HTMLDivElement | null>(null);

	function choose(v: T) {
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
		title={`Sort: ${currentLabel}`}
		aria-label={`Sort: ${currentLabel}`}
		class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-2 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
	>
		<Icon name="sort" class="h-4 w-4 shrink-0 text-slate-500" />
		<!-- The label would crowd the filter box on a phone; the icon carries it there. -->
		<span class="hidden max-w-40 truncate sm:inline">{currentLabel}</span>
	</button>

	{#if open}
		<div
			role="menu"
			class="absolute right-0 z-30 mt-1 min-w-56 overflow-hidden rounded-md border border-slate-200 bg-white py-1 shadow-lg dark:border-slate-700 dark:bg-slate-900"
		>
			{#each options as opt (opt.value)}
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
