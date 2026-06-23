<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';

	type MapMarker = { lat: number; lng: number; label?: string; href?: string };

	let {
		markers = [],
		height = 'h-72',
		zoom = 13
	}: { markers?: MapMarker[]; height?: string; zoom?: number } = $props();

	let mapEl: HTMLDivElement;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let map: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let L: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let layerGroup: any = null;

	function validMarkers(): MapMarker[] {
		return markers.filter((m) => m.lat != null && m.lng != null);
	}

	function render() {
		if (!map || !L || !layerGroup) return;
		layerGroup.clearLayers();
		const valid = validMarkers();
		const latlngs: [number, number][] = [];
		for (const m of valid) {
			const mk = L.marker([m.lat, m.lng]);
			if (m.label) mk.bindTooltip(m.label);
			if (m.href) {
				const href = m.href;
				mk.on('click', () => goto(href));
			}
			mk.addTo(layerGroup);
			latlngs.push([m.lat, m.lng]);
		}
		if (latlngs.length > 1) {
			map.fitBounds(latlngs, { padding: [30, 30], maxZoom: 16 });
		} else if (latlngs.length === 1) {
			map.setView(latlngs[0], zoom);
		}
	}

	async function init() {
		const leaflet = await import('leaflet');
		await import('leaflet/dist/leaflet.css');
		L = leaflet.default ?? leaflet;

		const valid = validMarkers();
		const center: [number, number] = valid.length ? [valid[0].lat, valid[0].lng] : [20, 0];
		map = L.map(mapEl).setView(center, valid.length ? zoom : 2);
		L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
			maxZoom: 19,
			attribution: '&copy; OpenStreetMap contributors'
		}).addTo(map);
		layerGroup = L.layerGroup().addTo(map);
		render();
		setTimeout(() => map?.invalidateSize(), 50);
	}

	// Re-render markers whenever the input changes after the map is ready.
	$effect(() => {
		// Touch markers so this effect re-runs on change.
		void markers;
		render();
	});

	onMount(init);
	onDestroy(() => {
		map?.remove();
		map = null;
	});
</script>

<div
	bind:this={mapEl}
	class={`w-full rounded-md border border-slate-200 dark:border-slate-800 ${height}`}
></div>
