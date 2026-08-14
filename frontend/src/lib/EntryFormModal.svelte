<script lang="ts">
	/**
	 * The add/edit entry form. Shared by the item page, where the item is fixed
	 * by the route, and by quick add, where it is chosen here — so both stay
	 * the same form rather than drifting apart.
	 *
	 * Attachment changes are staged and only applied on save, so cancelling
	 * leaves a stored entry untouched and never creates one as a side effect.
	 */
	import Icon from '$lib/Icon.svelte';
	import Modal from '$lib/Modal.svelte';
	import AttachmentList, { type AttachmentView } from '$lib/AttachmentList.svelte';
	import ItemPicker from '$lib/ItemPicker.svelte';
	import { ATTACHMENT_ACCEPT, attachmentError } from '$lib/attachments';
	import SuggestInput from '$lib/SuggestInput.svelte';
	import {
		getEntrySuggestions,
		createEntry,
		updateEntry,
		uploadEntryAttachment,
		deleteEntryAttachment,
		type Entry,
		type EntryInput,
		type ItemWithCollection
	} from '$lib/api';

	let {
		open = $bindable(false),
		entry = null,
		itemId = null,
		items = [],
		loadingItems = false,
		currency = 'EUR',
		selectedItemId = $bindable<number | null>(null),
		onsaved
	}: {
		open?: boolean;
		/** Entry being edited, or null to create a new one. */
		entry?: Entry | null;
		/** Fixed item. When null the form shows the item picker. */
		itemId?: number | null;
		/** Options for the picker; ignored when `itemId` is set. */
		items?: ItemWithCollection[];
		loadingItems?: boolean;
		/**
		 * Currency shown next to the amount. When the picker is used the parent
		 * derives this from `selectedItemId`, since an item does not carry its
		 * collection's currency.
		 */
		currency?: string;
		/** The picked item, exposed so the parent can resolve its currency. */
		selectedItemId?: number | null;
		onsaved?: (saved: Entry) => void | Promise<void>;
	} = $props();

	const needsItemPicker = $derived(itemId == null);

	let name = $state('');
	let amount = $state(0);
	let kind = $state<'debit' | 'credit'>('debit');
	let note = $state('');
	let from = $state('');
	let to = $state('');

	// Past values for the free-text fields, fetched once per dialog opening so
	// the list reflects entries added since the page loaded.
	let suggestions = $state<{ name: string[]; from: string[]; to: string[] }>({
		name: [],
		from: [],
		to: []
	});

	async function loadSuggestions() {
		try {
			const [nameValues, fromValues, toValues] = await Promise.all([
				getEntrySuggestions('name'),
				getEntrySuggestions('from'),
				getEntrySuggestions('to')
			]);
			suggestions = { name: nameValues, from: fromValues, to: toValues };
		} catch {
			// Suggestions are a convenience; the form works without them.
			suggestions = { name: [], from: [], to: [] };
		}
	}
	let date = $state('');
	let saving = $state(false);
	let error = $state('');

	let attachInput = $state<HTMLInputElement | null>(null);
	let attachmentMsg = $state('');
	let pendingFiles = $state<{ file: File; url: string }[]>([]);
	let removedPaths = $state<string[]>([]);

	const targetItemId = $derived(itemId ?? selectedItemId);

	const attachments = $derived<AttachmentView[]>([
		...(entry?.attachments ?? []).filter((a) => !removedPaths.includes(a.path)),
		...pendingFiles.map((p) => ({ name: p.file.name, path: p.url, pending: true }))
	]);

	const hasStagedChanges = $derived(pendingFiles.length > 0 || removedPaths.length > 0);

	function todayISO() {
		return new Date().toISOString().slice(0, 10);
	}

	export function clearStaged() {
		for (const p of pendingFiles) URL.revokeObjectURL(p.url);
		pendingFiles = [];
		removedPaths = [];
		attachmentMsg = '';
	}

	// Reload the fields from `entry` each time the dialog is opened, so a
	// reopened dialog never shows the previous edit's values.
	let wasOpen = $state(false);
	$effect(() => {
		if (open && !wasOpen) {
			clearStaged();
			error = '';
			selectedItemId = null;
			name = entry?.name ?? '';
			amount = entry?.amount ?? 0;
			kind = entry?.kind === 'credit' ? 'credit' : 'debit';
			note = entry?.note ?? '';
			from = entry?.from ?? '';
			to = entry?.to ?? '';
			date = entry?.occurredOn ? entry.occurredOn.slice(0, 10) : todayISO();
			void loadSuggestions();
		}
		wasOpen = open;
	});

	function onFileChange(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file) return;
		attachmentMsg = attachmentError(file);
		if (attachmentMsg) return;
		pendingFiles = [...pendingFiles, { file, url: URL.createObjectURL(file) }];
	}

	function onRemove(att: AttachmentView) {
		attachmentMsg = '';
		if (att.pending) {
			const staged = pendingFiles.find((p) => p.url === att.path);
			if (staged) URL.revokeObjectURL(staged.url);
			pendingFiles = pendingFiles.filter((p) => p.url !== att.path);
			return;
		}
		if (!removedPaths.includes(att.path)) removedPaths = [...removedPaths, att.path];
	}

	function cancel() {
		clearStaged();
		open = false;
	}

	async function save() {
		if (targetItemId == null) {
			error = 'Choose an item for this entry.';
			return;
		}
		saving = true;
		error = '';
		attachmentMsg = '';
		const payload: EntryInput = {
			name: name.trim(),
			amount: Number(amount),
			kind,
			note: note.trim(),
			from: from.trim(),
			to: to.trim(),
			occurredOn: date,
			attachments: (entry?.attachments ?? []).filter((a) => !removedPaths.includes(a.path))
		};
		try {
			let saved = entry
				? await updateEntry(entry.id, payload)
				: await createEntry(targetItemId, payload);

			// Attachment endpoints need an entry id, so they run after the save.
			for (const path of removedPaths) saved = await deleteEntryAttachment(saved.id, path);
			for (const staged of pendingFiles) saved = await uploadEntryAttachment(saved.id, staged.file);

			clearStaged();
			open = false;
			await onsaved?.(saved);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save entry';
		} finally {
			saving = false;
		}
	}
