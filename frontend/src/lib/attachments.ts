/**
 * Attachment and image file types.
 *
 * These lists mirror allowedAttachmentExt / allowedImageExt in
 * handlers_inventory.go. Checking here too means an unsupported file is
 * reported immediately and by name, rather than only after a round trip that
 * fails with a message the user may not be scrolled to.
 */

export const IMAGE_EXTENSIONS = ['.jpg', '.jpeg', '.png', '.gif', '.webp'] as const;

export const ATTACHMENT_EXTENSIONS = [
	...IMAGE_EXTENSIONS,
	'.pdf',
	'.txt',
	'.csv',
	'.doc',
	'.docx',
	'.xls',
	'.xlsx',
	'.zip'
] as const;

/** `accept` values for the file pickers, so the OS dialog filters too. */
export const ATTACHMENT_ACCEPT = ATTACHMENT_EXTENSIONS.join(',');
export const IMAGE_ACCEPT = IMAGE_EXTENSIONS.join(',');

/** Lowercase extension including the dot, or '' when there is none. */
export function extensionOf(nameOrPath: string): string {
	const clean = nameOrPath.split('?')[0].split('#')[0];
	const base = clean.slice(clean.lastIndexOf('/') + 1);
	const dot = base.lastIndexOf('.');
	return dot <= 0 ? '' : base.slice(dot).toLowerCase();
}

export type AttachmentKind = 'image' | 'pdf' | 'other';

/**
 * Classify by original filename where possible. Stored paths keep the
 * extension, so either works, but the name is what a pending upload has.
 */
export function attachmentKind(nameOrPath: string): AttachmentKind {
	const ext = extensionOf(nameOrPath);
	if ((IMAGE_EXTENSIONS as readonly string[]).includes(ext)) return 'image';
	if (ext === '.pdf') return 'pdf';
	return 'other';
}

export function isImageFile(nameOrPath: string): boolean {
	return attachmentKind(nameOrPath) === 'image';
}

/** Matches maxUploadBytes in handlers_inventory.go. */
export const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;

function tooLarge(file: File): string {
	if (file.size <= MAX_UPLOAD_BYTES) return '';
	const mb = (file.size / (1024 * 1024)).toFixed(1);
	return `“${file.name}” is ${mb} MB. The limit is ${MAX_UPLOAD_BYTES / (1024 * 1024)} MB.`;
}

function humanList(exts: readonly string[]): string {
	return exts.map((e) => e.slice(1).toUpperCase()).join(', ');
}

/** Returns an error message for an unsupported file, or '' when it is fine. */
export function attachmentError(file: File): string {
	const ext = extensionOf(file.name);
	if (!ext) {
		return `“${file.name}” has no file extension. Supported types: ${humanList(ATTACHMENT_EXTENSIONS)}.`;
	}
	if (!(ATTACHMENT_EXTENSIONS as readonly string[]).includes(ext)) {
		return `${ext.slice(1).toUpperCase()} files are not supported. Choose one of: ${humanList(ATTACHMENT_EXTENSIONS)}.`;
	}
	return tooLarge(file);
}

/** Same, for the item image gallery, which accepts images only. */
export function imageError(file: File): string {
	const ext = extensionOf(file.name);
	if (!ext || !(IMAGE_EXTENSIONS as readonly string[]).includes(ext)) {
		return `Images must be one of: ${humanList(IMAGE_EXTENSIONS)}.`;
	}
	return tooLarge(file);
}
