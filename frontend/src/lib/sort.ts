/**
 * Sorting for the collections and items lists.
 *
 * The lists are fetched whole (no pagination), so sorting happens client-side
 * and the choice is remembered per device in localStorage — the same place the
 * card/list view preference lives.
 */

export type SortOption =
	| 'updated-desc'
	| 'updated-asc'
	| 'created-desc'
	| 'created-asc'
	| 'name-asc'
	| 'name-desc';

/** Most recently modified first — what both lists open on. */
export const DEFAULT_SORT: SortOption = 'updated-desc';

export const SORT_OPTIONS: { value: SortOption; label: string }[] = [
	{ value: 'updated-desc', label: 'Last modified' },
	{ value: 'updated-asc', label: 'Least recently modified' },
	{ value: 'created-desc', label: 'Newest first' },
	{ value: 'created-asc', label: 'Oldest first' },
	{ value: 'name-asc', label: 'Name (A–Z)' },
	{ value: 'name-desc', label: 'Name (Z–A)' }
];

function isSortOption(v: string): v is SortOption {
	return SORT_OPTIONS.some((o) => o.value === v);
}

/** Read a stored preference, falling back to the default when absent or stale. */
export function loadSort(key: string): SortOption {
	try {
		const raw = localStorage.getItem(key);
		if (raw && isSortOption(raw)) return raw;
	} catch {
		// Private mode or blocked storage: fall back to the default.
	}
	return DEFAULT_SORT;
}

/** Persist any of the sort vocabularies; the value is only ever a string. */
export function saveSort(key: string, option: SortOption | SearchSortOption): void {
	try {
		localStorage.setItem(key, option);
	} catch {
		// Non-fatal: the choice just will not persist.
	}
}

/** The fields every sortable record shares. */
type Sortable = { name: string; createdAt: string; updatedAt: string };

/**
 * Return a sorted copy. Timestamps are ISO-ish strings from SQLite, which sort
 * correctly as plain strings, but parsing keeps it correct if the format ever
 * changes. Names use a locale-aware, case-insensitive comparison so "apple"
 * and "Apple" sort together.
 */
export function sortRecords<T extends Sortable>(records: T[], option: SortOption): T[] {
	const collator = new Intl.Collator(undefined, { sensitivity: 'base', numeric: true });
	const time = (v: string) => {
		const t = Date.parse(v);
		return Number.isNaN(t) ? 0 : t;
	};

	const out = [...records];
	out.sort((a, b) => {
		switch (option) {
			case 'name-asc':
				return collator.compare(a.name, b.name);
			case 'name-desc':
				return collator.compare(b.name, a.name);
			case 'created-asc':
				return time(a.createdAt) - time(b.createdAt);
			case 'created-desc':
				return time(b.createdAt) - time(a.createdAt);
			case 'updated-asc':
				return time(a.updatedAt) - time(b.updatedAt);
			case 'updated-desc':
			default:
				return time(b.updatedAt) - time(a.updatedAt);
		}
	});
	return out;
}


// ---------- Search ----------

/**
 * Search results carry no timestamps, so they get their own, smaller set.
 * "Relevance" is the order the backend returns (FTS ranking), which is what
 * the page showed before this control existed.
 */
export type SearchSortOption = 'relevance' | 'name-asc' | 'name-desc';

export const DEFAULT_SEARCH_SORT: SearchSortOption = 'relevance';

export const SEARCH_SORT_OPTIONS: { value: SearchSortOption; label: string }[] = [
	{ value: 'relevance', label: 'Best match' },
	{ value: 'name-asc', label: 'Name (A–Z)' },
	{ value: 'name-desc', label: 'Name (Z–A)' }
];

export function loadSearchSort(key: string): SearchSortOption {
	try {
		const raw = localStorage.getItem(key);
		if (raw && SEARCH_SORT_OPTIONS.some((o) => o.value === raw)) {
			return raw as SearchSortOption;
		}
	} catch {
		// Private mode or blocked storage: fall back to the default.
	}
	return DEFAULT_SEARCH_SORT;
}

/** Return a sorted copy, leaving the backend's ranking alone for 'relevance'. */
export function sortSearchResults<T extends { name: string }>(
	results: T[],
	option: SearchSortOption
): T[] {
	if (option === 'relevance') return results;
	const collator = new Intl.Collator(undefined, { sensitivity: 'base', numeric: true });
	const out = [...results];
	out.sort((a, b) =>
		option === 'name-asc' ? collator.compare(a.name, b.name) : collator.compare(b.name, a.name)
	);
	return out;
}
