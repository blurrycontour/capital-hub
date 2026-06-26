<script lang="ts">
	import { onMount } from 'svelte';
	import {
		listNotifications,
		markNotificationRead,
		markNotificationUnread,
		deleteNotification,
		markAllNotificationsRead,
		markAllNotificationsUnread,
		deleteAllNotifications,
		type NotificationItem
	} from '$lib/api';
	import { notifCount } from '$lib/notifCount.svelte';
	import Icon from '$lib/Icon.svelte';

	let items = $state<NotificationItem[]>([]);
	let loading = $state(true);
	let error = $state('');
	let selected = $state<Set<number>>(new Set());
	let expanded = $state<Set<number>>(new Set());

	const unreadCount = $derived(items.filter((n) => !n.readAt).length);
	const allSelected = $derived(items.length > 0 && selected.size === items.length);
	const hasSelection = $derived(selected.size > 0);

	onMount(reload);

	async function reload() {
		loading = true;
		error = '';
		try {
			items = await listNotifications(100);
			// Drop selections/expansions that no longer exist.
			selected = new Set([...selected].filter((id) => items.some((n) => n.id === id)));
			expanded = new Set([...expanded].filter((id) => items.some((n) => n.id === id)));
			void notifCount.refresh();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load notifications';
		} finally {
			loading = false;
		}
	}

	function toggleExpand(id: number) {
		const next = new Set(expanded);
		if (next.has(id)) next.delete(id);
		else next.add(id);
		expanded = next;
	}

	function toggleSelect(id: number) {
		const next = new Set(selected);
		if (next.has(id)) next.delete(id);
		else next.add(id);
		selected = next;
	}

	function toggleSelectAll() {
		selected = allSelected ? new Set() : new Set(items.map((n) => n.id));
	}

	async function markRead(item: NotificationItem) {
		if (item.readAt) return;
		try {
			await markNotificationRead(item.id);
			item.readAt = new Date().toISOString();
			items = [...items];
			notifCount.adjust(-1);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to mark as read';
		}
	}

	async function markUnread(item: NotificationItem) {
		if (!item.readAt) return;
		try {
			await markNotificationUnread(item.id);
			item.readAt = undefined;
			items = [...items];
			notifCount.adjust(1);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to mark as unread';
		}
	}

	async function remove(item: NotificationItem) {
		try {
			await deleteNotification(item.id);
			items = items.filter((n) => n.id !== item.id);
			const next = new Set(selected);
			next.delete(item.id);
			selected = next;
			void notifCount.refresh();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete notification';
		}
	}

	// Bulk actions act on the current selection, or on everything when nothing
	// is selected.
	function targets(): NotificationItem[] {
		return hasSelection ? items.filter((n) => selected.has(n.id)) : items;
	}

	async function bulkMarkRead() {
		const list = targets().filter((n) => !n.readAt);
		try {
			if (hasSelection) {
				await Promise.all(list.map((n) => markNotificationRead(n.id)));
			} else {
				await markAllNotificationsRead();
			}
			const now = new Date().toISOString();
			for (const n of list) n.readAt = now;
			items = [...items];
			void notifCount.refresh();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to mark as read';
		}
	}

	async function bulkMarkUnread() {
		const list = targets().filter((n) => n.readAt);
		try {
			if (hasSelection) {
				await Promise.all(list.map((n) => markNotificationUnread(n.id)));
			} else {
				await markAllNotificationsUnread();
			}
			for (const n of list) n.readAt = undefined;
			items = [...items];
			void notifCount.refresh();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to mark as unread';
		}
	}

	async function bulkDelete() {
		const list = targets();
		try {
			if (hasSelection) {
				await Promise.all(list.map((n) => deleteNotification(n.id)));
				const ids = new Set(list.map((n) => n.id));
				items = items.filter((n) => !ids.has(n.id));
			} else {
				await deleteAllNotifications();
				items = [];
			}
			selected = new Set();
			void notifCount.refresh();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete notifications';
		}
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleString();
	}
</script>

