import { json } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const BACKEND = env.BACKEND_URL ?? 'http://localhost:8080';

export const POST: RequestHandler = async ({ cookies }) => {
	const token = cookies.get('session');
	if (token) {
		await fetch(`${BACKEND}/auth/logout`, {
			method: 'POST',
			headers: { Authorization: `Bearer ${token}` }
		}).catch(() => {});
	}

	cookies.delete('session', { path: '/' });
	return json({ ok: true });
};
