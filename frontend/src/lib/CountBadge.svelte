<script lang="ts">
	// A compact pill for a single count (items, entries, collections). Counts
	// carry far less information than the money totals they used to sit beside,
	// so they read as badges rather than taking a full stat card each — which on
	// a phone meant one whole row per number.
	import Icon, { type IconName } from '$lib/Icon.svelte';

	let {
		icon,
		value,
		label,
		title
	}: {
		icon: IconName;
		value: number | string;
		label: string;
		title?: string;
	} = $props();
</script>

<span
	class="inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-2.5 py-1 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-300"
	{title}
>
	<Icon name={icon} class="h-3.5 w-3.5 shrink-0 text-slate-500 dark:text-slate-400" />
	<span class="font-semibold text-slate-900 dark:text-slate-100">{value}</span>
	{label}
</span>