<section class="mx-auto max-w-3xl space-y-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<div class="flex items-center gap-2">
			<h1 class="text-2xl font-bold">Notifications</h1>
			{#if unreadCount > 0}
				<span class="rounded-full bg-sky-600 px-2 py-0.5 text-xs font-semibold text-white">
					{unreadCount} new
				</span>
			{/if}
		</div>
		<button
			type="button"
			onclick={() => void reload()}
			class="rounded-md border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
		>
			Refresh
		</button>
	</div>

	{#if error}
		<div
			class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-700 dark:bg-red-950/40 dark:text-red-200"
		>
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="text-sm text-slate-500">Loading notifications...</div>
	{:else if items.length === 0}
		<div
			class="rounded-lg border border-slate-200 p-4 text-sm text-slate-600 dark:border-slate-800 dark:text-slate-400"
		>
			No notifications yet.
		</div>
	{:else}
		<!-- Toolbar -->
		<div
			class="flex flex-wrap items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 dark:border-slate-800"
		>
			<label class="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
				<input
					type="checkbox"
					class="h-4 w-4 rounded border-slate-300"
					checked={allSelected}
					onchange={toggleSelectAll}
				/>
				{#if hasSelection}
					{selected.size} selected
				{:else}
					Select all
				{/if}
			</label>
			<div class="ml-auto flex flex-wrap items-center gap-1.5">
				<button
					type="button"
					onclick={() => void bulkMarkRead()}
					class="inline-flex items-center gap-1 rounded-md border border-slate-300 px-2.5 py-1 text-xs hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
				>
					<Icon name="check" class="h-3.5 w-3.5" />
					{hasSelection ? 'Mark read' : 'Mark all read'}
				</button>
				<button
					type="button"
					onclick={() => void bulkMarkUnread()}
					class="inline-flex items-center gap-1 rounded-md border border-slate-300 px-2.5 py-1 text-xs hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
				>
					<Icon name="envelope" class="h-3.5 w-3.5" />
					{hasSelection ? 'Mark unread' : 'Mark all unread'}
				</button>
				<button
					type="button"
					onclick={() => void bulkDelete()}
					class="inline-flex items-center gap-1 rounded-md border border-rose-300 px-2.5 py-1 text-xs text-rose-600 hover:bg-rose-50 dark:border-rose-800 dark:text-rose-400 dark:hover:bg-rose-950/40"
				>
					<Icon name="trash" class="h-3.5 w-3.5" />
					{hasSelection ? 'Delete' : 'Delete all'}
				</button>
			</div>
		</div>

		<ul class="space-y-2">
			{#each items as item (item.id)}
				{@const isExpanded = expanded.has(item.id)}
				<li
					class="rounded-lg border border-slate-200 dark:border-slate-800"
					class:bg-sky-50={!item.readAt}
					class:dark:bg-sky-950={!item.readAt}
				>
					<div class="flex items-start gap-3 p-3 sm:p-4">
						<input
							type="checkbox"
							class="mt-1 h-4 w-4 shrink-0 rounded border-slate-300"
							checked={selected.has(item.id)}
							onchange={() => toggleSelect(item.id)}
							aria-label="Select notification"
						/>

						<!-- Clicking the body toggles expand. -->
						<button
							type="button"
							onclick={() => toggleExpand(item.id)}
							class="min-w-0 flex-1 text-left"
						>
							<div class="flex items-center gap-2">
								{#if !item.readAt}
									<span class="h-2 w-2 shrink-0 rounded-full bg-sky-600" aria-hidden="true"></span>
								{/if}
								<span class="truncate font-medium">{item.title}</span>
							</div>
							<div
								class="mt-1 text-sm text-slate-600 dark:text-slate-400"
								class:truncate={!isExpanded}
								class:whitespace-pre-wrap={isExpanded}
								class:break-words={isExpanded}
							>
								{item.body}
							</div>
							<div class="mt-2 flex items-center gap-2 text-xs text-slate-500">
								<span>{formatDate(item.createdAt)}</span>
								{#if item.link && isExpanded}
									<a
										href={item.link}
										class="inline-flex items-center gap-0.5 text-sky-600 hover:underline"
										onclick={(e) => e.stopPropagation()}
									>
										Open <Icon name="arrow-right" class="h-3 w-3" />
									</a>
								{/if}
							</div>
						</button>

						<div class="flex shrink-0 items-center gap-1">
							{#if item.readAt}
								<button
									type="button"
									title="Mark as unread"
									aria-label="Mark as unread"
									onclick={() => void markUnread(item)}
									class="rounded-md p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-slate-800"
								>
									<Icon name="envelope" class="h-4 w-4" />
								</button>
							{:else}
								<button
									type="button"
									title="Mark as read"
									aria-label="Mark as read"
									onclick={() => void markRead(item)}
									class="rounded-md p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-slate-800"
								>
									<Icon name="check" class="h-4 w-4" />
								</button>
							{/if}
							<button
								type="button"
								title="Delete"
								aria-label="Delete"
								onclick={() => void remove(item)}
								class="rounded-md p-1.5 text-slate-500 hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/40"
							>
								<Icon name="trash" class="h-4 w-4" />
							</button>
						</div>
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</section>
