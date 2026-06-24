export type ApiUser = {
	id: number;
	username: string;
	email: string;
	displayName: string;
	avatarPath: string;
	isAdmin: boolean;
	isActive: boolean;
	role: string;
};

export type NotificationItem = {
	id: number;
	type: string;
	title: string;
	body: string;
	link: string;
	createdAt: string;
	readAt?: string;
};

export type SMTPSettings = {
	host: string;
	port: number;
	username: string;
	from: string;
	useTls: boolean;
	passwordSet: boolean;
};

async function parseJSON<T>(res: Response): Promise<T> {
	if (!res.ok) {
		let message = `request failed (${res.status})`;
		try {
			const body = (await res.json()) as { error?: string };
			if (body.error) message = body.error;
		} catch {
			// keep fallback message
		}
		throw new Error(message);
	}
	return (await res.json()) as T;
}

export async function fetchMe(): Promise<ApiUser | null> {
	const res = await fetch('/api/v1/auth/me');
	if (res.status === 401) return null;
	const body = await parseJSON<{ user: ApiUser }>(res);
	return body.user;
}

export async function fetchProviders(): Promise<{ oidcEnabled: boolean; oidcProviderName: string }> {
	const res = await fetch('/api/v1/auth/providers');
	return parseJSON<{ oidcEnabled: boolean; oidcProviderName: string }>(res);
}

export async function updateProfile(payload: {
	displayName: string;
	email: string;
}): Promise<ApiUser> {
	const csrf = await fetchCSRFToken();
	const res = await fetch('/api/v1/auth/me', {
		method: 'PATCH',
		headers: {
			'Content-Type': 'application/json',
			'X-CSRF-Token': csrf
		},
		body: JSON.stringify(payload)
	});
	const body = await parseJSON<{ user: ApiUser }>(res);
	return body.user;
}

export async function uploadAvatar(file: File): Promise<ApiUser> {
	const csrf = await fetchCSRFToken();
	const form = new FormData();
	form.append('file', file);
	const res = await fetch('/api/v1/auth/me/avatar', {
		method: 'POST',
		headers: { 'X-CSRF-Token': csrf },
		body: form
	});
	const body = await parseJSON<{ user: ApiUser }>(res);
	return body.user;
}

export type UserPreferences = {
	includeSharedInStats: boolean;
};

export async function getPreferences(): Promise<UserPreferences> {
	const res = await fetch('/api/v1/auth/me/preferences');
	return parseJSON<UserPreferences>(res);
}

export async function updatePreferences(prefs: UserPreferences): Promise<UserPreferences> {
	return mutate<UserPreferences>('/api/v1/auth/me/preferences', 'PATCH', prefs);
}

export async function login(identifier: string, password: string): Promise<ApiUser> {
	const res = await fetch('/api/v1/auth/login', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ identifier, password })
	});
	const body = await parseJSON<{ user: ApiUser; csrfToken?: string }>(res);
	return body.user;
}

export async function fetchCSRFToken(): Promise<string> {
	const res = await fetch('/api/v1/auth/csrf');
	const body = await parseJSON<{ token: string }>(res);
	return body.token;
}

export async function logout(): Promise<void> {
	const csrf = await fetchCSRFToken();
	const res = await fetch('/api/v1/auth/logout', {
		method: 'POST',
		headers: { 'X-CSRF-Token': csrf }
	});
	await parseJSON<{ ok: boolean }>(res);
}

export async function listNotifications(limit = 50): Promise<NotificationItem[]> {
	const res = await fetch(`/api/v1/notifications?limit=${limit}`);
	const body = await parseJSON<{ notifications: NotificationItem[] }>(res);
	return body.notifications;
}

export async function markNotificationRead(id: number): Promise<void> {
	const csrf = await fetchCSRFToken();
	const res = await fetch(`/api/v1/notifications/${id}/read`, {
		method: 'POST',
		headers: { 'X-CSRF-Token': csrf }
	});
	await parseJSON<{ ok: boolean }>(res);
}

export async function listUsers(): Promise<ApiUser[]> {
	const res = await fetch('/api/v1/admin/users');
	const body = await parseJSON<{ users: ApiUser[] }>(res);
	return body.users;
}

export async function getSMTPSettings(): Promise<SMTPSettings> {
	const res = await fetch('/api/v1/admin/settings/smtp');
	return parseJSON<SMTPSettings>(res);
}

