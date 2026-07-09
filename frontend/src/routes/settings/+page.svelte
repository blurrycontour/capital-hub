<script lang="ts">
	import {
		getPreferences,
		updatePreferences,
		setAmountDecimals,
		setNumberFormat,
		getVersion,
		type UserPreferences,
		type NumberFormat
	} from '$lib/api';
	import Icon from '$lib/Icon.svelte';
	import { onMount } from 'svelte';

	// Dashboard stats preference.
	let includeSharedInStats = $state(false);
	let savingPrefs = $state(false);
	let prefsError = $state('');

	// Display & notification preferences.
	let amountDecimals = $state(0);
	let numberFormat = $state<NumberFormat>('international');
	let notifyCollectionShared = $state(true);
	let notifyItemAdded = $state(true);
	let notifyEntryAdded = $state(true);

	// Build version (baked at build time).
	let appVersion = $state('');

	onMount(async () => {
		try {
			const prefs = await getPreferences();
			includeSharedInStats = prefs.includeSharedInStats;
			amountDecimals = prefs.amountDecimals;
			numberFormat = prefs.numberFormat;
			notifyCollectionShared = prefs.notifyCollectionShared;
			notifyItemAdded = prefs.notifyItemAdded;
			notifyEntryAdded = prefs.notifyEntryAdded;
		} catch {
			/* keep default */
		}
		try {
			appVersion = await getVersion();
		} catch {
			/* version is best-effort */
		}
	});

	// Snapshot the current preference values into a payload for the API.
	function currentPrefs(): UserPreferences {
		return {
			includeSharedInStats,
			amountDecimals,
			numberFormat,
			notifyCollectionShared,
			notifyItemAdded,
			notifyEntryAdded
		};
	}

	// Persist the given preferences and reflect the server's normalized result
	// back into local state (and apply the rounding precision app-wide).
	async function savePreferences(next: UserPreferences) {
		savingPrefs = true;
		prefsError = '';
		try {
			const prefs = await updatePreferences(next);
			includeSharedInStats = prefs.includeSharedInStats;
			amountDecimals = prefs.amountDecimals;
			numberFormat = prefs.numberFormat;
			notifyCollectionShared = prefs.notifyCollectionShared;
			notifyItemAdded = prefs.notifyItemAdded;
			notifyEntryAdded = prefs.notifyEntryAdded;
			setAmountDecimals(prefs.amountDecimals);
			setNumberFormat(prefs.numberFormat);
		} catch (err) {
			prefsError = err instanceof Error ? err.message : 'Failed to update preference';
		} finally {
			savingPrefs = false;
		}
	}

	async function toggleIncludeShared() {
		await savePreferences({ ...currentPrefs(), includeSharedInStats: !includeSharedInStats });
	}

	async function setDecimals(n: number) {
		await savePreferences({ ...currentPrefs(), amountDecimals: n });
	}

	async function setMoneyFormat(f: NumberFormat) {
		await savePreferences({ ...currentPrefs(), numberFormat: f });
	}

	async function toggleNotify(
		key: 'notifyCollectionShared' | 'notifyItemAdded' | 'notifyEntryAdded'
	) {
		await savePreferences({ ...currentPrefs(), [key]: !currentPrefs()[key] });
	}
</script>

