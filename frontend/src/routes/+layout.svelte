<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { browser } from '$app/environment';
	import { applyTheme, getInitialTheme, type Theme } from '$lib/theme';
	import { auth } from '$lib/auth.svelte';
	import { breadcrumbs as crumbStore } from '$lib/breadcrumb.svelte';
	import { notifCount } from '$lib/notifCount.svelte';
	import Icon, { type IconName } from '$lib/Icon.svelte';
	import { useRegisterSW } from 'virtual:pwa-register/svelte';
	import { getPreferences, setAmountDecimals, setNumberFormat } from '$lib/api';

	let { children } = $props();
	let theme = $state<Theme>('light');
	let authError = $state('');
	// User account menu (top-right) modal.
	let userMenuOpen = $state(false);

	// PWA service-worker update prompt.
	const { needRefresh, updateServiceWorker } = useRegisterSW();
	const swUpdateAvailable = $derived($needRefresh);

	// Sidebar layout state (persisted).
	let collapsed = $state(false);
	// Mobile off-canvas drawer state (separate from desktop collapse).
	let mobileOpen = $state(false);
	let isMobile = $state(false);
	// On mobile the top bar hides while scrolling down and reappears on scroll up.
	let headerHidden = $state(false);

	const SIDEBAR_KEY = 'ch-sidebar';

	type NavItem = { href: string; label: string; icon: IconName; adminOnly?: boolean };
	const navItems: NavItem[] = [
		{ href: '/', label: 'Dashboard', icon: 'dashboard' },
		{ href: '/collections', label: 'Collections', icon: 'collections' },
		{ href: '/items', label: 'Items', icon: 'cube' },
		{ href: '/search', label: 'Search', icon: 'search' },
		{ href: '/notifications', label: 'Notifications', icon: 'bell' },
		{ href: '/help', label: 'Help', icon: 'help' }
	];

	// The sidebar is expanded unless explicitly collapsed (pure toggle).
	// On mobile the off-canvas drawer always shows full-width labels.
	const expanded = $derived(isMobile ? true : !collapsed);

	// Permanent bottom navigation on mobile (excludes Help).
	const bottomNavItems = $derived(navItems.filter((item) => item.href !== '/help'));

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
		const crumbs: { label: string; href: string }[] = [{ label: 'Home', href: '/' }];
		// A page may register a full custom trail (e.g. an item nested under its
		// collection) that doesn't match the URL structure.
		if (crumbStore.trail) {
			return [...crumbs, ...crumbStore.trail];
		}
		const path = $page.url.pathname;
		let acc = '';
		for (const seg of path.split('/').filter(Boolean)) {
			acc += `/${seg}`;
			const fallback = seg.charAt(0).toUpperCase() + seg.slice(1).replace(/-/g, ' ');
			const label = crumbStore.overrides[acc] ?? fallback;
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

		// Track viewport so the sidebar can switch between desktop collapse and
		// mobile off-canvas drawer behaviour.
		const mq = window.matchMedia('(max-width: 767px)');
		const apply = () => (isMobile = mq.matches);
		apply();
		mq.addEventListener('change', apply);

		// Hide the top bar while scrolling down on mobile, reveal on scroll up.
		let lastScrollY = window.scrollY;
		const onScroll = () => {
			const y = window.scrollY;
			if (!mq.matches) {
				headerHidden = false;
			} else if (y > lastScrollY && y > 64) {
				headerHidden = true;
			} else if (y < lastScrollY) {
				headerHidden = false;
			}
			lastScrollY = y;
		};
		window.addEventListener('scroll', onScroll, { passive: true });

		return () => {
			mq.removeEventListener('change', apply);
			window.removeEventListener('scroll', onScroll);
		};
	});

	// Close the mobile drawer whenever the route changes.
	$effect(() => {
		void $page.url.pathname;
		mobileOpen = false;
	});

	async function init() {
		authError = '';
		try {
			await auth.refresh();
		} catch (err) {
			authError = err instanceof Error ? err.message : 'Failed to load auth state';
		}
		guard();
		if (auth.isAuthenticated) {
			void notifCount.refresh();
			void loadPreferences();
		}
	}

	// Load user preferences (currently the money rounding precision) and apply
	// them globally so every currency value renders consistently.
	async function loadPreferences() {
		try {
			const prefs = await getPreferences();
			setAmountDecimals(prefs.amountDecimals);
			setNumberFormat(prefs.numberFormat);
		} catch {
			// Non-fatal: fall back to the default precision.
		}
	}

	// Poll for the unread notification count while signed in.
	$effect(() => {
		if (!auth.isAuthenticated) return;
		const id = setInterval(() => void notifCount.refresh(), 60_000);
		return () => clearInterval(id);
	});

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

	// The header button toggles the off-canvas drawer on mobile and the
	// icon/label collapse on desktop.
	function toggleSidebar() {
		if (isMobile) {
			mobileOpen = !mobileOpen;
		} else {
			toggleCollapsed();
		}
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
			{#if mobileOpen}
				<!-- Mobile drawer backdrop -->
				<button
					type="button"
					aria-label="Close menu"
					onclick={() => (mobileOpen = false)}
					class="fixed inset-0 z-30 bg-slate-900/50 backdrop-blur-sm md:hidden"
				></button>
			{/if}
			<aside
				class="fixed inset-y-0 left-0 z-40 flex h-screen shrink-0 flex-col border-r border-slate-200 bg-slate-100 transition-transform duration-200 md:sticky md:top-0 md:z-auto md:translate-x-0 md:transition-[width] dark:border-slate-800 dark:bg-slate-900"
				class:w-64={expanded}
				class:w-16={!expanded}
				class:-translate-x-full={!mobileOpen}
			>
				<div
					class="flex h-14 shrink-0 items-center border-b border-slate-200 px-3 dark:border-slate-800 text-xl"
					class:justify-center={!expanded}
				>
					<div class="flex min-w-0 items-center gap-2">
						{#if expanded}
							<img src="/logo-text.svg" alt="Capital Hub" class="h-7 w-auto rounded" />
						{:else}
							<span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg">
								<img src="/logo.svg" alt="Capital Hub" class="h-10 w-10" />
							</span>
						{/if}
					</div>
				</div>

				<nav class="flex-1 space-y-1 overflow-y-auto p-2 text-base">
					{#each navItems as item (item.href)}
						{#if !item.adminOnly || auth.user?.isAdmin}
							{@const active =
								item.href === '/'
									? $page.url.pathname === '/'
									: $page.url.pathname.startsWith(item.href)}
							<a
								href={item.href}
								title={item.label}
								class="relative flex items-center gap-3 rounded-md px-3 py-2 hover:bg-slate-100 dark:hover:bg-slate-800"
								class:justify-center={!expanded}
								class:bg-slate-100={active}
								class:dark:bg-slate-800={active}
								class:font-semibold={active}
								class:text-sky-600={active}
							>
								<Icon name={item.icon} class="h-5 w-5 shrink-0" />
								{#if expanded}
									<span class="truncate">{item.label}</span>
									{#if item.href === '/notifications' && notifCount.value > 0}
										<span
											class="ml-auto shrink-0 rounded-full bg-sky-600 px-1.5 py-0.5 text-xs font-semibold leading-none text-white"
										>
											{notifCount.value > 99 ? '99+' : notifCount.value}
										</span>
									{/if}
								{:else if item.href === '/notifications' && notifCount.value > 0}
									<span class="absolute right-1 top-1 h-2 w-2 rounded-full bg-sky-600"></span>
								{/if}
							</a>
						{/if}
					{/each}
				</nav>
			</aside>

			<div class="flex min-w-0 flex-1 flex-col">
				<header
					class="sticky top-0 z-30 flex h-14 items-center justify-between border-b border-slate-200 bg-slate-50 px-4 transition-transform duration-200 sm:px-6 dark:border-slate-800 dark:bg-slate-950"
					class:-translate-y-full={headerHidden && isMobile}
				>
					<div class="flex min-w-0 items-center gap-2">
						<button
							type="button"
							onclick={toggleSidebar}
							title={expanded ? 'Collapse sidebar' : 'Expand sidebar'}
							aria-label={expanded ? 'Collapse sidebar' : 'Expand sidebar'}
							class="shrink-0 rounded-md p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-900 dark:hover:bg-slate-800 dark:hover:text-slate-100"
						>
							<Icon name="panel-left" class="h-5 w-5" />
						</button>
						<nav class="flex min-w-0 items-center gap-1 text-sm" aria-label="Breadcrumb">
							{#each breadcrumbs as crumb, i (crumb.href)}
								{#if i > 0}
									<Icon name="chevron-divider" class="h-3.5 w-3.5 shrink-0 text-slate-400" />
								{/if}
								{#if i === breadcrumbs.length - 1}
									<span class="truncate font-medium text-slate-900 dark:text-slate-100">{crumb.label}</span>
								{:else}
									<a
										href={crumb.href}
										class="shrink-0 rounded px-1 text-slate-500 hover:text-sky-600 hover:underline"
									>
										{crumb.label}
									</a>
								{/if}
							{/each}
						</nav>
					</div>
					<div class="flex items-center gap-4">
						<button
							type="button"
							onclick={toggleTheme}
							title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
							aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
							class="rounded-md border border-slate-300 p-2 text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
						>
							<Icon name={theme === 'dark' ? 'sun' : 'moon'} class="h-5 w-5" />
						</button>

						{#if auth.user}
							<div class="relative">
								<div class="flex items-center justify-center">
									<button
										type="button"
										onclick={() => (userMenuOpen = !userMenuOpen)}
										title="Account menu"
										aria-label="Open account menu"
										aria-expanded={userMenuOpen}
										class="rounded-full ring-2 ring-transparent transition hover:ring-sky-500/40 focus:outline-none focus-visible:ring-sky-500"
									>
										{#if auth.user.avatarPath}
											<img
												src={auth.user.avatarPath}
												alt="Profile"
												class="h-9 w-9 shrink-0 rounded-full object-cover"
											/>
										{:else}
											<span
												class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-sky-600 text-sm font-semibold text-white"
											>
												{initials}
											</span>
										{/if}
									</button>
								</div>

								{#if userMenuOpen}
									<!-- Click-away backdrop with blur -->
									<button
										type="button"
										aria-label="Close account menu"
										onclick={() => (userMenuOpen = false)}
										class="fixed inset-0 z-40 cursor-default bg-slate-900/20 backdrop-blur-sm"
									></button>

									<!-- Dropdown attached to the button -->
									<div
										class="absolute right-0 top-full z-50 mt-2 w-64 overflow-hidden rounded-xl border border-slate-200 bg-white text-center shadow-xl dark:border-slate-800 dark:bg-slate-900"
										role="menu"
									>
										<div
											class="flex flex-col items-center gap-2 border-b border-slate-200 px-4 py-4 dark:border-slate-800"
										>
											{#if auth.user.avatarPath}
												<img
													src={auth.user.avatarPath}
													alt="Profile"
													class="h-14 w-14 rounded-full object-cover"
												/>
											{:else}
												<span
													class="flex h-14 w-14 items-center justify-center rounded-full bg-sky-600 text-lg font-semibold text-white"
												>
													{initials}
												</span>
											{/if}
											<div class="min-w-0">
												<p class="truncate text-sm font-semibold">
													{auth.user.displayName || auth.user.username}
												</p>
												<p class="truncate text-xs text-slate-500">{auth.user.email}</p>
											</div>
										</div>

										<div class="p-2">
											<a
												href="/account"
												onclick={() => (userMenuOpen = false)}
												class="flex items-center justify-center gap-2 rounded-md px-3 py-2.5 text-sm hover:bg-slate-100 dark:hover:bg-slate-800"
												role="menuitem"
											>
												<Icon name="user" class="h-5 w-5 shrink-0 text-slate-500" />
												<span>Account</span>
											</a>
											<a
												href="/settings"
												onclick={() => (userMenuOpen = false)}
												class="flex items-center justify-center gap-2 rounded-md px-3 py-2.5 text-sm hover:bg-slate-100 dark:hover:bg-slate-800"
												role="menuitem"
											>
												<Icon name="cog" class="h-5 w-5 shrink-0 text-slate-500" />
												<span>Settings</span>
											</a>
											{#if auth.user.isAdmin}
												<a
													href="/admin/settings"
													onclick={() => (userMenuOpen = false)}
													class="flex items-center justify-center gap-2 rounded-md px-3 py-2.5 text-sm hover:bg-slate-100 dark:hover:bg-slate-800"
													role="menuitem"
												>
													<Icon name="shield" class="h-5 w-5 shrink-0 text-slate-500" />
													<span>Admin Panel</span>
												</a>
											{/if}
										</div>

										<div class="border-t border-slate-200 p-2 dark:border-slate-800">
											<button
												type="button"
												onclick={() => {
													userMenuOpen = false;
													void doLogout();
												}}
												class="flex w-full items-center justify-center gap-2 rounded-md px-3 py-2.5 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-red-600 dark:text-slate-300 dark:hover:bg-slate-800"
												role="menuitem"
											>
												<Icon name="logout" class="h-4 w-4" />
												Logout
											</button>
										</div>
									</div>
								{/if}
							</div>
						{/if}
					</div>
				</header>

				<main class="flex-1 p-4 pb-20 md:pb-4">
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

			<!-- Permanent bottom navigation (mobile only) -->
			<nav
				class="fixed inset-x-0 bottom-0 z-40 flex items-stretch border-t border-slate-200 bg-white md:hidden dark:border-slate-800 dark:bg-slate-900"
				aria-label="Primary"
			>
				{#each bottomNavItems as item (item.href)}
					{@const active =
						item.href === '/'
							? $page.url.pathname === '/'
							: $page.url.pathname.startsWith(item.href)}
					<a
						href={item.href}
						title={item.label}
						aria-label={item.label}
						class={`relative flex flex-1 flex-col items-center justify-center gap-0.5 py-2 text-[0.65rem] font-medium hover:text-sky-600 ${
							active
								? 'text-sky-600 dark:text-sky-400'
								: 'text-slate-500 dark:text-slate-400'
						}`}
					>
						<span class="relative">
							<Icon name={item.icon} class="h-6 w-6" />
							{#if item.href === '/notifications' && notifCount.value > 0}
								<span
									class="absolute -right-2 -top-1.5 rounded-full bg-sky-600 px-1 py-0.5 text-[0.6rem] font-semibold leading-none text-white"
								>
									{notifCount.value > 99 ? '99+' : notifCount.value}
								</span>
							{/if}
						</span>
						<span class="truncate">{item.label}</span>
					</a>
				{/each}
			</nav>
		</div>
	{:else if isLoginRoute}
		{@render children()}
	{:else}
		<div class="flex min-h-screen items-center justify-center text-sm text-slate-500">
			Redirecting...
		</div>
	{/if}

	{#if swUpdateAvailable}
		<div
			class="fixed bottom-4 left-1/2 z-50 flex -translate-x-1/2 items-center gap-3 rounded-lg border border-slate-200 bg-white px-4 py-3 shadow-lg dark:border-slate-700 dark:bg-slate-900"
		>
			<span class="text-sm text-slate-700 dark:text-slate-200">A new version is available.</span>
			<button
				type="button"
				onclick={() => updateServiceWorker(true)}
				class="rounded-md bg-sky-600 px-3 py-1 text-sm font-medium text-white hover:bg-sky-500"
			>
				Update
			</button>
			<button
				type="button"
				onclick={() => needRefresh.set(false)}
				aria-label="Dismiss update notification"
				class="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
			>
				<Icon name="close" class="h-4 w-4" />
			</button>
		</div>
	{/if}
</div>