export async function updateSMTPSettings(payload: {
	host: string;
	port: number;
	username: string;
	password: string;
	from: string;
	useTls: boolean;
}): Promise<SMTPSettings> {
	const csrf = await fetchCSRFToken();
	const res = await fetch('/api/v1/admin/settings/smtp', {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json',
			'X-CSRF-Token': csrf
		},
		body: JSON.stringify(payload)
	});
	return parseJSON<SMTPSettings>(res);
}

export async function testSMTP(to: string): Promise<void> {
	const csrf = await fetchCSRFToken();
	const res = await fetch('/api/v1/admin/settings/smtp/test', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			'X-CSRF-Token': csrf
		},
		body: JSON.stringify({ to })
	});
	await parseJSON<{ ok: boolean }>(res);
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
	const csrf = await fetchCSRFToken();
	const res = await fetch('/api/v1/auth/me/password', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			'X-CSRF-Token': csrf
		},
		body: JSON.stringify({ currentPassword, newPassword })
	});
	await parseJSON<{ ok: boolean }>(res);
}

export async function adminCreateUser(payload: {
	username: string;
	email: string;
	displayName: string;
	password: string;
	role: string;
}): Promise<ApiUser> {
	const csrf = await fetchCSRFToken();
	const res = await fetch('/api/v1/admin/users', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			'X-CSRF-Token': csrf
		},
		body: JSON.stringify(payload)
	});
	const body = await parseJSON<{ user: ApiUser }>(res);
	return body.user;
}

export async function adminUpdateUser(id: number, role: string, isActive: boolean): Promise<ApiUser> {
	const csrf = await fetchCSRFToken();
	const res = await fetch(`/api/v1/admin/users/${id}`, {
		method: 'PATCH',
		headers: {
			'Content-Type': 'application/json',
			'X-CSRF-Token': csrf
		},
		body: JSON.stringify({ role, isActive })
	});
	const body = await parseJSON<{ user: ApiUser }>(res);
	return body.user;
}

export async function adminDeleteUser(id: number): Promise<void> {
	const csrf = await fetchCSRFToken();
	const res = await fetch(`/api/v1/admin/users/${id}`, {
		method: 'DELETE',
		headers: { 'X-CSRF-Token': csrf }
	});
	await parseJSON<{ ok: boolean }>(res);
}

export type OIDCSettings = {
	enabled: boolean;
	issuerUrl: string;
	clientId: string;
	clientSecretSet: boolean;
	redirectUrl: string;
	adminGroup: string;
	providerName: string;
	allowRegistration: boolean;
	envFields: string[];
};

export async function getOIDCSettings(): Promise<OIDCSettings> {
	const res = await fetch('/api/v1/admin/settings/oidc');
	return parseJSON<OIDCSettings>(res);
}

export async function updateOIDCSettings(payload: {
	enabled: boolean;
	issuerUrl: string;
	clientId: string;
	clientSecret: string;
	redirectUrl: string;
	adminGroup: string;
	providerName: string;
	allowRegistration: boolean;
}): Promise<OIDCSettings> {
	const csrf = await fetchCSRFToken();
	const res = await fetch('/api/v1/admin/settings/oidc', {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json',
			'X-CSRF-Token': csrf
		},
		body: JSON.stringify(payload)
	});
	return parseJSON<OIDCSettings>(res);
}

// ---------- Inventory: collections, items, entries ----------

export type CustomField = {
	label: string;
	value: string;
};

export type Attachment = {
	name: string;
	path: string;
};

export type Collection = {
	id: number;
	name: string;
	description: string;
	currency: string;
	locationLat: number | null;
	locationLng: number | null;
	locationLabel: string;
	customFields: CustomField[];
	createdAt: string;
	updatedAt: string;
	createdBy: string;
	updatedBy: string;
	itemCount: number;
	ownerName: string;
	shared: boolean;
	accessLevel: 'owner' | 'write' | 'read';
	shareCount: number;
};

export type CollectionShare = {
	userId: number;
	username: string;
	email: string;
	displayName: string;
	access: 'read' | 'write';
};

