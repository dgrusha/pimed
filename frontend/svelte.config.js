import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		// Pure static SPA, embedded into the Go binary via go:embed.
		adapter: adapter({
			pages: '../backend/web/dist',
			assets: '../backend/web/dist',
			fallback: 'index.html',
			precompress: false,
			strict: true
		}),
		// Content-Security-Policy. SvelteKit hashes its inline bootstrap <script> and
		// emits the policy as a <meta http-equiv> tag in index.html. The Go server
		// sends the same policy as a header (plus frame-ancestors, which a meta tag
		// cannot carry), computing the inline-script hash from the embedded
		// index.html at startup — see backend/internal/platform/spa. Keep both lists
		// in sync. style-src needs 'unsafe-inline' for Svelte's inline styles.
		csp: {
			mode: 'hash',
			directives: {
				'default-src': ['self'],
				'script-src': ['self'],
				'style-src': ['self', 'unsafe-inline'],
				'img-src': ['self', 'data:', 'blob:'],
				'media-src': ['self', 'blob:'],
				'connect-src': ['self'],
				'font-src': ['self'],
				'object-src': ['none'],
				'base-uri': ['none'],
				'form-action': ['self']
			}
		}
	}
};

export default config;
