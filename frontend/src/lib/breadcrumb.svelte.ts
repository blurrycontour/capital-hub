/**
 * Breadcrumb label overrides. Pages can register a human-friendly label for a
 * given path segment (e.g. show a collection's name instead of its numeric id)
 * and the layout's breadcrumb trail will use it.
 *
 * Pages can alternatively register a full custom trail when the URL structure
 * doesn't match the desired breadcrumb hierarchy (e.g. an item page nested
 * under its collection).
 */
export type Crumb = { label: string; href: string };

let overrides = $state<Record<string, string>>({});
let trail = $state<Crumb[] | null>(null);

export const breadcrumbs = {
	get overrides() {
		return overrides;
	},
	get trail() {
		return trail;
	},
	set(path: string, label: string) {
		overrides = { ...overrides, [path]: label };
	},
	clear(path: string) {
		const next = { ...overrides };
		delete next[path];
		overrides = next;
	},
	setTrail(crumbs: Crumb[]) {
		trail = crumbs;
	},
	clearTrail() {
		trail = null;
	}
};
