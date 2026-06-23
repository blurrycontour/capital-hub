<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { fetchMe, fetchProviders, login } from '$lib/api';

	let identifier = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state('');
	let oidcEnabled = $state(false);

	onMount(async () => {
		const me = await fetchMe();
		if (me) {
			await goto('/');
			return;
		}
		try {
			const providers = await fetchProviders();
			oidcEnabled = providers.oidcEnabled;
		} catch {
			oidcEnabled = false;
		}
	});

	async function submit() {
		error = '';
		loading = true;
		try {
			await login(identifier, password);
			await goto('/');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<section class="mx-auto max-w-md space-y-4">
	<h1 class="text-2xl font-bold">Sign in</h1>
	<p class="text-sm text-slate-600 dark:text-slate-400">Use username/email + password or OIDC.</p>

	{#if error}
		<div class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-700 dark:bg-red-950/40 dark:text-red-200">
			{error}
		</div>
	{/if}

	<form
		onsubmit={(e) => {
			e.preventDefault();
			void submit();
		}}
		class="space-y-3 rounded-lg border border-slate-200 p-4 dark:border-slate-800"
	>
		<label class="block space-y-1">
			<span class="text-sm">Username or email</span>
			<input
				bind:value={identifier}
				required
				class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900"
			/>
		</label>

		<label class="block space-y-1">
			<span class="text-sm">Password</span>
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
		<a
			href="/api/v1/auth/oidc/login"
			class="block rounded-md border border-slate-300 px-4 py-2 text-center text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
		>
			Sign in with OIDC
		</a>
	{/if}
</section>
