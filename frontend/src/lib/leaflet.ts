/**
 * Loads Leaflet (and its CSS) and fixes the default marker icon paths.
 *
 * Leaflet's default icon references `marker-icon.png`, `marker-icon-2x.png` and
 * `marker-shadow.png` using relative URLs that break when the library is run
 * through a bundler such as Vite. Importing the images explicitly lets the
 * bundler emit them with correct hashed URLs, which we then register on
 * `L.Icon.Default`.
 */

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let cached: any = null;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function loadLeaflet(): Promise<any> {
	if (cached) return cached;

	const leaflet = await import('leaflet');
	await import('leaflet/dist/leaflet.css');

	const markerIcon2x = (await import('leaflet/dist/images/marker-icon-2x.png')).default;
	const markerIcon = (await import('leaflet/dist/images/marker-icon.png')).default;
	const markerShadow = (await import('leaflet/dist/images/marker-shadow.png')).default;

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	const L = (leaflet as any).default ?? leaflet;

	L.Icon.Default.mergeOptions({
		iconRetinaUrl: markerIcon2x,
		iconUrl: markerIcon,
		shadowUrl: markerShadow
	});

	cached = L;
	return L;
}