export type Item = {
	id: number;
	collectionId: number;
	name: string;
	description: string;
	imagePath: string;
	images: string[];
	locationLat: number | null;
	locationLng: number | null;
	locationLabel: string;
	attachments: Attachment[];
	customFields: CustomField[];
	createdAt: string;
	updatedAt: string;
	createdBy: string;
	updatedBy: string;
	entryCount: number;
};

export type Entry = {
	id: number;
	itemId: number;
	name: string;
	amount: number;
	currency: string;
	note: string;
	occurredOn: string;
	attachments: Attachment[];
	createdAt: string;
	updatedAt: string;
	createdBy: string;
	updatedBy: string;
};

export type CurrencyTotal = {
	currency: string;
	total: number;
	entries: number;
};

export type Stats = {
	itemCount: number;
	entryCount: number;
	totals: CurrencyTotal[];
};

export type PortfolioSummary = {
	collectionCount: number;
	itemCount: number;
	entryCount: number;
	totals: CurrencyTotal[];
};

export type SearchResult = {
	type: 'collection' | 'item';
	id: number;
	name: string;
	description: string;
	collectionId: number;
	collectionName: string;
};

export type ItemInput = {
	name: string;
	description: string;
	images: string[];
	locationLat: number | null;
	locationLng: number | null;
	locationLabel: string;
	attachments: Attachment[];
	customFields: CustomField[];
};

export type EntryInput = {
	name: string;
	amount: number;
	note: string;
	occurredOn: string;
	attachments: Attachment[];
};

export type CollectionInput = {
	name: string;
	description: string;
	currency: string;
	locationLat: number | null;
	locationLng: number | null;
	locationLabel: string;
	customFields: CustomField[];
};

async function mutate<T>(url: string, method: string, payload?: unknown): Promise<T> {
	const csrf = await fetchCSRFToken();
	const headers: Record<string, string> = { 'X-CSRF-Token': csrf };
	let body: string | undefined;
	if (payload !== undefined) {
		headers['Content-Type'] = 'application/json';
		body = JSON.stringify(payload);
	}
	const res = await fetch(url, { method, headers, body });
	return parseJSON<T>(res);
}

// Collections

export async function listCollections(): Promise<Collection[]> {
	const res = await fetch('/api/v1/collections');
	const body = await parseJSON<{ collections: Collection[] }>(res);
	return body.collections;
}

export async function getCollection(id: number): Promise<Collection> {
	const res = await fetch(`/api/v1/collections/${id}`);
	const body = await parseJSON<{ collection: Collection }>(res);
	return body.collection;
}

export async function createCollection(payload: CollectionInput): Promise<Collection> {
	const body = await mutate<{ collection: Collection }>('/api/v1/collections', 'POST', payload);
	return body.collection;
}

export async function updateCollection(
	id: number,
	payload: CollectionInput
): Promise<Collection> {
	const body = await mutate<{ collection: Collection }>(
		`/api/v1/collections/${id}`,
		'PATCH',
		payload
	);
	return body.collection;
}

export async function deleteCollection(id: number): Promise<void> {
	await mutate<{ ok: boolean }>(`/api/v1/collections/${id}`, 'DELETE');
}

export async function listCollectionShares(id: number): Promise<CollectionShare[]> {
	const res = await fetch(`/api/v1/collections/${id}/shares`);
	const body = await parseJSON<{ shares: CollectionShare[] }>(res);
	return body.shares;
}

export async function shareCollection(
	id: number,
	identifier: string,
	access: 'read' | 'write'
): Promise<CollectionShare> {
	const body = await mutate<{ share: CollectionShare }>(
		`/api/v1/collections/${id}/shares`,
		'POST',
		{ identifier, access }
	);
	return body.share;
}

export async function unshareCollection(id: number, userId: number): Promise<void> {
	await mutate<{ ok: boolean }>(`/api/v1/collections/${id}/shares/${userId}`, 'DELETE');
}

export async function getCollectionStats(id: number): Promise<Stats> {
	const res = await fetch(`/api/v1/collections/${id}/stats`);
	const body = await parseJSON<{ stats: Stats }>(res);
	return body.stats;
}

// Items

export async function listItems(collectionId: number): Promise<Item[]> {
	const res = await fetch(`/api/v1/collections/${collectionId}/items`);
	const body = await parseJSON<{ items: Item[] }>(res);
	return body.items;
}

export async function getItem(id: number): Promise<Item> {
	const res = await fetch(`/api/v1/items/${id}`);
	const body = await parseJSON<{ item: Item }>(res);
	return body.item;
}

