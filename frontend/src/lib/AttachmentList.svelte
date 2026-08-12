<script lang="ts" module>
	// Extensions the backend accepts and a browser can render inline. Kept in
	// sync with allowedAttachmentExt in handlers_inventory.go; anything else
	// (pdf, doc, zip, …) is offered as a download instead of a preview.
	const IMAGE_EXT = ['.jpg', '.jpeg', '.png', '.gif', '.webp'];

	export function isImageAttachment(path: string): boolean {
		const clean = path.split('?')[0].toLowerCase();
		return IMAGE_EXT.some((ext) => clean.endsWith(ext));
	}
</script>

<script lang="ts">
	// Attachments as thumbnails rather than bare links. Images open in the
	// built-in preview; everything else still opens the file directly.
	import Icon from '$lib/Icon.svelte';
	import Lightbox from '$lib/Lightbox.svelte';
	import type { Attachment } from '$lib/api';

	let {
		attachments = [],
		ondelete,
		deleting = false,
		size = 'md'
	}: {
		attachments?: Attachment[];
		/** Omit to render read-only (no remove buttons). */
		ondelete?: (path: string) => void;
		deleting?: boolean;
		/** `sm` is used inside the entries table, where space is tighter. */
		size?: 'sm' | 'md';
	} = $props();

	const images = $derived(attachments.filter((a) => isImageAttachment(a.path)));

	let lightboxOpen = $state(false);
	let lightboxIndex = $state(0);

	function openPreview(path: string) {
		const i = images.findIndex((a) => a.path === path);
		lightboxIndex = i >= 0 ? i : 0;
		lightboxOpen = true;
	}

	const tile = $derived(size === 'sm' ? 'h-16 w-16' : 'h-20 w-20');
</script>

<ul class="flex flex-wrap gap-2">
	{#each attachments as att (att.path)}
		{@const isImage = isImageAttachment(att.path)}
		<li class="group relative">
			{#if isImage}
				<button
					type="button"
					onclick={() => openPreview(att.path)}
					title={`Preview ${att.name}`}
					class={`block overflow-hidden rounded-md border border-slate-200 hover:border-sky-400 dark:border-slate-700 dark:hover:border-sky-600 ${tile}`}
				>
					<img src={att.path} alt={att.name} class="h-full w-full object-cover" />
				</button>
			{:else}
				<a
					href={att.path}
					target="_blank"
					rel="noopener noreferrer"
					title={`Open ${att.name}`}
					class={`flex flex-col items-center justify-center gap-1 rounded-md border border-slate-200 p-1 text-center hover:border-sky-400 hover:bg-slate-50 dark:border-slate-700 dark:hover:border-sky-600 dark:hover:bg-slate-800 ${tile}`}
				>
					<Icon name="document" class="h-6 w-6 shrink-0 text-slate-400" />
					<span class="w-full truncate text-[0.65rem] leading-tight text-slate-500">{att.name}</span>
				</a>
			{/if}

			{#if ondelete}
				<button
					type="button"
					class="absolute -right-1.5 -top-1.5 rounded-full border border-slate-200 bg-white p-0.5 text-slate-400 shadow-sm hover:border-rose-300 hover:text-rose-600 disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900 dark:hover:border-rose-800 dark:hover:text-rose-400"
					aria-label={`Delete ${att.name}`}
					title={`Delete ${att.name}`}
					onclick={() => ondelete?.(att.path)}
					disabled={deleting}
				>
					<Icon name="close" class="h-3.5 w-3.5" />
				</button>
			{/if}

			{#if isImage}
				<!-- Images have no visible filename; expose it to assistive tech. -->
				<span class="sr-only">{att.name}</span>
			{/if}
		</li>
	{/each}
</ul>

<Lightbox
	images={images.map((a) => ({ src: a.path, name: a.name }))}
	bind:index={lightboxIndex}
	bind:open={lightboxOpen}
/>
