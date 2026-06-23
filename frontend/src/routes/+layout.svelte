<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { browser } from '$app/environment';
	import { applyTheme, getInitialTheme, type Theme } from '$lib/theme';
	import { auth } from '$lib/auth.svelte';
	import Icon, { type IconName } from '$lib/Icon.svelte';

	let { children } = $props();
	let theme = $state<Theme>('light');
	let authError = $state('');

	// Sidebar layout state (persisted).
	let collapsed = $state(false);
	let hovering = $state(false);

	const SIDEBAR_KEY = 'ch-sidebar';

	type NavItem = { href: string; label: string; icon: IconName; adminOnly?: boolean };
	const navItems: NavItem[] = [
		{ href: '/', label: 'Dashboard', icon: 'dashboard' },
		{ href: '/collections', label: 'Collections', icon: 'collections' },
		{ href: '/search', label: 'Search', icon: 'search' },
		{ href: '/notifications', label: 'Notifications', icon: 'bell' }
	];

	// A collapsed sidebar temporarily expands as an overlay while hovered.
	const expanded = $derived(!collapsed || hovering);

	const initials = $derived.by(() => {
		const u = auth.user;
		if (!u) return '?';
		const source = (u.displayName || u.username || '').trim();
		if (!source) return '?';
		const parts = source.split(/\s+/);
		const letters = parts.length > 1 ? parts[0][0] + parts[parts.length - 1][0] : source.slice(0, 2);
		return letters.toUpperCase();
	});

	const breadcrumbs = $derived.by(() => {
		const path = $page.url.pathname;
		const crumbs: { label: string; href: string }[] = [{ label: 'Home', href: '/' }];
		let acc = '';
		for (const seg of path.split('/').filter(Boolean)) {
			acc += `/${seg}`;
			const label = seg.charAt(0).toUpperCase() + seg.slice(1).replace(/-/g, ' ');
			crumbs.push({ label, href: acc });
		}
		return crumbs;
	});

	const isLoginRoute = $derived($page.url.pathname.startsWith('/login'));
	const showChrome = $derived(auth.isAuthenticated && !isLoginRoute);

	onMount(() => {
		theme = getInitialTheme();
		applyTheme(theme);
		loadSidebarState();
		void init();
	});

	async function init() {
		authError = '';
		try {
			await auth.refresh();
		} catch (err) {
			authError = err instanceof Error ? err.message : 'Failed to load auth state';
		}
		guard();
	}

	// Redirect unauthenticated users to the login page so nothing but the
	// sign-in screen is reachable while signed out.
	$effect(() => {
		if (auth.loaded) guard();
	});

	function guard() {
		if (!browser) return;
		if (auth.loaded && !auth.isAuthenticated && !isLoginRoute) {
			void goto('/login');
		}
	}

	function loadSidebarState() {
		if (!browser) return;
		try {
			const raw = localStorage.getItem(SIDEBAR_KEY);
			if (raw) {
				const parsed = JSON.parse(raw) as { collapsed?: boolean };
				collapsed = parsed.collapsed ?? false;
			}
		} catch {
			// ignore malformed state
		}
	}

	function saveSidebarState() {
		if (!browser) return;
		localStorage.setItem(SIDEBAR_KEY, JSON.stringify({ collapsed }));
	}

	function toggleCollapsed() {
		collapsed = !collapsed;
		saveSidebarState();
	}

	function toggleTheme() {
		theme = theme === 'dark' ? 'light' : 'dark';
		applyTheme(theme);
	}

	async function doLogout() {
		try {
			await auth.logout();
			await goto('/login');
		} catch (err) {
			authError = err instanceof Error ? err.message : 'Logout failed';
		}
	}
</script>

