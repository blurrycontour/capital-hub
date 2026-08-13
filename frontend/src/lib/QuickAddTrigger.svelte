<script lang="ts">
	/**
	 * Quick add trigger. On mobile a floating button above the bottom
	 * navigation that unfolds in place into three labelled actions over a
	 * blurred page; on desktop a sidebar button with a small popover, since a
	 * floating button would sit oddly beside a permanent sidebar.
	 *
	 * The dialogs themselves are rendered by QuickAddModals at the layout root.
	 */
	import Icon, { type IconName } from '$lib/Icon.svelte';
	import { quickAdd, type QuickAddKind } from '$lib/quickAdd.svelte';

	let {
		variant = 'fab',
		collapsed = false
	}: {
		variant?: 'fab' | 'sidebar';
		/** Sidebar only: render icon-only when the sidebar is collapsed. */
		collapsed?: boolean;
	} = $props();

	let openMenu = $state(false);

	type Action = { key: QuickAddKind; label: string; icon: IconName; tone: string };
	// Colours match each type's identity elsewhere in the app: collections are
	// sky, items emerald, entries amber.
	const actions: Action[] = [
		{ key: 'entry', label: 'Entry', icon: 'list', tone: 'bg-amber-500 hover:bg-amber-600' },
		{ key: 'item', label: 'Item', icon: 'cube', tone: 'bg-emerald-600 hover:bg-emerald-700' },
		{
			key: 'collection',
			label: 'Collection',
			icon: 'collections',
			tone: 'bg-sky-600 hover:bg-sky-700'
		}
	];

	function choose(kind: QuickAddKind) {
		openMenu = false;
		quickAdd.open(kind);
	}

	$effect(() => {
		if (!openMenu) return;
		function onKey(e: KeyboardEvent) {
			if (e.key === 'Escape') openMenu = false;
		}
		document.addEventListener('keydown', onKey);
		return () => document.removeEventListener('keydown', onKey);
	});
</script>

{#if variant === 'sidebar'}
	<div class="relative px-2 pb-2">
		<button
			type="button"
			onclick={() => (openMenu = !openMenu)}
			aria-haspopup="menu"
			aria-expanded={openMenu}
			title="Quick add"
			aria-label="Quick add"
			class="flex w-full items-center gap-3 rounded-md bg-sky-600 px-3 py-2 font-medium text-white hover:bg-sky-700"
			class:justify-center={collapsed}
		>
			<Icon name="plus" class="h-5 w-5 shrink-0" />
			{#if !collapsed}<span class="truncate">Quick add</span>{/if}
		</button>

		{#if openMenu}
			<button
				type="button"
				class="fixed inset-0 z-40 cursor-default"
				aria-label="Close quick add"
				onclick={() => (openMenu = false)}
			></button>
			<div
				role="menu"
				class="absolute bottom-full left-2 right-2 z-50 mb-1 overflow-hidden rounded-md border border-slate-200 bg-white py-1 shadow-lg dark:border-slate-700 dark:bg-slate-900"
			>
				{#each actions as action (action.key)}
					<button
						type="button"
						role="menuitem"
						onclick={() => choose(action.key)}
						class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-slate-100 dark:hover:bg-slate-800"
					>
						<span
							class={`flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-white ${action.tone}`}
						>
							<Icon name={action.icon} class="h-3.5 w-3.5" />
						</span>
						{action.label}
					</button>
				{/each}
			</div>
		{/if}
	</div>
{:else}
	{#if openMenu}
		<button
			type="button"
			aria-label="Close quick add"
			onclick={() => (openMenu = false)}
			class="fixed inset-0 z-40 bg-slate-900/30 backdrop-blur-sm md:hidden"
		></button>
	{/if}

	<div class="ch-quick-add fixed right-4 z-50 flex flex-col items-end gap-3 md:hidden">
		{#if openMenu}
			{#each actions as action, i (action.key)}
				<div
					class="ch-quick-add-action flex items-center gap-3"
					style={`animation-delay:${i * 40}ms`}
				>
					<span
						class="rounded-md bg-white px-2 py-1 text-sm font-medium shadow-sm dark:bg-slate-800 dark:text-slate-100"
					>
						{action.label}
					</span>
					<button
						type="button"
						onclick={() => choose(action.key)}
						aria-label={`Add ${action.label.toLowerCase()}`}
						class={`flex h-12 w-12 items-center justify-center rounded-full text-white shadow-lg ${action.tone}`}
					>
						<Icon name={action.icon} class="h-5 w-5" />
					</button>
				</div>
			{/each}
		{/if}

		<button
			type="button"
			onclick={() => (openMenu = !openMenu)}
			aria-haspopup="menu"
			aria-expanded={openMenu}
			aria-label={openMenu ? 'Close quick add' : 'Quick add'}
			class="flex h-14 w-14 items-center justify-center rounded-full bg-sky-600 text-white shadow-lg transition-transform hover:bg-sky-700"
			class:rotate-45={openMenu}
		>
			<Icon name="plus" class="h-6 w-6" />
		</button>
	</div>
{/if}
