<script lang="ts">
	// Marks an attachment's file type. PDFs get the conventional document-with-a
	// -PDF-label mark rather than the generic document outline, which is drawn
	// here instead of in Icon.svelte because it needs more than one path.
	import Icon from '$lib/Icon.svelte';
	import type { AttachmentKind } from '$lib/attachments';

	let {
		kind,
		class: cls = 'h-6 w-6'
	}: {
		kind: AttachmentKind;
		class?: string;
	} = $props();
</script>

{#if kind === 'pdf'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		viewBox="0 0 24 24"
		class={`${cls} text-rose-500 dark:text-rose-400`}
		aria-hidden="true"
	>
		<path
			d="M14 2.75H7A2.25 2.25 0 0 0 4.75 5v14A2.25 2.25 0 0 0 7 21.25h10A2.25 2.25 0 0 0 19.25 19V8L14 2.75Z"
			fill="none"
			stroke="currentColor"
			stroke-width="1.5"
			stroke-linejoin="round"
		/>
		<path
			d="M13.75 3v3.5a1 1 0 0 0 1 1h4"
			fill="none"
			stroke="currentColor"
			stroke-width="1.5"
			stroke-linejoin="round"
		/>
		<rect x="2.5" y="12.25" width="14" height="6.5" rx="1.5" fill="currentColor" />
		<text
			x="9.5"
			y="17"
			text-anchor="middle"
			font-size="4.9"
			font-weight="700"
			letter-spacing="0.2"
			fill="#fff"
			font-family="ui-sans-serif, system-ui, sans-serif">PDF</text
		>
	</svg>
{:else if kind === 'image'}
	<Icon name="photo" class={`${cls} text-slate-400`} />
{:else}
	<Icon name="document" class={`${cls} text-slate-400`} />
{/if}
