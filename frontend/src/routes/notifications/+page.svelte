<script lang="ts">
	import { onMount } from 'svelte';
	import { listNotifications, markNotificationRead, type NotificationItem } from '$lib/api';

	let items = $state<NotificationItem[]>([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		await reload();
	});

	async function reload() {
		loading = true;
		error = '';
		try {
			items = await listNotifications(100);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load notifications';
		} finally {
			loading = false;
		}
	}

	async function markRead(item: NotificationItem) {
		try {
			await markNotificationRead(item.id);
			item.readAt = new Date().toISOString();
			items = [...items];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to mark as read';
		}
	}
</script>

<section class="mx-auto max-w-3xl space-y-4">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">Notifications</h1>
		<button
			type="button"
			onclick={() => void reload()}
			class="rounded-md border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
		>
			Refresh
		</button>
	</div>

	{#if error}
		<div class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-700 dark:bg-red-950/40 dark:text-red-200">
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="text-sm text-slate-500">Loading notifications...</div>
	{:else if items.length === 0}
		<div class="rounded-lg border border-slate-200 p-4 text-sm text-slate-600 dark:border-slate-800 dark:text-slate-400">
			No notifications yet.
		</div>
	{:else}
		<ul class="space-y-2">
			{#each items as item}
				<li class="rounded-lg border border-slate-200 p-4 dark:border-slate-800">
					<div class="flex items-start justify-between gap-3">
						<div>
							<div class="font-medium">{item.title}</div>
							<div class="mt-1 text-sm text-slate-600 dark:text-slate-400">{item.body}</div>
							<div class="mt-2 text-xs text-slate-500">{new Date(item.createdAt).toLocaleString()}</div>
						</div>
						{#if !item.readAt}
							<button
								type="button"
								onclick={() => void markRead(item)}
								class="rounded-md border border-slate-300 px-2 py-1 text-xs hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
							>
								Mark read
							</button>
						{:else}
							<span class="rounded bg-slate-100 px-2 py-1 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-300">Read</span>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</section>
