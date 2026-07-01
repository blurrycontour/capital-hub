<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import Icon from '$lib/Icon.svelte';
	import { auth } from '$lib/auth.svelte';
	import { breadcrumbs } from '$lib/breadcrumb.svelte';
	import {
		adminCreateUser,
		adminDeleteUser,
		adminUpdateUser,
		getOIDCSettings,
		getSMTPSettings,
		listUsers,
		testSMTP,
		updateOIDCSettings,
		updateSMTPSettings,
		type ApiUser,
		type OIDCSettings,
		type SMTPSettings
	} from '$lib/api';

	const currentUser = $derived(auth.user);

	let users = $state<ApiUser[]>([]);
	let smtp = $state<SMTPSettings>({
		host: '',
		port: 587,
		username: '',
		from: '',
		useTls: true,
		passwordSet: false
	});
	let oidc = $state<OIDCSettings>({
		enabled: false,
		issuerUrl: '',
		clientId: '',
		clientSecretSet: false,
		redirectUrl: '',
		adminGroup: '',
		providerName: 'OIDC',
		allowRegistration: true,
		envFields: []
	});

	let smtpPassword = $state('');
	let oidcClientSecret = $state('');
	let testTo = $state('');
	let loading = $state(true);
	let savingSmtp = $state(false);
	let savingOidc = $state(false);
	let testing = $state(false);
	let error = $state('');
	let success = $state('');

	// Backend health status.
	let backendStatus = $state<'loading' | string>('loading');

	// User editing state
	let editingUser = $state<ApiUser | null>(null);
	let editRole = $state('');
	let editActive = $state(true);
	let savingUser = $state(false);

	// New user creation state
	let showCreateUser = $state(false);
	let newUser = $state({
		username: '',
		email: '',
		displayName: '',
		password: '',
		role: 'editor'
	});
	let creatingUser = $state(false);

	onMount(async () => {
		breadcrumbs.setTrail([{ label: 'Administration', href: '/admin/settings' }]);
		await reload();
		try {
			const res = await fetch('/api/v1/health');
			const data = await res.json();
			backendStatus = data.status ?? 'unknown';
		} catch {
			backendStatus = 'unreachable';
		}
	});

	onDestroy(() => breadcrumbs.clearTrail());

	async function reload() {
		loading = true;
		error = '';
		success = '';
		try {
			const [smtpSettings, userList, oidcSettings] = await Promise.all([
				getSMTPSettings(),
				listUsers(),
				getOIDCSettings()
			]);
			smtp = smtpSettings;
			users = userList;
			oidc = oidcSettings;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load admin settings';
		} finally {
			loading = false;
		}
	}

	async function saveSMTP() {
		error = '';
		success = '';
		savingSmtp = true;
		try {
			smtp = await updateSMTPSettings({
				host: smtp.host,
				port: smtp.port,
				username: smtp.username,
				password: smtpPassword,
				from: smtp.from,
				useTls: smtp.useTls
			});
			smtpPassword = '';
			success = 'SMTP settings saved';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save SMTP settings';
		} finally {
			savingSmtp = false;
		}
	}

	async function runSMTPTest() {
		error = '';
		success = '';
		testing = true;
		try {
			await testSMTP(testTo);
			success = 'Test email sent';
		} catch (err) {
			error = err instanceof Error ? err.message : 'SMTP test failed';
		} finally {
			testing = false;
		}
	}

	async function saveOIDC() {
		error = '';
		success = '';
		savingOidc = true;
		try {
			oidc = await updateOIDCSettings({
				enabled: oidc.enabled,
				issuerUrl: oidc.issuerUrl,
				clientId: oidc.clientId,
				clientSecret: oidcClientSecret,
				redirectUrl: oidc.redirectUrl,
				adminGroup: oidc.adminGroup,
				providerName: oidc.providerName,
				allowRegistration: oidc.allowRegistration
			});
			oidcClientSecret = '';
			success = 'OIDC settings saved';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save OIDC settings';
		} finally {
			savingOidc = false;
		}
	}

	function startEditUser(u: ApiUser) {
		editingUser = u;
		editRole = u.role || (u.isAdmin ? 'administrator' : 'editor');
		editActive = u.isActive;
	}

	function cancelEditUser() {
		editingUser = null;
	}

	async function createUser() {
		creatingUser = true;
		error = '';
		success = '';
		try {
			const created = await adminCreateUser(newUser);
			users = [...users, created];
			showCreateUser = false;
			newUser = { username: '', email: '', displayName: '', password: '', role: 'editor' };
			success = 'User created';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create user';
		} finally {
			creatingUser = false;
		}
	}

	async function saveUserEdit() {
		if (!editingUser) return;
		savingUser = true;
		error = '';
		success = '';
		try {
			const updated = await adminUpdateUser(editingUser.id, editRole, editActive);
			users = users.map((u) => (u.id === updated.id ? updated : u));
			editingUser = null;
			success = 'User updated';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update user';
		} finally {
			savingUser = false;
		}
	}

	async function deleteUser(u: ApiUser) {
		if (!confirm(`Delete user "${u.displayName || u.username}"? This cannot be undone.`)) return;
		error = '';
		success = '';
		try {
			await adminDeleteUser(u.id);
			users = users.filter((x) => x.id !== u.id);
			success = 'User deleted';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete user';
		}
	}

	const oidcEnvSet = $derived(new Set(oidc.envFields));
