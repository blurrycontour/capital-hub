<script lang="ts" module>
	import type { Attachment } from '$lib/api';

	/**
	 * An attachment as shown in the UI. `pending` marks one that has been
	 * chosen but not uploaded yet — its `path` is a local object URL, so it
	 * previews correctly but must never be treated as a server path.
	 */
	export type AttachmentView = Attachment & { pending?: boolean };
</script>

<script lang="ts">
	// Attachments as thumbnails rather than bare links. Images open in the
	// built-in preview; PDFs get their own marker; everything else opens
	// directly.
	import Icon from '$lib/Icon.svelte';
	import Lightbox from '$lib/Lightbox.svelte';
	import FileTypeIcon from '$lib/FileTypeIcon.svelte';
	import { attachmentKind } from '$lib/attachments';

	let {
		attachments = [],
		ondelete,
		deleting = false,
		size = 'md'
	}: {
		attachments?: AttachmentView[];
		/** Omit to render read-only (no remove buttons). */
		ondelete?: (attachment: AttachmentView) => void;
		deleting?: boolean;
		/** `sm` is used inside the entries table, where space is tighter. */
		size?: 'sm' | 'md';
	} = $props();

	// Kind is read from the original filename, which a pending file has even
	// though its object-URL path does not.
	const kindOf = (a: AttachmentView) => attachmentKind(a.name || a.path);
	const images = $derived(attachments.filter((a) => kindOf(a) === 'image'));

	let lightboxOpen = $state(false);
	let lightboxIndex = $state(0);

	function openPreview(att: AttachmentView) {
		const i = images.findIndex((a) => a.path === att.path);
		lightboxIndex = i >= 0 ? i : 0;
		lightboxOpen = true;
	}

	const tile = $derived(size === 'sm' ? 'h-16 w-16' : 'h-20 w-20');
</script>

<ul class="flex flex-wrap gap-2">
	{#each attachments as att (att.path)}
		{@const kind = kindOf(att)}
		<li class="relative">
			{#if kind === 'image'}
				<button
					type="button"
					onclick={() => openPreview(att)}
					title={`Preview ${att.name}`}
					class={`block overflow-hidden rounded-md border border-slate-200 hover:border-sky-400 dark:border-slate-700 dark:hover:border-sky-600 ${tile}`}
				>
					<img src={att.path} alt={att.name} class="h-full w-full object-cover" />
				</button>
			{:else}
				<svelte:element
					this={att.pending ? 'div' : 'a'}
					href={att.pending ? undefined : att.path}
					target={att.pending ? undefined : '_blank'}
					rel={att.pending ? undefined : 'noopener noreferrer'}
					title={att.pending ? `${att.name} (not uploaded yet)` : `Open ${att.name}`}
					class={`flex flex-col items-center justify-center gap-1 rounded-md border border-slate-200 p-1 text-center dark:border-slate-700 ${
						att.pending
							? ''
							: 'hover:border-sky-400 hover:bg-slate-50 dark:hover:border-sky-600 dark:hover:bg-slate-800'
					} ${tile}`}
				>
					<FileTypeIcon {kind} class="h-6 w-6 shrink-0" />
					<span class="w-full truncate text-[0.65rem] leading-tight text-slate-500">{att.name}</span>
				</svelte:element>
			{/if}

			{#if att.pending}
				<!-- Marks a file that is only staged; saving is what uploads it. -->
				<span
					class="absolute bottom-0 left-0 right-0 rounded-b-md bg-sky-600/90 py-0.5 text-center text-[0.6rem] font-medium leading-none text-white"
				>
					New
				</span>
			{/if}

			{#if ondelete}
				<button
					type="button"
					class="absolute -right-1.5 -top-1.5 rounded-full border border-slate-200 bg-white p-0.5 text-slate-400 shadow-sm hover:border-rose-300 hover:text-rose-600 disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900 dark:hover:border-rose-800 dark:hover:text-rose-400"
					aria-label={`Remove ${att.name}`}
					title={`Remove ${att.name}`}
					onclick={() => ondelete?.(att)}
					disabled={deleting}
				>
					<Icon name="close" class="h-3.5 w-3.5" />
				</button>
			{/if}

			{#if kind === 'image'}
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