export async function createItem(collectionId: number, payload: ItemInput): Promise<Item> {
	const body = await mutate<{ item: Item }>(
		`/api/v1/collections/${collectionId}/items`,
		'POST',
		payload
	);
	return body.item;
}

export async function updateItem(id: number, payload: ItemInput): Promise<Item> {
	const body = await mutate<{ item: Item }>(`/api/v1/items/${id}`, 'PATCH', payload);
	return body.item;
}

export async function deleteItem(id: number): Promise<void> {
	await mutate<{ ok: boolean }>(`/api/v1/items/${id}`, 'DELETE');
}

export async function getItemStats(id: number): Promise<Stats> {
	const res = await fetch(`/api/v1/items/${id}/stats`);
	const body = await parseJSON<{ stats: Stats }>(res);
	return body.stats;
}

export async function uploadItemImage(id: number, file: File): Promise<Item> {
	const csrf = await fetchCSRFToken();
	const form = new FormData();
	form.append('file', file);
	const res = await fetch(`/api/v1/items/${id}/image`, {
		method: 'POST',
		headers: { 'X-CSRF-Token': csrf },
		body: form
	});
	const body = await parseJSON<{ item: Item }>(res);
	return body.item;
}

export async function deleteItemImage(id: number, path: string): Promise<Item> {
	const body = await mutate<{ item: Item }>(`/api/v1/items/${id}/image`, 'DELETE', { path });
	return body.item;
}

export async function uploadItemAttachment(id: number, file: File): Promise<Item> {
	const csrf = await fetchCSRFToken();
	const form = new FormData();
	form.append('file', file);
	const res = await fetch(`/api/v1/items/${id}/attachments`, {
		method: 'POST',
		headers: { 'X-CSRF-Token': csrf },
		body: form
	});
	const body = await parseJSON<{ item: Item }>(res);
	return body.item;
}

// Entries

export async function listEntries(itemId: number): Promise<Entry[]> {
	const res = await fetch(`/api/v1/items/${itemId}/entries`);
	const body = await parseJSON<{ entries: Entry[] }>(res);
	return body.entries;
}

export async function createEntry(itemId: number, payload: EntryInput): Promise<Entry> {
	const body = await mutate<{ entry: Entry }>(
		`/api/v1/items/${itemId}/entries`,
		'POST',
		payload
	);
	return body.entry;
}

export async function updateEntry(id: number, payload: EntryInput): Promise<Entry> {
	const body = await mutate<{ entry: Entry }>(`/api/v1/entries/${id}`, 'PATCH', payload);
	return body.entry;
}

export async function uploadEntryAttachment(id: number, file: File): Promise<Entry> {
	const csrf = await fetchCSRFToken();
	const form = new FormData();
	form.append('file', file);
	const res = await fetch(`/api/v1/entries/${id}/attachments`, {
		method: 'POST',
		headers: { 'X-CSRF-Token': csrf },
		body: form
	});
	const body = await parseJSON<{ entry: Entry }>(res);
	return body.entry;
}

export async function deleteEntry(id: number): Promise<void> {
	await mutate<{ ok: boolean }>(`/api/v1/entries/${id}`, 'DELETE');
}

// Search & portfolio

export async function search(query: string): Promise<SearchResult[]> {
	const res = await fetch(`/api/v1/search?q=${encodeURIComponent(query)}`);
	const body = await parseJSON<{ results: SearchResult[] }>(res);
	return body.results;
}

export async function getPortfolioStats(): Promise<PortfolioSummary> {
	const res = await fetch('/api/v1/stats/portfolio');
	const body = await parseJSON<{ stats: PortfolioSummary }>(res);
	return body.stats;
}

// Common currency codes offered in the UI.
export const CURRENCIES = [
	'USD',
	'EUR',
	'GBP',
	'INR',
	'SEK',
	'NOK',
	'DKK',
	'JPY',
	'CNY',
	'CHF',
	'CAD',
	'AUD',
	'SGD',
	'AED'
];

// Format a currency total for display.
export function formatCurrency(amount: number, currency: string): string {
	try {
		return new Intl.NumberFormat(undefined, {
			style: 'currency',
			currency
		}).format(amount);
	} catch {
		return `${amount.toLocaleString()} ${currency}`;
	}
}
