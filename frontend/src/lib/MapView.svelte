<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { loadLeaflet } from '$lib/leaflet';
	import Icon from '$lib/Icon.svelte';

	type MapMarker = { lat: number; lng: number; label?: string; href?: string };

	let {
		markers = [],
		height = 'h-72',
		zoom = 13
	}: { markers?: MapMarker[]; height?: string; zoom?: number } = $props();

	// Fullscreen overlay toggle.
	let expanded = $state(false);

	let mapEl: HTMLDivElement;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let map: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let L: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let layerGroup: any = null;

	let hint = $state('');
	let hintTimer: ReturnType<typeof setTimeout> | undefined;
	let renderFrame: number | undefined;

	function showHint(message: string) {
		hint = message;
		clearTimeout(hintTimer);
		hintTimer = setTimeout(() => (hint = ''), 1400);
	}

	function validMarkers(): MapMarker[] {
		return markers.filter((m) => m.lat != null && m.lng != null);
	}

	// Defer Leaflet rendering to the next animation frame, collapsing rapid
	// reactive updates into a single render and keeping it out of the flush.
	function scheduleRender() {
		if (renderFrame !== undefined) cancelAnimationFrame(renderFrame);
		renderFrame = requestAnimationFrame(() => {
			renderFrame = undefined;
			render();
		});
	}

	function render() {
		if (!map || !L || !layerGroup) return;
		try {
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
			// `animate: false` avoids Leaflet's requestAnimationFrame-driven pan/zoom
			// transitions. Those can conflict with another Leaflet map being torn down
			// in the same tick (e.g. a LocationPicker inside a modal that just closed)
			// and lock up the main thread.
			if (latlngs.length > 1) {
				map.fitBounds(latlngs, { padding: [30, 30], maxZoom: 16, animate: false });
			} else if (latlngs.length === 1) {
				map.setView(latlngs[0], zoom, { animate: false });
			}
		} catch {
			/* ignore Leaflet render errors so the reactive flush is never aborted */
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

	// Toggle the fullscreen overlay. Leaflet needs invalidateSize after the
	// container resizes, then we re-fit the markers into the new viewport.
	function toggleExpand() {
		expanded = !expanded;
		setTimeout(() => {
			map?.invalidateSize();
			render();
		}, 80);
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && expanded) toggleExpand();
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
	// The actual Leaflet work is deferred to the next animation frame so it never
	// runs synchronously inside Svelte's reactive flush. That keeps the rest of
	// the flush (e.g. a header name update) from being blocked, and lets any
	// other Leaflet map (such as a LocationPicker in a modal that just closed)
	// finish tearing down first — avoiding a rAF conflict that can freeze the UI.
	$effect(() => {
		// Track each marker's coordinates/label so edits (move/clear) re-render.
		for (const m of markers) {
			void m.lat;
			void m.lng;
			void m.label;
		}
		void markers.length;
		scheduleRender();
	});

	onMount(init);
	onDestroy(() => {
		clearTimeout(hintTimer);
		if (renderFrame !== undefined) cancelAnimationFrame(renderFrame);
		mapEl?.removeEventListener('wheel', onWheel);
		mapEl?.removeEventListener('touchstart', onTouchStart);
		try {
			map?.remove();
		} catch {
			/* ignore Leaflet teardown errors */
		}
		map = null;
	});
</script>

<!-- `isolate` creates a stacking context so Leaflet's internal high z-index
     panes/controls never paint above app modals. -->
<svelte:window on:keydown={expanded ? onKeydown : undefined} />

<div
	class={expanded
		? 'fixed inset-0 z-[1000] flex flex-col bg-slate-900/60 p-4 backdrop-blur-sm'
		: `relative isolate ${height}`}
>
	<div class="relative h-full w-full flex-1">
		<div
			bind:this={mapEl}
			class="h-full w-full rounded-md border border-slate-200 dark:border-slate-800"
		></div>

		<button
			type="button"
			onclick={toggleExpand}
			title={expanded ? 'Minimize map' : 'Expand map'}
			aria-label={expanded ? 'Minimize map' : 'Expand map'}
			class="absolute right-2 top-2 z-[500] rounded-md border border-slate-300 bg-white/90 p-1.5 text-slate-600 shadow-sm hover:bg-white hover:text-slate-900 dark:border-slate-600 dark:bg-slate-800/90 dark:text-slate-300 dark:hover:bg-slate-800"
		>
			<Icon name={expanded ? 'minimize' : 'expand'} class="h-4 w-4" />
		</button>

		{#if hint}
			<div class="pointer-events-none absolute inset-0 z-[400] flex items-center justify-center">
				<span class="rounded-md bg-slate-900/70 px-3 py-1.5 text-sm font-medium text-white">
					{hint}
				</span>
			</div>
		{/if}
	</div>
</div>
