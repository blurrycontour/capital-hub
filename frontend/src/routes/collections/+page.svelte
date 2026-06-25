<script lang="ts">
	import { onMount } from 'svelte';
	import Icon from '$lib/Icon.svelte';
	import Modal from '$lib/Modal.svelte';
	import LocationPicker from '$lib/LocationPicker.svelte';
	import CustomFieldsEditor from '$lib/CustomFieldsEditor.svelte';
	import {
		listCollections,
		createCollection,
		CURRENCIES,
		type Collection,
		type CustomField
	} from '$lib/api';

	let collections = $state<Collection[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Card vs. list view (persisted).
	const VIEW_KEY = 'ch-view-collections';
	let view = $state<'card' | 'list'>('card');

	function setView(v: 'card' | 'list') {
		view = v;
		try {
			localStorage.setItem(VIEW_KEY, v);
		} catch {
			/* ignore */
		}
	}

	// Create modal state.
	let createModal = $state(false);
	let cName = $state('');
	let cDescription = $state('');
	let cCurrency = $state('USD');
	let cLat = $state<number | null>(null);
	let cLng = $state<number | null>(null);
	let cLabel = $state('');
	let cFields = $state<CustomField[]>([]);
	let cUseLocation = $state(false);
	let saving = $state(false);

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
		} catch {
			/* ignore */
		}
	});

	function openCreate() {
		cName = '';
		cDescription = '';
		cCurrency = 'USD';
		cLat = null;
		cLng = null;
		cLabel = '';
		cFields = [];
		cUseLocation = false;
		error = '';
		createModal = true;
	}

	async function save() {
		if (!cName.trim()) return;
		saving = true;
		error = '';
		try {
			await createCollection({
				name: cName.trim(),
				description: cDescription.trim(),
				currency: cCurrency,
				locationLat: cUseLocation ? cLat : null,
				locationLng: cUseLocation ? cLng : null,
				locationLabel: cUseLocation ? cLabel.trim() : '',
				customFields: cFields.filter((f) => f.label.trim() || f.value.trim())
			});
			createModal = false;
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create collection';
		} finally {
			saving = false;
		}
	}
</script>

<section class="mx-auto max-w-5xl space-y-6">
	<header class="flex flex-wrap items-center justify-between gap-3">
		<h1 class="text-2xl font-bold">Collections</h1>
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
	{:else if view === 'list'}
		<ul class="divide-y divide-slate-200 overflow-hidden rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
			{#each collections as c (c.id)}
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
										title={`Shared by ${c.ownerName} (${c.accessLevel === 'write' ? 'can edit' : 'read only'})`}
									>
										<Icon name="users" class="h-3 w-3" />
										Shared
									</span>
								{:else if c.shareCount > 0}
									<span
										class="inline-flex items-center gap-1 rounded-full bg-violet-100 px-2 py-0.5 font-medium text-violet-700 dark:bg-violet-950/40 dark:text-violet-300"
										title={`Shared with ${c.shareCount} ${c.shareCount === 1 ? 'person' : 'people'}`}
									>
										<Icon name="users" class="h-3 w-3" />
										Shared
									</span>
								{:else}
									<span
										class="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 font-medium text-slate-500 dark:bg-slate-800 dark:text-slate-400"
										title="Private — only visible to you"
									>
										<Icon name="user" class="h-3 w-3" />
										Private
									</span>
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
			{#each collections as c (c.id)}
				<li>
					<a
						href={`/collections/${c.id}`}
						class="flex h-full flex-col rounded-lg border border-slate-200 p-4 transition hover:border-sky-400 hover:shadow-sm dark:border-slate-800 dark:hover:border-sky-600"
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
									title={`Shared by ${c.ownerName} (${c.accessLevel === 'write' ? 'can edit' : 'read only'})`}
								>
									<Icon name="users" class="h-3 w-3" />
									Shared
								</span>
							{:else if c.shareCount > 0}
								<span
									class="inline-flex items-center gap-1 rounded-full bg-violet-100 px-2 py-0.5 font-medium text-violet-700 dark:bg-violet-950/40 dark:text-violet-300"
									title={`Shared with ${c.shareCount} ${c.shareCount === 1 ? 'person' : 'people'}`}
								>
									<Icon name="users" class="h-3 w-3" />
									Shared
								</span>
							{:else}
								<span
									class="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 font-medium text-slate-500 dark:bg-slate-800 dark:text-slate-400"
									title="Private — only visible to you"
								>
									<Icon name="user" class="h-3 w-3" />
									Private
								</span>
							{/if}
						</div>
					</a>
				</li>
			{/each}
		</ul>
	{/if}
</section>

<!-- New collection modal -->
<Modal title="New collection" bind:open={createModal}>
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
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Currency</span>
			<select
				bind:value={cCurrency}
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
				<CustomFieldsEditor bind:fields={cFields} />
			</div>
		</div>

		<label class="flex items-center gap-2 text-sm">
			<input type="checkbox" bind:checked={cUseLocation} class="rounded" />
			<span class="text-slate-600 dark:text-slate-400">Add a location</span>
		</label>
		{#if cUseLocation}
			<LocationPicker bind:lat={cLat} bind:lng={cLng} bind:label={cLabel} />
		{/if}
	</div>
	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => (createModal = false)}
		>
			Cancel
		</button>
		<button
			type="button"
			class="rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-60"
			onclick={save}
			disabled={saving || !cName.trim()}
		>
			{saving ? 'Saving…' : 'Create'}
		</button>
	{/snippet}
</Modal>
