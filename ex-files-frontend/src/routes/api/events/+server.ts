import { env } from '$env/dynamic/private';
import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

const BACKEND_URL = env.BACKEND_URL ?? 'http://localhost:8080';

export const GET: RequestHandler = async ({ url, cookies, request, fetch }) => {
	const session = cookies.get('session');
	if (!session) error(401, 'unauthorized');

	const upstreamUrl = new URL(`${BACKEND_URL}/events`);
	const documentId = url.searchParams.get('documentId');
	if (documentId) upstreamUrl.searchParams.set('documentId', documentId);

	const upstream = await fetch(upstreamUrl, {
		headers: {
			Authorization: `Bearer ${session}`,
			Accept: 'text/event-stream'
		},
		signal: request.signal
	});

	if (!upstream.ok || !upstream.body) {
		error(upstream.status || 502, 'upstream sse failed');
	}

	return new Response(upstream.body, {
		status: 200,
		headers: {
			'Content-Type': 'text/event-stream',
			'Cache-Control': 'no-cache, no-transform',
			Connection: 'keep-alive',
			'X-Accel-Buffering': 'no'
		}
	});
};
