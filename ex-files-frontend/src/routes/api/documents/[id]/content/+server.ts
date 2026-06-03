import { env } from '$env/dynamic/private';
import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

const BACKEND_URL = env.BACKEND_URL ?? 'http://localhost:8080';

export const GET: RequestHandler = async ({ params, cookies, request, fetch }) => {
	const session = cookies.get('session');
	if (!session) error(401, 'unauthorized');

	const upstream = await fetch(`${BACKEND_URL}/documents/${params.id}/file`, {
		headers: {
			Authorization: `Bearer ${session}`
		},
		signal: request.signal
	});

	if (!upstream.ok || !upstream.body) {
		error(upstream.status || 502, 'download failed');
	}

	const headers = new Headers();
	for (const h of ['content-type', 'content-length', 'content-disposition']) {
		const v = upstream.headers.get(h);
		if (v) headers.set(h, v);
	}

	return new Response(upstream.body, { status: 200, headers });
};
