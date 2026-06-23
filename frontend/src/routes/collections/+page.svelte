<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import Icon from '$lib/Icon.svelte';
	import Modal from '$lib/Modal.svelte';
	import {
		listCollections,
		createCollection,
		updateCollection,
		deleteCollection,
		getCollectionStats,
		listItems,
		createItem,
		formatCurrency,
		type Collection,
		type Item,
		type Stats
	} from '$lib/api';

	let collections = $state<Collection[]>([]);
	let selectedId = $state<number | null>(null);
	let items = $state<Item[]>([]);
	let stats = $state<Stats | null>(null);
	let loading = $state(true);
	let error = $state('');

	const selected = $derived(collections.find((c) => c.id === selectedId) ?? null);

	// Collection create/edit modal
	let collectionModal = $state(false);
	let editingCollection = $state<Collection | null>(null);
	let cName = $state('');
	let cDescription = $state('');
	let savingCollection = $state(false);

	// Item create modal
	let itemModal = $state(false);
	let iName = $state('');
	let iDescription = $state('');
	let savingItem = $state(false);

	let deleteModal = $state(false);
	let deleting = $state(false);

	async function loadCollections(preferId: number | null = null) {
		collections = await listCollections();
		if (collections.length === 0) {
			selectedId = null;
			return;
		}
		const wanted = preferId ?? selectedId;
		selectedId = collections.some((c) => c.id === wanted) ? wanted! : collections[0].id;
	}

	async function loadSelected() {
		if (selectedId == null) {
			items = [];
			stats = null;
			return;
		}
		[items, stats] = await Promise.all([listItems(selectedId), getCollectionStats(selectedId)]);
	}

	onMount(async () => {
		try {
			const param = $page.url.searchParams.get('c');
			const preferId = param ? Number(param) : null;
			await loadCollections(preferId);
			await loadSelected();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load collections';
		} finally {
			loading = false;
		}
	});

	async function switchCollection(id: number) {
		selectedId = id;
		error = '';
		const url = new URL($page.url);
		url.searchParams.set('c', String(id));
		await goto(`${url.pathname}?${url.searchParams.toString()}`, {
			replaceState: true,
			keepFocus: true,
			noScroll: true
		});
		try {
			await loadSelected();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load collection';
		}
	}

	function openCreateCollection() {
		editingCollection = null;
		cName = '';
		cDescription = '';
		collectionModal = true;
	}

	function openEditCollection() {
		if (!selected) return;
		editingCollection = selected;
		cName = selected.name;
		cDescription = selected.description;
		collectionModal = true;
	}

	async function saveCollection() {
		if (!cName.trim()) return;
		savingCollection = true;
		error = '';
		try {
			if (editingCollection) {
				await updateCollection(editingCollection.id, {
					name: cName.trim(),
					description: cDescription.trim()
				});
				await loadCollections(editingCollection.id);
			} else {
				const created = await createCollection({
					name: cName.trim(),
					description: cDescription.trim()
				});
				await loadCollections(created.id);
				await switchCollection(created.id);
			}
			await loadSelected();
			collectionModal = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save collection';
		} finally {
			savingCollection = false;
		}
	}

	async function confirmDeleteCollection() {
		if (!selected) return;
		deleting = true;
		error = '';
		try {
			await deleteCollection(selected.id);
			deleteModal = false;
			await loadCollections(null);
			await loadSelected();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete collection';
		} finally {
			deleting = false;
		}
	}

	function openCreateItem() {
		iName = '';
		iDescription = '';
		itemModal = true;
	}

	async function saveItem() {
		if (!selectedId || !iName.trim()) return;
		savingItem = true;
		error = '';
		try {
			await createItem(selectedId, {
				name: iName.trim(),
				description: iDescription.trim(),
				locationLat: null,
				locationLng: null,
				locationLabel: ''
			});
			itemModal = false;
			await loadSelected();
			await loadCollections(selectedId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create item';
		} finally {
			savingItem = false;
		}
	}
</script>

<section class="mx-auto max-w-5xl space-y-6">
	<header class="flex flex-wrap items-center justify-between gap-3">
		<h1 class="text-2xl font-bold">Collections</h1>
		<div class="flex items-center gap-2">
			{#if collections.length > 0}
				<div class="relative">
					<select
						class="appearance-none rounded-md border border-slate-300 bg-white py-1.5 pl-3 pr-9 text-sm dark:border-slate-700 dark:bg-slate-800"
						value={selectedId}
						onchange={(e) => switchCollection(Number((e.target as HTMLSelectElement).value))}
					>
						{#each collections as c (c.id)}
							<option value={c.id}>{c.name}</option>
						{/each}
					</select>
					<span class="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 text-slate-500">
						<Icon name="chevron-down" class="h-4 w-4" />
					</span>
				</div>
			{/if}
			<button
				type="button"
				class="inline-flex items-center gap-1.5 rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700"
				onclick={openCreateCollection}
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
				onclick={openCreateCollection}
			>
				<Icon name="plus" class="h-4 w-4" />
				New collection
			</button>
		</div>
	{:else if selected}
		<!-- Collection header + actions -->
		<div class="rounded-lg border border-slate-200 p-5 dark:border-slate-800">
			<div class="flex flex-wrap items-start justify-between gap-3">
				<div>
					<h2 class="text-xl font-semibold">{selected.name}</h2>
					{#if selected.description}
						<p class="mt-1 text-sm text-slate-600 dark:text-slate-400">{selected.description}</p>
					{/if}
					<p class="mt-2 text-xs text-slate-500">
						Updated {new Date(selected.updatedAt).toLocaleString()}
						{#if selected.updatedBy}· by {selected.updatedBy}{/if}
					</p>
				</div>
				<div class="flex items-center gap-2">
					<button
						type="button"
						class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
						onclick={openEditCollection}
					>
						<Icon name="pencil" class="h-4 w-4" /> Edit
					</button>
					<button
						type="button"
						class="inline-flex items-center gap-1.5 rounded-md border border-rose-300 px-2.5 py-1.5 text-sm text-rose-600 hover:bg-rose-50 dark:border-rose-800 dark:text-rose-400 dark:hover:bg-rose-950/40"
						onclick={() => (deleteModal = true)}
					>
						<Icon name="trash" class="h-4 w-4" /> Delete
					</button>
				</div>
			</div>

			<!-- Stats -->
			{#if stats}
				<div class="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
					<div class="rounded-md border border-slate-200 p-3 dark:border-slate-800">
						<div class="text-xs text-slate-500">Items</div>
						<div class="text-lg font-semibold">{stats.itemCount}</div>
					</div>
					<div class="rounded-md border border-slate-200 p-3 dark:border-slate-800">
						<div class="text-xs text-slate-500">Entries</div>
						<div class="text-lg font-semibold">{stats.entryCount}</div>
					</div>
					{#each stats.totals as t (t.currency)}
						<div class="rounded-md border border-slate-200 p-3 dark:border-slate-800">
							<div class="text-xs text-slate-500">{t.currency} total</div>
							<div class="text-lg font-semibold">{formatCurrency(t.total, t.currency)}</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Items -->
		<div class="flex items-center justify-between">
			<h3 class="text-lg font-semibold">Items</h3>
			<button
				type="button"
				class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
				onclick={openCreateItem}
			>
				<Icon name="plus" class="h-4 w-4" /> Add item
			</button>
		</div>

		{#if items.length === 0}
			<p class="rounded-lg border border-dashed border-slate-300 p-6 text-center text-sm text-slate-500 dark:border-slate-700">
				No items in this collection yet.
			</p>
		{:else}
			<ul class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
				{#each items as item (item.id)}
					<li>
						<a
							href={`/items/${item.id}`}
							class="block overflow-hidden rounded-lg border border-slate-200 transition hover:border-sky-400 hover:shadow-sm dark:border-slate-800 dark:hover:border-sky-600"
						>
							<div class="flex h-32 items-center justify-center bg-slate-100 dark:bg-slate-800">
								{#if item.imagePath}
									<img src={item.imagePath} alt={item.name} class="h-full w-full object-cover" />
								{:else}
									<Icon name="cube" class="h-10 w-10 text-slate-400" />
								{/if}
							</div>
							<div class="p-3">
								<div class="font-medium">{item.name}</div>
								{#if item.description}
									<p class="mt-0.5 line-clamp-2 text-xs text-slate-500">{item.description}</p>
								{/if}
								<div class="mt-2 flex items-center gap-3 text-xs text-slate-500">
									<span class="inline-flex items-center gap-1">
										<Icon name="activity" class="h-3.5 w-3.5" />
										{item.entryCount} entries
									</span>
									{#if item.locationLat != null && item.locationLng != null}
										<span class="inline-flex items-center gap-1">
											<Icon name="map-pin" class="h-3.5 w-3.5" />
											{item.locationLabel || 'Located'}
										</span>
									{/if}
								</div>
							</div>
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	{/if}
</section>

<!-- Collection create/edit modal -->
<Modal
	title={editingCollection ? 'Edit collection' : 'New collection'}
	bind:open={collectionModal}
>
	<div class="space-y-3">
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Name</span>
			<input
				type="text"
				bind:value={cName}
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
				placeholder="e.g. Coin collection"
			/>
		</label>
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Description</span>
			<textarea
				bind:value={cDescription}
				rows="3"
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
			></textarea>
		</label>
	</div>
	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => (collectionModal = false)}
		>
			Cancel
		</button>
		<button
			type="button"
			class="rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-60"
			onclick={saveCollection}
			disabled={savingCollection || !cName.trim()}
		>
			{savingCollection ? 'Saving…' : 'Save'}
		</button>
	{/snippet}
</Modal>

<!-- Item create modal -->
<Modal title="Add item" bind:open={itemModal}>
	<div class="space-y-3">
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Name</span>
			<input
				type="text"
				bind:value={iName}
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
				placeholder="e.g. 1921 Silver Dollar"
			/>
		</label>
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Description</span>
			<textarea
				bind:value={iDescription}
				rows="3"
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
			></textarea>
		</label>
		<p class="text-xs text-slate-500">
			You can add a photo, location and transaction entries after creating the item.
		</p>
	</div>
	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => (itemModal = false)}
		>
			Cancel
		</button>
		<button
			type="button"
			class="rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-60"
			onclick={saveItem}
			disabled={savingItem || !iName.trim()}
		>
			{savingItem ? 'Creating…' : 'Create'}
		</button>
	{/snippet}
</Modal>

<!-- Delete confirm modal -->
<Modal title="Delete collection" bind:open={deleteModal}>
	<p class="text-sm text-slate-600 dark:text-slate-400">
		Are you sure you want to delete <strong>{selected?.name}</strong>? This will remove all items
		and entries in it. This action cannot be undone.
	</p>
	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => (deleteModal = false)}
		>
			Cancel
		</button>
		<button
			type="button"
			class="rounded-md bg-rose-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-rose-700 disabled:opacity-60"
			onclick={confirmDeleteCollection}
			disabled={deleting}
		>
			{deleting ? 'Deleting…' : 'Delete'}
		</button>
	{/snippet}
</Modal>
