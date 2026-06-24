<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { changePassword, updateProfile, uploadAvatar, getPreferences, updatePreferences } from '$lib/api';
	import Icon from '$lib/Icon.svelte';
	import { onMount } from 'svelte';

	const user = $derived(auth.user);

	// Editable form state, seeded from the current user.
	let displayName = $state('');
	let email = $state('');
	let saving = $state(false);
	let error = $state('');
	let success = $state('');
	let seededFor = $state<number | null>(null);

	// Password change state.
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let changingPassword = $state(false);
	let passwordError = $state('');
	let passwordSuccess = $state('');

	// Avatar upload state.
	let avatarInput = $state<HTMLInputElement | null>(null);
	let uploadingAvatar = $state(false);

	// Dashboard stats preference.
	let includeSharedInStats = $state(false);
	let savingPrefs = $state(false);
	let prefsError = $state('');

	onMount(async () => {
		try {
			const prefs = await getPreferences();
			includeSharedInStats = prefs.includeSharedInStats;
		} catch {
			/* keep default */
		}
	});

	async function toggleIncludeShared() {
		const next = !includeSharedInStats;
		savingPrefs = true;
		prefsError = '';
		try {
			const prefs = await updatePreferences({ includeSharedInStats: next });
			includeSharedInStats = prefs.includeSharedInStats;
		} catch (err) {
			prefsError = err instanceof Error ? err.message : 'Failed to update preference';
		} finally {
			savingPrefs = false;
		}
	}

	const initials = $derived.by(() => {
		const source = (user?.displayName || user?.username || '').trim();
		if (!source) return '?';
		const parts = source.split(/\s+/);
		const letters =
			parts.length > 1 ? parts[0][0] + parts[parts.length - 1][0] : source.slice(0, 2);
		return letters.toUpperCase();
	});

	async function onAvatarChange(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		uploadingAvatar = true;
		error = '';
		success = '';
		try {
			const updated = await uploadAvatar(file);
			auth.set(updated);
			success = 'Profile picture updated';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to upload picture';
		} finally {
			uploadingAvatar = false;
			input.value = '';
		}
	}

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

	async function savePassword() {
		passwordError = '';
		passwordSuccess = '';
		if (newPassword !== confirmPassword) {
			passwordError = 'New passwords do not match';
			return;
		}
		changingPassword = true;
		try {
			await changePassword(currentPassword, newPassword);
			passwordSuccess = 'Password changed successfully';
			currentPassword = '';
			newPassword = '';
			confirmPassword = '';
		} catch (err) {
			passwordError = err instanceof Error ? err.message : 'Failed to change password';
		} finally {
			changingPassword = false;
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

			<!-- Profile picture -->
			<div class="flex items-center gap-4">
				{#if user.avatarPath}
					<img
						src={user.avatarPath}
						alt="Profile"
						class="h-16 w-16 rounded-full object-cover"
					/>
				{:else}
					<span
						class="flex h-16 w-16 items-center justify-center rounded-full bg-sky-600 text-lg font-semibold text-white"
					>
						{initials}
					</span>
				{/if}
				<div>
					<input
						bind:this={avatarInput}
						type="file"
						accept="image/*"
						class="hidden"
						onchange={onAvatarChange}
					/>
					<button
						type="button"
						onclick={() => avatarInput?.click()}
						disabled={uploadingAvatar}
						class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 disabled:opacity-60 dark:border-slate-700 dark:hover:bg-slate-800"
					>
						<Icon name="photo" class="h-4 w-4" />
						{uploadingAvatar ? 'Uploading…' : user.avatarPath ? 'Change picture' : 'Upload picture'}
					</button>
					<p class="mt-1 text-xs text-slate-500">JPG, PNG, GIF or WebP.</p>
				</div>
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
					<dd class="text-sm capitalize">{user.role || (user.isAdmin ? 'administrator' : 'editor')}</dd>
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

		<!-- Change password -->
		<form
			onsubmit={(e) => {
				e.preventDefault();
				void savePassword();
			}}
			class="space-y-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800"
		>
			<div class="flex items-center gap-2">
				<Icon name="shield" class="h-5 w-5 text-slate-500" />
				<h2 class="text-lg font-semibold">Change password</h2>
			</div>

			{#if passwordError}
				<div class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-700 dark:bg-red-950/40 dark:text-red-200">
					{passwordError}
				</div>
			{/if}
			{#if passwordSuccess}
				<div class="rounded-md border border-emerald-300 bg-emerald-50 px-3 py-2 text-sm text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200">
					{passwordSuccess}
				</div>
			{/if}

			<div class="grid gap-4 sm:grid-cols-3">
				<label class="space-y-1">
					<span class="text-sm font-medium">Current password</span>
					<input
						type="password"
						bind:value={currentPassword}
						required
						class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900"
					/>
				</label>
				<label class="space-y-1">
					<span class="text-sm font-medium">New password</span>
					<input
						type="password"
						bind:value={newPassword}
						required
						class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900"
					/>
				</label>
				<label class="space-y-1">
					<span class="text-sm font-medium">Confirm new password</span>
					<input
						type="password"
						bind:value={confirmPassword}
						required
						class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900"
					/>
				</label>
			</div>

			<div class="border-t border-slate-200 pt-4 dark:border-slate-800">
				<button
					type="submit"
					disabled={changingPassword}
					class="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-500 disabled:cursor-not-allowed disabled:opacity-50"
				>
					{changingPassword ? 'Saving...' : 'Change password'}
				</button>
			</div>
		</form>

		<!-- Preferences -->
		<section class="space-y-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800">
			<div class="flex items-center gap-2">
				<Icon name="dashboard" class="h-5 w-5 text-slate-500" />
				<h2 class="text-lg font-semibold">Dashboard</h2>
			</div>

			{#if prefsError}
				<div
					class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-700 dark:bg-red-950/40 dark:text-red-200"
				>
					{prefsError}
				</div>
			{/if}

			<div class="flex items-start justify-between gap-4">
				<div class="space-y-0.5">
					<p class="text-sm font-medium">Include shared collections in statistics</p>
					<p class="text-sm text-slate-500">
						When enabled, collections shared with you also contribute to the totals on your
						dashboard.
					</p>
				</div>
				<button
					type="button"
					role="switch"
					aria-checked={includeSharedInStats}
					aria-label="Include shared collections in statistics"
					disabled={savingPrefs}
					onclick={toggleIncludeShared}
					class={`relative mt-1 inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors disabled:opacity-50 ${
						includeSharedInStats ? 'bg-sky-600' : 'bg-slate-300 dark:bg-slate-700'
					}`}
				>
					<span
						class={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${
							includeSharedInStats ? 'translate-x-5' : 'translate-x-0.5'
						}`}
					></span>
				</button>
			</div>
		</section>
	{:else}
		<div class="rounded-lg border border-slate-200 p-4 text-sm text-slate-500 dark:border-slate-800">
			Loading profile...
		</div>
	{/if}
</section>
