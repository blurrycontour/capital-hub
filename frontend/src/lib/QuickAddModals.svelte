<script lang="ts">
	// The three quick-add dialogs, rendered once at the layout root so they are
	// never nested inside the transformed sidebar (see quickAdd.svelte.ts).
	import { goto } from '$app/navigation';
	import CreateCollectionModal from '$lib/CreateCollectionModal.svelte';
	import CreateItemModal from '$lib/CreateItemModal.svelte';
	import EntryFormModal from '$lib/EntryFormModal.svelte';
	import { quickAdd } from '$lib/quickAdd.svelte';

	let entryItemId = $state<number | null>(null);

	// The dialogs own their `open` flag so their Cancel/Escape/backdrop paths
	// keep working; these two effects keep that in step with the shared state.
	let collectionOpen = $state(false);
	let itemOpen = $state(false);
	let entryOpen = $state(false);

	$effect(() => {
		collectionOpen = quickAdd.active === 'collection';
		itemOpen = quickAdd.active === 'item';
		entryOpen = quickAdd.active === 'entry';
	});

	// Only clears once every dialog is closed, so this cannot fight the effect
	// above while one is being opened.
	$effect(() => {
		if (!collectionOpen && !itemOpen && !entryOpen && quickAdd.active) quickAdd.close();
	});
</script>

<CreateCollectionModal
	bind:open={collectionOpen}
	oncreated={(created) => goto(`/collections/${created.id}`)}
/>

<CreateItemModal
	bind:open={itemOpen}
	collections={quickAdd.collections}
	loadingCollections={quickAdd.loadingCollections}
	oncreated={(created) => goto(`/collections/${created.collectionId}/items/${created.id}`)}
/>

<EntryFormModal
	bind:open={entryOpen}
	bind:selectedItemId={entryItemId}
	items={quickAdd.items}
	loadingItems={quickAdd.loadingItems}
	currency={quickAdd.currencyOf(entryItemId)}
	onsaved={(saved) => {
		const href = quickAdd.itemHref(saved.itemId);
		if (href) return goto(`${href}#entry-${saved.id}`);
	}}
/>
