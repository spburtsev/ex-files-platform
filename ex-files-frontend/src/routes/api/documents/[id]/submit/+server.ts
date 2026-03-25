import { json } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const BACKEND = env.BACKEND_URL ?? 'http://localhost:8080';

export const POST: RequestHandler = async ({ cookies, params }) => {
	const token = cookies.get('session');
	const res = await fetch(`${BACKEND}/documents/${params.id}/submit`, {
		method: 'POST',
		headers: { Authorization: `Bearer ${token}` }
	});
	const data = await res.json().catch(() => ({}));
	return json(data, { status: res.status });
};
