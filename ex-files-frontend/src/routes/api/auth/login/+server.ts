import { json } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const BACKEND = env.BACKEND_URL ?? 'http://localhost:8080';

export const POST: RequestHandler = async ({ request, cookies }) => {
	const body = await request.json();

	const res = await fetch(`${BACKEND}/auth/login`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(body)
	});

	if (!res.ok) {
		const err = await res.json().catch(() => ({ error: 'Login failed' }));
		return json(err, { status: res.status });
	}

	const data = await res.json();
	cookies.set('session', data.token, {
		httpOnly: true,
		sameSite: 'lax',
		path: '/',
		maxAge: 8 * 60 * 60
	});

	return json({ ok: true });
};
