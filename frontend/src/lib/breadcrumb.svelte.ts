/**
 * Breadcrumb label overrides. Pages can register a human-friendly label for a
 * given path segment (e.g. show a collection's name instead of its numeric id)
 * and the layout's breadcrumb trail will use it.
 */
let overrides = $state<Record<string, string>>({});

export const breadcrumbs = {
	get overrides() {
		return overrides;
	},
	set(path: string, label: string) {
		overrides = { ...overrides, [path]: label };
	},
	clear(path: string) {
		const next = { ...overrides };
		delete next[path];
		overrides = next;
	}
};
