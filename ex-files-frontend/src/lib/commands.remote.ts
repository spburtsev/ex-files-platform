import { command, getRequestEvent } from '$app/server';
import { env } from '$env/dynamic/private';
import { fromBinary } from '@bufbuild/protobuf';
import { LoginResponseSchema, RegisterResponseSchema } from '$lib/gen/auth/v1/auth_pb';
import { CreateWorkspaceResponseSchema } from '$lib/gen/workspaces/v1/workspaces_pb';
import { GetDownloadURLResponseSchema } from '$lib/gen/documents/v1/documents_pb';

const BACKEND = env.BACKEND_URL ?? 'http://localhost:8080';

const NETWORK_ERROR = 'Unable to reach the server. Please try again later.';

async function parseJsonError(res: Response, fallback: string) {
	const data = await res.json().catch(() => ({}));
	return (data as Record<string, string>).error ?? fallback;
}

/** Wrapper around event.fetch that returns null instead of throwing on network failure. */
async function safeFetch(url: string, init?: RequestInit) {
	try {
		const event = getRequestEvent();
		return await event.fetch(url, init);
	} catch {
		return null;
	}
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

export const login = command(
	'unchecked',
	async (credentials: { email: string; password: string }) => {
		const res = await safeFetch(`${BACKEND}/auth/login`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(credentials)
		});
		if (!res) return { ok: false as const, error: NETWORK_ERROR };
		if (!res.ok) {
			return { ok: false as const, error: await parseJsonError(res, 'Invalid email or password') };
		}
		const r = fromBinary(LoginResponseSchema, new Uint8Array(await res.arrayBuffer()));
		const event = getRequestEvent();
		event.cookies.set('session', r.token, {
			path: '/',
			httpOnly: true,
			maxAge: 8 * 60 * 60,
			sameSite: 'lax'
		});
		return { ok: true as const };
	}
);

export const register = command(
	'unchecked',
	async (details: { name: string; email: string; password: string }) => {
		const res = await safeFetch(`${BACKEND}/auth/register`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(details)
		});
		if (!res) return { ok: false as const, error: NETWORK_ERROR };
		if (!res.ok) {
			return {
				ok: false as const,
				error: await parseJsonError(res, 'Registration failed. Please try again.')
			};
		}
		const r = fromBinary(RegisterResponseSchema, new Uint8Array(await res.arrayBuffer()));
		const event = getRequestEvent();
		event.cookies.set('session', r.token, {
			path: '/',
			httpOnly: true,
			maxAge: 8 * 60 * 60,
			sameSite: 'lax'
		});
		return { ok: true as const };
	}
);

export const logout = command(async () => {
	const event = getRequestEvent();
	try {
		await event.fetch(`${BACKEND}/auth/logout`, { method: 'POST' });
	} catch {
		// Even if backend is down, clear the local session cookie
	}
	event.cookies.delete('session', { path: '/' });
});

// ---------------------------------------------------------------------------
// Workspaces
// ---------------------------------------------------------------------------

export const createWorkspace = command('unchecked', async (name: string) => {
	const res = await safeFetch(`${BACKEND}/workspaces`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ name })
	});
	if (!res) return { ok: false as const, error: NETWORK_ERROR };
	if (!res.ok) {
		return { ok: false as const, error: await parseJsonError(res, 'Failed to create workspace') };
	}
	const r = fromBinary(CreateWorkspaceResponseSchema, new Uint8Array(await res.arrayBuffer()));
	return { ok: true as const, workspace: { id: r.workspace!.id } };
});

export const updateWorkspace = command(
	'unchecked',
	async ({ id, name }: { id: string; name: string }) => {
		const res = await safeFetch(`${BACKEND}/workspaces/${id}`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ name })
		});
		if (!res) return { ok: false as const, error: NETWORK_ERROR };
		if (!res.ok) {
			return { ok: false as const, error: await parseJsonError(res, 'Failed to update workspace') };
		}
		return { ok: true as const };
	}
);

export const deleteWorkspace = command('unchecked', async (id: string) => {
	const res = await safeFetch(`${BACKEND}/workspaces/${id}`, { method: 'DELETE' });
	return { ok: res?.ok ?? false };
});

export const addWorkspaceMember = command(
	'unchecked',
	async ({ workspaceId, userId }: { workspaceId: string; userId: number }) => {
		const res = await safeFetch(`${BACKEND}/workspaces/${workspaceId}/members`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ user_id: userId })
		});
		if (!res) return { ok: false as const, error: NETWORK_ERROR };
		if (!res.ok) {
			return { ok: false as const, error: await parseJsonError(res, 'Failed to add member') };
		}
		return { ok: true as const };
	}
);

