<script lang="ts">
	// Full-screen image preview. Opened from attachment thumbnails so an image
	// can be viewed without navigating away from the item and losing scroll
	// position, which is what following the raw file link used to do.
	import Icon from '$lib/Icon.svelte';

	let {
		images = [],
		index = $bindable(0),
		open = $bindable(false)
	}: {
		// Each entry is a displayable image: `src` is the stored path, `name`
		// the original filename shown in the caption.
		images?: { src: string; name: string }[];
		index?: number;
		open?: boolean;
	} = $props();

	const current = $derived(images[index]);
	const hasMany = $derived(images.length > 1);

	function close() {
		open = false;
	}

	function prev() {
		if (hasMany) index = (index - 1 + images.length) % images.length;
	}

	function next() {
		if (hasMany) index = (index + 1) % images.length;
	}

	function onkeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') close();
		else if (e.key === 'ArrowLeft') prev();
		else if (e.key === 'ArrowRight') next();
	}

	// Keep the page behind the overlay from scrolling while it is open.
	$effect(() => {
		if (!open) return;
		const previous = document.body.style.overflow;
		document.body.style.overflow = 'hidden';
		return () => {
			document.body.style.overflow = previous;
		};
	});
</script>

<svelte:window on:keydown={open ? onkeydown : undefined} />

{#if open && current}
	<!-- Above the map's fullscreen layer (z-1000) so it wins from anywhere. -->
	<div class="ch-overlay-inset fixed inset-0 z-[1100] flex flex-col bg-slate-950/90 backdrop-blur-sm" role="presentation">
		<button
			type="button"
			class="absolute inset-0 cursor-default"
			aria-label="Close preview"
			onclick={close}
		></button>

		<div class="pointer-events-none relative flex items-center justify-between gap-3 text-white">
			<p class="pointer-events-auto min-w-0 flex-1 truncate text-sm" title={current.name}>
				{current.name}
			</p>
			<div class="pointer-events-auto flex shrink-0 items-center gap-1">
				<a
					href={current.src}
					download={current.name}
					title="Download"
					aria-label="Download"
					class="rounded-full p-2 text-white/80 hover:bg-white/10 hover:text-white"
				>
					<Icon name="download" class="h-5 w-5" />
				</a>
				<button
					type="button"
					class="rounded-full p-2 text-white/80 hover:bg-white/10 hover:text-white"
					aria-label="Close preview"
					onclick={close}
				>
					<Icon name="close" class="h-5 w-5" />
				</button>
			</div>
		</div>

		<div class="pointer-events-none relative flex min-h-0 flex-1 items-center justify-center">
			<img
				src={current.src}
				alt={current.name}
				class="pointer-events-auto max-h-full max-w-full object-contain"
			/>

			{#if hasMany}
				<button
					type="button"
					class="pointer-events-auto absolute left-0 top-1/2 -translate-y-1/2 rounded-full bg-slate-900/70 p-2 text-white hover:bg-slate-900"
					aria-label="Previous image"
					onclick={prev}
				>
					<Icon name="chevron-left" class="h-6 w-6" />
				</button>
				<button
					type="button"
					class="pointer-events-auto absolute right-0 top-1/2 -translate-y-1/2 rounded-full bg-slate-900/70 p-2 text-white hover:bg-slate-900"
					aria-label="Next image"
					onclick={next}
				>
					<Icon name="chevron-right" class="h-6 w-6" />
				</button>
			{/if}
		</div>

		{#if hasMany}
			<p class="pointer-events-none relative pt-2 text-center text-xs text-white/70">
				{index + 1} / {images.length}
			</p>
		{/if}
	</div>
{/if}
