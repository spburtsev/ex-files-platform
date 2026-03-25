import { json } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const BACKEND = env.BACKEND_URL ?? 'http://localhost:8080';

export const POST: RequestHandler = async ({ request, cookies, params }) => {
	const token = cookies.get('session');
	const res = await fetch(`${BACKEND}/workspaces/${params.id}/members`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			Authorization: `Bearer ${token}`
		},
		body: await request.text()
	});

	const data = await res.json().catch(() => ({}));
	return json(data, { status: res.status });
};