</script>

<section class="mx-auto max-w-5xl space-y-6">
	<header>
		<div class="flex flex-wrap items-center justify-between gap-2">
			<h1 class="text-2xl font-bold">Admin Settings</h1>
			<span
				class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium"
				class:bg-emerald-100={backendStatus === 'ok'}
				class:text-emerald-800={backendStatus === 'ok'}
				class:bg-amber-100={backendStatus !== 'ok' && backendStatus !== 'loading'}
				class:text-amber-800={backendStatus !== 'ok' && backendStatus !== 'loading'}
				class:bg-slate-100={backendStatus === 'loading'}
				class:text-slate-600={backendStatus === 'loading'}
			>
				<span class="h-1.5 w-1.5 rounded-full bg-current"></span>
				Backend {backendStatus}
			</span>
		</div>
		<p class="text-sm text-slate-600 dark:text-slate-400">
			Manage server configuration and user accounts.
		</p>
	</header>

	{#if error}
		<div class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-700 dark:bg-red-950/40 dark:text-red-200">
			{error}
		</div>
	{/if}
	{#if success}
		<div class="rounded-md border border-emerald-300 bg-emerald-50 px-3 py-2 text-sm text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200">
			{success}
		</div>
	{/if}

	{#if loading}
		<div class="text-sm text-slate-500">Loading admin settings...</div>
	{:else}
		<!-- Email / SMTP -->
		<div class="space-y-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800">
			<div class="flex items-center gap-2 border-b border-slate-200 pb-3 dark:border-slate-800">
				<Icon name="bell" class="h-5 w-5 text-slate-500" />
				<div>
					<h2 class="text-lg font-semibold">Email (SMTP)</h2>
					<p class="text-xs text-slate-500">Outbound mail server used for notifications.</p>
				</div>
			</div>

			<div class="grid gap-4 md:grid-cols-2">
				<label class="space-y-1">
					<span class="text-sm font-medium">Host</span>
					<input bind:value={smtp.host} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900" />
				</label>
				<label class="space-y-1">
					<span class="text-sm font-medium">Port</span>
					<input type="number" bind:value={smtp.port} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900" />
				</label>
				<label class="space-y-1">
					<span class="text-sm font-medium">Username</span>
					<input bind:value={smtp.username} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900" />
				</label>
				<label class="space-y-1">
					<span class="text-sm font-medium">Password {smtp.passwordSet ? '(already set)' : ''}</span>
					<input type="password" bind:value={smtpPassword} placeholder="Leave blank to keep current" class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900" />
				</label>
				<label class="space-y-1">
					<span class="text-sm font-medium">From email</span>
					<input bind:value={smtp.from} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900" />
				</label>
				<label class="flex items-center gap-2 self-end pb-2 text-sm">
					<input type="checkbox" bind:checked={smtp.useTls} /> Use TLS
				</label>
			</div>

			<div class="border-t border-slate-200 pt-4 dark:border-slate-800">
				<button
					type="button"
					onclick={() => void saveSMTP()}
					disabled={savingSmtp}
					class="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-500 disabled:cursor-not-allowed disabled:opacity-60"
				>
					{savingSmtp ? 'Saving...' : 'Save SMTP settings'}
				</button>
			</div>

			<div class="space-y-2 rounded-md bg-slate-50 p-4 dark:bg-slate-800/40">
				<div class="text-sm font-medium">Send a test email</div>
				<div class="flex flex-wrap gap-2">
					<input bind:value={testTo} placeholder="recipient@example.com" class="min-w-0 flex-1 rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900" />
					<button
						type="button"
						onclick={() => void runSMTPTest()}
						disabled={testing}
						class="rounded-md border border-slate-300 px-3 py-2 text-sm hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:hover:bg-slate-800"
					>
						{testing ? 'Sending...' : 'Send test email'}
					</button>
				</div>
			</div>
		</div>

		<!-- OIDC settings -->
		<div class="space-y-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800">
			<div class="flex items-center gap-2 border-b border-slate-200 pb-3 dark:border-slate-800">
				<Icon name="shield" class="h-5 w-5 text-slate-500" />
				<div>
					<h2 class="text-lg font-semibold">OIDC / SSO</h2>
					<p class="text-xs text-slate-500">
						Single sign-on via an external identity provider. Fields marked
						<span class="font-medium text-amber-600 dark:text-amber-400">env</span>
						are controlled by environment variables and cannot be changed here.
					</p>
				</div>
			</div>

			<div class="grid gap-4 md:grid-cols-2">
				<label class="flex items-center gap-2 text-sm {oidcEnvSet.has('enabled') ? 'opacity-60' : ''}">
					<input type="checkbox" bind:checked={oidc.enabled} disabled={oidcEnvSet.has('enabled')} />
					<span class="font-medium">Enable OIDC login</span>
					{#if oidcEnvSet.has('enabled')}<span class="ml-1 rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">env</span>{/if}
				</label>
				<label class="flex items-center gap-2 text-sm {oidcEnvSet.has('allowRegistration') ? 'opacity-60' : ''}">
					<input type="checkbox" bind:checked={oidc.allowRegistration} disabled={oidcEnvSet.has('allowRegistration')} />
					<span class="font-medium">Allow new user registration via OIDC</span>
					{#if oidcEnvSet.has('allowRegistration')}<span class="ml-1 rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">env</span>{/if}
				</label>
			</div>

			<div class="grid gap-4 md:grid-cols-2">
				<label class="space-y-1 {oidcEnvSet.has('providerName') ? 'opacity-60' : ''}">
					<span class="text-sm font-medium">
						Provider name
						{#if oidcEnvSet.has('providerName')}<span class="ml-1 rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">env</span>{/if}
					</span>
					<input bind:value={oidc.providerName} disabled={oidcEnvSet.has('providerName')} placeholder="e.g. Authelia" class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 disabled:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:disabled:bg-slate-800" />
					<p class="text-xs text-slate-500">Shown on the login button: "Sign in with …"</p>
				</label>
				<label class="space-y-1 {oidcEnvSet.has('issuerUrl') ? 'opacity-60' : ''}">
					<span class="text-sm font-medium">
						Issuer URL
						{#if oidcEnvSet.has('issuerUrl')}<span class="ml-1 rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">env</span>{/if}
					</span>
					<input bind:value={oidc.issuerUrl} disabled={oidcEnvSet.has('issuerUrl')} placeholder="https://auth.example.com" class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 disabled:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:disabled:bg-slate-800" />
				</label>
				<label class="space-y-1 {oidcEnvSet.has('clientId') ? 'opacity-60' : ''}">
					<span class="text-sm font-medium">
						Client ID
						{#if oidcEnvSet.has('clientId')}<span class="ml-1 rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">env</span>{/if}
					</span>
					<input bind:value={oidc.clientId} disabled={oidcEnvSet.has('clientId')} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 disabled:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:disabled:bg-slate-800" />
				</label>
				<label class="space-y-1 {oidcEnvSet.has('clientSecret') ? 'opacity-60' : ''}">
					<span class="text-sm font-medium">
						Client secret {oidc.clientSecretSet ? '(already set)' : ''}
						{#if oidcEnvSet.has('clientSecret')}<span class="ml-1 rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">env</span>{/if}
					</span>
					<input type="password" bind:value={oidcClientSecret} disabled={oidcEnvSet.has('clientSecret')} placeholder="Leave blank to keep current" class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 disabled:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:disabled:bg-slate-800" />
				</label>
				<label class="space-y-1 {oidcEnvSet.has('redirectUrl') ? 'opacity-60' : ''}">
					<span class="text-sm font-medium">
						Redirect URI
						{#if oidcEnvSet.has('redirectUrl')}<span class="ml-1 rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">env</span>{/if}
					</span>
					<input bind:value={oidc.redirectUrl} disabled={oidcEnvSet.has('redirectUrl')} placeholder="https://app.example.com/api/v1/auth/oidc/callback" class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 disabled:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:disabled:bg-slate-800" />
					<p class="text-xs text-slate-500">Must end with <code>/api/v1/auth/oidc/callback</code></p>
				</label>
				<label class="space-y-1 {oidcEnvSet.has('adminGroup') ? 'opacity-60' : ''}">
					<span class="text-sm font-medium">
						Admin group (optional)
						{#if oidcEnvSet.has('adminGroup')}<span class="ml-1 rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">env</span>{/if}
					</span>
					<input bind:value={oidc.adminGroup} disabled={oidcEnvSet.has('adminGroup')} placeholder="admins" class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 disabled:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:disabled:bg-slate-800" />
					<p class="text-xs text-slate-500">Group claim value that grants administrator role.</p>
				</label>
			</div>

			<div class="border-t border-slate-200 pt-4 dark:border-slate-800">
				<button
					type="button"
					onclick={() => void saveOIDC()}
					disabled={savingOidc}
					class="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-500 disabled:cursor-not-allowed disabled:opacity-60"
				>
					{savingOidc ? 'Saving...' : 'Save OIDC settings'}
				</button>
			</div>
		</div>

		<!-- User management -->
		<div class="space-y-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800">
			<div class="flex items-center justify-between border-b border-slate-200 pb-3 dark:border-slate-800">
				<div class="flex items-center gap-2">
					<Icon name="user" class="h-5 w-5 text-slate-500" />
					<div>
						<h2 class="text-lg font-semibold">User management</h2>
						<p class="text-xs text-slate-500">Accounts with access to this instance.</p>
					</div>
				</div>
				<span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300">
					{users.length} user{users.length === 1 ? '' : 's'}
				</span>
			</div>

			<div>
				<button
					type="button"
					onclick={() => (showCreateUser = !showCreateUser)}
					class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
				>
					{showCreateUser ? 'Cancel' : '+ Add user'}
				</button>
			</div>

			{#if showCreateUser}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						void createUser();
					}}
					class="space-y-4 rounded-md border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/30"
				>
					<div class="grid gap-4 md:grid-cols-2">
						<label class="space-y-1">
							<span class="text-sm font-medium">Username</span>
							<input bind:value={newUser.username} required class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900" />
						</label>
						<label class="space-y-1">
							<span class="text-sm font-medium">Email</span>
							<input type="email" bind:value={newUser.email} required class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900" />
						</label>
						<label class="space-y-1">
							<span class="text-sm font-medium">Display name</span>
							<input bind:value={newUser.displayName} placeholder="Optional" class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900" />
						</label>
						<label class="space-y-1">
							<span class="text-sm font-medium">Password</span>
							<input type="password" bind:value={newUser.password} required minlength="8" placeholder="At least 8 characters" class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900" />
						</label>
						<label class="space-y-1">
							<span class="text-sm font-medium">Role</span>
							<select bind:value={newUser.role} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900">
								<option value="administrator">Administrator</option>
								<option value="editor">Editor</option>
								<option value="reader">Reader</option>
							</select>
						</label>
					</div>
					<button
						type="submit"
						disabled={creatingUser}
						class="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-500 disabled:cursor-not-allowed disabled:opacity-60"
					>
						{creatingUser ? 'Creating...' : 'Create user'}
					</button>
				</form>
			{/if}

			{#if users.length === 0}
				<div class="text-sm text-slate-500">No users found.</div>
			{:else}
				<div class="overflow-x-auto">
					<table class="w-full text-left text-sm">
						<thead class="text-xs uppercase tracking-wide text-slate-500">
							<tr class="border-b border-slate-200 dark:border-slate-800">
								<th class="py-2 pr-4 font-medium">Name</th>
								<th class="py-2 pr-4 font-medium">Email</th>
								<th class="py-2 pr-4 font-medium">Role</th>
								<th class="py-2 pr-4 font-medium">Status</th>
								<th class="py-2 font-medium">Actions</th>
							</tr>
						</thead>
						<tbody>
							{#each users as u (u.id)}
								{#if editingUser?.id === u.id}
									<tr class="border-b border-slate-100 bg-slate-50 dark:border-slate-800/60 dark:bg-slate-800/30">
										<td class="py-2 pr-4 font-medium">{u.displayName || u.username}</td>
										<td class="py-2 pr-4 text-slate-500">{u.email}</td>
										<td class="py-2 pr-4">
											<select
												bind:value={editRole}
												class="rounded border border-slate-300 bg-white px-2 py-1 text-sm dark:border-slate-600 dark:bg-slate-900"
											>
												<option value="administrator">Administrator</option>
												<option value="editor">Editor</option>
												<option value="reader">Reader</option>
											</select>
										</td>
										<td class="py-2 pr-4">
											<label class="flex items-center gap-1.5 text-sm">
												<input type="checkbox" bind:checked={editActive} />
												Active
											</label>
										</td>
										<td class="py-2">
											<div class="flex gap-2">
												<button
													type="button"
													onclick={() => void saveUserEdit()}
													disabled={savingUser}
													class="rounded bg-sky-600 px-2 py-1 text-xs text-white hover:bg-sky-500 disabled:opacity-60"
												>
													Save
												</button>
												<button
													type="button"
													onclick={cancelEditUser}
													class="rounded border border-slate-300 px-2 py-1 text-xs hover:bg-slate-100 dark:border-slate-600 dark:hover:bg-slate-800"
												>
													Cancel
												</button>
											</div>
										</td>
									</tr>
								{:else}
									<tr class="border-b border-slate-100 last:border-0 dark:border-slate-800/60">
										<td class="py-2 pr-4 font-medium">{u.displayName || u.username}</td>
										<td class="py-2 pr-4 text-slate-500">{u.email}</td>
										<td class="py-2 pr-4 capitalize">{u.role || (u.isAdmin ? 'administrator' : 'editor')}</td>
										<td class="py-2 pr-4">
											<span
												class="rounded px-2 py-0.5 text-xs"
												class:bg-emerald-100={u.isActive}
												class:text-emerald-800={u.isActive}
												class:bg-slate-200={!u.isActive}
												class:text-slate-700={!u.isActive}
											>
												{u.isActive ? 'Active' : 'Inactive'}
											</span>
										</td>
										<td class="py-2">
											<div class="flex gap-2">
												<button
													type="button"
													onclick={() => startEditUser(u)}
													class="rounded border border-slate-300 px-2 py-1 text-xs hover:bg-slate-100 dark:border-slate-600 dark:hover:bg-slate-800"
												>
													Edit
												</button>
												{#if u.id !== currentUser?.id}
													<button
														type="button"
														onclick={() => void deleteUser(u)}
														class="rounded border border-red-300 px-2 py-1 text-xs text-red-600 hover:bg-red-50 dark:border-red-700 dark:text-red-400 dark:hover:bg-red-950/30"
													>
														Delete
													</button>
												{/if}
											</div>
										</td>
									</tr>
								{/if}
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>
	{/if}
</section>
