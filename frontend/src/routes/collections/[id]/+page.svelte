<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import Icon from '$lib/Icon.svelte';
	import Modal from '$lib/Modal.svelte';
	import LocationPicker from '$lib/LocationPicker.svelte';
	import CustomFieldsEditor from '$lib/CustomFieldsEditor.svelte';
	import MapView from '$lib/MapView.svelte';
	import { breadcrumbs } from '$lib/breadcrumb.svelte';
	import {
		getCollection,
		updateCollection,
		deleteCollection,
		getCollectionStats,
		listItems,
		createItem,
		formatCurrency,
		CURRENCIES,
		type Collection,
		type Item,
		type Stats,
		type CustomField
	} from '$lib/api';

	const collectionId = $derived(Number($page.params.id));

	let collection = $state<Collection | null>(null);
	let items = $state<Item[]>([]);
	let stats = $state<Stats | null>(null);
	let loading = $state(true);
	let error = $state('');

	// Edit collection modal.
	let editModal = $state(false);
	let eName = $state('');
	let eDescription = $state('');
	let eCurrency = $state('USD');
	let eLat = $state<number | null>(null);
	let eLng = $state<number | null>(null);
	let eLabel = $state('');
	let eFields = $state<CustomField[]>([]);
	let eUseLocation = $state(false);
	let savingEdit = $state(false);

	// Add item modal.
	let itemModal = $state(false);
	let iName = $state('');
	let iDescription = $state('');
	let iLat = $state<number | null>(null);
	let iLng = $state<number | null>(null);
	let iLabel = $state('');
	let iFields = $state<CustomField[]>([]);
	let iUseLocation = $state(false);
	let savingItem = $state(false);

	let deleteModal = $state(false);
	let deleting = $state(false);

	const mapMarkers = $derived.by(() => {
		const markers: { lat: number; lng: number; label?: string; href?: string }[] = [];
		if (collection?.locationLat != null && collection?.locationLng != null) {
			markers.push({
				lat: collection.locationLat,
				lng: collection.locationLng,
				label: collection.locationLabel || collection.name
			});
		}
		for (const it of items) {
			if (it.locationLat != null && it.locationLng != null) {
				markers.push({
					lat: it.locationLat,
					lng: it.locationLng,
					label: it.name,
					href: `/items/${it.id}`
				});
			}
		}
		return markers;
	});

	async function load() {
		loading = true;
		error = '';
		try {
			[collection, items, stats] = await Promise.all([
				getCollection(collectionId),
				listItems(collectionId),
				getCollectionStats(collectionId)
			]);
			breadcrumbs.set(`/collections/${collectionId}`, collection.name);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load collection';
		} finally {
			loading = false;
		}
	}

	onMount(load);
	onDestroy(() => breadcrumbs.clear(`/collections/${collectionId}`));

	function openEdit() {
		if (!collection) return;
		eName = collection.name;
		eDescription = collection.description;
		eCurrency = collection.currency;
		eLat = collection.locationLat;
		eLng = collection.locationLng;
		eLabel = collection.locationLabel;
		eFields = collection.customFields.map((f) => ({ ...f }));
		eUseLocation = collection.locationLat != null && collection.locationLng != null;
		editModal = true;
	}

	async function saveEdit() {
		if (!collection || !eName.trim()) return;
		savingEdit = true;
		error = '';
		try {
			collection = await updateCollection(collection.id, {
				name: eName.trim(),
				description: eDescription.trim(),
				currency: eCurrency,
				locationLat: eUseLocation ? eLat : null,
				locationLng: eUseLocation ? eLng : null,
				locationLabel: eUseLocation ? eLabel.trim() : '',
				customFields: eFields.filter((f) => f.label.trim() || f.value.trim())
			});
			breadcrumbs.set(`/collections/${collection.id}`, collection.name);
			editModal = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save collection';
		} finally {
			savingEdit = false;
		}
	}

	async function confirmDelete() {
		if (!collection) return;
		deleting = true;
		error = '';
		try {
			await deleteCollection(collection.id);
			await goto('/collections');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete collection';
			deleting = false;
		}
	}

	function openCreateItem() {
		iName = '';
		iDescription = '';
		iLat = null;
		iLng = null;
		iLabel = '';
		iFields = [];
		iUseLocation = false;
		itemModal = true;
	}

	async function saveItem() {
		if (!collection || !iName.trim()) return;
		savingItem = true;
		error = '';
		try {
			await createItem(collection.id, {
				name: iName.trim(),
				description: iDescription.trim(),
				locationLat: iUseLocation ? iLat : null,
				locationLng: iUseLocation ? iLng : null,
				locationLabel: iUseLocation ? iLabel.trim() : '',
				attachments: [],
				customFields: iFields.filter((f) => f.label.trim() || f.value.trim())
			});
			itemModal = false;
			[items, stats] = await Promise.all([
				listItems(collection.id),
				getCollectionStats(collection.id)
			]);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create item';
		} finally {
			savingItem = false;
		}
	}
</script>

