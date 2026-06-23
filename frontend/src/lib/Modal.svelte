<script lang="ts">
	import Icon from '$lib/Icon.svelte';
	import type { Snippet } from 'svelte';

	let {
		title,
		open = $bindable(false),
		onclose,
		children,
		footer
	}: {
		title: string;
		open?: boolean;
		onclose?: () => void;
		children: Snippet;
		footer?: Snippet;
	} = $props();

	function close() {
		open = false;
		onclose?.();
	}

	function onkeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') close();
	}
</script>

<svelte:window on:keydown={open ? onkeydown : undefined} />

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4"
		role="presentation"
	>
		<button
			type="button"
			class="absolute inset-0 cursor-default bg-slate-900/50 backdrop-blur-sm"
			aria-label="Close dialog"
			onclick={close}
		></button>
		<div
			class="relative z-10 flex max-h-[90vh] w-full max-w-lg flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl dark:border-slate-800 dark:bg-slate-900"
			role="dialog"
			aria-modal="true"
			aria-label={title}
		>
			<div
				class="flex items-center justify-between border-b border-slate-200 px-5 py-3 dark:border-slate-800"
			>
				<h2 class="text-lg font-semibold">{title}</h2>
				<button
					type="button"
					class="rounded-md p-1 text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-slate-800 dark:hover:text-slate-200"
					aria-label="Close"
					onclick={close}
				>
					<Icon name="close" class="h-5 w-5" />
				</button>
			</div>
			<div class="flex-1 overflow-y-auto px-5 py-4">
				{@render children()}
			</div>
			{#if footer}
				<div
					class="flex items-center justify-end gap-2 border-t border-slate-200 px-5 py-3 dark:border-slate-800"
				>
					{@render footer()}
				</div>
			{/if}
		</div>
	</div>
{/if}
