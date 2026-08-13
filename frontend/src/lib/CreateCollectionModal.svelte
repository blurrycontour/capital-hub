<script lang="ts">
	// The "new collection" form, shared by the collections page and quick add so
	// there is a single definition of what creating a collection looks like.
	import Modal from '$lib/Modal.svelte';
	import CustomFieldsEditor from '$lib/CustomFieldsEditor.svelte';
	import { createCollection, CURRENCIES, type Collection, type CustomField } from '$lib/api';

	let {
		open = $bindable(false),
		oncreated
	}: {
		open?: boolean;
		oncreated?: (created: Collection) => void | Promise<void>;
	} = $props();

	let name = $state('');
	let description = $state('');
	let currency = $state('EUR');
	let fields = $state<CustomField[]>([]);
	let saving = $state(false);
	let error = $state('');

	// Start from a clean form each time the dialog opens.
	let wasOpen = $state(false);
	$effect(() => {
		if (open && !wasOpen) {
			name = '';
			description = '';
			currency = 'EUR';
			fields = [];
			error = '';
		}
		wasOpen = open;
	});

	async function save() {
		if (!name.trim()) return;
		saving = true;
		error = '';
		try {
			const created = await createCollection({
				name: name.trim(),
				description: description.trim(),
				currency,
				locationLat: null,
				locationLng: null,
				locationLabel: '',
				customFields: fields.filter((f) => f.label.trim() || f.value.trim())
			});
			open = false;
			await oncreated?.(created);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create collection';
		} finally {
			saving = false;
		}
	}
</script>

<Modal title="New collection" bind:open>
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
			<span class="text-slate-600 dark:text-slate-400">Name</span>
			<input
				type="text"
				bind:value={name}
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
				placeholder="e.g. Coin collection"
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
		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Currency</span>
			<select
				bind:value={currency}
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
				<CustomFieldsEditor bind:fields />
			</div>
		</div>
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
			disabled={saving || !name.trim()}
		>
			{saving ? 'Saving…' : 'Create'}
		</button>
	{/snippet}
</Modal>
