<script lang="ts">
	// A compact pill for a single count (items, entries, collections). Counts
	// carry far less information than the money totals they used to sit beside,
	// so they read as badges rather than taking a full stat card each — which on
	// a phone meant one whole row per number.
	import Icon, { type IconName } from '$lib/Icon.svelte';

	// Per-entity icon colours. Collections and items reuse the hues already used
	// for their page-header tiles; entries had no colour of its own, so it takes
	// amber — the one family not already carrying a meaning here (sky, emerald,
	// violet and rose are collections, items, shared and debit respectively).
	// Written as whole class strings because Tailwind only sees literal names.
	export type BadgeTone = 'collections' | 'items' | 'entries' | 'neutral';

	const TONES: Record<BadgeTone, string> = {
		collections: 'text-sky-600 dark:text-sky-400',
		items: 'text-emerald-600 dark:text-emerald-400',
		entries: 'text-amber-600 dark:text-amber-400',
		neutral: 'text-slate-500 dark:text-slate-400'
	};

	let {
		icon,
		value,
		label,
		tone = 'neutral',
		title
	}: {
		icon: IconName;
		value: number | string;
		label: string;
		tone?: BadgeTone;
		title?: string;
	} = $props();
</script>

<span
	class="inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-2.5 py-1 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-300"
	{title}
>
	<Icon name={icon} class={`h-3.5 w-3.5 shrink-0 ${TONES[tone]}`} />
	<span class="font-semibold text-slate-900 dark:text-slate-100">{value}</span>
	{label}
</span>
