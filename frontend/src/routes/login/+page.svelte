<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { fetchMe, fetchProviders, login } from '$lib/api';
	import { auth } from '$lib/auth.svelte';
	import Icon from '$lib/Icon.svelte';

	let identifier = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state('');
	let oidcEnabled = $state(false);
	let oidcProviderName = $state('OIDC');

	onMount(async () => {
		const me = await fetchMe();
		if (me) {
			auth.set(me);
			await goto('/');
			return;
		}
		try {
			const providers = await fetchProviders();
			oidcEnabled = providers.oidcEnabled;
			oidcProviderName = providers.oidcProviderName || 'OIDC';
		} catch {
			oidcEnabled = false;
		}
	});

	async function submit() {
		error = '';
		loading = true;
		try {
			const user = await login(identifier, password);
			auth.set(user);
			await goto('/');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<section class="grid min-h-screen w-full md:grid-cols-2">
	<!-- Left: brand + welcome -->
	<div
		class="hidden flex-col justify-center gap-6 bg-gradient-to-br from-sky-700 to-slate-900 p-12 text-white md:flex"
	>
		<div class="flex items-center gap-3">
			<span class="flex h-12 w-12 items-center justify-center rounded-xl bg-white/15">
				<Icon name="logo" class="h-7 w-7" />
			</span>
			<span class="text-2xl font-bold">Capital Hub</span>
		</div>
		<div class="space-y-3">
			<h1 class="text-3xl font-bold leading-tight">Welcome to Capital Hub</h1>
			<p class="max-w-md text-sky-100/90">
				Self-hosted asset management. Track collections, organize your portfolio, and stay on top of
				notifications — all in one place.
			</p>
		</div>
	</div>

	<!-- Right: sign-in form -->
	<div class="flex items-center justify-center p-6">
		<div class="w-full max-w-md space-y-4">
			<div class="flex items-center justify-center gap-3 md:hidden">
				<span class="flex h-10 w-10 items-center justify-center rounded-lg bg-sky-600 text-white">
					<Icon name="logo" class="h-6 w-6" />
				</span>
				<span class="text-3xl font-bold">Capital Hub</span>
			</div>

			<div class="flex items-center justify-center gap-3">
				<h2 class="text-center text-l">Sign in to your account</h2>
			</div>

			{#if error}
				<div
					class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-700 dark:bg-red-950/40 dark:text-red-200"
				>
					{error}
				</div>
			{/if}

			<form
				onsubmit={(e) => {
					e.preventDefault();
					void submit();
				}}
						class="space-y-4 rounded-lg border border-slate-200 p-4 dark:border-slate-800"
			>
				<label class="block space-y-1">
					<p class="text-sm">Username or email</p>
					<input
						bind:value={identifier}
						required
						class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900"
					/>
				</label>

				<label class="block space-y-1">
					<p class="text-sm">Password</p>
					<input
						type="password"
						bind:value={password}
						required
						class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900"
					/>
				</label>

				<button
					type="submit"
					disabled={loading}
					class="w-full rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-500 disabled:cursor-not-allowed disabled:opacity-60"
				>
					{loading ? 'Signing in...' : 'Sign in'}
				</button>
			</form>

			{#if oidcEnabled}
				<div class="flex items-center gap-3">
					<hr class="flex-1 border-slate-300 dark:border-slate-700" />
					<span class="shrink-0 text-sm text-slate-500 dark:text-slate-400">Or continue with</span>
					<hr class="flex-1 border-slate-300 dark:border-slate-700" />
				</div>
				<a
					href="/api/v1/auth/oidc/login"
					class="block rounded-md border border-slate-300 px-4 py-2 text-center text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
				>
					Sign in with {oidcProviderName}
				</a>
			{/if}
		</div>
	</div>
</section>
