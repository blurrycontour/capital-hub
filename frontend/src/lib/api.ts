export type ApiUser = {
	id: number;
	username: string;
	email: string;
	displayName: string;
	isAdmin: boolean;
	isActive: boolean;
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

export async function fetchProviders(): Promise<{ oidcEnabled: boolean }> {
	const res = await fetch('/api/v1/auth/providers');
	return parseJSON<{ oidcEnabled: boolean }>(res);
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
