<script lang="ts">
	import Icon from '$lib/Icon.svelte';

	let {
		images = [],
		onadd,
		ondelete,
		uploading = false
	}: {
		images?: string[];
		onadd?: (file: File) => void;
		ondelete?: (path: string) => void;
		uploading?: boolean;
	} = $props();

	let index = $state(0);
	let fileInput = $state<HTMLInputElement | null>(null);

	// Keep the active slide in range as images are added/removed.
	$effect(() => {
		if (index > images.length - 1) index = Math.max(0, images.length - 1);
	});

	const current = $derived(images[index] ?? '');

	function prev() {
		if (images.length === 0) return;
		index = (index - 1 + images.length) % images.length;
	}

	function next() {
		if (images.length === 0) return;
		index = (index + 1) % images.length;
	}

	function onFileChange(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (file) onadd?.(file);
		input.value = '';
	}
</script>

<div class="space-y-3">
	<div
		class="relative flex h-72 items-center justify-center overflow-hidden rounded-lg border border-slate-200 bg-slate-100 dark:border-slate-800 dark:bg-slate-800"
	>
		{#if current}
			<img src={current} alt={`Image ${index + 1}`} class="h-full w-full object-contain" />
			{#if images.length > 1}
				<button
					type="button"
					class="absolute left-2 top-1/2 -translate-y-1/2 rounded-full bg-slate-900/60 p-1.5 text-white hover:bg-slate-900/80"
					aria-label="Previous image"
					onclick={prev}
				>
					<Icon name="chevron-left" class="h-5 w-5" />
				</button>
				<button
					type="button"
					class="absolute right-2 top-1/2 -translate-y-1/2 rounded-full bg-slate-900/60 p-1.5 text-white hover:bg-slate-900/80"
					aria-label="Next image"
					onclick={next}
				>
					<Icon name="chevron-right" class="h-5 w-5" />
				</button>
				<span
					class="absolute bottom-2 right-2 rounded-full bg-slate-900/60 px-2 py-0.5 text-xs text-white"
				>
					{index + 1} / {images.length}
				</span>
			{/if}
			{#if ondelete}
				<button
					type="button"
					class="absolute right-2 top-2 rounded-full bg-slate-900/60 p-1.5 text-white hover:bg-rose-600"
					aria-label="Delete this image"
					onclick={() => ondelete?.(current)}
				>
					<Icon name="trash" class="h-4 w-4" />
				</button>
			{/if}
		{:else}
			<Icon name="photo" class="h-12 w-12 text-slate-400" />
		{/if}
	</div>

	<!-- Thumbnails -->
	{#if images.length > 1}
		<div class="flex flex-wrap gap-2">
			{#each images as img, i (img)}
				<button
					type="button"
					class="h-14 w-14 overflow-hidden rounded-md border-2"
					class:border-sky-500={i === index}
					class:border-transparent={i !== index}
					aria-label={`Show image ${i + 1}`}
					onclick={() => (index = i)}
				>
					<img src={img} alt={`Thumbnail ${i + 1}`} class="h-full w-full object-cover" />
				</button>
			{/each}
		</div>
	{/if}

	{#if onadd}
		<input bind:this={fileInput} type="file" accept="image/*" class="hidden" onchange={onFileChange} />
		<button
			type="button"
			class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm hover:bg-slate-100 disabled:opacity-60 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={() => fileInput?.click()}
			disabled={uploading}
		>
			<Icon name="photo" class="h-4 w-4" />
			{uploading ? 'Uploading…' : 'Add image'}
		</button>
	{/if}
</div>
