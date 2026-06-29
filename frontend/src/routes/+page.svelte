<script lang="ts">
	import { onMount } from 'svelte';
	import {
		listNotifications,
		getPortfolioStats,
		getRecentItems,
		formatCurrency,
		type NotificationItem,
		type Item
	} from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import Icon, { type IconName } from '$lib/Icon.svelte';

	const user = $derived(auth.user);

	let notifications = $state<NotificationItem[]>([]);
	let recentItems = $state<Item[]>([]);
	let error = $state('');

	// Portfolio summary loaded from the backend.
	const summary = $state({ collections: 0, items: 0, entries: 0 });
	let totals = $state<{ currency: string; total: number }[]>([]);

	const valueLabel = $derived(
		totals.length === 0
			? formatCurrency(0, 'USD')
			: totals.map((t) => formatCurrency(t.total, t.currency)).join(' · ')
	);

	const cards: { label: string; icon: IconName; value: string }[] = $derived([
		{ label: 'Total Collections', icon: 'collections', value: String(summary.collections) },
		{ label: 'Total Items', icon: 'cube', value: String(summary.items) },
		{
			label: 'Total Value',
			icon: 'currency',
			value: valueLabel
		}
	]);

	// Compact "x ago" label for the recent-items list.
	function timeAgo(iso: string): string {
		const then = new Date(iso).getTime();
		if (Number.isNaN(then)) return '';
		const secs = Math.max(0, Math.round((Date.now() - then) / 1000));
		if (secs < 60) return 'just now';
		const mins = Math.round(secs / 60);
		if (mins < 60) return `${mins}m ago`;
		const hours = Math.round(mins / 60);
		if (hours < 24) return `${hours}h ago`;
		const days = Math.round(hours / 24);
		if (days < 30) return `${days}d ago`;
		return new Date(iso).toLocaleDateString();
	}

	onMount(async () => {
		try {
			if (auth.user) {
				const [notes, portfolio, recent] = await Promise.all([
					listNotifications(5),
					getPortfolioStats(),
					getRecentItems(6)
				]);
				notifications = notes;
				recentItems = recent;
				summary.collections = portfolio.collectionCount;
				summary.items = portfolio.itemCount;
				summary.entries = portfolio.entryCount;
				totals = portfolio.totals;
			}
		} catch {
			error = 'Failed to load dashboard data';
		}
	});
</script>

<section class="mx-auto max-w-5xl space-y-6">
	<header class="flex flex-wrap items-end justify-between gap-2">
		<div>
			<h1 class="text-2xl font-bold">
				Welcome{user ? `, ${user.displayName || user.username}` : ''}
			</h1>
			<p class="text-sm text-slate-600 dark:text-slate-400">Here's an overview of your capitals!</p>
		</div>
		<a
			href="/help"
			class="inline-flex items-center gap-1.5 rounded-md p-2 text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-slate-800 dark:hover:text-slate-200"
			title="Help"
			aria-label="Help"
		>
			<Icon name="help" class="h-6 w-6" />
		</a>
	</header>

	{#if error}
		<div
			class="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200"
		>
			{error}
		</div>
	{/if}

	<!-- Summary cards -->
	<div class="grid gap-4 sm:grid-cols-3">
		{#each cards as card (card.label)}
			<div class="rounded-lg border border-slate-200 p-5 dark:border-slate-800">
				<div class="flex items-center justify-between">
					<span class="text-sm text-slate-500">{card.label}</span>
					<span class="text-sky-600 dark:text-sky-400">
						<Icon name={card.icon} class="h-5 w-5" />
					</span>
				</div>
				<div class="mt-2 text-2xl font-bold">{card.value}</div>
			</div>
		{/each}
	</div>

	<!-- Recently modified items -->
	{#if recentItems.length > 0}
		<div class="space-y-3 rounded-lg border border-slate-200 p-5 dark:border-slate-800">
			<h2 class="text-lg font-semibold">Recent Activity</h2>
			<ul class="grid gap-2 sm:grid-cols-2">
				{#each recentItems as it (it.id)}
					<li class="min-w-0">
						<a
							href={`/collections/${it.collectionId}/items/${it.id}`}
							class="flex min-w-0 items-center gap-3 rounded-md border border-slate-200 p-2.5 transition hover:border-sky-400 hover:shadow-sm dark:border-slate-800 dark:hover:border-sky-600"
						>
							{#if it.imagePath}
								<img
									src={it.imagePath}
									alt={it.name}
									class="h-10 w-10 shrink-0 rounded object-cover"
								/>
							{:else}
								<span
									class="flex h-10 w-10 shrink-0 items-center justify-center rounded bg-sky-100 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300"
								>
									<Icon name="cube" class="h-5 w-5" />
								</span>
							{/if}
							<span class="min-w-0 flex-1">
								<span class="block truncate text-sm font-medium">{it.name}</span>
								<span class="block truncate text-xs text-slate-500">{timeAgo(it.updatedAt)}</span>
							</span>
						</a>
					</li>
				{/each}
			</ul>
		</div>
	{/if}

	<!-- Quick links + activity -->
	<div class="grid gap-4 md:grid-cols-2">
		<div class="space-y-3 rounded-lg border border-slate-200 p-5 dark:border-slate-800">
			<h2 class="text-lg font-semibold">Quick actions</h2>
			<div class="flex flex-wrap gap-2">
				<a
					href="/collections"
					class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
					>Browse collections</a
				>
				<a
					href="/notifications"
					class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
					>Notifications</a
				>
				{#if user?.isAdmin}
					<a
						href="/admin/settings"
						class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
						>Admin panel</a
					>
				{/if}
			</div>
		</div>

		<div class="space-y-3 rounded-lg border border-slate-200 p-5 dark:border-slate-800">
			<h2 class="text-lg font-semibold">Recent notifications</h2>
			{#if notifications.length === 0}
				<p class="text-sm text-slate-500">No recent notifications.</p>
			{:else}
				<ul class="space-y-2">
					{#each notifications as n (n.id)}
						<li class="text-sm">
							<span class="font-medium">{n.title}</span>
							<span class="block truncate text-slate-500">{n.body}</span>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</div>
</section>
