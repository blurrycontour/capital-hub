<script lang="ts">
	import { onMount } from 'svelte';
	import Icon from '$lib/Icon.svelte';
	import {
		getSMTPSettings,
		listUsers,
		testSMTP,
		updateSMTPSettings,
		type ApiUser,
		type SMTPSettings
	} from '$lib/api';

	let users = $state<ApiUser[]>([]);
	let smtp = $state<SMTPSettings>({
		host: '',
		port: 587,
		username: '',
		from: '',
		useTls: true,
		passwordSet: false
	});
	let password = $state('');
	let testTo = $state('');
	let loading = $state(true);
	let saving = $state(false);
	let testing = $state(false);
	let error = $state('');
	let success = $state('');

	onMount(async () => {
		await reload();
	});

	async function reload() {
		loading = true;
		error = '';
		success = '';
		try {
			const [settings, userList] = await Promise.all([getSMTPSettings(), listUsers()]);
			smtp = settings;
			users = userList;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load admin settings';
		} finally {
			loading = false;
		}
	}

	async function saveSMTP() {
		error = '';
		success = '';
		saving = true;
		try {
			smtp = await updateSMTPSettings({
				host: smtp.host,
				port: smtp.port,
				username: smtp.username,
				password,
				from: smtp.from,
				useTls: smtp.useTls
			});
			password = '';
			success = 'SMTP settings saved';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save SMTP settings';
		} finally {
			saving = false;
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
</script>

<section class="mx-auto max-w-5xl space-y-6">
	<header>
		<h1 class="text-2xl font-bold">Admin Settings</h1>
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
					<input type="password" bind:value={password} placeholder="Leave blank to keep current" class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-sky-500 dark:border-slate-700 dark:bg-slate-900" />
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
					disabled={saving}
					class="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-500 disabled:cursor-not-allowed disabled:opacity-60"
				>
					{saving ? 'Saving...' : 'Save SMTP settings'}
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
								<th class="py-2 font-medium">Status</th>
							</tr>
						</thead>
						<tbody>
							{#each users as u (u.id)}
								<tr class="border-b border-slate-100 last:border-0 dark:border-slate-800/60">
									<td class="py-2 pr-4 font-medium">{u.displayName || u.username}</td>
									<td class="py-2 pr-4 text-slate-500">{u.email}</td>
									<td class="py-2 pr-4">{u.isAdmin ? 'Admin' : 'User'}</td>
									<td class="py-2">
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
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>
	{/if}
</section>
