<script lang="ts">
	import { onMount } from 'svelte';
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

<section class="mx-auto max-w-4xl space-y-6">
	<h1 class="text-2xl font-bold">Admin Settings</h1>

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
		<div class="grid gap-6 md:grid-cols-2">
			<div class="space-y-4 rounded-lg border border-slate-200 p-4 dark:border-slate-800">
				<h2 class="text-lg font-semibold">SMTP</h2>
				<div class="grid gap-3">
					<label class="space-y-1">
						<span class="text-sm">Host</span>
						<input bind:value={smtp.host} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-900" />
					</label>
					<label class="space-y-1">
						<span class="text-sm">Port</span>
						<input type="number" bind:value={smtp.port} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-900" />
					</label>
					<label class="space-y-1">
						<span class="text-sm">Username</span>
						<input bind:value={smtp.username} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-900" />
					</label>
					<label class="space-y-1">
						<span class="text-sm">Password {smtp.passwordSet ? '(already set)' : ''}</span>
						<input type="password" bind:value={password} placeholder="Leave blank to keep current" class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-900" />
					</label>
					<label class="space-y-1">
						<span class="text-sm">From email</span>
						<input bind:value={smtp.from} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-900" />
					</label>
					<label class="inline-flex items-center gap-2 text-sm">
						<input type="checkbox" bind:checked={smtp.useTls} /> Use TLS
					</label>
				</div>
				<button
					type="button"
					onclick={() => void saveSMTP()}
					disabled={saving}
					class="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-500 disabled:cursor-not-allowed disabled:opacity-60"
				>
					{saving ? 'Saving...' : 'Save SMTP'}
				</button>

				<div class="space-y-2 border-t border-slate-200 pt-3 dark:border-slate-800">
					<div class="text-sm font-medium">Test email</div>
					<input bind:value={testTo} placeholder="recipient@example.com" class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-900" />
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

			<div class="space-y-3 rounded-lg border border-slate-200 p-4 dark:border-slate-800">
				<h2 class="text-lg font-semibold">Users</h2>
				{#if users.length === 0}
					<div class="text-sm text-slate-500">No users found.</div>
				{:else}
					<ul class="space-y-2">
						{#each users as u}
							<li class="rounded-md border border-slate-200 p-3 text-sm dark:border-slate-800">
								<div class="font-medium">{u.displayName || u.username}</div>
								<div class="text-slate-500">{u.email}</div>
								<div class="mt-1 text-xs">
									{u.isAdmin ? 'Admin' : 'User'} · {u.isActive ? 'Active' : 'Inactive'}
								</div>
							</li>
						{/each}
					</ul>
				{/if}
			</div>
		</div>
	{/if}
</section>
