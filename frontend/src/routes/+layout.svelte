<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { applyTheme, getInitialTheme, type Theme } from '$lib/theme';
	import { fetchMe, logout, type ApiUser } from '$lib/api';

	let { children } = $props();
	let theme = $state<Theme>('light');
	let sidebarOpen = $state(true);
	let user = $state<ApiUser | null>(null);
	let authLoading = $state(true);
	let authError = $state('');

	onMount(() => {
		theme = getInitialTheme();
		applyTheme(theme);
		void refreshAuth();
	});

	function toggleTheme() {
		theme = theme === 'dark' ? 'light' : 'dark';
		applyTheme(theme);
	}

	async function refreshAuth() {
		authLoading = true;
		authError = '';
		try {
			user = await fetchMe();
		} catch (err) {
			authError = err instanceof Error ? err.message : 'Failed to load auth state';
			user = null;
		} finally {
			authLoading = false;
		}
	}

	async function doLogout() {
		try {
			await logout();
			user = null;
			await goto('/login');
		} catch (err) {
			authError = err instanceof Error ? err.message : 'Logout failed';
		}
	}
</script>

<div class="min-h-full bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
	<div class="flex min-h-full">
		<aside
			class="border-r border-slate-200 bg-white transition-all dark:border-slate-800 dark:bg-slate-900"
			class:w-72={sidebarOpen}
			class:w-16={!sidebarOpen}
		>
			<div class="flex h-14 items-center justify-between border-b border-slate-200 px-3 dark:border-slate-800">
				<span class="font-semibold" class:hidden={!sidebarOpen}>Capital-Hub</span>
				<button
					type="button"
					onclick={() => (sidebarOpen = !sidebarOpen)}
					class="rounded-md border border-slate-300 px-2 py-1 text-xs hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
				>
					{sidebarOpen ? 'Collapse' : 'Expand'}
				</button>
			</div>

			<nav class="space-y-1 p-2 text-sm">
				<a
					href="/"
					class="block rounded-md px-3 py-2 hover:bg-slate-100 dark:hover:bg-slate-800"
					class:font-semibold={$page.url.pathname === '/'}
				>
					Dashboard
				</a>
				<a
					href="/notifications"
					class="block rounded-md px-3 py-2 hover:bg-slate-100 dark:hover:bg-slate-800"
					class:font-semibold={$page.url.pathname.startsWith('/notifications')}
				>
					Notifications
				</a>
				{#if user?.isAdmin}
					<a
						href="/admin/settings"
						class="block rounded-md px-3 py-2 hover:bg-slate-100 dark:hover:bg-slate-800"
						class:font-semibold={$page.url.pathname.startsWith('/admin/settings')}
					>
						Admin Settings
					</a>
				{/if}
			</nav>

			<div class="mt-auto border-t border-slate-200 p-3 text-xs dark:border-slate-800">
				{#if authLoading}
					<div class="text-slate-500">Loading session...</div>
				{:else if user}
					<div class="truncate font-medium">{user.displayName || user.username}</div>
					<div class="truncate text-slate-500">{user.email}</div>
				{:else}
					<div class="text-slate-500">Not signed in</div>
				{/if}
			</div>
		</aside>

		<div class="flex min-w-0 flex-1 flex-col">
			<header class="flex h-14 items-center justify-between border-b border-slate-200 px-4 dark:border-slate-800">
				<div class="text-sm text-slate-500">{$page.url.pathname}</div>
				<div class="flex items-center gap-2">
					<button
						type="button"
						onclick={toggleTheme}
						class="rounded-md border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
					>
						{theme === 'dark' ? 'Light' : 'Dark'} mode
					</button>
					{#if user}
						<button
							type="button"
							onclick={doLogout}
							class="rounded-md border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
						>
							Logout
						</button>
					{:else}
						<a
							href="/login"
							class="rounded-md border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
						>
							Login
						</a>
					{/if}
				</div>
			</header>

			<main class="flex-1 p-4">
				{#if authError}
					<div class="mb-4 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200">
						{authError}
					</div>
				{/if}
				{@render children()}
			</main>
		</div>
	</div>
</div>
