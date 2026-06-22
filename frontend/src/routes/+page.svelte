<script lang="ts">
	import { onMount } from 'svelte';

	let status = $state<'loading' | string>('loading');

	onMount(async () => {
		try {
			const res = await fetch('/api/v1/health');
			const data = await res.json();
			status = data.status ?? 'unknown';
		} catch {
			status = 'unreachable';
		}
	});
</script>

<section class="mx-auto max-w-2xl space-y-4">
	<h1 class="text-2xl font-bold">Welcome to Capital&#8209;Hub</h1>
	<p class="text-slate-600 dark:text-slate-400">
		Self-hosted asset management. The interface is under construction.
	</p>
	<div class="rounded-lg border border-slate-200 p-4 dark:border-slate-800">
		<span class="text-sm font-medium">Backend status:</span>
		<span
			class="ml-2 rounded px-2 py-0.5 text-sm"
			class:bg-green-100={status === 'ok'}
			class:text-green-800={status === 'ok'}
			class:bg-amber-100={status !== 'ok' && status !== 'loading'}
			class:text-amber-800={status !== 'ok' && status !== 'loading'}
		>
			{status}
		</span>
	</div>
</section>
