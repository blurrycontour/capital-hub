import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit(),
		VitePWA({
			// The service worker file is written into build/ alongside the SPA
			// and served by the Go backend as a static file.
			registerType: 'prompt', // show update prompt instead of silently replacing
			injectRegister: 'script',
			includeAssets: ['new-logo.svg'],
			manifest: {
				name: 'Capital Hub',
				short_name: 'Capital Hub',
				description: 'Track and manage your collections and inventory.',
				theme_color: '#0f172a',
				background_color: '#0f172a',
				display: 'standalone',
				scope: '/',
				start_url: '/',
				icons: [
					{
						src: 'pwa-192.png',
						sizes: '192x192',
						type: 'image/png'
					},
					{
						src: 'pwa-512.png',
						sizes: '512x512',
						type: 'image/png',
						purpose: 'any maskable'
					}
				]
			},
			workbox: {
				// Cache the SPA shell and all hashed static assets.
				globPatterns: ['**/*.{js,css,html,svg,png,webp,ico}'],
				// Serve the SPA index.html for any navigation not matched by
				// the precache (mirrors the Go server's fallback behaviour).
				navigateFallback: 'index.html',
				navigateFallbackDenylist: [/^\/api\//]
			},
			devOptions: {
				enabled: false // keep dev server fast; SW only active in production build
			}
		})
	],
	server: {
		port: 5173,
		// Proxy API calls to the Go backend during development.
		proxy: {
			'/api': {
				target: 'http://localhost:8080',
				changeOrigin: true
			},
			'/healthz': {
				target: 'http://localhost:8080',
				changeOrigin: true
			}
		}
	}
});
