/// <reference types="@sveltejs/kit" />
/// <reference lib="webworker" />

import { build, files, version } from '$service-worker';

declare const self: ServiceWorkerGlobalScope;

const CACHE = `app-${version}`;
// Pre-built JS, CSS, fonts + static files. We only intercept these - anything
// else (HTML pages, /api/*, the SSE stream at /api/events, etc.) goes through
// the browser's native fetch with no service-worker involvement. Intercepting
// streaming responses (SSE) used to break the worker because cache.put would
// try to consume an unbounded body.
const ASSETS = new Set<string>([...build, ...files]);

self.addEventListener('install', (event) => {
	event.waitUntil(caches.open(CACHE).then((cache) => cache.addAll([...ASSETS])));
	self.skipWaiting();
});

self.addEventListener('activate', (event) => {
	event.waitUntil(
		caches
			.keys()
			.then((keys) =>
				Promise.all(keys.filter((key) => key !== CACHE).map((key) => caches.delete(key)))
			)
	);
});

self.addEventListener('fetch', (event) => {
	if (event.request.method !== 'GET') return;

	const url = new URL(event.request.url);
	if (url.protocol !== 'http:' && url.protocol !== 'https:') return;

	// Bypass anything we didn't ship as a build asset. Critically, this
	// excludes /api/events (text/event-stream) - letting the SW touch a
	// streaming body causes the request to be cancelled by the browser.
	if (url.origin !== self.location.origin || !ASSETS.has(url.pathname)) return;

	event.respondWith(
		caches.open(CACHE).then(async (cache) => {
			const cached = await cache.match(url.pathname);
			if (cached) return cached;
			try {
				const response = await fetch(event.request);
				if (response.ok) cache.put(event.request, response.clone());
				return response;
			} catch (err) {
				if (cached) return cached;
				throw err;
			}
		})
	);
});
