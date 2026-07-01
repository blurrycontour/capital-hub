<script lang="ts">
	import Icon, { type IconName } from '$lib/Icon.svelte';
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
		if (r.type === 'collection') return `/collections/${r.id}`;
		if (r.type === 'entry')
			return `/collections/${r.collectionId}/items/${r.itemId}#entry-${r.id}`;
		return `/collections/${r.collectionId}/items/${r.id}`;
	}

	// Split a snippet on the highlight markers (\x01 open, \x02 close) emitted by
	// the backend into safe text parts. Rendering as text (never HTML) keeps this
	// XSS-free even though the source is user content.
	function snippetParts(snippet: string): { text: string; mark: boolean }[] {
		const parts: { text: string; mark: boolean }[] = [];
		let i = 0;
		while (i < snippet.length) {
			const open = snippet.indexOf('\x01', i);
			if (open === -1) {
				parts.push({ text: snippet.slice(i), mark: false });
				break;
			}
			if (open > i) parts.push({ text: snippet.slice(i, open), mark: false });
			const close = snippet.indexOf('\x02', open + 1);
			if (close === -1) {
				parts.push({ text: snippet.slice(open + 1), mark: true });
				break;
			}
			parts.push({ text: snippet.slice(open + 1, close), mark: true });
			i = close + 1;
		}
		return parts;
	}

	type GroupDef = {
		key: SearchResult['type'];
		label: string;
		icon: IconName;
		iconClass: string;
		badgeClass: string;
		markClass: string;
	};

	// Full class literals so Tailwind's compiler keeps them.
	const groupDefs: GroupDef[] = [
		{
			key: 'collection',
			label: 'Collections',
			icon: 'collections',
			iconClass: 'text-sky-500',
			badgeClass: 'bg-sky-100 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300',
			markClass: 'bg-sky-100 text-sky-800 dark:bg-sky-900/50 dark:text-sky-200'
		},
		{
			key: 'item',
			label: 'Items',
			icon: 'cube',
			iconClass: 'text-emerald-500',
			badgeClass: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300',
			markClass: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-200'
		},
		{
			key: 'entry',
			label: 'Entries',
			icon: 'list',
			iconClass: 'text-amber-500',
			badgeClass: 'bg-amber-100 text-amber-700 dark:bg-amber-950/50 dark:text-amber-300',
			markClass: 'bg-amber-100 text-amber-800 dark:bg-amber-900/50 dark:text-amber-200'
		}
	];

	const groups = $derived(
		groupDefs
			.map((g) => ({ ...g, items: results.filter((r) => r.type === g.key) }))
			.filter((g) => g.items.length > 0)
	);
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
			placeholder="Search collections, items and entries..."
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
	{:else if groups.length > 0}
		<div class="space-y-6">
			{#each groups as g (g.key)}
				<div>
					<h2
						class="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-slate-500"
					>
						<Icon name={g.icon} class="h-4 w-4 {g.iconClass}" />
						{g.label}
						<span class="text-slate-400">({g.items.length})</span>
					</h2>
					<ul
						class="divide-y divide-slate-100 rounded-lg border border-slate-200 dark:divide-slate-800 dark:border-slate-800"
					>
						{#each g.items as r (r.type + r.id)}
							<li>
								<a
									href={hrefFor(r)}
									class="flex items-start gap-3 px-4 py-3 hover:bg-slate-50 dark:hover:bg-slate-800/40"
								>
									<span class="mt-0.5 {g.iconClass}">
										<Icon name={g.icon} class="h-5 w-5" />
									</span>
									<span class="min-w-0 flex-1">
										<span class="block truncate font-medium">
											{r.name || (r.type === 'entry' ? 'Entry' : 'Untitled')}
										</span>
										{#if r.type === 'entry'}
											<span class="block truncate text-xs text-slate-500">
												{r.itemName} · {r.collectionName}
											</span>
										{:else if r.type === 'item'}
											<span class="block truncate text-xs text-slate-500">in {r.collectionName}</span>
										{/if}
										{#if r.snippet}
											<span class="mt-0.5 block truncate text-xs text-slate-500"
												>{#each snippetParts(r.snippet) as part}{#if part.mark}<mark
															class="rounded px-0.5 {g.markClass}">{part.text}</mark
														>{:else}{part.text}{/if}{/each}</span
											>
										{/if}
									</span>
									<span class="rounded-full px-2 py-0.5 text-xs capitalize {g.badgeClass}">
										{r.type}
									</span>
								</a>
							</li>
						{/each}
					</ul>
				</div>
			{/each}
		</div>
	{:else}
		<div
			class="rounded-lg border border-dashed border-slate-300 p-10 text-center text-sm text-slate-500 dark:border-slate-700"
		>
			{searched ? `No results for "${query}".` : 'Start typing to search.'}
		</div>
	{/if}
</section>
