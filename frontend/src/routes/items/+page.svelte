<script lang="ts">
	import { onMount } from 'svelte';
	import Icon from '$lib/Icon.svelte';
	import { listAllItems, type ItemWithCollection } from '$lib/api';

	let items = $state<ItemWithCollection[]>([]);
	let loading = $state(true);
	let error = $state('');
	let query = $state('');

	// Card vs. list view (persisted).
	const VIEW_KEY = 'ch-view-items';
	let view = $state<'card' | 'list'>('card');

	function setView(v: 'card' | 'list') {
		view = v;
		try {
			localStorage.setItem(VIEW_KEY, v);
		} catch {
			/* ignore */
		}
	}

	const filtered = $derived.by(() => {
		const q = query.trim().toLowerCase();
		if (!q) return items;
		return items.filter(
			(it) =>
				it.name.toLowerCase().includes(q) ||
				it.description.toLowerCase().includes(q) ||
				it.collectionName.toLowerCase().includes(q)
		);
	});

	async function load() {
		loading = true;
		try {
			items = await listAllItems();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load items';
		} finally {
			loading = false;
		}
	}

	onMount(load);

	onMount(() => {
		try {
			const raw = localStorage.getItem(VIEW_KEY);
			if (raw === 'list' || raw === 'card') view = raw;
		} catch {
			/* ignore */
		}
	});

	function itemHref(it: ItemWithCollection): string {
		return `/collections/${it.collectionId}/items/${it.id}`;
	}
</script>

<section class="mx-auto max-w-5xl space-y-6">
	<header class="flex flex-wrap items-center justify-between gap-3">
		<h1 class="text-2xl font-bold">Items</h1>
		<div class="flex items-center gap-2">
			<div class="inline-flex rounded-md border border-slate-300 p-0.5 dark:border-slate-700">
				<button
					type="button"
					class="rounded p-1.5"
					class:bg-slate-200={view === 'card'}
					class:dark:bg-slate-700={view === 'card'}
					class:text-slate-500={view !== 'card'}
					title="Card view"
					aria-label="Card view"
					onclick={() => setView('card')}
				>
					<Icon name="grid" class="h-4 w-4" />
				</button>
				<button
					type="button"
					class="rounded p-1.5"
					class:bg-slate-200={view === 'list'}
					class:dark:bg-slate-700={view === 'list'}
					class:text-slate-500={view !== 'list'}
					title="List view"
					aria-label="List view"
					onclick={() => setView('list')}
				>
					<Icon name="list" class="h-4 w-4" />
				</button>
			</div>
		</div>
	</header>

	{#if error}
		<div
			class="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200"
		>
			{error}
		</div>
	{/if}

	{#if loading}
		<p class="text-sm text-slate-500">Loading…</p>
	{:else if items.length === 0}
		<div
			class="flex flex-col items-center gap-3 rounded-lg border border-dashed border-slate-300 p-10 text-center text-slate-500 dark:border-slate-700"
		>
			<Icon name="cube" class="h-10 w-10" />
			<p class="text-sm">No items yet. Add items to your collections to see them here.</p>
		</div>
	{:else}
		<input
			type="search"
			bind:value={query}
			placeholder="Filter items…"
			class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
		/>

		{#if filtered.length === 0}
			<p
				class="rounded-lg border border-dashed border-slate-300 p-6 text-center text-sm text-slate-500 dark:border-slate-700"
			>
				No items match “{query}”.
			</p>
		{:else if view === 'list'}
			<ul
				class="divide-y divide-slate-200 overflow-hidden rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800"
			>
				{#each filtered as it (it.id)}
					<li>
						<a
							href={itemHref(it)}
							class="flex items-center gap-3 px-4 py-3 transition hover:bg-slate-50 dark:hover:bg-slate-800/60"
						>
							<span
								class="flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-md bg-slate-100 text-slate-500 dark:bg-slate-800"
							>
								{#if it.imagePath}
									<img src={it.imagePath} alt={it.name} class="h-full w-full object-cover" />
								{:else}
									<Icon name="cube" class="h-4 w-4" />
								{/if}
							</span>
							<div class="min-w-0 flex-1">
								<p class="truncate font-medium">{it.name}</p>
								{#if it.description}
									<p class="truncate text-sm text-slate-500">{it.description}</p>
								{/if}
								<div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
									<span
										class="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300"
									>
										<Icon name="collections" class="h-3.5 w-3.5" />
										{it.collectionName}
									</span>
									<span class="inline-flex items-center gap-1">
										<Icon name="list" class="h-3.5 w-3.5" />
										{it.entryCount}
										{it.entryCount === 1 ? 'entry' : 'entries'}
									</span>
								</div>
							</div>
						</a>
					</li>
				{/each}
			</ul>
		{:else}
			<ul class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each filtered as it (it.id)}
					<li class="min-w-0">
						<a
							href={itemHref(it)}
							class="flex h-full min-w-0 flex-col rounded-lg border border-slate-200 p-4 transition hover:border-sky-400 hover:shadow-sm dark:border-slate-800 dark:hover:border-sky-600"
						>
							<h2 class="break-words font-semibold">{it.name}</h2>
							{#if it.description}
								<p class="mt-1 line-clamp-3 flex-1 overflow-hidden break-words text-sm text-slate-500">
									{it.description}
								</p>
							{/if}
							<div class="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-xs text-slate-500">
								<span
									class="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300"
									title="Collection"
								>
									<Icon name="collections" class="h-3.5 w-3.5" />
									{it.collectionName}
								</span>
								<span class="inline-flex items-center gap-1">
									<Icon name="list" class="h-3.5 w-3.5" />
									{it.entryCount}
									{it.entryCount === 1 ? 'entry' : 'entries'}
								</span>
							</div>
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	{/if}
</section>
