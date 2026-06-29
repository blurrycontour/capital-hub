<script lang="ts">
	import { onMount, onDestroy, tick } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import Icon from '$lib/Icon.svelte';
	import Modal from '$lib/Modal.svelte';
	import LocationPicker from '$lib/LocationPicker.svelte';
	import CustomFieldsEditor from '$lib/CustomFieldsEditor.svelte';
	import MapView from '$lib/MapView.svelte';
	import ImageGallery from '$lib/ImageGallery.svelte';
	import ConfirmDeleteModal from '$lib/ConfirmDeleteModal.svelte';
	import Dropdown from '$lib/Dropdown.svelte';
	import { breadcrumbs } from '$lib/breadcrumb.svelte';
	import {
		getItem,
		updateItem,
		deleteItem,
		moveItem,
		listCollections,
		uploadItemImage,
		deleteItemImage,
		setItemCover,
		uploadItemAttachment,
		deleteItemAttachment,
		uploadEntryAttachment,
		deleteEntryAttachment,
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

	const itemId = $derived(Number($page.params.itemId));

	let item = $state<Item | null>(null);
	let collection = $state<Collection | null>(null);
	let entries = $state<Entry[]>([]);
	let stats = $state<Stats | null>(null);
	let loading = $state(true);
	let error = $state('');

	const currency = $derived(collection?.currency ?? 'USD');

	// Write access derived from the parent collection's access level.
	const canWrite = $derived(
		collection?.accessLevel === 'owner' || collection?.accessLevel === 'write'
	);

	// Edit item modal
	let editModal = $state(false);
	let eName = $state('');
	let eDescription = $state('');
	let eLat = $state<number | null>(null);
	let eLng = $state<number | null>(null);
	let eLabel = $state('');
	let eUseLocation = $state(false);
	let eFields = $state<CustomField[]>([]);
	let savingItem = $state(false);

	let deleteItemModal = $state(false);
	let deletingItem = $state(false);

	// Move-to-collection modal.
	let moveModal = $state(false);
	let moveTargets = $state<Collection[]>([]);
	let moveTargetId = $state<number | null>(null);
	let movingItem = $state(false);
	let moveError = $state('');
	let loadingMoveTargets = $state(false);

	// Image upload
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

	// Expanded entry row (shows full note + edit/delete actions).
	let expandedEntryId = $state<number | null>(null);

	function toggleExpand(id: number) {
		expandedEntryId = expandedEntryId === id ? null : id;
	}

	// Entry sorting
	let sort = $state<{ key: 'occurredOn' | 'name' | 'amount'; dir: 'asc' | 'desc' }>({
		key: 'occurredOn',
		dir: 'desc'
	});

	const sortedEntries = $derived.by(() => {
		const list = [...entries];
		const { key, dir } = sort;
		list.sort((a, b) => {
			let cmp = 0;
			if (key === 'amount') {
				cmp = a.amount - b.amount;
			} else if (key === 'name') {
				cmp = a.name.localeCompare(b.name);
			} else {
				cmp = (a.occurredOn || '').localeCompare(b.occurredOn || '');
			}
			return dir === 'asc' ? cmp : -cmp;
		});
		return list;
	});

	function toggleSort(key: 'occurredOn' | 'name' | 'amount') {
		if (sort.key === key) {
			sort = { key, dir: sort.dir === 'asc' ? 'desc' : 'asc' };
		} else {
			sort = { key, dir: 'asc' };
		}
	}

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
		try {
			collection = await getCollection(item.collectionId);
		} catch {
			collection = null;
		}
		updateTrail();
	}

	// Nest the item under its collection: Collections > <collection> > Items > <item>.
	// Must be async and await tick() before calling setTrail. Calling setTrail
	// synchronously in the same flush as a modal close triggers a layout re-render
	// that can race with Leaflet teardown (LocationPicker onDestroy rAF callbacks)
	// and freeze the main thread. tick() ensures the current flush — including any
	// DOM mutations and Leaflet cleanup — fully completes before the trail updates.
	async function updateTrail() {
		await tick();
		if (!item) return;
		const collectionId = item.collectionId;
		breadcrumbs.setTrail([
			{ label: 'Collections', href: '/collections' },
			{ label: collection?.name ?? 'Collection', href: `/collections/${collectionId}` },
			{ label: item.name, href: `/collections/${collectionId}/items/${itemId}` }
		]);
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

	onDestroy(() => breadcrumbs.clearTrail());

	function openEdit() {
		if (!item) return;
		eName = item.name;
		eDescription = item.description;
		eLat = item.locationLat;
		eLng = item.locationLng;
		eLabel = item.locationLabel;
		eUseLocation = item.locationLat != null && item.locationLng != null;
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
				locationLat: eUseLocation ? eLat : null,
				locationLng: eUseLocation ? eLng : null,
				locationLabel: eUseLocation ? eLabel.trim() : '',
				images: item.images,
				attachments: item.attachments,
				customFields: eFields.filter((f) => f.label.trim() || f.value.trim())
			});
			editModal = false;
			updateTrail(); // fire-and-forget: runs after tick(), past the modal-close flush
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

	async function openMove() {
		if (!item) return;
		moveModal = true;
		moveError = '';
		moveTargetId = null;
		loadingMoveTargets = true;
		try {
			const all = await listCollections();
			// Only collections the user can write to, excluding the current one.
			moveTargets = all.filter(
				(c) =>
					c.id !== item!.collectionId &&
					(c.accessLevel === 'owner' || c.accessLevel === 'write')
			);
		} catch (e) {
			moveError = e instanceof Error ? e.message : 'Failed to load collections';
		} finally {
			loadingMoveTargets = false;
		}
	}

	// The currency the entries will be re-stamped with after a move, when it
	// differs from the current collection's currency.
	const moveTarget = $derived(moveTargets.find((c) => c.id === moveTargetId) ?? null);
	const moveCurrencyChanges = $derived(
		moveTarget != null && collection != null && moveTarget.currency !== collection.currency
	);

	async function confirmMove() {
		if (!item || moveTargetId == null) return;
		movingItem = true;
		moveError = '';
		try {
			const updated = await moveItem(item.id, moveTargetId);
			// Reflect the move first, then close the modal on the next tick (see the
			// note in saveItem for why the ordering matters).
			item = updated;
			collection = await getCollection(updated.collectionId);
			moveModal = false;
			// Keep the item open but reflect its new collection in the URL.
			await goto(`/collections/${updated.collectionId}/items/${updated.id}`, {
				replaceState: true,
				noScroll: true,
				keepFocus: true
			});
			updateTrail();
		} catch (e) {
			moveError = e instanceof Error ? e.message : 'Failed to move item';
		} finally {
			movingItem = false;
		}
	}

	async function onAddImage(file: File) {
		if (!item) return;
		uploading = true;
		error = '';
		try {
			item = await uploadItemImage(item.id, file);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to upload image';
		} finally {
			uploading = false;
		}
	}

	// Path of the image awaiting delete confirmation (null = no prompt).
	let deleteImageTarget = $state<string | null>(null);

	async function onSetCover(path: string) {
		if (!item) return;
		uploading = true;
		error = '';
		try {
			item = await setItemCover(item.id, path);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to set display picture';
		} finally {
			uploading = false;
		}
	}

	async function confirmDeleteImage() {
		if (!item || deleteImageTarget == null) return;
		uploading = true;
		error = '';
		try {
			item = await deleteItemImage(item.id, deleteImageTarget);
			deleteImageTarget = null;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete image';
		} finally {
			uploading = false;
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

	// Path of the attachment awaiting delete confirmation (null = no prompt).
	let deleteAttachmentTarget = $state<string | null>(null);

	async function confirmDeleteItemAttachment() {
		if (!item || deleteAttachmentTarget == null) return;
		uploadingAttachment = true;
		error = '';
		try {
			item = await deleteItemAttachment(item.id, deleteAttachmentTarget);
			deleteAttachmentTarget = null;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete attachment';
		} finally {
			uploadingAttachment = false;
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
			entryModal = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save entry';
		} finally {
			savingEntry = false;
		}
	}

	async function onEntryAttachmentChange(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file || !item) return;
		uploadingEntryAttachment = true;
		error = '';
		try {
			// During the "add entry" flow there is no entry yet to attach to, so
			// persist the entry first (using the values entered so far), then upload
			// the attachment against the freshly created entry.
			if (!editingEntry) {
				editingEntry = await createEntry(item.id, {
					name: enName.trim(),
					amount: Number(enAmount),
					note: enNote.trim(),
					occurredOn: enDate,
					attachments: []
				});
			}
			editingEntry = await uploadEntryAttachment(editingEntry.id, file);
			[entries, stats] = await Promise.all([listEntries(item.id), getItemStats(item.id)]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to upload attachment';
		} finally {
			uploadingEntryAttachment = false;
			input.value = '';
		}
	}

	async function onDeleteEntryAttachment(path: string) {
		if (!editingEntry || !item) return;
		uploadingEntryAttachment = true;
		error = '';
		try {
			editingEntry = await deleteEntryAttachment(editingEntry.id, path);
			[entries, stats] = await Promise.all([listEntries(item.id), getItemStats(item.id)]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete attachment';
		} finally {
			uploadingEntryAttachment = false;
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
		{#if error}
			<div
				class="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200"
			>
				{error}
			</div>
		{/if}

		<div class="space-y-3">
			<!-- Details -->
			<div class="space-y-3">
				<div class="flex items-center justify-between gap-2">
					<div class="flex min-w-0 items-center gap-2">
						<span
							class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300"
						>
							<Icon name="cube" class="h-5 w-5" />
						</span>
						<span class="text-xs font-semibold uppercase tracking-wide text-slate-400"
							>Item</span
						>
					</div>
					<div class="flex shrink-0 items-center gap-2">
						<button
							type="button"
							class="rounded-md p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-slate-800"
							aria-label="Item metadata"
							onclick={() => (metadataModal = true)}
						>
							<Icon name="info" class="h-5 w-5" />
						</button>
						{#if canWrite}
							<Dropdown label="Options">
								<button
									type="button"
									role="menuitem"
									class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-slate-100 dark:hover:bg-slate-800"
									onclick={openEdit}
								>
									<Icon name="pencil" class="h-4 w-4 text-slate-500" /> Edit
								</button>
								<button
									type="button"
									role="menuitem"
									class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-slate-100 dark:hover:bg-slate-800"
									onclick={openMove}
								>
									<Icon name="arrow-right" class="h-4 w-4 text-slate-500" /> Move to…
								</button>
								<button
									type="button"
									role="menuitem"
									class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-rose-600 hover:bg-rose-50 dark:text-rose-400 dark:hover:bg-rose-950/40"
									onclick={() => (deleteItemModal = true)}
								>
									<Icon name="trash" class="h-4 w-4" /> Delete
								</button>
							</Dropdown>
						{/if}
					</div>
				</div>

				<h1 class="break-words text-2xl font-bold">{item.name}</h1>

				{#if item.description}
					<p class="text-sm break-words text-slate-600 dark:text-slate-400">{item.description}</p>
				{/if}

				{#if item.locationLat != null && item.locationLng != null}
					<p class="inline-flex items-center gap-1.5 text-sm text-slate-500">
						<Icon name="map-pin" class="h-4 w-4" />
						{item.locationLabel || `${item.locationLat.toFixed(5)}, ${item.locationLng.toFixed(5)}`}
					</p>
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

		<!-- Details (custom fields) -->
		{#if item.customFields.length > 0}
			<div class="space-y-3">
				<h2 class="mb-3 text-lg font-semibold">Details</h2>
				<dl class="grid gap-x-6 gap-y-2 sm:grid-cols-2">
					{#each item.customFields as field (field.label + field.value)}
						<div class="flex justify-between gap-3 border-b border-slate-100 py-1 text-sm dark:border-slate-800">
							<dt class="text-slate-500">{field.label}</dt>
							<dd class="font-medium whitespace-pre-wrap break-words">{field.value}</dd>
						</div>
					{/each}
				</dl>
			</div>
		{/if}

		<!-- Images -->
		<div>
			<h2 class="mb-2 text-lg font-semibold">Images</h2>
			<ImageGallery
				images={item.images}
				coverPath={item.imagePath}
				onadd={canWrite ? onAddImage : undefined}
				ondelete={canWrite ? (path) => (deleteImageTarget = path) : undefined}
				onsetcover={canWrite ? onSetCover : undefined}
				{uploading}
			/>
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
				{#if canWrite}
					<button
						type="button"
						class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm hover:bg-slate-100 disabled:opacity-60 dark:border-slate-700 dark:hover:bg-slate-800"
						onclick={() => attachInput?.click()}
						disabled={uploadingAttachment}
					>
						<Icon name="plus" class="h-4 w-4" />
						{uploadingAttachment ? 'Uploading…' : 'Add attachment'}
					</button>
				{/if}
			</div>
			{#if item.attachments.length === 0}
				<p class="text-sm text-slate-500">No attachments.</p>
			{:else}
				<ul class="flex flex-wrap gap-2">
					{#each item.attachments as att (att.path)}
						<li
							class="inline-flex items-center gap-1 rounded-md border border-slate-200 pr-1 text-sm dark:border-slate-800"
						>
							<a
								href={att.path}
								target="_self"
								title="Open attachment in same tab"
								rel="noopener noreferrer"
								class="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1.5 hover:bg-slate-100 dark:hover:bg-slate-800"
							>
								<Icon name="photo" class="h-4 w-4 text-slate-400" />
								{att.name}
							</a>
							{#if canWrite}
								<button
									type="button"
									class="rounded p-1 text-slate-400 hover:bg-rose-50 hover:text-rose-600 disabled:opacity-60 dark:hover:bg-rose-950/40 dark:hover:text-rose-400"
									aria-label="Delete attachment"
									onclick={() => (deleteAttachmentTarget = att.path)}
									disabled={uploadingAttachment}
								>
									<Icon name="close" class="h-4 w-4" />
								</button>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}
		</div>

		<!-- Entries -->
		<div class="flex items-center justify-between">
			<h2 class="text-lg font-semibold">Entries</h2>
			{#if canWrite}
				<button
					type="button"
					class="inline-flex items-center gap-1.5 rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700"
					onclick={openCreateEntry}
				>
					<Icon name="plus" class="h-4 w-4" /> Add entry
				</button>
			{/if}
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
						class="bg-slate-50 text-left text-xs text-slate-500 dark:bg-slate-800/50"
					>
						<tr>
							<th class="px-3 py-3 font-medium">
								<button
									type="button"
									class="inline-flex items-center gap-1 hover:text-slate-700 dark:hover:text-slate-300"
									onclick={() => toggleSort('occurredOn')}
								>
									Date
									{#if sort.key === 'occurredOn'}
										<Icon
											name="chevron-down"
											class={`h-3.5 w-3.5 ${sort.dir === 'asc' ? 'rotate-180' : ''}`}
										/>
									{/if}
								</button>
							</th>
							<th class="px-3 py-3 font-medium">
								<button
									type="button"
									class="inline-flex items-center gap-1 hover:text-slate-700 dark:hover:text-slate-300"
									onclick={() => toggleSort('name')}
								>
									Name
									{#if sort.key === 'name'}
										<Icon
											name="chevron-down"
											class={`h-3.5 w-3.5 ${sort.dir === 'asc' ? 'rotate-180' : ''}`}
										/>
									{/if}
								</button>
							</th>
							<th class="px-3 py-3 font-medium">
								<button
									type="button"
									class="ml-auto inline-flex items-center gap-1 hover:text-slate-700 dark:hover:text-slate-300"
									onclick={() => toggleSort('amount')}
								>
									Amount
									{#if sort.key === 'amount'}
										<Icon
											name="chevron-down"
											class={`h-3.5 w-3.5 ${sort.dir === 'asc' ? 'rotate-180' : ''}`}
										/>
									{/if}
								</button>
							</th>
							<th class="px-3 py-3 font-medium">Note</th>
							<th class="px-3 py-3 font-medium">
								<Icon name="photo" class="mx-auto h-4 w-4" />
								<span class="sr-only">Attachments</span>
							</th>
							<th class="w-8 px-3 py-3"></th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-100 dark:divide-slate-800">
						{#each sortedEntries as entry (entry.id)}
							{@const isOpen = expandedEntryId === entry.id}
							<tr
								class="cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-800/40"
								role="button"
								tabindex="0"
								aria-expanded={isOpen}
								onclick={() => toggleExpand(entry.id)}
								onkeydown={(e) => {
									if (e.key === 'Enter' || e.key === ' ') {
										e.preventDefault();
										toggleExpand(entry.id);
									}
								}}
							>
								<td class="px-3 py-4 whitespace-nowrap">{entry.occurredOn}</td>
								<td class="px-3 py-4 font-medium">{entry.name || '—'}</td>
								<td class="px-3 py-4 font-medium whitespace-nowrap">
									{formatCurrency(entry.amount, entry.currency)}
								</td>
								<td class="max-w-[1px] px-3 py-4 text-slate-600 dark:text-slate-400">
									<span class="block truncate">{entry.note}</span>
								</td>
								<td class="px-3 py-4 text-center text-slate-500">
									{#if entry.attachments.length > 0}
										{entry.attachments.length}
									{:else}
										<span class="text-slate-300 dark:text-slate-600">—</span>
									{/if}
								</td>
								<td class="px-3 py-4 text-right text-slate-400">
									<Icon
										name="chevron-down"
										class={`h-4 w-4 transition-transform ${isOpen ? 'rotate-180' : ''}`}
									/>
								</td>
							</tr>
							{#if isOpen}
								<tr class="bg-slate-50/60 dark:bg-slate-800/30">
									<td colspan="6" class="px-3 py-3">
										<div class="space-y-3">
											<div>
												<p class="text-xs font-medium uppercase text-slate-400">Note</p>
												<p class="mt-0.5 text-sm whitespace-pre-wrap text-slate-600 dark:text-slate-300">
													{entry.note || '—'}
												</p>
											</div>
											{#if entry.attachments.length > 0}
												<div>
													<p class="text-xs font-medium uppercase text-slate-400">Attachments</p>
													<ul class="mt-1 flex flex-wrap gap-2">
														{#each entry.attachments as att (att.path)}
															<li>
																<a
																	href={att.path}
																	target="_self"
																	title="Open attachment in same tab"
																	rel="noopener noreferrer"
																	class="inline-flex items-center gap-1.5 rounded-md border border-slate-200 px-2 py-1 text-xs hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
																>
																	<Icon name="photo" class="h-3.5 w-3.5 text-slate-400" />
																	{att.name}
																</a>
															</li>
														{/each}
													</ul>
												</div>
											{/if}
											{#if canWrite}
												<div class="flex items-center gap-2">
													<button
														type="button"
														class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
														onclick={() => openEditEntry(entry)}
													>
														<Icon name="pencil" class="h-4 w-4" /> Edit
													</button>
													<button
														type="button"
														class="inline-flex items-center gap-1.5 rounded-md border border-rose-300 px-2.5 py-1.5 text-sm text-rose-600 hover:bg-rose-50 dark:border-rose-900/60 dark:text-rose-400 dark:hover:bg-rose-950/40"
														onclick={() => (deleteEntryTarget = entry)}
													>
														<Icon name="trash" class="h-4 w-4" /> Delete
													</button>
												</div>
											{/if}
										</div>
									</td>
								</tr>
							{/if}
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
	<!-- Extra vertical space  -->
	<div class="h-10"></div>
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
			<label class="flex items-center gap-2 text-sm">
				<input type="checkbox" bind:checked={eUseLocation} class="rounded" />
				<span class="text-slate-600 dark:text-slate-400">Set a location</span>
			</label>
			{#if eUseLocation}
				<div class="mt-2">
					<LocationPicker bind:lat={eLat} bind:lng={eLng} bind:label={eLabel} />
				</div>
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
					disabled={uploadingEntryAttachment}
				>
					<Icon name="plus" class="h-3.5 w-3.5" />
					{uploadingEntryAttachment ? 'Uploading…' : 'Add'}
				</button>
			</div>
			{#if !editingEntry}
				<p class="mt-1 text-xs text-slate-500">
					Adding a file will save this entry automatically.
				</p>
			{:else if editingEntry.attachments.length === 0}
				<p class="mt-1 text-xs text-slate-500">No attachments.</p>
			{:else}
				<ul class="mt-1 flex flex-wrap gap-2">
					{#each editingEntry.attachments as att (att.path)}
						<li
							class="inline-flex items-center gap-1 rounded-md border border-slate-200 pr-0.5 text-xs dark:border-slate-800"
						>
							<p
								title={att.path}
								class="inline-flex items-center gap-1.5 rounded-md px-2 py-1 hover:bg-slate-100 dark:hover:bg-slate-800"
							>
								<Icon name="photo" class="h-3.5 w-3.5 text-slate-400" />
								{att.name}
							</p>
							<button
								type="button"
								class="rounded p-0.5 text-slate-400 hover:bg-rose-50 hover:text-rose-600 disabled:opacity-60 dark:hover:bg-rose-950/40 dark:hover:text-rose-400"
								aria-label="Delete attachment"
								onclick={() => onDeleteEntryAttachment(att.path)}
								disabled={uploadingEntryAttachment}
							>
								<Icon name="close" class="h-3.5 w-3.5" />
							</button>
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
{#if item}
	<ConfirmDeleteModal
		bind:open={deleteItemModal}
		name={item.name}
		title="Delete item"
		message="This will delete the item and all of its entries. This action cannot be undone."
		deleting={deletingItem}
		onconfirm={confirmDeleteItem}
	/>
{/if}

<!-- Move item modal -->
<Modal title="Move item" bind:open={moveModal}>
	<div class="space-y-3">
		{#if moveError}
			<div
				class="rounded-md border border-rose-300 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-800 dark:bg-rose-950/40 dark:text-rose-300"
			>
				{moveError}
			</div>
		{/if}
		{#if loadingMoveTargets}
			<p class="text-sm text-slate-500">Loading collections…</p>
		{:else if moveTargets.length === 0}
			<p class="text-sm text-slate-500">
				There are no other collections you can move this item to.
			</p>
		{:else}
			<label class="block space-y-1 text-sm">
				<span class="font-medium">Move to collection</span>
				<select
					bind:value={moveTargetId}
					class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-900"
				>
					<option value={null} disabled selected>Select a collection…</option>
					{#each moveTargets as c (c.id)}
						<option value={c.id}>{c.name} ({c.currency})</option>
					{/each}
				</select>
			</label>
			{#if moveCurrencyChanges && moveTarget}
				<div
					class="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200"
				>
					The target collection uses {moveTarget.currency}, so this item's entries will be
					changed from {collection?.currency} to {moveTarget.currency}.
				</div>
			{/if}
		{/if}
	</div>
	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-4 py-2 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => (moveModal = false)}
		>
			Cancel
		</button>
		<button
			type="button"
			class="inline-flex items-center gap-1.5 rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-500 disabled:cursor-not-allowed disabled:opacity-50"
			disabled={moveTargetId == null || movingItem}
			onclick={confirmMove}
		>
			<Icon name="arrow-right" class="h-4 w-4" />
			{movingItem ? 'Moving…' : 'Move'}
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

<!-- Delete image confirmation -->
<Modal
	title="Delete image"
	open={deleteImageTarget !== null}
	onclose={() => (deleteImageTarget = null)}
>
	<p class="text-sm text-slate-600 dark:text-slate-400">
		Delete this image? This cannot be undone.
	</p>
	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => (deleteImageTarget = null)}
		>
			Cancel
		</button>
		<button
			type="button"
			class="rounded-md bg-rose-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-rose-700 disabled:opacity-60"
			onclick={confirmDeleteImage}
			disabled={uploading}
		>
			{uploading ? 'Deleting…' : 'Delete'}
		</button>
	{/snippet}
</Modal>

<!-- Delete attachment confirmation -->
<Modal
	title="Delete attachment"
	open={deleteAttachmentTarget !== null}
	onclose={() => (deleteAttachmentTarget = null)}
>
	<p class="text-sm text-slate-600 dark:text-slate-400">
		Delete this attachment? This cannot be undone.
	</p>
	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => (deleteAttachmentTarget = null)}
		>
			Cancel
		</button>
		<button
			type="button"
			class="rounded-md bg-rose-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-rose-700 disabled:opacity-60"
			onclick={confirmDeleteItemAttachment}
			disabled={uploadingAttachment}
		>
			{uploadingAttachment ? 'Deleting…' : 'Delete'}
		</button>
	{/snippet}
</Modal>
