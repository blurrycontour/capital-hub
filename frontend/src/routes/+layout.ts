// Run as a pure single-page app: no SSR, no prerendering. The Go backend
// serves the built static files with an index.html fallback.
export const ssr = false;
export const prerender = false;
export const trailingSlash = 'never';
