<script lang="ts">
	import Icon from '$lib/Icon.svelte';

	let query = $state('');
</script>

<section class="mx-auto max-w-4xl space-y-4">
	<h1 class="text-2xl font-bold">Search</h1>
	<div class="relative">
		<span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">
			<Icon name="search" class="h-5 w-5" />
		</span>
		<input
			bind:value={query}
			placeholder="Search assets, collections, notifications..."
			class="w-full rounded-md border border-slate-300 bg-white py-2 pl-10 pr-3 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900"
		/>
	</div>
	<div
		class="rounded-lg border border-dashed border-slate-300 p-10 text-center text-sm text-slate-500 dark:border-slate-700"
	>
		{query ? `No results for "${query}" yet — search is under construction.` : 'Start typing to search.'}
	</div>
</section>
