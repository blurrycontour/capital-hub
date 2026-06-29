<script lang="ts">
	import Icon from '$lib/Icon.svelte';

	// The three data objects, top to bottom in the hierarchy.
	const objects = [
		{
			icon: 'collections' as const,
			name: 'Collection',
			color: 'bg-sky-100 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300',
			desc: 'A group of related things you want to track — e.g. “Coins”, “Watches” or “Real estate”. Each collection has its own currency, so totals add up consistently.'
		},
		{
			icon: 'cube' as const,
			name: 'Item',
			color: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300',
			desc: 'A single thing inside a collection — e.g. a specific coin. Items can hold photos, attachments, a location and custom fields.'
		},
		{
			icon: 'list' as const,
			name: 'Entry',
			color: 'bg-amber-100 text-amber-700 dark:bg-amber-950/50 dark:text-amber-300',
			desc: 'A dated value record on an item — a purchase, sale or valuation. Entries are what build up an item’s value over time.'
		}
	];

	// First-entry walkthrough.
	const steps = [
		{
			title: 'Create a collection',
			desc: 'Go to Collections and add one. Pick a name and the currency its values should be tracked in.'
		},
		{
			title: 'Add an item',
			desc: 'Open the collection and add an item — or use “Add item” on the Items page and choose the collection there.'
		},
		{
			title: 'Add an entry',
			desc: 'Open the item and add an entry with an amount and date. That’s your first valuation recorded.'
		}
	];

	// What you can do with your data.
	const actions = [
		{
			icon: 'plus' as const,
			title: 'Create & organise',
			desc: 'Build collections, items and entries with photos, attachments, locations and custom fields.'
		},
		{
			icon: 'share' as const,
			title: 'Share',
			desc: 'Invite other users to a collection with read-only or read-write access.'
		},
		{
			icon: 'bell' as const,
			title: 'Get notified',
			desc: 'Receive notifications about activity such as collections shared with you.'
		},
		{
			icon: 'search' as const,
			title: 'Search',
			desc: 'Quickly find any collection or item across everything you can access.'
		}
	];
</script>

<section class="mx-auto max-w-3xl space-y-10">
	<header class="space-y-1">
		<h1 class="text-2xl font-bold">Help</h1>
		<p class="text-sm text-slate-600 dark:text-slate-400">
			A quick tour of how Capital Hub organises your data and how to get started.
		</p>
	</header>

	<!-- Data structure -->
	<div class="space-y-4">
		<h2 class="text-lg font-semibold">How your data is organised</h2>
		<p class="text-sm text-slate-600 dark:text-slate-400">
			Capital Hub nests data in three levels: a <strong>collection</strong> contains
			<strong>items</strong>, and each item contains <strong>entries</strong>.
		</p>

		<div class="space-y-3">
			{#each objects as obj, i (obj.name)}
				<div
					class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-slate-800"
				>
					<span
						class={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${obj.color}`}
					>
						<Icon name={obj.icon} class="h-6 w-6" />
					</span>
					<div class="min-w-0">
						<h3 class="font-semibold">{obj.name}</h3>
						<p class="mt-0.5 text-sm text-slate-600 dark:text-slate-400">{obj.desc}</p>
					</div>
				</div>
				{#if i < objects.length - 1}
					<div class="flex justify-center text-slate-300 dark:text-slate-600">
						<Icon name="chevron-down" class="h-5 w-5" />
					</div>
				{/if}
			{/each}
		</div>
	</div>

	<!-- First entry steps -->
	<div class="space-y-4">
		<h2 class="text-lg font-semibold">Record your first entry in 3 steps</h2>
		<ol class="space-y-3">
			{#each steps as step, i (step.title)}
				<li class="flex items-start gap-3">
					<span
						class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-sky-600 text-sm font-semibold text-white"
					>
						{i + 1}
					</span>
					<div class="min-w-0">
						<h3 class="font-medium">{step.title}</h3>
						<p class="mt-0.5 text-sm text-slate-600 dark:text-slate-400">{step.desc}</p>
					</div>
				</li>
			{/each}
		</ol>
	</div>

	<!-- What you can do -->
	<div class="space-y-4">
		<h2 class="text-lg font-semibold">What you can do</h2>
		<div class="grid gap-3 sm:grid-cols-2">
			{#each actions as action (action.title)}
				<div class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-slate-800">
					<span
						class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-300"
					>
						<Icon name={action.icon} class="h-5 w-5" />
					</span>
					<div class="min-w-0">
						<h3 class="font-medium">{action.title}</h3>
						<p class="mt-0.5 text-sm text-slate-600 dark:text-slate-400">{action.desc}</p>
					</div>
				</div>
			{/each}
		</div>
	</div>

	<div class="flex flex-wrap gap-3">
		<a
			href="/collections"
			class="inline-flex items-center gap-1.5 rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700"
		>
			<Icon name="collections" class="h-4 w-4" /> Go to Collections
		</a>
		<a
			href="/items"
			class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-4 py-2 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
		>
			<Icon name="cube" class="h-4 w-4" /> Go to Items
		</a>
	</div>
</section>
