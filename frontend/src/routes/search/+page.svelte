<script lang="ts">
	import Icon from '$lib/Icon.svelte';
	import { search, type SearchResult } from '$lib/api';

	let query = $state('');
	let results = $state<SearchResult[]>([]);
	let searching = $state(false);
	let searched = $state(false);
	let error = $state('');
	let timer: ReturnType<typeof setTimeout> | undefined;

	async function runSearch(q: string) {
		const trimmed = q.trim();
		if (!trimmed) {
			results = [];
			searched = false;
			return;
		}
		searching = true;
		error = '';
		try {
			results = await search(trimmed);
			searched = true;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Search failed';
		} finally {
			searching = false;
		}
	}

	function onInput() {
		clearTimeout(timer);
		timer = setTimeout(() => runSearch(query), 250);
	}

	function hrefFor(r: SearchResult): string {
		return r.type === 'collection' ? `/collections?c=${r.id}` : `/items/${r.id}`;
	}
</script>

<section class="mx-auto max-w-4xl space-y-4">
	<h1 class="text-2xl font-bold">Search</h1>
	<div class="relative">
		<span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">
			<Icon name="search" class="h-5 w-5" />
		</span>
		<input
			bind:value={query}
			oninput={onInput}
			placeholder="Search collections and items..."
			class="w-full rounded-md border border-slate-300 bg-white py-2 pl-10 pr-3 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900"
		/>
	</div>

	{#if error}
		<div
			class="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200"
		>
			{error}
		</div>
	{/if}

	{#if searching}
		<p class="text-sm text-slate-500">Searching…</p>
	{:else if results.length > 0}
		<ul class="divide-y divide-slate-100 rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
			{#each results as r (r.type + r.id)}
				<li>
					<a
						href={hrefFor(r)}
						class="flex items-center gap-3 px-4 py-3 hover:bg-slate-50 dark:hover:bg-slate-800/40"
					>
						<span class="text-slate-400">
							<Icon name={r.type === 'collection' ? 'collections' : 'cube'} class="h-5 w-5" />
						</span>
						<span class="min-w-0 flex-1">
							<span class="block truncate font-medium">{r.name}</span>
							{#if r.type === 'item' && r.collectionName}
								<span class="block truncate text-xs text-slate-500">in {r.collectionName}</span>
							{:else if r.description}
								<span class="block truncate text-xs text-slate-500">{r.description}</span>
							{/if}
						</span>
						<span class="rounded-full bg-slate-100 px-2 py-0.5 text-xs capitalize text-slate-500 dark:bg-slate-800">
							{r.type}
						</span>
					</a>
				</li>
			{/each}
		</ul>
	{:else}
		<div
			class="rounded-lg border border-dashed border-slate-300 p-10 text-center text-sm text-slate-500 dark:border-slate-700"
		>
			{searched ? `No results for "${query}".` : 'Start typing to search.'}
		</div>
	{/if}
</section>