export const removeWorkspaceMember = command(
	'unchecked',
	async ({ workspaceId, userId }: { workspaceId: string; userId: bigint }) => {
		const res = await safeFetch(`${BACKEND}/workspaces/${workspaceId}/members/${userId}`, {
			method: 'DELETE'
		});
		return { ok: res?.ok ?? false };
	}
);

// ---------------------------------------------------------------------------
// Documents
// ---------------------------------------------------------------------------

export const uploadDocument = command(
	'unchecked',
	async ({ workspaceId, file }: { workspaceId: string; file: File }) => {
		const form = new FormData();
		form.append('file', file);
		const res = await safeFetch(`${BACKEND}/workspaces/${workspaceId}/documents`, {
			method: 'POST',
			body: form
		});
		if (!res) return { ok: false as const, error: NETWORK_ERROR };
		if (!res.ok) {
			return { ok: false as const, error: await parseJsonError(res, 'Upload failed') };
		}
		return { ok: true as const };
	}
);

export const deleteDocument = command('unchecked', async (id: string) => {
	const res = await safeFetch(`${BACKEND}/documents/${id}`, { method: 'DELETE' });
	return { ok: res?.ok ?? false };
});

export const uploadDocumentVersion = command(
	'unchecked',
	async ({ docId, file }: { docId: string; file: File }) => {
		const form = new FormData();
		form.append('file', file);
		const res = await safeFetch(`${BACKEND}/documents/${docId}/versions`, {
			method: 'POST',
			body: form
		});
		if (!res) return { ok: false as const, error: NETWORK_ERROR };
		if (!res.ok) {
			return { ok: false as const, error: await parseJsonError(res, 'Upload failed') };
		}
		return { ok: true as const };
	}
);

export const getDocumentDownloadUrl = command(
	'unchecked',
	async ({ docId, versionId }: { docId: string; versionId: number }) => {
		const res = await safeFetch(`${BACKEND}/documents/${docId}/versions/${versionId}/download`);
		if (!res || !res.ok) return { url: null };
		const r = fromBinary(GetDownloadURLResponseSchema, new Uint8Array(await res.arrayBuffer()));
		return { url: r.url };
	}
);

// ---------------------------------------------------------------------------
// Document workflow
// ---------------------------------------------------------------------------

export const submitDocument = command('unchecked', async (id: string) => {
	const res = await safeFetch(`${BACKEND}/documents/${id}/submit`, { method: 'POST' });
	if (!res) return { ok: false as const, error: NETWORK_ERROR };
	if (!res.ok) {
		return { ok: false as const, error: await parseJsonError(res, 'Action failed') };
	}
	return { ok: true as const };
});

export const resubmitDocument = command('unchecked', async (id: string) => {
	const res = await safeFetch(`${BACKEND}/documents/${id}/resubmit`, { method: 'POST' });
	if (!res) return { ok: false as const, error: NETWORK_ERROR };
	if (!res.ok) {
		return { ok: false as const, error: await parseJsonError(res, 'Action failed') };
	}
	return { ok: true as const };
});

export const approveDocument = command('unchecked', async (id: string) => {
	const res = await safeFetch(`${BACKEND}/documents/${id}/approve`, { method: 'POST' });
	if (!res) return { ok: false as const, error: NETWORK_ERROR };
	if (!res.ok) {
		return { ok: false as const, error: await parseJsonError(res, 'Action failed') };
	}
	return { ok: true as const };
});

export const rejectDocument = command(
	'unchecked',
	async ({ id, note }: { id: string; note: string }) => {
		const res = await safeFetch(`${BACKEND}/documents/${id}/reject`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ note })
		});
		if (!res) return { ok: false as const, error: NETWORK_ERROR };
		if (!res.ok) {
			return { ok: false as const, error: await parseJsonError(res, 'Action failed') };
		}
		return { ok: true as const };
	}
);

export const requestDocumentChanges = command(
	'unchecked',
	async ({ id, note }: { id: string; note: string }) => {
		const res = await safeFetch(`${BACKEND}/documents/${id}/request-changes`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ note })
		});
		if (!res) return { ok: false as const, error: NETWORK_ERROR };
		if (!res.ok) {
			return { ok: false as const, error: await parseJsonError(res, 'Action failed') };
		}
		return { ok: true as const };
	}
);

export const assignDocumentReviewer = command(
	'unchecked',
	async ({ id, reviewerId }: { id: string; reviewerId: number }) => {
		const res = await safeFetch(`${BACKEND}/documents/${id}/reviewer`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ reviewer_id: reviewerId })
		});
		if (!res) return { ok: false as const, error: NETWORK_ERROR };
		if (!res.ok) {
			return { ok: false as const, error: await parseJsonError(res, 'Failed to assign reviewer') };
		}
		return { ok: true as const };
	}
);
