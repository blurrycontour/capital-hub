/**
 * Shared reactive auth state. Acts as the single source of truth for the
 * signed-in user so that login/logout propagate to every component (notably
 * the layout chrome) without each one fetching independently.
 */
import { fetchMe, logout as apiLogout, type ApiUser } from './api';

let user = $state<ApiUser | null>(null);
let loading = $state(true);
let loaded = $state(false);

export const auth = {
	get user() {
		return user;
	},
	get loading() {
		return loading;
	},
	get loaded() {
		return loaded;
	},
	get isAuthenticated() {
		return user !== null;
	},
	set(next: ApiUser | null) {
		user = next;
		loaded = true;
		loading = false;
	},
	async refresh() {
		loading = true;
		try {
			user = await fetchMe();
		} catch {
			user = null;
		} finally {
			loading = false;
			loaded = true;
		}
	},
	async logout() {
		await apiLogout();
		user = null;
	}
};