</script>

<Modal title={entry ? 'Edit entry' : 'Add entry'} bind:open onclose={clearStaged}>
	<div class="space-y-3">
		{#if error}
			<div
				class="rounded-md border border-rose-300 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-800 dark:bg-rose-950/40 dark:text-rose-300"
				role="alert"
			>
				{error}
			</div>
		{/if}

		{#if needsItemPicker}
			<div class="block text-sm">
				<span class="text-slate-600 dark:text-slate-400">Item</span>
				<div class="mt-1">
					<ItemPicker {items} bind:value={selectedItemId} loading={loadingItems} disabled={saving} />
				</div>
			</div>
		{/if}

		<div class="grid grid-cols-2 gap-3">
			<div class="block text-sm">
				<label for="entry-name" class="text-slate-600 dark:text-slate-400">Name</label>
				<SuggestInput
					id="entry-name"
					bind:value={name}
					suggestions={suggestions.name}
					placeholder="e.g. Purchase"
					disabled={saving}
				/>
			</div>
			<label class="block text-sm">
				<span class="text-slate-600 dark:text-slate-400">Date</span>
				<input
					type="date"
					bind:value={date}
					class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
				/>
			</label>
		</div>

		<fieldset class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Type</span>
			<div class="mt-1 flex gap-2">
				<label
					class={`flex flex-1 cursor-pointer items-center justify-center gap-1.5 rounded-md border px-3 py-2 text-sm font-medium transition-colors ${
						kind === 'debit'
							? 'border-rose-500 bg-rose-50 text-rose-700 dark:border-rose-500 dark:bg-rose-500/10 dark:text-rose-300'
							: 'border-slate-300 text-slate-600 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800'
					}`}
				>
					<input type="radio" name="entry-kind" value="debit" bind:group={kind} class="sr-only" />
					<Icon name="minus" class="h-4 w-4" />
					Debit
				</label>
				<label
					class={`flex flex-1 cursor-pointer items-center justify-center gap-1.5 rounded-md border px-3 py-2 text-sm font-medium transition-colors ${
						kind === 'credit'
							? 'border-emerald-500 bg-emerald-50 text-emerald-700 dark:border-emerald-500 dark:bg-emerald-500/10 dark:text-emerald-300'
							: 'border-slate-300 text-slate-600 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800'
					}`}
				>
					<input type="radio" name="entry-kind" value="credit" bind:group={kind} class="sr-only" />
					<Icon name="plus" class="h-4 w-4" />
					Credit
				</label>
			</div>
		</fieldset>

		<!-- Optional counterparty fields. Both offer previously used values. -->
		<div class="grid grid-cols-2 gap-3">
			<div class="block text-sm">
				<label for="entry-from" class="text-slate-600 dark:text-slate-400">
					From <span class="text-slate-400">(optional)</span>
				</label>
				<SuggestInput
					id="entry-from"
					bind:value={from}
					suggestions={suggestions.from}
					placeholder="e.g. Bank"
					disabled={saving}
				/>
			</div>
			<div class="block text-sm">
				<label for="entry-to" class="text-slate-600 dark:text-slate-400">
					To <span class="text-slate-400">(optional)</span>
				</label>
				<SuggestInput
					id="entry-to"
					bind:value={to}
					suggestions={suggestions.to}
					placeholder="e.g. Dealer"
					disabled={saving}
				/>
			</div>
		</div>

		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Amount ({currency})</span>
			<input
				type="number"
				step="any"
				bind:value={amount}
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
			/>
		</label>

		<label class="block text-sm">
			<span class="text-slate-600 dark:text-slate-400">Note</span>
			<textarea
				bind:value={note}
				rows="2"
				class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
			></textarea>
		</label>

		<div class="text-sm">
			<div class="flex items-center justify-between">
				<span class="text-slate-600 dark:text-slate-400">Attachments</span>
				<input
					bind:this={attachInput}
					type="file"
					accept={ATTACHMENT_ACCEPT}
					class="hidden"
					onchange={onFileChange}
				/>
				<button
					type="button"
					class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2 py-1 text-xs hover:bg-slate-100 disabled:opacity-60 dark:border-slate-700 dark:hover:bg-slate-800"
					onclick={() => attachInput?.click()}
					disabled={saving}
				>
					<Icon name="plus" class="h-3.5 w-3.5" /> Add
				</button>
			</div>

			{#if attachmentMsg}
				<p
					class="mt-1 rounded-md border border-amber-300 bg-amber-50 px-2 py-1 text-xs text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200"
					role="alert"
				>
					{attachmentMsg}
				</p>
			{/if}

			{#if attachments.length === 0}
				<p class="mt-1 text-xs text-slate-500">No attachments.</p>
			{:else}
				<div class="mt-1">
					<AttachmentList {attachments} ondelete={onRemove} deleting={saving} size="sm" />
				</div>
			{/if}

			{#if hasStagedChanges}
				<p class="mt-1 text-xs text-slate-500">Attachment changes are applied when you save.</p>
			{/if}
		</div>
	</div>

	{#snippet footer()}
		<button
			type="button"
			class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={cancel}
		>
			Cancel
		</button>
		<button
			type="button"
			class="rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-60"
			onclick={save}
			disabled={saving}
		>
			{saving ? 'Saving…' : 'Save'}
		</button>
	{/snippet}
</Modal>
