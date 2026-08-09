/**
 * Collection permission helpers.
 *
 * Access levels are ordered weakest to strongest: read < write < full < owner.
 * Always ask these helpers rather than comparing levels inline — a level added
 * later is otherwise silently excluded from every `=== 'write'` check.
 */
import type { CollectionAccess } from './api';

export type AccessLevel = CollectionAccess | 'owner';

/** Can view, but change nothing. */
export function canEditContents(level: AccessLevel | undefined): boolean {
	return level === 'write' || level === 'full' || level === 'owner';
}

/** Can edit the collection's own details (name, currency, custom fields, …). */
export function canEditCollection(level: AccessLevel | undefined): boolean {
	return level === 'full' || level === 'owner';
}

/** Can delete the collection and manage who it is shared with. Owner only. */
export function isOwner(level: AccessLevel | undefined): boolean {
	return level === 'owner';
}

/** Title-case label for a share level, e.g. in the "people with access" list. */
export const ACCESS_LABELS: Record<CollectionAccess, string> = {
	read: 'Read',
	write: 'Write',
	full: 'Full control'
};

/** Sentence fragment describing what a level allows, for tooltips. */
export const ACCESS_DESCRIPTIONS: Record<CollectionAccess, string> = {
	read: 'read only',
	write: 'can edit items and entries',
	full: 'full control'
};

/** Options for the share form, ordered weakest to strongest. */
export const SHARE_ACCESS_OPTIONS: { value: CollectionAccess; label: string; hint: string }[] = [
	{ value: 'read', label: 'Read', hint: 'View the collection and its contents' },
	{ value: 'write', label: 'Write', hint: 'Also add, edit and delete items and entries' },
	{ value: 'full', label: 'Full control', hint: "Also edit the collection's own details" }
];

export function accessLabel(level: AccessLevel | undefined): string {
	if (level === 'owner') return 'Owner';
	return level ? ACCESS_LABELS[level] : '';
}

export function accessDescription(level: AccessLevel | undefined): string {
	if (level === 'owner') return 'owner';
	return level ? ACCESS_DESCRIPTIONS[level] : '';
}
