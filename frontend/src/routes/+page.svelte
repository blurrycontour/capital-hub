<script lang="ts">
	import { onMount } from 'svelte';
	import { listNotifications, getPortfolioStats, formatCurrency, type NotificationItem } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import Icon, { type IconName } from '$lib/Icon.svelte';

	const user = $derived(auth.user);

	let status = $state<'loading' | string>('loading');
	let notifications = $state<NotificationItem[]>([]);
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
		{ label: 'Collections', icon: 'collections', value: String(summary.collections) },
		{ label: 'Total items', icon: 'cube', value: String(summary.items) },
		{
			label: 'Total value',
			icon: 'currency',
			value: valueLabel
		}
	]);

	onMount(async () => {
		try {
			const res = await fetch('/api/v1/health');
			const data = await res.json();
			status = data.status ?? 'unknown';
			if (auth.user) {
				const [notes, portfolio] = await Promise.all([
					listNotifications(5),
					getPortfolioStats()
				]);
				notifications = notes;
				summary.collections = portfolio.collectionCount;
				summary.items = portfolio.itemCount;
				summary.entries = portfolio.entryCount;
				totals = portfolio.totals;
			}
		} catch {
			status = 'unreachable';
			error = 'Failed to load dashboard data';
		}
	});
</script>

<section class="mx-auto max-w-5xl space-y-6">
	<header class="flex flex-wrap items-end justify-between gap-2">
		<div>
			<h1 class="text-2xl font-bold">
				Welcome back{user ? `, ${user.displayName || user.username}` : ''}
			</h1>
			<p class="text-sm text-slate-600 dark:text-slate-400">Here's an overview of your portfolio.</p>
		</div>
		<span
			class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium"
			class:bg-emerald-100={status === 'ok'}
			class:text-emerald-800={status === 'ok'}
			class:bg-amber-100={status !== 'ok' && status !== 'loading'}
			class:text-amber-800={status !== 'ok' && status !== 'loading'}
			class:bg-slate-100={status === 'loading'}
			class:text-slate-600={status === 'loading'}
		>
			<span class="h-1.5 w-1.5 rounded-full bg-current"></span>
			Backend {status}
		</span>
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
