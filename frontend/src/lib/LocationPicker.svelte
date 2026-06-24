<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import Icon from '$lib/Icon.svelte';
	import { loadLeaflet } from '$lib/leaflet';

	let {
		lat = $bindable(null),
		lng = $bindable(null),
		label = $bindable('')
	}: {
		lat?: number | null;
		lng?: number | null;
		label?: string;
	} = $props();

	let mapEl: HTMLDivElement;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let map: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let marker: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let L: any = null;
	let locating = $state(false);
	let geoError = $state('');

	const defaultCenter: [number, number] = [20, 0];

	function setMarker(la: number, ln: number) {
		lat = la;
		lng = ln;
		if (!map || !L) return;
		if (marker) {
			marker.setLatLng([la, ln]);
		} else {
			marker = L.marker([la, ln]).addTo(map);
		}
	}

	async function initMap() {
		L = await loadLeaflet();

		const start: [number, number] =
			lat != null && lng != null ? [lat, lng] : defaultCenter;
		const zoom = lat != null && lng != null ? 13 : 2;

		map = L.map(mapEl).setView(start, zoom);
		L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
			maxZoom: 19,
			attribution: '&copy; OpenStreetMap contributors'
		}).addTo(map);

		if (lat != null && lng != null) {
			marker = L.marker([lat, lng]).addTo(map);
		}

		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		map.on('click', (e: any) => {
			setMarker(e.latlng.lat, e.latlng.lng);
		});

		// Leaflet sizing fix when rendered inside a modal.
		setTimeout(() => map?.invalidateSize(), 50);
	}

	function useCurrentLocation() {
		geoError = '';
		if (!navigator.geolocation) {
			geoError = 'Geolocation is not supported by this browser.';
			return;
		}
		locating = true;
		navigator.geolocation.getCurrentPosition(
			(pos) => {
				locating = false;
				const la = pos.coords.latitude;
				const ln = pos.coords.longitude;
				setMarker(la, ln);
				map?.setView([la, ln], 14);
			},
			(err) => {
				locating = false;
				geoError = err.message || 'Unable to retrieve your location.';
			},
			{ enableHighAccuracy: true, timeout: 10000 }
		);
	}

	function clearLocation() {
		lat = null;
		lng = null;
		label = '';
		if (marker) {
			map?.removeLayer(marker);
			marker = null;
		}
	}

	function onLatInput(e: Event) {
		const v = parseFloat((e.target as HTMLInputElement).value);
		if (!Number.isNaN(v) && lng != null) setMarker(v, lng);
		else lat = Number.isNaN(v) ? null : v;
	}

	function onLngInput(e: Event) {
		const v = parseFloat((e.target as HTMLInputElement).value);
		if (!Number.isNaN(v) && lat != null) setMarker(lat, v);
		else lng = Number.isNaN(v) ? null : v;
	}

	onMount(() => {
		initMap();
	});

	onDestroy(() => {
		try {
			map?.remove();
		} catch {
			/* ignore Leaflet teardown errors */
		}
		map = null;
	});
</script>

<div class="space-y-3">
	<div class="flex flex-wrap items-center gap-2">
		<button
			type="button"
			class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm hover:bg-slate-100 disabled:opacity-60 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={useCurrentLocation}
			disabled={locating}
		>
			<Icon name="crosshair" class="h-4 w-4" />
			{locating ? 'Locating…' : 'Use my location'}
		</button>
		<button
			type="button"
			class="inline-flex items-center gap-1.5 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
			onclick={clearLocation}
		>
			Clear
		</button>
		<span class="text-xs text-slate-500">Click the map to drop a pin.</span>
	</div>

	{#if geoError}
		<p class="text-xs text-amber-600 dark:text-amber-400">{geoError}</p>
	{/if}

	<div
		bind:this={mapEl}
		class="isolate h-64 w-full rounded-md border border-slate-200 dark:border-slate-800"
	></div>

	<div class="grid grid-cols-2 gap-2">
		<label class="block text-xs">
			<span class="text-slate-500">Latitude</span>
			<input
				type="number"
				step="any"
				value={lat ?? ''}
				oninput={onLatInput}
				class="mt-1 w-full rounded-md border border-slate-300 px-2 py-1 text-sm dark:border-slate-700 dark:bg-slate-800"
			/>
		</label>
		<label class="block text-xs">
			<span class="text-slate-500">Longitude</span>
			<input
				type="number"
				step="any"
				value={lng ?? ''}
				oninput={onLngInput}
				class="mt-1 w-full rounded-md border border-slate-300 px-2 py-1 text-sm dark:border-slate-700 dark:bg-slate-800"
			/>
		</label>
	</div>

	<label class="block text-xs">
		<span class="text-slate-500">Label (optional)</span>
		<input
			type="text"
			bind:value={label}
			placeholder="e.g. Home safe, Bank vault"
			class="mt-1 w-full rounded-md border border-slate-300 px-2 py-1 text-sm dark:border-slate-700 dark:bg-slate-800"
		/>
	</label>
</div>
