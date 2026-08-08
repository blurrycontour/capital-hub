/**
 * Money display preferences and formatting.
 *
 * These live in a `.svelte.ts` module so the settings can be `$state`. User
 * preferences load asynchronously after the layout mounts, and the settings
 * page can change them at any time; plain module variables would leave any
 * component that already rendered showing the previous setting, so the same
 * amount could appear with different precision on different pages.
 */

export type NumberFormat = 'international' | 'indian' | 'european';

// Number of decimal places used when formatting money (0–2, default 0).
let amountDecimals = $state(0);

// Digit grouping and decimal separator style (default 'international').
let numberFormat = $state<NumberFormat>('international');

const NUMBER_FORMAT_LOCALES: Record<NumberFormat, string> = {
	international: 'en-US', // 1,234,567.89
	indian: 'en-IN', // 12,34,567.89
	european: 'de-DE' // 1.234.567,89
};

export function setAmountDecimals(n: number): void {
	amountDecimals = Math.min(2, Math.max(0, Math.trunc(n || 0)));
}

export function getAmountDecimals(): number {
	return amountDecimals;
}

export function setNumberFormat(f: string): void {
	numberFormat = f === 'indian' || f === 'european' ? f : 'international';
}

export function getNumberFormat(): NumberFormat {
	return numberFormat;
}

// Format a currency total for display.
export function formatCurrency(amount: number, currency: string): string {
	const locale = NUMBER_FORMAT_LOCALES[numberFormat];
	try {
		return new Intl.NumberFormat(locale, {
			style: 'currency',
			currency,
			minimumFractionDigits: amountDecimals,
			maximumFractionDigits: amountDecimals
		}).format(amount);
	} catch {
		return `${amount.toLocaleString(locale, {
			minimumFractionDigits: amountDecimals,
			maximumFractionDigits: amountDecimals
		})} ${currency}`;
	}
}
