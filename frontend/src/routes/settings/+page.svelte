<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { updateProfile } from '$lib/api';
	import Icon from '$lib/Icon.svelte';

	const user = $derived(auth.user);

	// Editable form state, seeded from the current user.
	let displayName = $state('');
	let email = $state('');
	let saving = $state(false);
	let error = $state('');
	let success = $state('');
	let seededFor = $state<number | null>(null);

	// Seed the form once per user without clobbering in-progress edits.
	$effect(() => {
		if (user && seededFor !== user.id) {
			displayName = user.displayName;
			email = user.email;
			seededFor = user.id;
		}
	});

	const dirty = $derived(!!user && (displayName !== user.displayName || email !== user.email));

	function reset() {
		if (!user) return;
		displayName = user.displayName;
		email = user.email;
		error = '';
		success = '';
	}

	async function save() {
		error = '';
		success = '';
		saving = true;
		try {
			const updated = await updateProfile({ displayName, email });
			auth.set(updated);
			success = 'Profile updated';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update profile';
		} finally {
			saving = false;
		}
	}
</script>

<section class="mx-auto max-w-3xl space-y-6">
	<header>
		<h1 class="text-2xl font-bold">User Settings</h1>
		<p class="text-sm text-slate-600 dark:text-slate-400">
			Manage your profile and account information.
		</p>
	</header>

	{#if error}
		<div
			class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-700 dark:bg-red-950/40 dark:text-red-200"
		>
			{error}
		</div>
	{/if}
	{#if success}
		<div
			class="rounded-md border border-emerald-300 bg-emerald-50 px-3 py-2 text-sm text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200"
		>
			{success}
		</div>
	{/if}

	{#if user}
		<!-- Profile (editable) -->
		<form
			onsubmit={(e) => {
				e.preventDefault();
				void save();
			}}
			class="space-y-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800"
		>
			<div class="flex items-center gap-2">
				<Icon name="user" class="h-5 w-5 text-slate-500" />
				<h2 class="text-lg font-semibold">Profile</h2>
			</div>

			<div class="grid gap-4 sm:grid-cols-2">
				<label class="space-y-1">
					<span class="text-sm font-medium">Display name</span>
					<input
						bind:value={displayName}
						placeholder="Your name"
						class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900"
					/>
				</label>
				<label class="space-y-1">
					<span class="text-sm font-medium">Email</span>
					<input
						type="email"
						bind:value={email}
						placeholder="you@example.com"
						class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900"
					/>
				</label>
			</div>

			<div class="flex items-center gap-2 border-t border-slate-200 pt-4 dark:border-slate-800">
				<button
					type="submit"
					disabled={!dirty || saving}
					class="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-500 disabled:cursor-not-allowed disabled:opacity-50"
				>
					{saving ? 'Saving...' : 'Save changes'}
				</button>
				<button
					type="button"
					onclick={reset}
					disabled={!dirty || saving}
					class="rounded-md border border-slate-300 px-4 py-2 text-sm hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800"
				>
					Cancel
				</button>
				{#if dirty}
					<span class="text-xs text-amber-600 dark:text-amber-400">Unsaved changes</span>
				{/if}
			</div>
		</form>

		<!-- Account (read-only) -->
		<div class="space-y-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800">
			<div class="flex items-center gap-2">
				<Icon name="shield" class="h-5 w-5 text-slate-500" />
				<h2 class="text-lg font-semibold">Account</h2>
			</div>
			<dl class="grid gap-4 sm:grid-cols-3">
				<div>
					<dt class="text-xs uppercase tracking-wide text-slate-500">Username</dt>
					<dd class="text-sm">{user.username}</dd>
				</div>
				<div>
					<dt class="text-xs uppercase tracking-wide text-slate-500">Role</dt>
					<dd class="text-sm">{user.isAdmin ? 'Administrator' : 'Member'}</dd>
				</div>
				<div>
					<dt class="text-xs uppercase tracking-wide text-slate-500">Status</dt>
					<dd class="text-sm">
						<span
							class="rounded px-2 py-0.5 text-xs"
							class:bg-emerald-100={user.isActive}
							class:text-emerald-800={user.isActive}
							class:bg-slate-200={!user.isActive}
							class:text-slate-700={!user.isActive}
						>
							{user.isActive ? 'Active' : 'Inactive'}
						</span>
					</dd>
				</div>
			</dl>
		</div>
	{:else}
		<div class="rounded-lg border border-slate-200 p-4 text-sm text-slate-500 dark:border-slate-800">
			Loading profile...
		</div>
	{/if}
</section>
