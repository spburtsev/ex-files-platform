import { query, getRequestEvent } from '$app/server';
import { env } from '$env/dynamic/private';
import { fromBinary } from '@bufbuild/protobuf';
import {
	GetIssuesResponseSchema,
	GetUsersResponseSchema,
	GetIssueResponseSchema
} from '$lib/gen/issues/v1/issues_pb';
import { MeResponseSchema } from '$lib/gen/auth/v1/auth_pb';
import {
	GetWorkspacesResponseSchema,
	GetWorkspaceResponseSchema
} from '$lib/gen/workspaces/v1/workspaces_pb';
import {
	ListDocumentsResponseSchema,
	GetDocumentResponseSchema
} from '$lib/gen/documents/v1/documents_pb';
import { GetAuditLogResponseSchema } from '$lib/gen/audit/v1/audit_pb';

const BACKEND = env.BACKEND_URL ?? 'http://localhost:8080';

/** Fetch a URL and return the raw bytes, or null on any failure (network error, non-2xx). */
async function fetchProto(url: string, fetchFn: typeof fetch) {
	try {
		const res = await fetchFn(url);
		if (!res.ok) return null;
		return new Uint8Array(await res.arrayBuffer());
	} catch {
		return null;
	}
}

/** Fetch a URL and return the Response, or null on network error. */
async function safeFetch(url: string, fetchFn: typeof fetch, init?: RequestInit) {
	try {
		return await fetchFn(url, init);
	} catch {
		return null;
	}
}

function paginationFromHeaders(res: Response | null) {
	if (!res) return { total: 0, totalPages: 0, page: 1, perPage: 20 };
	return {
		total: Number(res.headers.get('X-Total-Count') ?? 0),
		totalPages: Number(res.headers.get('X-Total-Pages') ?? 1),
		page: Number(res.headers.get('X-Page') ?? 1),
		perPage: Number(res.headers.get('X-Per-Page') ?? 20)
	};
}

export const getMe = query(async () => {
	const { fetch } = getRequestEvent();
	const res = await safeFetch(`${BACKEND}/auth/me`, fetch);
	if (!res) return { user: null, error: 'Unable to reach the server' as const };
	if (!res.ok) return { user: null, error: null };
	const bytes = new Uint8Array(await res.arrayBuffer());
	const r = fromBinary(MeResponseSchema, bytes);
	return { user: r.user ?? null, error: null };
});

export const getUsers = query(async () => {
	const { fetch } = getRequestEvent();
	const bytes = await fetchProto(`${BACKEND}/users`, fetch);
	if (!bytes) return [];
	const r = fromBinary(GetUsersResponseSchema, bytes);
	return r.users;
});

export const getIssues = query('unchecked', async (workspaceId: string) => {
	const { fetch } = getRequestEvent();
	const bytes = await fetchProto(`${BACKEND}/workspaces/${workspaceId}/issues`, fetch);
	if (!bytes) return [];
	const r = fromBinary(GetIssuesResponseSchema, bytes);
	return r.issues;
});

export const getIssue = query('unchecked', async (id: string) => {
	const { fetch } = getRequestEvent();
	const bytes = await fetchProto(`${BACKEND}/issues/${id}`, fetch);
	if (!bytes) return null;
	return fromBinary(GetIssueResponseSchema, bytes);
});

// ---------------------------------------------------------------------------
// Workspace queries
// ---------------------------------------------------------------------------

export const getWorkspaces = query('unchecked', async (page: number = 1) => {
	const { fetch } = getRequestEvent();
	const res = await safeFetch(`${BACKEND}/workspaces?page=${page}&per_page=20`, fetch);
	if (!res || !res.ok) return { workspaces: [] as never[], ...paginationFromHeaders(res) };
	const bytes = new Uint8Array(await res.arrayBuffer());
	const r = fromBinary(GetWorkspacesResponseSchema, bytes);
	return {
		workspaces: r.workspaces,
		...paginationFromHeaders(res)
	};
});

export const getWorkspaceDetail = query('unchecked', async (id: string) => {
	const { fetch } = getRequestEvent();
	const res = await safeFetch(`${BACKEND}/workspaces/${id}`, fetch);
	if (!res || !res.ok) return null;
	const bytes = new Uint8Array(await res.arrayBuffer());
	const r = fromBinary(GetWorkspaceResponseSchema, bytes);
	return r.workspace ?? null;
});

// getSystemUsers: auth endpoint still returns JSON (no ListUsersResponse proto)
export const getSystemUsers = query(async () => {
	const { fetch } = getRequestEvent();
	const res = await safeFetch(`${BACKEND}/auth/users`, fetch);
	if (!res || !res.ok) return [];
	const data = await res.json();
	return (data.users ?? []) as Array<{ id: number; name: string; email: string; role: number }>;
});

// ---------------------------------------------------------------------------
// Document queries
// ---------------------------------------------------------------------------

export const getDocuments = query('unchecked', async (queryStr: string) => {
	const sep = queryStr.indexOf('?');
	const issueId = sep === -1 ? queryStr : queryStr.slice(0, sep);
	const qs = sep === -1 ? '' : queryStr.slice(sep + 1);
	const sp = new URLSearchParams(qs);
	if (!sp.has('page')) sp.set('page', '1');
	if (!sp.has('per_page')) sp.set('per_page', '20');
	const { fetch } = getRequestEvent();
	const res = await safeFetch(`${BACKEND}/issues/${issueId}/documents?${sp}`, fetch);
	if (!res || !res.ok) return { documents: [] as never[], ...paginationFromHeaders(res) };
	const bytes = new Uint8Array(await res.arrayBuffer());
	const r = fromBinary(ListDocumentsResponseSchema, bytes);
	return {
		documents: r.documents,
		...paginationFromHeaders(res)
	};
});

export const getDocumentDetail = query('unchecked', async (docId: string) => {
	const { fetch } = getRequestEvent();
	const res = await safeFetch(`${BACKEND}/documents/${docId}`, fetch);
	if (!res || !res.ok) return null;
	const bytes = new Uint8Array(await res.arrayBuffer());
	const r = fromBinary(GetDocumentResponseSchema, bytes);
	return r.document ?? null;
});

// ---------------------------------------------------------------------------
// Audit query
// ---------------------------------------------------------------------------

export const getAuditLog = query('unchecked', async (queryStr: string = '') => {
	const sp = new URLSearchParams(queryStr);
	if (!sp.has('page')) sp.set('page', '1');
	if (!sp.has('per_page')) sp.set('per_page', '25');

	const from = sp.get('from');
	const to = sp.get('to');
	if (from) sp.set('from', new Date(from).toISOString());
	if (to) {
		const d = new Date(to);
		d.setHours(23, 59, 59, 999);
		sp.set('to', d.toISOString());
	}

	const { fetch } = getRequestEvent();
	const res = await safeFetch(`${BACKEND}/audit?${sp}`, fetch);
	if (!res || !res.ok)
		return { entries: [] as never[], total: 0, totalPages: 0, page: 1, perPage: 25 };
	const bytes = new Uint8Array(await res.arrayBuffer());
	const r = fromBinary(GetAuditLogResponseSchema, bytes);
	return {
		entries: r.entries,
		...paginationFromHeaders(res)
	};
});
