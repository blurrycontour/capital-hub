<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import {
		changePassword,
		updateProfile,
		uploadAvatar,
		requestAccountDeletion,
		confirmAccountDeletion,
		getVersion
	} from '$lib/api';
	import Icon from '$lib/Icon.svelte';
	import Modal from '$lib/Modal.svelte';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	const user = $derived(auth.user);

	// Build version (baked at build time).
	let appVersion = $state('');

	onMount(async () => {
		try {
			appVersion = await getVersion();
		} catch {
			/* version is best-effort */
		}
	});

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

	// Account deletion flow (email-code confirmation).
	let deleteModalOpen = $state(false);
	let deleteStage = $state<'confirm' | 'code'>('confirm');
	let deleteCode = $state('');
	let sendingCode = $state(false);
	let deletingAccount = $state(false);
	let deleteError = $state('');

	function openDeleteModal() {
		deleteStage = 'confirm';
		deleteCode = '';
		deleteError = '';
		deleteModalOpen = true;
	}

	async function sendDeletionCode() {
		sendingCode = true;
		deleteError = '';
		try {
			await requestAccountDeletion();
			deleteStage = 'code';
		} catch (err) {
			deleteError = err instanceof Error ? err.message : 'Failed to send confirmation code';
		} finally {
			sendingCode = false;
		}
	}

	async function confirmDelete() {
		if (!deleteCode.trim()) {
			deleteError = 'Enter the confirmation code from your email';
			return;
		}
		deletingAccount = true;
		deleteError = '';
		try {
			await confirmAccountDeletion(deleteCode.trim());
			deleteModalOpen = false;
			auth.set(null);
			await goto('/login');
		} catch (err) {
			deleteError = err instanceof Error ? err.message : 'Failed to delete account';
		} finally {
			deletingAccount = false;
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
			const prepared = await downscaleImage(file, 512);
			const updated = await uploadAvatar(prepared);
			auth.set(updated);
			success = 'Profile picture updated';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to upload picture';
		} finally {
			uploadingAvatar = false;
			input.value = '';
		}
	}

	// Downscale a profile picture in the browser before upload so very large
	// photos don't get stored at full resolution. Returns the original file when
	// it's already small enough or can't be processed (e.g. unsupported format).
	async function downscaleImage(file: File, maxSize: number): Promise<File> {
		if (!file.type.startsWith('image/') || file.type === 'image/gif') return file;
		try {
			const bitmap = await createImageBitmap(file);
			const { width, height } = bitmap;
			if (Math.max(width, height) <= maxSize) {
				bitmap.close();
				return file;
			}
			const scale = maxSize / Math.max(width, height);
			const w = Math.round(width * scale);
			const h = Math.round(height * scale);
			const canvas = document.createElement('canvas');
			canvas.width = w;
			canvas.height = h;
			const ctx = canvas.getContext('2d');
			if (!ctx) {
				bitmap.close();
				return file;
			}
			ctx.drawImage(bitmap, 0, 0, w, h);
			bitmap.close();
			const blob = await new Promise<Blob | null>((resolve) =>
				canvas.toBlob(resolve, 'image/jpeg', 0.9)
			);
			if (!blob) return file;
			const name = file.name.replace(/\.[^.]+$/, '') + '.jpg';
			return new File([blob], name, { type: 'image/jpeg' });
		} catch {
			return file;
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

	// OIDC-only accounts have no local password and cannot change one.
	const isOidcOnly = $derived(!!user && !user.hasPassword);

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
		<h1 class="text-2xl font-bold">Account</h1>
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
					<img src={user.avatarPath} alt="Profile" class="h-16 w-16 rounded-full object-cover" />
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
				<h2 class="text-lg font-semibold">Account details</h2>
			</div>
			<dl class="grid gap-4 sm:grid-cols-3">
				<div>
					<dt class="text-xs uppercase tracking-wide text-slate-500">Username</dt>
					<dd class="text-sm">{user.username}</dd>
				</div>
				<div>
					<dt class="text-xs uppercase tracking-wide text-slate-500">Role</dt>
					<dd class="text-sm capitalize">
						{user.role || (user.isAdmin ? 'administrator' : 'editor')}
					</dd>
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
				<div>
					<dt class="text-xs uppercase tracking-wide text-slate-500">Sign-in</dt>
					<dd class="text-sm">
						{#if isOidcOnly}
							<span
								class="inline-flex items-center gap-1 rounded bg-indigo-100 px-2 py-0.5 text-xs font-medium text-indigo-700 dark:bg-indigo-950/40 dark:text-indigo-300"
							>
								<Icon name="shield" class="h-3.5 w-3.5" />
								OIDC
							</span>
						{:else}
							<span class="text-slate-600 dark:text-slate-300">Password</span>
						{/if}
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

			{#if isOidcOnly}
				<div
					class="rounded-md border border-indigo-200 bg-indigo-50 px-3 py-2 text-sm text-indigo-800 dark:border-indigo-900/60 dark:bg-indigo-950/40 dark:text-indigo-200"
				>
					OIDC Authetication
				</div>
			{/if}

			{#if passwordError}
				<div
					class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-700 dark:bg-red-950/40 dark:text-red-200"
				>
					{passwordError}
				</div>
			{/if}
			{#if passwordSuccess}
				<div
					class="rounded-md border border-emerald-300 bg-emerald-50 px-3 py-2 text-sm text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200"
				>
					{passwordSuccess}
				</div>
			{/if}

			<div class="grid gap-4 sm:grid-cols-3" class:opacity-50={isOidcOnly}>
				<label class="space-y-1">
					<span class="text-sm font-medium">Current password</span>
					<input
						type="password"
						bind:value={currentPassword}
						required
						disabled={isOidcOnly}
						class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 disabled:cursor-not-allowed dark:border-slate-700 dark:bg-slate-900"
					/>
				</label>
				<label class="space-y-1">
					<span class="text-sm font-medium">New password</span>
					<input
						type="password"
						bind:value={newPassword}
						required
						disabled={isOidcOnly}
						class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 disabled:cursor-not-allowed dark:border-slate-700 dark:bg-slate-900"
					/>
				</label>
				<label class="space-y-1">
					<span class="text-sm font-medium">Confirm new password</span>
					<input
						type="password"
						bind:value={confirmPassword}
						required
						disabled={isOidcOnly}
						class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 disabled:cursor-not-allowed dark:border-slate-700 dark:bg-slate-900"
					/>
				</label>
			</div>

			<div class="border-t border-slate-200 pt-4 dark:border-slate-800">
				<button
					type="submit"
					disabled={changingPassword || isOidcOnly}
					class="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-500 disabled:cursor-not-allowed disabled:opacity-50"
				>
					{changingPassword ? 'Saving...' : 'Change password'}
				</button>
			</div>
		</form>

		<!-- Danger zone: delete account -->
		<section class="space-y-4 rounded-lg border border-rose-300 p-5 dark:border-rose-900/60">
			<div class="flex items-center gap-2">
				<Icon name="trash" class="h-5 w-5 text-rose-500" />
				<h2 class="text-lg font-semibold text-rose-700 dark:text-rose-400">Delete account</h2>
			</div>
			<p class="text-sm text-slate-600 dark:text-slate-400">
				Permanently delete your account and all collections, items and entries you own. This cannot
				be undone. We'll email you a confirmation code to verify it's really you.
			</p>
			<button
				type="button"
				onclick={openDeleteModal}
				class="inline-flex items-center gap-1.5 rounded-md border border-rose-300 px-4 py-2 text-sm font-medium text-rose-700 hover:bg-rose-50 dark:border-rose-900/60 dark:text-rose-400 dark:hover:bg-rose-950/40"
			>
				<Icon name="trash" class="h-4 w-4" />
				Delete my account
			</button>
		</section>
	{:else}
		<div class="rounded-lg border border-slate-200 p-4 text-sm text-slate-500 dark:border-slate-800">
			Loading profile...
		</div>
	{/if}

	<!-- App version -->
	<p class="flex items-center justify-center gap-1.5 text-xs text-slate-400">
		<Icon name="tag" class="h-3.5 w-3.5" />
		<span>Version:</span>
		<span class="font-mono">{appVersion || '—'}</span>
	</p>
</section>

<Modal bind:open={deleteModalOpen} title="Delete account">
	<div class="space-y-4">
		{#if deleteError}
			<div
				class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-700 dark:bg-red-950/40 dark:text-red-200"
			>
				{deleteError}
			</div>
		{/if}

		{#if deleteStage === 'confirm'}
			<p class="text-sm text-slate-600 dark:text-slate-300">
				This will permanently delete your account along with every collection, item and entry you
				own. This action cannot be undone.
			</p>
			<p class="text-sm text-slate-600 dark:text-slate-300">
				To continue, we'll send a confirmation code to your email address.
			</p>
		{:else}
			<p class="text-sm text-slate-600 dark:text-slate-300">
				We sent a confirmation code to your email. Enter it below to permanently delete your
				account.
			</p>
			<label class="block space-y-1">
				<span class="text-sm font-medium">Confirmation code</span>
				<input
					bind:value={deleteCode}
					inputmode="numeric"
					autocomplete="one-time-code"
					placeholder="123456"
					class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm tracking-widest outline-none focus:border-rose-500 dark:border-slate-700 dark:bg-slate-900"
				/>
			</label>
		{/if}
	</div>

	{#snippet footer()}
		<button
			type="button"
			onclick={() => (deleteModalOpen = false)}
			class="rounded-md border border-slate-300 px-4 py-2 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
		>
			Cancel
		</button>
		{#if deleteStage === 'confirm'}
			<button
				type="button"
				onclick={sendDeletionCode}
				disabled={sendingCode}
				class="rounded-md bg-rose-600 px-4 py-2 text-sm font-medium text-white hover:bg-rose-500 disabled:cursor-not-allowed disabled:opacity-50"
			>
				{sendingCode ? 'Sending…' : 'Send confirmation code'}
			</button>
		{:else}
			<button
				type="button"
				onclick={confirmDelete}
				disabled={deletingAccount}
				class="rounded-md bg-rose-600 px-4 py-2 text-sm font-medium text-white hover:bg-rose-500 disabled:cursor-not-allowed disabled:opacity-50"
			>
				{deletingAccount ? 'Deleting…' : 'Delete account'}
			</button>
		{/if}
	{/snippet}
</Modal>
