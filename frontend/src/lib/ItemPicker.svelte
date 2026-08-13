<script lang="ts">
	// Searchable item chooser for the quick-add entry form, where the item is
	// not implied by the page. Shows the owning collection alongside each item,
	// since item names repeat across collections.
	import Icon from '$lib/Icon.svelte';
	import type { ItemWithCollection } from '$lib/api';

	let {
		items = [],
		value = $bindable<number | null>(null),
		loading = false,
		disabled = false
	}: {
		items?: ItemWithCollection[];
		value?: number | null;
		loading?: boolean;
		disabled?: boolean;
	} = $props();

	let open = $state(false);
	let query = $state('');
	let el = $state<HTMLDivElement | null>(null);
	let input = $state<HTMLInputElement | null>(null);
	let highlighted = $state(0);

	const selected = $derived(items.find((i) => i.id === value) ?? null);

	const matches = $derived.by(() => {
		const q = query.trim().toLowerCase();
		if (!q) return items;
		return items.filter(
			(i) =>
				i.name.toLowerCase().includes(q) || (i.collectionName ?? '').toLowerCase().includes(q)
		);
	});

	// Keep the highlight inside the filtered list as it shrinks.
	$effect(() => {
		if (highlighted >= matches.length) highlighted = Math.max(0, matches.length - 1);
	});

	function openList() {
		if (disabled) return;
		open = true;
		query = '';
		highlighted = Math.max(
			0,
			matches.findIndex((i) => i.id === value)
		);
		// Focus after the list renders so typing filters immediately.
		queueMicrotask(() => input?.focus());
	}

	function choose(id: number) {
		value = id;
		open = false;
	}

	function onkeydown(e: KeyboardEvent) {
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			highlighted = Math.min(highlighted + 1, matches.length - 1);
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			highlighted = Math.max(highlighted - 1, 0);
		} else if (e.key === 'Enter') {
			e.preventDefault();
			const pick = matches[highlighted];
			if (pick) choose(pick.id);
		} else if (e.key === 'Escape') {
			e.preventDefault();
			open = false;
		}
	}

	$effect(() => {
		if (!open) return;
		function onDocClick(ev: MouseEvent) {
			if (el && !el.contains(ev.target as Node)) open = false;
		}
		document.addEventListener('click', onDocClick, true);
		return () => document.removeEventListener('click', onDocClick, true);
	});
</script>

<div bind:this={el} class="relative">
	{#if open}
		<input
			bind:this={input}
			type="text"
			bind:value={query}
			{onkeydown}
			placeholder="Search items…"
			role="combobox"
			aria-expanded="true"
			aria-controls="item-picker-list"
			aria-autocomplete="list"
			class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900"
		/>
	{:else}
		<button
			type="button"
			onclick={openList}
			disabled={disabled || loading}
			aria-haspopup="listbox"
			class="flex w-full items-center justify-between gap-2 rounded-md border border-slate-300 bg-white px-3 py-2 text-left text-sm disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900"
		>
			{#if loading}
				<span class="text-slate-500">Loading items…</span>
			{:else if selected}
				<span class="min-w-0">
					<span class="block truncate">{selected.name}</span>
					<span class="block truncate text-xs text-slate-500">in {selected.collectionName}</span>
				</span>
			{:else}
				<span class="text-slate-500">Select an item…</span>
			{/if}
			<Icon name="chevron-down" class="h-4 w-4 shrink-0 text-slate-400" />
		</button>
	{/if}

	{#if open}
		<ul
			id="item-picker-list"
			role="listbox"
			class="ch-no-scrollbar absolute z-30 mt-1 max-h-60 w-full overflow-y-auto rounded-md border border-slate-200 bg-white py-1 shadow-lg dark:border-slate-700 dark:bg-slate-900"
		>
			{#if matches.length === 0}
				<li class="px-3 py-2 text-sm text-slate-500">No matching items.</li>
			{:else}
				{#each matches as it, i (it.id)}
					<li>
						<button
							type="button"
							role="option"
							aria-selected={it.id === value}
							onclick={() => choose(it.id)}
							onmouseenter={() => (highlighted = i)}
							class={`flex w-full flex-col items-start px-3 py-2 text-left text-sm ${
								i === highlighted ? 'bg-slate-100 dark:bg-slate-800' : ''
							}`}
						>
							<span class="w-full truncate {it.id === value ? 'font-semibold text-sky-600 dark:text-sky-400' : ''}">
								{it.name}
							</span>
							<span class="flex w-full items-center gap-1 truncate text-xs text-slate-500">
								<Icon name="collections" class="h-3 w-3 shrink-0" />
								{it.collectionName}
							</span>
						</button>
					</li>
				{/each}
			{/if}
		</ul>
	{/if}
</div>
