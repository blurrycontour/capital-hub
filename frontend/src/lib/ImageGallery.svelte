<script lang="ts">
	import Icon from '$lib/Icon.svelte';

	let {
		images = [],
		coverPath = '',
		onadd,
		ondelete,
		onsetcover,
		uploading = false
	}: {
		images?: string[];
		coverPath?: string;
		onadd?: (file: File) => void;
		ondelete?: (path: string) => void;
		onsetcover?: (path: string) => void;
		uploading?: boolean;
	} = $props();

	// Path of the image currently in view. Tracking by path (not index) keeps
	// the view on the same image when the list is reordered — e.g. after
	// choosing a new cover, which moves that image to the front.
	let activePath = $state('');
	let fileInput = $state<HTMLInputElement | null>(null);

	// Index is derived from the active path so a reorder never changes which
	// image is shown. Falls back to the first image when the path is unknown.
	const index = $derived.by(() => {
		const i = images.indexOf(activePath);
		return i >= 0 ? i : 0;
	});

	// Keep the active path pointing at a real image (initial load, additions,
	// removals) without disturbing it on a mere reorder.
	$effect(() => {
		if (!images.includes(activePath)) {
			activePath = images[0] ?? '';
		}
	});

	const current = $derived(images[index] ?? '');
	const isCover = $derived(current !== '' && current === coverPath);

	function prev() {
		if (images.length === 0) return;
		activePath = images[(index - 1 + images.length) % images.length];
	}

	function next() {
		if (images.length === 0) return;
		activePath = images[(index + 1) % images.length];
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
			{#if onsetcover}
				<button
					type="button"
					class={`absolute left-2 top-2 rounded-full p-1.5 text-white disabled:cursor-default ${
						isCover ? 'bg-sky-700' : 'bg-slate-900/60 hover:bg-sky-700'
					}`}
					aria-label={isCover ? 'Current display picture' : 'Set as display picture'}
					title={isCover ? 'Current display picture' : 'Set as display picture'}
					onclick={() => onsetcover?.(current)}
					disabled={isCover}
				>
					<Icon name="star" class="h-4 w-4" filled={isCover} />
				</button>
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
					onclick={() => (activePath = img)}
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
