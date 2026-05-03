import { query } from '$app/server';

import { apiOpts } from '$lib/api-client';
import {
	auditList,
	auditStats,
	authListUsers,
	authMe,
	commentsList,
	documentsGet,
	documentsGetFile,
	documentsList,
	issuesGet,
	issuesListByWorkspace,
	workspacesAssignableMembers,
	workspacesGet,
	workspacesList
} from '$lib/api';

function paginationFromHeaders(res: Response | undefined) {
	const h = res?.headers;
	return {
		total: Number(h?.get('X-Total-Count') ?? 0),
		totalPages: Number(h?.get('X-Total-Pages') ?? 1),
		page: Number(h?.get('X-Page') ?? 1),
		perPage: Number(h?.get('X-Per-Page') ?? 20)
	};
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

export const getMe = query(async () => {
	try {
		const r = await authMe(apiOpts());
		return r.data?.user ?? null;
	} catch (err) {
		console.error('Failed to fetch /auth/me', err);
		return null;
	}
});

export const getUsers = query(async () => {
	const r = await authListUsers(apiOpts());
	return r.data?.users ?? [];
});

// ---------------------------------------------------------------------------
// Workspaces
// ---------------------------------------------------------------------------

export const getWorkspaces = query(
	'unchecked',
	async ({
		page = 1,
		search = '',
		status = 'active'
	}: { page?: number; search?: string; status?: 'all' | 'active' | 'archived' } = {}) => {
		const r = await workspacesList({
			...apiOpts(),
			query: { page, perPage: 20, search: search || undefined, status }
		});
		return {
			workspaces: r.data?.workspaces ?? [],
			...paginationFromHeaders(r.response)
		};
	}
);

export const getWorkspaceDetail = query('unchecked', async (id: string) => {
	const r = await workspacesGet({ ...apiOpts(), path: { id } });
	return r.data?.workspace ?? null;
});

export const getAssignableMembers = query('unchecked', async (workspaceId: string) => {
	const r = await workspacesAssignableMembers({ ...apiOpts(), path: { id: workspaceId } });
	return r.data?.users ?? [];
});

// ---------------------------------------------------------------------------
// Issues
// ---------------------------------------------------------------------------

export const getIssues = query(
	'unchecked',
	async ({
		workspaceId,
		search = '',
		status = 'all',
		archived = false
	}: {
		workspaceId: string;
		search?: string;
		status?: 'all' | 'open' | 'resolved';
		archived?: boolean;
	}) => {
		const r = await issuesListByWorkspace({
			...apiOpts(),
			path: { id: workspaceId },
			query: { search: search || undefined, status, archived }
		});
		return r.data?.issues ?? [];
	}
);

export const getIssue = query('unchecked', async (id: string) => {
	const r = await issuesGet({ ...apiOpts(), path: { id } });
	return r.data ?? null;
});

// ---------------------------------------------------------------------------
// Documents
// ---------------------------------------------------------------------------

export const getDocuments = query('unchecked', async (queryStr: string) => {
	const sep = queryStr.indexOf('?');
	const issueId = sep === -1 ? queryStr : queryStr.slice(0, sep);
	const params = new URLSearchParams(sep === -1 ? '' : queryStr.slice(sep + 1));
	const r = await documentsList({
		...apiOpts(),
		path: { id: issueId },
		query: {
			page: Number(params.get('page') ?? 1),
			perPage: Number(params.get('per_page') ?? 20),
			search: params.get('search') ?? undefined,
			status: (params.get('status') as never) ?? undefined
		}
	});
	return {
		documents: r.data?.documents ?? [],
		...paginationFromHeaders(r.response)
	};
});

export const getDocumentDetail = query('unchecked', async (docId: string) => {
	const r = await documentsGet({ ...apiOpts(), path: { id: docId } });
	return r.data?.document ?? null;
});

export const getDocumentBytes = query(
	'unchecked',
	async (arg: { docId: string; versionId: string }) => {
		const r = await documentsGetFile({
			...apiOpts(),
			path: { id: arg.docId, versionId: arg.versionId },
			parseAs: 'blob'
		});
		if (!r.data) return new Uint8Array();
		const blob = r.data as unknown as Blob;
		return new Uint8Array(await blob.arrayBuffer());
	}
);

// ---------------------------------------------------------------------------
// Comments
// ---------------------------------------------------------------------------

export const getComments = query('unchecked', async (docId: string) => {
	const r = await commentsList({ ...apiOpts(), path: { id: docId } });
	return r.data?.comments ?? [];
});

// ---------------------------------------------------------------------------
// Audit
// ---------------------------------------------------------------------------

export const getAuditStats = query(async () => {
	const r = await auditStats(apiOpts());
	return r.data ?? null;
});

export const getAuditLog = query('unchecked', async (queryStr: string = '') => {
	const sp = new URLSearchParams(queryStr);
	const from = sp.get('from');
	const to = sp.get('to');
	const fromIso = from ? new Date(from).toISOString() : undefined;
	let toIso: string | undefined;
	if (to) {
		const d = new Date(to);
		d.setHours(23, 59, 59, 999);
		toIso = d.toISOString();
	}
	const r = await auditList({
		...apiOpts(),
		query: {
			page: Number(sp.get('page') ?? 1),
			perPage: Number(sp.get('per_page') ?? 25),
			action: sp.get('action') ?? undefined,
			targetType: sp.get('target_type') ?? undefined,
			actorId: sp.get('actor_id') ?? undefined,
			targetId: sp.get('target_id') ?? undefined,
			from: fromIso,
			to: toIso
		}
	});
	return {
		entries: r.data?.entries ?? [],
		...paginationFromHeaders(r.response)
	};
});
