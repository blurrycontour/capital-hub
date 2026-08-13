<script lang="ts">
	import { onMount } from 'svelte';
	import Icon from '$lib/Icon.svelte';
	import SortMenu from '$lib/SortMenu.svelte';
	import { loadSort, saveSort, sortRecords, type SortOption } from '$lib/sort';
	import { accessDescription } from '$lib/access';
	import CreateCollectionModal from '$lib/CreateCollectionModal.svelte';
	import {
		listCollections,
		type Collection
	} from '$lib/api';

	let collections = $state<Collection[]>([]);
	let loading = $state(true);
	let error = $state('');
	let query = $state('');

	// Card vs. list view (persisted).
	const VIEW_KEY = 'ch-view-collections';
	let view = $state<'card' | 'list'>('card');

	// Sort order (persisted per device, like the view preference).
	const SORT_KEY = 'ch-sort-collections';
	let sort = $state<SortOption>('updated-desc');

	function setSort(v: SortOption) {
		sort = v;
		saveSort(SORT_KEY, v);
	}

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
		const matches = !q
			? collections
			: collections.filter(
					(c) =>
						c.name.toLowerCase().includes(q) ||
						c.description.toLowerCase().includes(q) ||
						c.currency.toLowerCase().includes(q) ||
						c.ownerName.toLowerCase().includes(q)
				);
		return sortRecords(matches, sort);
	});

	// Create modal state (the form itself lives in CreateCollectionModal).
	let createModal = $state(false);

	async function load() {
		loading = true;
		try {
			collections = await listCollections();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load collections';
		} finally {
			loading = false;
		}
	}

	onMount(load);

	onMount(() => {
		try {
			const raw = localStorage.getItem(VIEW_KEY);
			if (raw === 'list' || raw === 'card') view = raw;
			sort = loadSort(SORT_KEY);
		} catch {
			/* ignore */
		}
	});

	function openCreate() {
		error = '';
		createModal = true;
	}
</script>

<svelte:head><title>Collections · Capital Hub</title></svelte:head>

