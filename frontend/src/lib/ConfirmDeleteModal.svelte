<script lang="ts">
	import Modal from '$lib/Modal.svelte';

	let {
		open = $bindable(false),
		name,
		title = 'Confirm delete',
		message,
		deleting = false,
		onconfirm
	}: {
		open?: boolean;
		name: string;
		title?: string;
		message?: string;
		deleting?: boolean;
		onconfirm: () => void;
	} = $props();

	let typed = $state('');

	const matches = $derived(typed.trim() === name.trim() && name.trim() !== '');

	// Reset the confirmation field whenever the dialog is reopened.
	$effect(() => {
		if (open) typed = '';
	});
</script>

<Modal {title} bind:open>
	<div class="space-y-3">
		<p class="text-sm text-slate-600 dark:text-slate-400">
			{message ?? 'This action cannot be undone.'}
		</p>
		<p class="text-sm text-slate-600 dark:text-slate-400">
			Type <strong class="break-all">{name}</strong> to confirm.
		</p>
		<input
			type="text"
			bind:value={typed}
			placeholder={name}
			autocomplete="off"
			class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
		/>
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
			class="rounded-md bg-rose-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-rose-700 disabled:cursor-not-allowed disabled:opacity-50"
			onclick={onconfirm}
			disabled={!matches || deleting}
		>
			{deleting ? 'Deleting…' : 'Delete'}
		</button>
	{/snippet}
</Modal>
