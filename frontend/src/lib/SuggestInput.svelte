<script lang="ts">
	/**
	 * A text field that offers past values for the same field.
	 *
	 * The full list appears on focus so common answers can be picked without
	 * typing, and narrows as the user types. It stays a plain text input — any
	 * value is allowed, the suggestions are only a shortcut.
	 */
	import Icon from '$lib/Icon.svelte';

	let {
		value = $bindable(''),
		suggestions = [],
		placeholder = '',
		id,
		disabled = false
	}: {
		value?: string;
		suggestions?: string[];
		placeholder?: string;
		id?: string;
		disabled?: boolean;
	} = $props();

	let open = $state(false);
	let highlighted = $state(-1);
	let el = $state<HTMLDivElement | null>(null);
	let input = $state<HTMLInputElement | null>(null);

	// Match on any part of the value so "bob" finds "Bob's Garage"; an exact
	// match is dropped because offering what is already typed is just noise.
	const matches = $derived.by(() => {
		const q = value.trim().toLowerCase();
		const list = q
			? suggestions.filter((s) => s.toLowerCase().includes(q))
			: suggestions;
		return list.filter((s) => s.toLowerCase() !== q).slice(0, 50);
	});

	const showList = $derived(open && matches.length > 0);

	// Keep the highlight valid as the list narrows under the user's typing.
	$effect(() => {
		if (highlighted >= matches.length) highlighted = -1;
	});

	function choose(s: string) {
		value = s;
		open = false;
		highlighted = -1;
		input?.focus();
	}

	function onkeydown(e: KeyboardEvent) {
		if (!showList) {
			// Let the user summon the list again after dismissing it.
			if (e.key === 'ArrowDown') {
				open = true;
				e.preventDefault();
			}
			return;
		}
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			highlighted = Math.min(highlighted + 1, matches.length - 1);
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			highlighted = Math.max(highlighted - 1, -1);
		} else if (e.key === 'Enter' && highlighted >= 0) {
			// Only intercept Enter while a suggestion is actually highlighted, so
			// it otherwise submits as usual.
			e.preventDefault();
			choose(matches[highlighted]);
		} else if (e.key === 'Escape') {
			e.preventDefault();
			open = false;
			highlighted = -1;
		}
	}

	let list = $state<HTMLUListElement | null>(null);

	$effect(() => {
		if (!showList || !list) return;
		// `nearest` scrolls the minimum needed, so the field does not jump when
		// the list already fits.
		const el = list;
		requestAnimationFrame(() => el.scrollIntoView({ block: 'nearest' }));
	});

	$effect(() => {
		if (!open) return;
		function onDocPointer(ev: PointerEvent) {
			if (el && !el.contains(ev.target as Node)) open = false;
		}
		document.addEventListener('pointerdown', onDocPointer, true);
		return () => document.removeEventListener('pointerdown', onDocPointer, true);
	});
</script>

<div bind:this={el} class="relative">
	<input
		bind:this={input}
		{id}
		type="text"
		bind:value
		{placeholder}
		{disabled}
		autocomplete="off"
		role="combobox"
		aria-expanded={showList}
		aria-autocomplete="list"
		aria-controls={id ? `${id}-suggestions` : undefined}
		onfocus={() => (open = true)}
		onclick={() => (open = true)}
		oninput={() => {
			open = true;
			highlighted = -1;
		}}
		{onkeydown}
		class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
	/>

	{#if showList}
		<ul
			bind:this={list}
			id={id ? `${id}-suggestions` : undefined}
			role="listbox"
			class="ch-no-scrollbar absolute z-30 mt-1 max-h-48 w-full overflow-y-auto rounded-md border border-slate-200 bg-white py-1 shadow-lg dark:border-slate-700 dark:bg-slate-900"
		>
			{#each matches as suggestion, i (suggestion)}
				<li role="presentation">
					<button
						type="button"
						role="option"
						aria-selected={i === highlighted}
						onmouseenter={() => (highlighted = i)}
						onclick={() => choose(suggestion)}
						class={`flex w-full items-center gap-2 px-3 py-2 text-left text-sm ${
							i === highlighted ? 'bg-slate-100 dark:bg-slate-800' : ''
						}`}
					>
						<Icon name="activity" class="h-3.5 w-3.5 shrink-0 text-slate-400" />
						<span class="truncate">{suggestion}</span>
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>
