<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import Icon from '$lib/Icon.svelte';
	import Modal from '$lib/Modal.svelte';
	import LocationPicker from '$lib/LocationPicker.svelte';
	import CustomFieldsEditor from '$lib/CustomFieldsEditor.svelte';
	import MapView from '$lib/MapView.svelte';
	import { breadcrumbs } from '$lib/breadcrumb.svelte';
	import {
		getItem,
		updateItem,
		deleteItem,
		uploadItemImage,
		uploadItemAttachment,
		uploadEntryAttachment,
		getItemStats,
		getCollection,
		listEntries,
		createEntry,
		updateEntry,
		deleteEntry,
		formatCurrency,
		type Item,
		type Entry,
		type Stats,
		type Collection,
		type EntryInput,
		type CustomField
	} from '$lib/api';

	const itemId = $derived(Number($page.params.id));

	let item = $state<Item | null>(null);
	let collection = $state<Collection | null>(null);
	let entries = $state<Entry[]>([]);
	let stats = $state<Stats | null>(null);
	let loading = $state(true);
	let error = $state('');

	const currency = $derived(collection?.currency ?? 'USD');

	// Edit item modal
	let editModal = $state(false);
	let eName = $state('');
	let eDescription = $state('');
	let eLat = $state<number | null>(null);
	let eLng = $state<number | null>(null);
	let eLabel = $state('');
	let eFields = $state<CustomField[]>([]);
	let savingItem = $state(false);

	let deleteItemModal = $state(false);
	let deletingItem = $state(false);

	// Image upload
	let fileInput = $state<HTMLInputElement | null>(null);
	let uploading = $state(false);

	// Item attachment upload
	let attachInput = $state<HTMLInputElement | null>(null);
	let uploadingAttachment = $state(false);

	// Entry modal
	let entryModal = $state(false);
	let editingEntry = $state<Entry | null>(null);
	let enName = $state('');
	let enAmount = $state(0);
	let enNote = $state('');
	let enDate = $state('');
	let savingEntry = $state(false);
	let entryAttachInput = $state<HTMLInputElement | null>(null);
	let uploadingEntryAttachment = $state(false);

	let deleteEntryTarget = $state<Entry | null>(null);
	let deletingEntry = $state(false);

	let metadataModal = $state(false);

	const itemMarkers = $derived.by(() =>
		item && item.locationLat != null && item.locationLng != null
			? [
					{
						lat: item.locationLat,
						lng: item.locationLng,
						label: item.locationLabel || item.name
					}
				]
			: []
	);

	async function loadAll() {
		[item, entries, stats] = await Promise.all([
			getItem(itemId),
			listEntries(itemId),
			getItemStats(itemId)
		]);
		breadcrumbs.set(`/items/${itemId}`, item.name);
		try {
			collection = await getCollection(item.collectionId);
		} catch {
			collection = null;
		}
	}

	onMount(async () => {
		try {
			await loadAll();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load item';
		} finally {
			loading = false;
		}
	});

	onDestroy(() => breadcrumbs.clear(`/items/${itemId}`));

	function openEdit() {
		if (!item) return;
		eName = item.name;
		eDescription = item.description;
		eLat = item.locationLat;
		eLng = item.locationLng;
		eLabel = item.locationLabel;
		eFields = item.customFields.map((f) => ({ ...f }));
		editModal = true;
	}

	async function saveItem() {
		if (!item || !eName.trim()) return;
		savingItem = true;
		error = '';
		try {
			item = await updateItem(item.id, {
				name: eName.trim(),
				description: eDescription.trim(),
				locationLat: eLat,
				locationLng: eLng,
				locationLabel: eLabel.trim(),
				attachments: item.attachments,
				customFields: eFields.filter((f) => f.label.trim() || f.value.trim())
			});
			breadcrumbs.set(`/items/${item.id}`, item.name);
			editModal = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save item';
		} finally {
			savingItem = false;
		}
	}

	async function confirmDeleteItem() {
		if (!item) return;
		deletingItem = true;
		try {
			const collectionId = item.collectionId;
			await deleteItem(item.id);
			await goto(`/collections/${collectionId}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete item';
			deletingItem = false;
		}
	}

	async function onFileChange(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file || !item) return;
		uploading = true;
		error = '';
		try {
			item = await uploadItemImage(item.id, file);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to upload image';
		} finally {
			uploading = false;
			input.value = '';
		}
	}

	async function onAttachmentChange(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file || !item) return;
		uploadingAttachment = true;
		error = '';
		try {
			item = await uploadItemAttachment(item.id, file);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to upload attachment';
		} finally {
			uploadingAttachment = false;
			input.value = '';
		}
	}

	function todayISO() {
		return new Date().toISOString().slice(0, 10);
	}

	function openCreateEntry() {
		editingEntry = null;
		enName = '';
		enAmount = 0;
		enNote = '';
		enDate = todayISO();
		entryModal = true;
	}

	function openEditEntry(entry: Entry) {
		editingEntry = entry;
		enName = entry.name;
		enAmount = entry.amount;
		enNote = entry.note;
		enDate = entry.occurredOn ? entry.occurredOn.slice(0, 10) : todayISO();
		entryModal = true;
	}

	async function saveEntry() {
		if (!item) return;
		savingEntry = true;
		error = '';
		const payload: EntryInput = {
			name: enName.trim(),
			amount: Number(enAmount),
			note: enNote.trim(),
			occurredOn: enDate,
			attachments: editingEntry?.attachments ?? []
		};
		try {
			if (editingEntry) {
				editingEntry = await updateEntry(editingEntry.id, payload);
			} else {
				editingEntry = await createEntry(item.id, payload);
			}
			[entries, stats] = await Promise.all([listEntries(item.id), getItemStats(item.id)]);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save entry';
		} finally {
			savingEntry = false;
		}
	}

	async function onEntryAttachmentChange(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file || !editingEntry) return;
		uploadingEntryAttachment = true;
		error = '';
		try {
			editingEntry = await uploadEntryAttachment(editingEntry.id, file);
			if (item) entries = await listEntries(item.id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to upload attachment';
		} finally {
			uploadingEntryAttachment = false;
			input.value = '';
		}
	}

	async function confirmDeleteEntry() {
		if (!deleteEntryTarget || !item) return;
		deletingEntry = true;
		try {
			await deleteEntry(deleteEntryTarget.id);
			deleteEntryTarget = null;
			[entries, stats] = await Promise.all([listEntries(item.id), getItemStats(item.id)]);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete entry';
		} finally {
			deletingEntry = false;
		}
	}
</script>

<section class="mx-auto max-w-4xl space-y-6">
	{#if loading}
		<p class="text-sm text-slate-500">Loading…</p>
	{:else if error && !item}
		<div
			class="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200"
		>
			{error}
		</div>
	{:else if item}
		<a
			href={`/collections/${item.collectionId}`}
			class="inline-flex items-center gap-1 text-sm text-slate-500 hover:text-slate-700 dark:hover:text-slate-300"
		>
			<Icon name="chevron-left" class="h-4 w-4" />
			{collection ? collection.name : 'Back to collection'}
		</a>

		{#if error}
			<div
				class="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200"
			>
				{error}
			</div>
		{/if}

		<div class="grid gap-6 md:grid-cols-[16rem_1fr]">
			<!-- Image + upload -->
			<div class="space-y-2">
				<div
					class="flex aspect-square items-center justify-center overflow-hidden rounded-lg border border-slate-200 bg-slate-100 dark:border-slate-800 dark:bg-slate-800"
				>
					{#if item.imagePath}
						<img src={item.imagePath} alt={item.name} class="h-full w-full object-cover" />
					{:else}
						<Icon name="photo" class="h-12 w-12 text-slate-400" />
					{/if}
				</div>
				<input
					bind:this={fileInput}
					type="file"
					accept="image/*"
					class="hidden"
					onchange={onFileChange}
				/>
				<button
					type="button"
					class="inline-flex w-full items-center justify-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm hover:bg-slate-100 disabled:opacity-60 dark:border-slate-700 dark:hover:bg-slate-800"
					onclick={() => fileInput?.click()}
					disabled={uploading}
				>
					<Icon name="photo" class="h-4 w-4" />
					{uploading ? 'Uploading…' : item.imagePath ? 'Replace photo' : 'Add photo'}
				</button>
			</div>

			<!-- Details -->
			<div class="space-y-3">
				<div class="flex flex-wrap items-start justify-between gap-2">
					<h1 class="text-2xl font-bold">{item.name}</h1>
					<div class="flex items-center gap-2">
						<button
							type="button"
							class="rounded-md p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-slate-800"
							aria-label="Item metadata"
							onclick={() => (metadataModal = true)}
						>
							<Icon name="info" class="h-5 w-5" />
						</button>
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
							onclick={() => (deleteItemModal = true)}
						>
							<Icon name="trash" class="h-4 w-4" />
						</button>
					</div>
				</div>

				{#if item.description}
					<p class="text-sm text-slate-600 dark:text-slate-400">{item.description}</p>
				{/if}

				{#if item.locationLat != null && item.locationLng != null}
					<p class="inline-flex items-center gap-1.5 text-sm text-slate-500">
						<Icon name="map-pin" class="h-4 w-4" />
						{item.locationLabel || `${item.locationLat.toFixed(5)}, ${item.locationLng.toFixed(5)}`}
					</p>
				{/if}

				<!-- Custom fields -->
				{#if item.customFields.length > 0}
					<dl class="grid gap-x-6 gap-y-1 sm:grid-cols-2">
						{#each item.customFields as field (field.label + field.value)}
							<div
								class="flex justify-between gap-3 border-b border-slate-100 py-1 text-sm dark:border-slate-800"
							>
								<dt class="text-slate-500">{field.label}</dt>
								<dd class="font-medium">{field.value}</dd>
							</div>
						{/each}
					</dl>
				{/if}

				<!-- Stats -->
				{#if stats}
					<div class="grid gap-3 sm:grid-cols-3">
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
		</div>

		<!-- Location map -->
		{#if itemMarkers.length > 0}
			<div>
				<h2 class="mb-2 text-lg font-semibold">Location</h2>
				<MapView markers={itemMarkers} height="h-64" />
			</div>
		{/if}

		<!-- Attachments -->
		<div>
			<div class="mb-2 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Attachments</h2>
				<input
					bind:this={attachInput}
					type="file"
					class="hidden"
					onchange={onAttachmentChange}
				/>
				<button
					type="button"
					class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm hover:bg-slate-100 disabled:opacity-60 dark:border-slate-700 dark:hover:bg-slate-800"
					onclick={() => attachInput?.click()}
					disabled={uploadingAttachment}
				>
					<Icon name="plus" class="h-4 w-4" />
					{uploadingAttachment ? 'Uploading…' : 'Add attachment'}
				</button>
			</div>
			{#if item.attachments.length === 0}
				<p class="text-sm text-slate-500">No attachments.</p>
			{:else}
				<ul class="flex flex-wrap gap-2">
					{#each item.attachments as att (att.path)}
						<li>
							<a
								href={att.path}
								target="_blank"
								rel="noopener noreferrer"
								class="inline-flex items-center gap-1.5 rounded-md border border-slate-200 px-2.5 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-800 dark:hover:bg-slate-800"
							>
								<Icon name="photo" class="h-4 w-4 text-slate-400" />
								{att.name}
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</div>

		<!-- Entries -->
		<div class="flex items-center justify-between">
			<h2 class="text-lg font-semibold">Entries</h2>
			<button
				type="button"
				class="inline-flex items-center gap-1.5 rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700"
				onclick={openCreateEntry}
			>
				<Icon name="plus" class="h-4 w-4" /> Add entry
			</button>
		</div>

		{#if entries.length === 0}
			<p
				class="rounded-lg border border-dashed border-slate-300 p-6 text-center text-sm text-slate-500 dark:border-slate-700"
			>
				No entries yet. Add a valuation, purchase or sale to start tracking value.
			</p>
		{:else}
			<div class="overflow-x-auto rounded-lg border border-slate-200 dark:border-slate-800">
				<table class="w-full text-sm">
					<thead
						class="bg-slate-50 text-left text-xs uppercase text-slate-500 dark:bg-slate-800/50"
					>
						<tr>
							<th class="px-3 py-2 font-medium">Date</th>
							<th class="px-3 py-2 font-medium">Name</th>
							<th class="px-3 py-2 font-medium">Note</th>
							<th class="px-3 py-2 text-right font-medium">Amount</th>
							<th class="px-3 py-2"></th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100 dark:divide-slate-800">
						{#each entries as entry (entry.id)}
							<tr class="hover:bg-slate-50 dark:hover:bg-slate-800/40">
								<td class="px-3 py-2 whitespace-nowrap">{entry.occurredOn}</td>
								<td class="px-3 py-2 font-medium">
									{entry.name || '—'}
									{#if entry.attachments.length > 0}
										<span class="ml-1 inline-flex items-center gap-0.5 text-xs text-slate-400">
											<Icon name="photo" class="h-3.5 w-3.5" />
											{entry.attachments.length}
										</span>
									{/if}
								</td>
								<td class="px-3 py-2 text-slate-600 dark:text-slate-400">{entry.note}</td>
								<td class="px-3 py-2 text-right font-medium whitespace-nowrap">
									{formatCurrency(entry.amount, entry.currency)}
								</td>
								<td class="px-3 py-2">
									<div class="flex items-center justify-end gap-1">
										<button
											type="button"
											class="rounded p-1 text-slate-500 hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-slate-700"
											aria-label="Edit entry"
											onclick={() => openEditEntry(entry)}
										>
											<Icon name="pencil" class="h-4 w-4" />
										</button>
										<button
											type="button"
											class="rounded p-1 text-rose-500 hover:bg-rose-100 dark:hover:bg-rose-950/40"
											aria-label="Delete entry"
											onclick={() => (deleteEntryTarget = entry)}
										>
											<Icon name="trash" class="h-4 w-4" />
										</button>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
</section>

<!-- Edit item modal -->
<Modal title="Edit item" bind:open={editModal}>
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
				rows="2"
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
			></textarea>
		</label>
		<div class="text-sm">
			<span class="text-slate-600 dark:text-slate-400">Custom fields</span>
			<div class="mt-1">
				<CustomFieldsEditor bind:fields={eFields} />
			</div>
		</div>
		<div>
			<span class="text-sm text-slate-600 dark:text-slate-400">Location</span>
			{#if editModal}
				<LocationPicker bind:lat={eLat} bind:lng={eLng} bind:label={eLabel} />
			{/if}
		</div>
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
			onclick={saveItem}
			disabled={savingItem || !eName.trim()}
		>
			{savingItem ? 'Saving…' : 'Save'}
		</button>
	{/snippet}
</Modal>

<!-- Entry modal -->
<Modal title={editingEntry ? 'Edit entry' : 'Add entry'} bind:open={entryModal}>
	<div class="space-y-3">
		<div class="grid grid-cols-2 gap-3">
			<label class="block text-sm">
				<span class="text-slate-600 dark:text-slate-400">Name</span>
				<input
					type="text"
					bind:value={enName}
					placeholder="e.g. Purchase"
					class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
				/>
			</label>
			<label class="block text-sm">
				<span class="text-slate-600 dark:text-slate-400">Date</span>
				<input
					type="date"
					bind:value={enDate}
					class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
				/>
			</label>
		</div>
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Amount ({currency})</span>
			<input
				type="number"
				step="any"
				bind:value={enAmount}
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
			/>
		</label>
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Note</span>
			<textarea
				bind:value={enNote}
				rows="2"
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
			></textarea>
		</label>

		<!-- Entry attachments (available once the entry exists) -->
		<div class="text-sm">
			<div class="flex items-center justify-between">
				<span class="text-slate-600 dark:text-slate-400">Attachments</span>
				<input
					bind:this={entryAttachInput}
					type="file"
					class="hidden"
					onchange={onEntryAttachmentChange}
				/>
				<button
					type="button"
					class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2 py-1 text-xs hover:bg-slate-100 disabled:opacity-60 dark:border-slate-700 dark:hover:bg-slate-800"
					onclick={() => entryAttachInput?.click()}
					disabled={!editingEntry || uploadingEntryAttachment}
					title={editingEntry ? '' : 'Save the entry first to add attachments'}
				>
					<Icon name="plus" class="h-3.5 w-3.5" />
					{uploadingEntryAttachment ? 'Uploading…' : 'Add'}
				</button>
			</div>
			{#if !editingEntry}
				<p class="mt-1 text-xs text-slate-500">Save the entry first to attach files.</p>
			{:else if editingEntry.attachments.length === 0}
				<p class="mt-1 text-xs text-slate-500">No attachments.</p>
			{:else}
				<ul class="mt-1 flex flex-wrap gap-2">
					{#each editingEntry.attachments as att (att.path)}
						<li>
							<a
								href={att.path}
								target="_blank"
								rel="noopener noreferrer"
								class="inline-flex items-center gap-1.5 rounded-md border border-slate-200 px-2 py-1 text-xs hover:bg-slate-100 dark:border-slate-800 dark:hover:bg-slate-800"
							>
								<Icon name="photo" class="h-3.5 w-3.5 text-slate-400" />
								{att.name}
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</div>
	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => (entryModal = false)}
		>
			Close
		</button>
		<button
			type="button"
			class="rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-60"
			onclick={saveEntry}
			disabled={savingEntry}
		>
			{savingEntry ? 'Saving…' : 'Save'}
		</button>
	{/snippet}
</Modal>

<!-- Metadata modal -->
<Modal title="Item metadata" bind:open={metadataModal}>
	{#if item}
		<dl class="space-y-2 text-sm">
			<div class="flex justify-between gap-4">
				<dt class="text-slate-500">Created</dt>
				<dd class="text-right">{new Date(item.createdAt).toLocaleString()}</dd>
			</div>
			<div class="flex justify-between gap-4">
				<dt class="text-slate-500">Created by</dt>
				<dd class="text-right">{item.createdBy || '—'}</dd>
			</div>
			<div class="flex justify-between gap-4">
				<dt class="text-slate-500">Last updated</dt>
				<dd class="text-right">{new Date(item.updatedAt).toLocaleString()}</dd>
			</div>
			<div class="flex justify-between gap-4">
				<dt class="text-slate-500">Updated by</dt>
				<dd class="text-right">{item.updatedBy || '—'}</dd>
			</div>
		</dl>
	{/if}
</Modal>

<!-- Delete item modal -->
<Modal title="Delete item" bind:open={deleteItemModal}>
	<p class="text-sm text-slate-600 dark:text-slate-400">
		Delete <strong>{item?.name}</strong> and all of its entries? This cannot be undone.
	</p>
	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => (deleteItemModal = false)}
		>
			Cancel
		</button>
		<button
			type="button"
			class="rounded-md bg-rose-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-rose-700 disabled:opacity-60"
			onclick={confirmDeleteItem}
			disabled={deletingItem}
		>
			{deletingItem ? 'Deleting…' : 'Delete'}
		</button>
	{/snippet}
</Modal>

<!-- Delete entry modal -->
<Modal
	title="Delete entry"
	open={deleteEntryTarget !== null}
	onclose={() => (deleteEntryTarget = null)}
>
	<p class="text-sm text-slate-600 dark:text-slate-400">
		Delete this entry? This cannot be undone.
	</p>
	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => (deleteEntryTarget = null)}
		>
			Cancel
		</button>
		<button
			type="button"
			class="rounded-md bg-rose-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-rose-700 disabled:opacity-60"
			onclick={confirmDeleteEntry}
			disabled={deletingEntry}
		>
			{deletingEntry ? 'Deleting…' : 'Delete'}
		</button>
	{/snippet}
</Modal>
