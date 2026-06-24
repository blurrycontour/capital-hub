<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { loadLeaflet } from '$lib/leaflet';

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

	let hint = $state('');
	let hintTimer: ReturnType<typeof setTimeout> | undefined;

	function showHint(message: string) {
		hint = message;
		clearTimeout(hintTimer);
		hintTimer = setTimeout(() => (hint = ''), 1400);
	}

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

	// Desktop: only zoom when Ctrl/Cmd is held, otherwise let the page scroll.
	function onWheel(e: WheelEvent) {
		if (e.ctrlKey || e.metaKey) {
			e.preventDefault();
			if (!map || !L) return;
			const latlng = map.mouseEventToLatLng(e);
			map.setZoomAround(latlng, map.getZoom() + (e.deltaY < 0 ? 1 : -1));
		} else {
			showHint('Use Ctrl + scroll to zoom');
		}
	}

	// Mobile: require two fingers to pan so a single finger scrolls the page.
	function onTouchStart(e: TouchEvent) {
		if (!map) return;
		if (e.touches.length >= 2) {
			map.dragging.enable();
		} else {
			map.dragging.disable();
		}
	}

	async function init() {
		L = await loadLeaflet();

		const valid = validMarkers();
		const center: [number, number] = valid.length ? [valid[0].lat, valid[0].lng] : [20, 0];
		map = L.map(mapEl, { scrollWheelZoom: false }).setView(center, valid.length ? zoom : 2);
		map.dragging.disable();
		L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
			maxZoom: 19,
			attribution: '&copy; OpenStreetMap contributors'
		}).addTo(map);
		layerGroup = L.layerGroup().addTo(map);

		mapEl.addEventListener('wheel', onWheel, { passive: false });
		mapEl.addEventListener('touchstart', onTouchStart);

		render();
		setTimeout(() => map?.invalidateSize(), 50);
	}

	// Re-render markers whenever the input changes after the map is ready.
	$effect(() => {
		void markers;
		render();
	});

	onMount(init);
	onDestroy(() => {
		clearTimeout(hintTimer);
		mapEl?.removeEventListener('wheel', onWheel);
		mapEl?.removeEventListener('touchstart', onTouchStart);
		map?.remove();
		map = null;
	});
</script>

<!-- `isolate` creates a stacking context so Leaflet's internal high z-index
     panes/controls never paint above app modals. -->
<div class={`relative isolate ${height}`}>
	<div
		bind:this={mapEl}
		class="h-full w-full rounded-md border border-slate-200 dark:border-slate-800"
	></div>
	{#if hint}
		<div class="pointer-events-none absolute inset-0 z-[400] flex items-center justify-center">
			<span class="rounded-md bg-slate-900/70 px-3 py-1.5 text-sm font-medium text-white">
				{hint}
			</span>
		</div>
	{/if}
</div>