<section class="mx-auto max-w-5xl space-y-6">
	<a
		href="/collections"
		class="inline-flex items-center gap-1 text-sm text-slate-500 hover:text-slate-700 dark:hover:text-slate-300"
	>
		<Icon name="chevron-left" class="h-4 w-4" /> All collections
	</a>

	{#if error}
		<div
			class="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200"
		>
			{error}
		</div>
	{/if}

	{#if loading}
		<p class="text-sm text-slate-500">Loading…</p>
	{:else if collection}
		<!-- Header -->
		<div class="rounded-lg border border-slate-200 p-5 dark:border-slate-800">
			<div class="flex flex-wrap items-start justify-between gap-3">
				<div>
					<div class="flex items-center gap-2">
						<h1 class="text-xl font-semibold">{collection.name}</h1>
						<span
							class="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300"
						>
							<Icon name="currency" class="h-3.5 w-3.5" />
							{collection.currency}
						</span>
					</div>
					{#if collection.description}
						<p class="mt-1 text-sm text-slate-600 dark:text-slate-400">{collection.description}</p>
					{/if}
					{#if collection.locationLat != null && collection.locationLng != null}
						<p class="mt-2 inline-flex items-center gap-1 text-xs text-slate-500">
							<Icon name="map-pin" class="h-3.5 w-3.5" />
							{collection.locationLabel || 'Located'}
						</p>
					{/if}
				</div>
				<div class="flex items-center gap-2">
					<button
						type="button"
						class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
						onclick={openEdit}
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

			{#if collection.customFields.length > 0}
				<dl class="mt-4 grid gap-x-6 gap-y-2 sm:grid-cols-2">
					{#each collection.customFields as field (field.label + field.value)}
						<div class="flex justify-between gap-3 border-b border-slate-100 py-1 text-sm dark:border-slate-800">
							<dt class="text-slate-500">{field.label}</dt>
							<dd class="font-medium">{field.value}</dd>
						</div>
					{/each}
				</dl>
			{/if}

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

		<!-- Map -->
		{#if mapMarkers.length > 0}
			<div>
				<h2 class="mb-2 text-lg font-semibold">Map</h2>
				<MapView markers={mapMarkers} />
			</div>
		{/if}

		<!-- Items -->
		<div class="flex items-center justify-between">
			<h2 class="text-lg font-semibold">Items</h2>
			<button
				type="button"
				class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
				onclick={openCreateItem}
			>
				<Icon name="plus" class="h-4 w-4" /> Add item
			</button>
		</div>

		{#if items.length === 0}
			<p
				class="rounded-lg border border-dashed border-slate-300 p-6 text-center text-sm text-slate-500 dark:border-slate-700"
			>
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

<!-- Edit collection modal -->
<Modal title="Edit collection" bind:open={editModal}>
	<div class="space-y-3">
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Name</span>
			<input
				type="text"
				bind:value={eName}
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
			/>
		</label>
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Description</span>
			<textarea
				bind:value={eDescription}
				rows="3"
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
			></textarea>
		</label>
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Currency</span>
			<select
				bind:value={eCurrency}
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
			>
				{#each CURRENCIES as code (code)}
					<option value={code}>{code}</option>
				{/each}
			</select>
		</label>
		<div class="text-sm">
			<span class="text-slate-600 dark:text-slate-400">Custom fields</span>
			<div class="mt-1">
				<CustomFieldsEditor bind:fields={eFields} />
			</div>
		</div>
		<label class="flex items-center gap-2 text-sm">
			<input type="checkbox" bind:checked={eUseLocation} class="rounded" />
			<span class="text-slate-600 dark:text-slate-400">Set a location</span>
		</label>
		{#if eUseLocation}
			<LocationPicker bind:lat={eLat} bind:lng={eLng} bind:label={eLabel} />
		{/if}
	</div>
	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => (editModal = false)}
		>
			Cancel
		</button>
		<button
			type="button"
			class="rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-60"
			onclick={saveEdit}
			disabled={savingEdit || !eName.trim()}
		>
			{savingEdit ? 'Saving…' : 'Save'}
		</button>
	{/snippet}
</Modal>

<!-- Add item modal -->
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
		<div class="text-sm">
			<span class="text-slate-600 dark:text-slate-400">Custom fields</span>
			<div class="mt-1">
				<CustomFieldsEditor bind:fields={iFields} />
			</div>
		</div>
		<label class="flex items-center gap-2 text-sm">
			<input type="checkbox" bind:checked={iUseLocation} class="rounded" />
			<span class="text-slate-600 dark:text-slate-400">Add a location</span>
		</label>
		{#if iUseLocation}
			<LocationPicker bind:lat={iLat} bind:lng={iLng} bind:label={iLabel} />
		{/if}
		<p class="text-xs text-slate-500">
			You can add a photo, attachments and transaction entries after creating the item.
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
		Are you sure you want to delete <strong>{collection?.name}</strong>? This will remove all items
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
			onclick={confirmDelete}
			disabled={deleting}
		>
			{deleting ? 'Deleting…' : 'Delete'}
		</button>
	{/snippet}
</Modal>