<section class="mx-auto max-w-3xl space-y-6">
	<header>
		<h1 class="text-2xl font-bold">Settings</h1>
		<p class="text-sm text-slate-600 dark:text-slate-400">
			Configure your dashboard and notification preferences.
		</p>
	</header>

	<!-- Dashboard preferences -->
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

		<div
			class="flex items-start justify-between gap-4 border-t border-slate-100 pt-4 dark:border-slate-800"
		>
			<div class="space-y-0.5">
				<p class="text-sm font-medium">Money rounding</p>
				<p class="text-sm text-slate-500">
					Number of decimal places used when displaying amounts in stats, collections, items and
					entries.
				</p>
			</div>
			<select
				aria-label="Money rounding decimal places"
				disabled={savingPrefs}
				value={amountDecimals}
				onchange={(e) => setDecimals(Number((e.currentTarget as HTMLSelectElement).value))}
				class="mt-1 rounded-md border border-slate-300 px-3 py-2 text-sm disabled:opacity-50 dark:border-slate-700 dark:bg-slate-800"
			>
				<option value={0}>0 (whole numbers)</option>
				<option value={1}>1 decimal place</option>
				<option value={2}>2 decimal places</option>
			</select>
		</div>

		<div
			class="flex items-start justify-between gap-4 border-t border-slate-100 pt-4 dark:border-slate-800"
		>
			<div class="space-y-0.5">
				<p class="text-sm font-medium">Number format</p>
				<p class="text-sm text-slate-500">
					Digit grouping and decimal separator used when displaying amounts.
				</p>
			</div>
			<select
				aria-label="Money number format"
				disabled={savingPrefs}
				value={numberFormat}
				onchange={(e) =>
					setMoneyFormat((e.currentTarget as HTMLSelectElement).value as NumberFormat)}
				class="mt-1 rounded-md border border-slate-300 px-3 py-2 text-sm disabled:opacity-50 dark:border-slate-700 dark:bg-slate-800"
			>
				<option value="international">International (1,234,567.89)</option>
				<option value="indian">Indian (12,34,567.89)</option>
				<option value="european">European (1.234.567,89)</option>
			</select>
		</div>
	</section>

	<!-- Notification preferences -->
	<section class="space-y-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800">
		<div class="flex items-center gap-2">
			<Icon name="bell" class="h-5 w-5 text-slate-500" />
			<h2 class="text-lg font-semibold">Notifications</h2>
		</div>

		<p class="text-sm text-slate-500">
			Choose which events on collections shared with you create a notification.
		</p>

		<div class="flex items-start justify-between gap-4">
			<div class="space-y-0.5">
				<p class="text-sm font-medium">A collection is shared with me</p>
				<p class="text-sm text-slate-500">
					Notify me when someone shares one of their collections with me.
				</p>
			</div>
			<button
				type="button"
				role="switch"
				aria-checked={notifyCollectionShared}
				aria-label="Notify when a collection is shared with me"
				disabled={savingPrefs}
				onclick={() => toggleNotify('notifyCollectionShared')}
				class={`relative mt-1 inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors disabled:opacity-50 ${
					notifyCollectionShared ? 'bg-sky-600' : 'bg-slate-300 dark:bg-slate-700'
				}`}
			>
				<span
					class={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${
						notifyCollectionShared ? 'translate-x-5' : 'translate-x-0.5'
					}`}
				></span>
			</button>
		</div>

		<div
			class="flex items-start justify-between gap-4 border-t border-slate-100 pt-4 dark:border-slate-800"
		>
			<div class="space-y-0.5">
				<p class="text-sm font-medium">An item is added to a shared collection</p>
				<p class="text-sm text-slate-500">
					Notify me when someone adds an item to a collection I have access to.
				</p>
			</div>
			<button
				type="button"
				role="switch"
				aria-checked={notifyItemAdded}
				aria-label="Notify when an item is added to a shared collection"
				disabled={savingPrefs}
				onclick={() => toggleNotify('notifyItemAdded')}
				class={`relative mt-1 inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors disabled:opacity-50 ${
					notifyItemAdded ? 'bg-sky-600' : 'bg-slate-300 dark:bg-slate-700'
				}`}
			>
				<span
					class={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${
						notifyItemAdded ? 'translate-x-5' : 'translate-x-0.5'
					}`}
				></span>
			</button>
		</div>

		<div
			class="flex items-start justify-between gap-4 border-t border-slate-100 pt-4 dark:border-slate-800"
		>
			<div class="space-y-0.5">
				<p class="text-sm font-medium">An entry is made in a shared collection</p>
				<p class="text-sm text-slate-500">
					Notify me when someone records an entry on an item in a collection I have access to.
				</p>
			</div>
			<button
				type="button"
				role="switch"
				aria-checked={notifyEntryAdded}
				aria-label="Notify when an entry is made in a shared collection"
				disabled={savingPrefs}
				onclick={() => toggleNotify('notifyEntryAdded')}
				class={`relative mt-1 inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors disabled:opacity-50 ${
					notifyEntryAdded ? 'bg-sky-600' : 'bg-slate-300 dark:bg-slate-700'
				}`}
			>
				<span
					class={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${
						notifyEntryAdded ? 'translate-x-5' : 'translate-x-0.5'
					}`}
				></span>
			</button>
		</div>
	</section>

	<!-- Developers -->
	<section class="space-y-4 rounded-lg border border-slate-200 p-5 dark:border-slate-800">
		<div class="flex items-center gap-2">
			<Icon name="cog" class="h-5 w-5 text-slate-500" />
			<h2 class="text-lg font-semibold">Developers</h2>
		</div>
		<div class="flex items-start justify-between gap-4">
			<div class="space-y-0.5">
				<p class="text-sm font-medium">API documentation</p>
				<p class="text-sm text-slate-500">
					Interactive Swagger UI for every endpoint. Try calls directly using your current
					session.
				</p>
			</div>
			<a
				href="/api/docs"
				target="_blank"
				rel="noopener"
				class="mt-1 inline-flex shrink-0 items-center gap-1.5 rounded-md border border-slate-300 px-3 py-2 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			>
				<Icon name="arrow-right" class="h-4 w-4" />
				Open API Docs
			</a>
		</div>
	</section>

	<!-- App version -->
	<p class="flex items-center justify-center gap-1.5 text-xs text-slate-400">
		<Icon name="tag" class="h-3.5 w-3.5" />
		<span>Version:</span>
		<span class="font-mono">{appVersion || '—'}</span>
	</p>
</section>
