import type { CreateClientConfig } from '$lib/api/client';
import { env } from '$env/dynamic/private';
import { getRequestEvent } from '$app/server';

export const createClientConfig: CreateClientConfig = (config) => ({
	...config,
	baseUrl: env.BACKEND_URL ?? 'http://localhost:8080'
});

/**
 * Per-call options for SDK functions. Provides the request-scoped fetch from
 * SvelteKit (so the dev-server proxy + cookies work) and forwards the session
 * cookie as a Bearer token so the backend can authenticate the call.
 */
export function apiOpts(): { fetch: typeof fetch; headers?: Record<string, string> } {
	const event = getRequestEvent();
	const session = event.cookies.get('session');
	return {
		fetch: event.fetch,
		headers: session ? { Authorization: `Bearer ${session}` } : undefined
	};
}
