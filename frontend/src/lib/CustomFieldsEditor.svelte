<script lang="ts">
	import Icon from '$lib/Icon.svelte';
	import type { CustomField } from '$lib/api';

	let { fields = $bindable([]) }: { fields: CustomField[] } = $props();

	function add() {
		fields = [...fields, { label: '', value: '' }];
	}

	function remove(i: number) {
		fields = fields.filter((_, idx) => idx !== i);
	}
</script>

<div class="space-y-2">
	{#each fields as field, i (i)}
		<div class="flex items-center gap-2">
			<input
				type="text"
				bind:value={field.label}
				placeholder="Label"
				class="w-1/3 rounded-md border border-slate-300 px-2 py-1 text-sm dark:border-slate-700 dark:bg-slate-800"
			/>
			<input
				type="text"
				bind:value={field.value}
				placeholder="Value"
				class="flex-1 rounded-md border border-slate-300 px-2 py-1 text-sm dark:border-slate-700 dark:bg-slate-800"
			/>
			<button
				type="button"
				class="rounded p-1 text-rose-500 hover:bg-rose-100 dark:hover:bg-rose-950/40"
				aria-label="Remove field"
				onclick={() => remove(i)}
			>
				<Icon name="trash" class="h-4 w-4" />
			</button>
		</div>
	{/each}
	<button
		type="button"
		class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
		onclick={add}
	>
		<Icon name="plus" class="h-4 w-4" /> Add field
	</button>
</div>
