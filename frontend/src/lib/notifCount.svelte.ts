// Singleton reactive unread-notification count shared between the sidebar and
// the notification page so both reflect the same value without an extra fetch.
import { getUnreadNotificationCount } from './api';

let count = $state(0);

export const notifCount = {
	get value() {
		return count;
	},
	async refresh() {
		try {
			count = await getUnreadNotificationCount();
		} catch {
			// non-fatal – badge simply stays stale
		}
	},
	/** Optimistically adjust the count without a round-trip. */
	adjust(delta: number) {
		count = Math.max(0, count + delta);
	},
	set(value: number) {
		count = Math.max(0, value);
	},
	reset() {
		count = 0;
	}
};
