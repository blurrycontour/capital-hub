<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchMe, listNotifications, type ApiUser, type NotificationItem } from '$lib/api';

	let status = $state<'loading' | string>('loading');
	let user = $state<ApiUser | null>(null);
	let notifications = $state<NotificationItem[]>([]);
	let error = $state('');

	onMount(async () => {
		try {
			const res = await fetch('/api/v1/health');
			const data = await res.json();
			status = data.status ?? 'unknown';
			user = await fetchMe();
			if (user) {
				notifications = await listNotifications(5);
			}
		} catch {
			status = 'unreachable';
			error = 'Failed to load dashboard data';
		}
	});
</script>

<section class="mx-auto max-w-2xl space-y-4">
	<h1 class="text-2xl font-bold">Welcome to Capital&#8209;Hub</h1>
	<p class="text-slate-600 dark:text-slate-400">
		Self-hosted asset management. The interface is under construction.
	</p>
	<div class="rounded-lg border border-slate-200 p-4 dark:border-slate-800">
		<span class="text-sm font-medium">Backend status:</span>
		<span
			class="ml-2 rounded px-2 py-0.5 text-sm"
			class:bg-green-100={status === 'ok'}
			class:text-green-800={status === 'ok'}
			class:bg-amber-100={status !== 'ok' && status !== 'loading'}
			class:text-amber-800={status !== 'ok' && status !== 'loading'}
		>
			{status}
		</span>
	</div>

	{#if error}
		<div class="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200">
			{error}
		</div>
	{/if}

	{#if user}
		<div class="rounded-lg border border-slate-200 p-4 dark:border-slate-800">
			<div class="mb-2 text-sm font-medium">Signed in as {user.displayName || user.username}</div>
			<div class="text-sm text-slate-600 dark:text-slate-400">Recent notifications: {notifications.length}</div>
			<div class="mt-3 flex flex-wrap gap-2">
				<a href="/notifications" class="rounded-md border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800">Open notifications</a>
				{#if user.isAdmin}
					<a href="/admin/settings" class="rounded-md border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800">Admin settings</a>
				{/if}
			</div>
		</div>
	{:else}
		<div class="rounded-lg border border-slate-200 p-4 dark:border-slate-800">
			<div class="mb-2 text-sm">You are not logged in.</div>
			<a href="/login" class="rounded-md border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800">Go to login</a>
		</div>
	{/if}
</section>