<div class="min-h-screen bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
	{#if !auth.loaded}
		<div class="flex min-h-screen items-center justify-center text-sm text-slate-500">
			Loading...
		</div>
	{:else if showChrome}
		<div class="flex min-h-screen">
			<aside
				class="sticky top-0 z-20 flex h-screen shrink-0 flex-col border-r border-slate-200 bg-white transition-all duration-150 dark:border-slate-800 dark:bg-slate-900"
				class:w-64={expanded}
				class:w-16={!expanded}
				onmouseenter={() => (hovering = true)}
				onmouseleave={() => (hovering = false)}
			>
				<div
					class="flex h-14 shrink-0 items-center border-b border-slate-200 px-3 dark:border-slate-800"
					class:justify-between={expanded}
					class:justify-center={!expanded}
				>
					{#if expanded}
						<div class="flex min-w-0 items-center gap-2">
							<span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-sky-600 text-white">
								<Icon name="logo" class="h-5 w-5" />
							</span>
							<span class="truncate font-semibold">Capital-Hub</span>
						</div>
						<button
							type="button"
							onclick={toggleCollapsed}
							title="Collapse sidebar"
							aria-label="Collapse sidebar"
							class="rounded-md p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-900 dark:hover:bg-slate-800 dark:hover:text-slate-100"
						>
							<Icon name="panel-left" class="h-5 w-5" />
						</button>
					{:else}
						<button
							type="button"
							onclick={toggleCollapsed}
							title="Expand sidebar"
							aria-label="Expand sidebar"
							class="flex h-8 w-8 items-center justify-center rounded-lg bg-sky-600 text-white hover:bg-sky-500"
						>
							<Icon name="logo" class="h-5 w-5" />
						</button>
					{/if}
				</div>

				<nav class="flex-1 space-y-1 overflow-y-auto p-2 text-sm">
					{#each navItems as item (item.href)}
						{#if !item.adminOnly || auth.user?.isAdmin}
							{@const active =
								item.href === '/'
									? $page.url.pathname === '/'
									: $page.url.pathname.startsWith(item.href)}
							<a
								href={item.href}
								title={item.label}
								class="flex items-center gap-3 rounded-md px-3 py-2 hover:bg-slate-100 dark:hover:bg-slate-800"
								class:justify-center={!expanded}
								class:bg-slate-100={active}
								class:dark:bg-slate-800={active}
								class:font-semibold={active}
								class:text-sky-600={active}
							>
								<Icon name={item.icon} class="h-5 w-5 shrink-0" />
								{#if expanded}<span class="truncate">{item.label}</span>{/if}
							</a>
						{/if}
					{/each}
				</nav>

				<div class="shrink-0 space-y-1 border-t border-slate-200 p-2 text-sm dark:border-slate-800">
					{#if auth.user?.isAdmin}
						<a
							href="/admin/settings"
							title="Admin Panel"
							class="flex items-center gap-3 rounded-md px-3 py-2 hover:bg-slate-100 dark:hover:bg-slate-800"
							class:justify-center={!expanded}
							class:bg-slate-100={$page.url.pathname.startsWith('/admin/settings')}
							class:dark:bg-slate-800={$page.url.pathname.startsWith('/admin/settings')}
							class:font-semibold={$page.url.pathname.startsWith('/admin/settings')}
							class:text-sky-600={$page.url.pathname.startsWith('/admin/settings')}
						>
							<Icon name="shield" class="h-5 w-5 shrink-0" />
							{#if expanded}<span class="truncate">Admin Panel</span>{/if}
						</a>
					{/if}

					<button
						type="button"
						onclick={doLogout}
						title="Logout"
						aria-label="Logout"
						class="flex w-full items-center gap-3 rounded-md px-3 py-2 text-slate-600 hover:bg-slate-100 hover:text-red-600 dark:text-slate-300 dark:hover:bg-slate-800"
						class:justify-center={!expanded}
					>
						<Icon name="logout" class="h-5 w-5 shrink-0" />
						{#if expanded}<span>Logout</span>{/if}
					</button>

					{#if auth.user}
						<a
							href="/settings"
							title="User Profile"
							class="mt-1 flex items-center gap-2 rounded-md px-2 py-2 hover:bg-slate-100 dark:hover:bg-slate-800"
							class:justify-center={!expanded}
							class:bg-slate-100={$page.url.pathname.startsWith('/settings')}
							class:dark:bg-slate-800={$page.url.pathname.startsWith('/settings')}
						>
							<span
								class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-sky-600 text-xs font-semibold text-white"
							>
								{initials}
							</span>
							{#if expanded}
								<span class="min-w-0">
									<span class="block truncate text-xs font-medium">
										{auth.user.displayName || auth.user.username}
									</span>
									<span class="block truncate text-xs text-slate-500">{auth.user.email}</span>
								</span>
							{/if}
						</a>
					{/if}
				</div>
			</aside>

			<div class="flex min-w-0 flex-1 flex-col">
				<header
					class="flex h-14 items-center justify-between border-b border-slate-200 px-4 dark:border-slate-800"
				>
					<nav class="flex items-center gap-1 text-sm" aria-label="Breadcrumb">
						{#each breadcrumbs as crumb, i (crumb.href)}
							{#if i > 0}
								<Icon name="chevron-divider" class="h-3.5 w-3.5 text-slate-400" />
							{/if}
							{#if i === breadcrumbs.length - 1}
								<span class="font-medium text-slate-900 dark:text-slate-100">{crumb.label}</span>
							{:else}
								<a
									href={crumb.href}
									class="rounded px-1 text-slate-500 hover:text-sky-600 hover:underline"
								>
									{crumb.label}
								</a>
							{/if}
						{/each}
					</nav>
					<div class="flex items-center gap-2">
						<button
							type="button"
							onclick={toggleTheme}
							title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
							aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
							class="rounded-md border border-slate-300 p-2 text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
						>
							<Icon name={theme === 'dark' ? 'sun' : 'moon'} class="h-5 w-5" />
						</button>
					</div>
				</header>

				<main class="flex-1 p-4">
					{#if authError}
						<div
							class="mb-4 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200"
						>
							{authError}
						</div>
					{/if}
					{@render children()}
				</main>
			</div>
		</div>
	{:else if isLoginRoute}
		{@render children()}
	{:else}
		<div class="flex min-h-screen items-center justify-center text-sm text-slate-500">
			Redirecting...
		</div>
	{/if}
</div>
