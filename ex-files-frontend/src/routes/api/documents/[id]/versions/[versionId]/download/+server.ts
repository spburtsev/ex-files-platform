import { json } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const BACKEND = env.BACKEND_URL ?? 'http://localhost:8080';

// Returns { url: string } — a presigned MinIO URL the client can open directly.
export const GET: RequestHandler = async ({ cookies, params }) => {
	const token = cookies.get('session');
	const res = await fetch(
		`${BACKEND}/documents/${params.id}/versions/${params.versionId}/download`,
		{ headers: { Authorization: `Bearer ${token}` } }
	);

	const data = await res.json().catch(() => ({}));
	return json(data, { status: res.status });
};
