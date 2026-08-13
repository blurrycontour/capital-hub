<script lang="ts">
	// The "add item" form, shared by the items page and quick add. The
	// collection is chosen here because neither entry point implies one.
	import Modal from '$lib/Modal.svelte';
	import CustomFieldsEditor from '$lib/CustomFieldsEditor.svelte';
	import LocationPicker from '$lib/LocationPicker.svelte';
	import { canEditContents } from '$lib/access';
	import { createItem, type Collection, type CustomField, type Item } from '$lib/api';

	let {
		open = $bindable(false),
		collections = [],
		loadingCollections = false,
		oncreated
	}: {
		open?: boolean;
		collections?: Collection[];
		loadingCollections?: boolean;
		oncreated?: (created: Item) => void | Promise<void>;
	} = $props();

	// Only collections the user can add items to.
	const writable = $derived(collections.filter((c) => canEditContents(c.accessLevel)));

	let collectionId = $state<number | null>(null);
	let name = $state('');
	let description = $state('');
	let lat = $state<number | null>(null);
	let lng = $state<number | null>(null);
	let label = $state('');
	let fields = $state<CustomField[]>([]);
	let useLocation = $state(false);
	let saving = $state(false);
	let error = $state('');

	let wasOpen = $state(false);
	$effect(() => {
		if (open && !wasOpen) {
			// Preselect when there is only one choice, saving a tap.
			collectionId = writable.length === 1 ? writable[0].id : null;
			name = '';
			description = '';
			lat = null;
			lng = null;
			label = '';
			fields = [];
			useLocation = false;
			error = '';
		}
		wasOpen = open;
	});

	async function save() {
		if (collectionId == null || !name.trim()) return;
		saving = true;
		error = '';
		try {
			const created = await createItem(collectionId, {
				name: name.trim(),
				description: description.trim(),
				locationLat: useLocation ? lat : null,
				locationLng: useLocation ? lng : null,
				locationLabel: useLocation ? label.trim() : '',
				images: [],
				attachments: [],
				customFields: fields.filter((f) => f.label.trim() || f.value.trim())
			});
			open = false;
			await oncreated?.(created);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create item';
		} finally {
			saving = false;
		}
	}
</script>

<Modal title="Add item" bind:open>
	<div class="space-y-3">
		{#if error}
			<div
				class="rounded-md border border-rose-300 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-800 dark:bg-rose-950/40 dark:text-rose-300"
				role="alert"
			>
				{error}
			</div>
		{/if}
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Collection</span>
			{#if loadingCollections}
				<p class="mt-1 text-sm text-slate-500">Loading collections…</p>
			{:else if writable.length === 0}
				<p class="mt-1 text-sm text-slate-500">
					You don’t have any collections you can add items to.
					<a href="/collections" class="text-sky-600 hover:underline dark:text-sky-400"
						>Create one first.</a
					>
				</p>
			{:else}
				<select
					bind:value={collectionId}
					class="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-900"
				>
					<option value={null} disabled selected>Select a collection…</option>
					{#each writable as c (c.id)}
						<option value={c.id}>{c.name} ({c.currency})</option>
					{/each}
				</select>
			{/if}
		</label>
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Name</span>
			<input
				type="text"
				bind:value={name}
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
				placeholder="e.g. 1921 Silver Dollar"
			/>
		</label>
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Description</span>
			<textarea
				bind:value={description}
				rows="3"
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
			></textarea>
		</label>
		<div class="text-sm">
			<span class="text-slate-600 dark:text-slate-400">Custom fields</span>
			<div class="mt-1">
				<CustomFieldsEditor bind:fields />
			</div>
		</div>
		<label class="flex items-center gap-2 text-sm">
			<input type="checkbox" bind:checked={useLocation} class="rounded" />
			<span class="text-slate-600 dark:text-slate-400">Add a location</span>
		</label>
		{#if useLocation}
			<LocationPicker bind:lat bind:lng bind:label />
		{/if}
		<p class="text-xs text-slate-500">
			You can add a photo, attachments and transaction entries after creating the item.
		</p>
	</div>
	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => (open = false)}
		>
			Cancel
		</button>
		<button
			type="button"
			class="rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-60"
			onclick={save}
			disabled={saving || collectionId == null || !name.trim()}
		>
			{saving ? 'Saving…' : 'Create'}
		</button>
	{/snippet}
</Modal>