<section class="mx-auto max-w-5xl space-y-6">
	<header class="flex flex-wrap items-center justify-between gap-3">
		<div class="flex items-center gap-2.5">
			<span
				class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-sky-100 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300"
			>
				<Icon name="collections" class="h-5 w-5" />
			</span>
			<h1 class="text-2xl font-bold">Collections</h1>
		</div>
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
			<button
				type="button"
				class="inline-flex items-center gap-1.5 rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700"
				onclick={openCreate}
			>
				<Icon name="plus" class="h-4 w-4" />
				New collection
			</button>
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
	{:else if collections.length === 0}
		<div
			class="flex flex-col items-center gap-3 rounded-lg border border-dashed border-slate-300 p-10 text-center text-slate-500 dark:border-slate-700"
		>
			<Icon name="collections" class="h-10 w-10" />
			<p class="text-sm">No collections yet. Create your first one to get started.</p>
			<button
				type="button"
				class="inline-flex items-center gap-1.5 rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700"
				onclick={openCreate}
			>
				<Icon name="plus" class="h-4 w-4" />
				New collection
			</button>
		</div>
	{:else}
		<div class="flex items-center gap-2">
			<input
				type="search"
				bind:value={query}
				placeholder="Filter collections…"
				class="min-w-0 flex-1 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
			/>
			<SortMenu value={sort} onchange={setSort} />
		</div>

		{#if filtered.length === 0}
			<p
				class="rounded-lg border border-dashed border-slate-300 p-6 text-center text-sm text-slate-500 dark:border-slate-700"
			>
				No collections match “{query}”.
			</p>
		{:else if view === 'list'}
			<ul class="divide-y divide-slate-200 overflow-hidden rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
				{#each filtered as c (c.id)}
				<li>
					<a
						href={`/collections/${c.id}`}
						class="flex items-center gap-3 px-4 py-3 transition hover:bg-slate-50 dark:hover:bg-slate-800/60"
					>
						<span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-slate-100 text-slate-500 dark:bg-slate-800">
							<Icon name="collections" class="h-4 w-4" />
						</span>
						<div class="min-w-0 flex-1">
							<p class="truncate font-medium">{c.name}</p>
							{#if c.description}
								<p class="truncate text-sm text-slate-500">{c.description}</p>
							{/if}
							<div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
								<span class="inline-flex items-center gap-1">
									<Icon name="cube" class="h-3.5 w-3.5" />
									{c.itemCount} items
								</span>
								<span
									class="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300"
								>
									<Icon name="currency" class="h-3.5 w-3.5" />
									{c.currency}
								</span>
								{#if c.shared}
									<span
										class="inline-flex items-center gap-1 rounded-full bg-violet-100 px-2 py-0.5 font-medium text-violet-700 dark:bg-violet-950/40 dark:text-violet-300"
										title={`Shared by ${c.ownerName} (${accessDescription(c.accessLevel)})`}
									>
										<Icon name="users" class="h-3 w-3" />
										Shared
									</span>
								{:else}
									<span
										class="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 font-medium text-slate-500 dark:bg-slate-800 dark:text-slate-400"
										title="You own this collection"
									>
										<Icon name="user" class="h-3 w-3" />
										Owned
									</span>
									{#if c.shareCount > 0}
										<span
											class="inline-flex items-center gap-1 rounded-full bg-sky-100 px-2 py-0.5 font-medium text-sky-700 dark:bg-sky-950/40 dark:text-sky-300"
											title={`Shared with ${c.shareCount} ${c.shareCount === 1 ? 'person' : 'people'}`}
										>
											<Icon name="users" class="h-3 w-3" />
											Shared
										</span>
									{/if}
								{/if}
								{#if c.locationLat != null && c.locationLng != null}
									<span class="inline-flex items-center gap-1">
										<Icon name="map-pin" class="h-3.5 w-3.5" />
										{c.locationLabel || 'Located'}
									</span>
								{/if}
							</div>
						</div>
					</a>
				</li>
			{/each}
		</ul>
	{:else}
		<ul class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each filtered as c (c.id)}
				<li class="min-w-0">
					<a
						href={`/collections/${c.id}`}
						class="flex h-full min-w-0 flex-col rounded-lg border border-slate-200 p-4 transition hover:border-sky-400 hover:shadow-sm dark:border-slate-800 dark:hover:border-sky-600"
					>
						<h2 class="break-words font-semibold">{c.name}</h2>
						{#if c.description}
							<p class="mt-1 line-clamp-3 flex-1 overflow-hidden break-words text-sm text-slate-500">{c.description}</p>
						{/if}
						<div class="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-xs text-slate-500">
							<span class="inline-flex items-center gap-1">
								<Icon name="cube" class="h-3.5 w-3.5" />
								{c.itemCount} items
							</span>
							{#if c.locationLat != null && c.locationLng != null}
								<span class="inline-flex items-center gap-1">
									<Icon name="map-pin" class="h-3.5 w-3.5" />
									{c.locationLabel || 'Located'}
								</span>
							{/if}
							<span
								class="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300"
							>
								<Icon name="currency" class="h-3.5 w-3.5" />
								{c.currency}
							</span>
							{#if c.shared}
								<span
									class="inline-flex items-center gap-1 rounded-full bg-violet-100 px-2 py-0.5 font-medium text-violet-700 dark:bg-violet-950/40 dark:text-violet-300"
									title={`Shared by ${c.ownerName} (${accessDescription(c.accessLevel)})`}
								>
									<Icon name="users" class="h-3 w-3" />
									Shared
								</span>
							{:else}
								<span
									class="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 font-medium text-slate-500 dark:bg-slate-800 dark:text-slate-400"
									title="You own this collection"
								>
									<Icon name="user" class="h-3 w-3" />
									Owned
								</span>
								{#if c.shareCount > 0}
									<span
										class="inline-flex items-center gap-1 rounded-full bg-sky-100 px-2 py-0.5 font-medium text-sky-700 dark:bg-sky-950/40 dark:text-sky-300"
										title={`Shared with ${c.shareCount} ${c.shareCount === 1 ? 'person' : 'people'}`}
									>
										<Icon name="users" class="h-3 w-3" />
										Shared
									</span>
								{/if}
							{/if}
						</div>
					</a>
				</li>
			{/each}
		</ul>
		{/if}
	{/if}
</section>

<!-- New collection modal -->
<CreateCollectionModal bind:open={createModal} oncreated={() => load()} />
