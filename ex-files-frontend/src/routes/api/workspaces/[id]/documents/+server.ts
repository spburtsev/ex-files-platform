import { json } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const BACKEND = env.BACKEND_URL ?? 'http://localhost:8080';

// Proxy multipart upload to the backend, forwarding the session token.
// The Content-Type header (including the multipart boundary) must be forwarded as-is.
export const POST: RequestHandler = async ({ request, cookies, params }) => {
	const token = cookies.get('session');
	const contentType = request.headers.get('content-type') ?? '';

	const res = await fetch(`${BACKEND}/workspaces/${params.id}/documents`, {
		method: 'POST',
		headers: {
			Authorization: `Bearer ${token}`,
			'Content-Type': contentType
		},
		body: await request.arrayBuffer()
	});

	const data = await res.json().catch(() => ({}));
	return json(data, { status: res.status });
};
