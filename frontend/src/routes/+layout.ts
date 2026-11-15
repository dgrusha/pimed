// Pure client-side SPA: no SSR, no prerendering. adapter-static emits only the
// `index.html` fallback, which the Go backend serves for every non-/api path.
export const ssr = false;
export const prerender = false;
