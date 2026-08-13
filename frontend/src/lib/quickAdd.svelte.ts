/**
 * Shared state for quick add.
 *
 * The trigger and the dialogs it opens live in different parts of the layout:
 * the desktop trigger sits inside the sidebar, which is `transform`ed and so
 * becomes the containing block for any `position: fixed` descendant — a dialog
 * rendered there would be trapped inside the sidebar's width. The dialogs are
 * therefore rendered once at the layout root and driven from here.
 */
import { listCollections, listAllItems, type Collection, type ItemWithCollection } from '$lib/api';

export type QuickAddKind = 'collection' | 'item' | 'entry';

let active = $state<QuickAddKind | null>(null);
let collections = $state<Collection[]>([]);
let items = $state<ItemWithCollection[]>([]);
let loadingCollections = $state(false);
let loadingItems = $state(false);

async function ensureCollections() {
	if (collections.length > 0 || loadingCollections) return;
	loadingCollections = true;
	try {
		collections = await listCollections();
	} catch {
		collections = [];
	} finally {
		loadingCollections = false;
	}
}

async function ensureItems() {
	if (items.length > 0 || loadingItems) return;
	loadingItems = true;
	try {
		items = await listAllItems();
	} catch {
		items = [];
	} finally {
		loadingItems = false;
	}
}

export const quickAdd = {
	get active() {
		return active;
	},
	get collections() {
		return collections;
	},
	get items() {
		return items;
	},
	get loadingCollections() {
		return loadingCollections;
	},
	get loadingItems() {
		return loadingItems;
	},

	/** Open one of the three forms, fetching only what that form needs. */
	open(kind: QuickAddKind) {
		active = kind;
		if (kind === 'item') void ensureCollections();
		if (kind === 'entry') {
			void ensureCollections();
			void ensureItems();
		}
	},

	close() {
		active = null;
	},

	/** Currency of the collection a given item belongs to. */
	currencyOf(itemId: number | null): string {
		const item = items.find((i) => i.id === itemId);
		if (!item) return 'EUR';
		return collections.find((c) => c.id === item.collectionId)?.currency ?? 'EUR';
	},

	/** Where to land after creating an entry against `itemId`. */
	itemHref(itemId: number): string | null {
		const item = items.find((i) => i.id === itemId);
		return item ? `/collections/${item.collectionId}/items/${item.id}` : null;
	}
};
