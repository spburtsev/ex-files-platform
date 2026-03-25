import { query, getRequestEvent } from '$app/server';
import { env } from '$env/dynamic/private';
import { fromBinary } from '@bufbuild/protobuf';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import {
	GetAssignmentsResponseSchema,
	GetUsersResponseSchema,
	GetAssignmentResponseSchema,
	Role
} from '$lib/gen/assignments/v1/assignments_pb';
import type { MockAssignment, MockUser } from '$lib/mock-data';

// ---------------------------------------------------------------------------
// Workspace types (proto JSON shape from c.JSON() with Go proto structs)
// ---------------------------------------------------------------------------

export interface ProtoTimestamp {
	seconds: number;
	nanos?: number;
}

export interface Workspace {
	id: number;
	name: string;
	managerId: number;
	createdAt?: ProtoTimestamp;
	updatedAt?: ProtoTimestamp;
}

export interface WorkspaceUser {
	id: number;
	name: string;
	email: string;
	role: number; // 1=root, 2=manager, 3=employee
	avatarUrl?: string;
}

export interface WorkspaceDetail {
	workspace: Workspace;
	manager: WorkspaceUser;
	members: WorkspaceUser[];
}

export interface WorkspaceListData {
	workspaces: Workspace[];
	total: number;
	totalPages: number;
	page: number;
	perPage: number;
}

export function protoTsToDate(ts?: ProtoTimestamp): Date | null {
	if (!ts) return null;
	return new Date(Number(ts.seconds) * 1000);
}

export function workspaceUserRole(role: number): 'root' | 'manager' | 'employee' {
	if (role === 1) return 'root';
	if (role === 2) return 'manager';
	return 'employee';
}

const BACKEND = env.BACKEND_URL ?? 'http://localhost:8080';

async function fetchProto(url: string): Promise<Uint8Array> {
	const res = await fetch(url);
	return new Uint8Array(await res.arrayBuffer());
}

function tsToIso(ts?: Timestamp): string | undefined {
	return ts ? new Date(Number(ts.seconds) * 1000).toISOString().slice(0, 19) : undefined;
}

export const getMe = query(async (): Promise<MockUser | null> => {
	const event = getRequestEvent();
	const token = event.cookies.get('session');
	if (!token) return null;
	const res = await fetch(`${BACKEND}/auth/me`, {
		headers: { Authorization: `Bearer ${token}` }
	});
	if (!res.ok) return null;
	const { user: u } = await res.json();
	return {
		id: String(u.id),
		name: u.name,
		email: u.email,
		// auth proto: ROLE_MANAGER=2, ROLE_EMPLOYEE=3, ROLE_ROOT=1
		role: u.role === 2 ? 'manager' : 'employee'
	};
});

export const getUsers = query(async (): Promise<MockUser[]> => {
	const r = fromBinary(GetUsersResponseSchema, await fetchProto(`${BACKEND}/users`));
	return r.users.map((u) => ({
		id: u.id,
		name: u.name,
		email: u.email,
		role: u.role === Role.MANAGER ? 'manager' : 'employee'
	}));
});

export const getAssignments = query(async (): Promise<MockAssignment[]> => {
	const r = fromBinary(GetAssignmentsResponseSchema, await fetchProto(`${BACKEND}/assignments`));
	return r.assignments.map((a) => ({
		id: a.id,
		creatorId: a.creatorId,
		assigneeId: a.assigneeId,
		title: a.title,
		description: a.description,
		deadline: tsToIso(a.deadline),
		resolved: a.resolved,
		commentsCount: a.commentsCount,
		versionsCount: a.versionsCount
	}));
});

export const getAssignment = query('unchecked', async (id: string) => {
	const r = fromBinary(
		GetAssignmentResponseSchema,
		await fetchProto(`${BACKEND}/assignments/${id}`)
	);
	const a = r.assignment!;
	const u = r.user;
	return {
		assignment: {
			id: a.id,
			creatorId: a.creatorId,
			assigneeId: a.assigneeId,
			title: a.title,
			description: a.description,
			deadline: tsToIso(a.deadline),
			resolved: a.resolved,
			commentsCount: a.commentsCount,
			versionsCount: a.versionsCount
		} satisfies MockAssignment,
		user: u
			? ({
					id: u.id,
					name: u.name,
					email: u.email,
					role: u.role === Role.MANAGER ? 'manager' : 'employee'
				} satisfies MockUser)
			: null
	};
});

// ---------------------------------------------------------------------------
// Workspace queries
// ---------------------------------------------------------------------------

export const getWorkspaces = query(
	'unchecked',
	async (page: number = 1): Promise<WorkspaceListData> => {
		const event = getRequestEvent();
		const token = event.cookies.get('session');
		const res = await fetch(`${BACKEND}/workspaces?page=${page}&per_page=20`, {
			headers: { Authorization: `Bearer ${token}` }
		});
		if (!res.ok) return { workspaces: [], total: 0, totalPages: 0, page: 1, perPage: 20 };
		const data = await res.json();
		return {
			workspaces: (data.workspaces ?? []) as Workspace[],
			total: Number(res.headers.get('X-Total-Count') ?? 0),
			totalPages: Number(res.headers.get('X-Total-Pages') ?? 1),
			page: Number(res.headers.get('X-Page') ?? 1),
			perPage: Number(res.headers.get('X-Per-Page') ?? 20)
		};
	}
);

export const getWorkspaceDetail = query(
	'unchecked',
	async (id: string): Promise<WorkspaceDetail | null> => {
		const event = getRequestEvent();
		const token = event.cookies.get('session');
		const res = await fetch(`${BACKEND}/workspaces/${id}`, {
			headers: { Authorization: `Bearer ${token}` }
		});
		if (!res.ok) return null;
		const data = await res.json();
		// Response shape: { workspace: WorkspaceDetail }
		return data.workspace as WorkspaceDetail;
	}
);

// ---------------------------------------------------------------------------
// Document types
// ---------------------------------------------------------------------------

export interface Document {
	id: number;
	name: string;
	mimeType: string;
	size: number;
	hash: string;
	status: string; // pending | in_review | approved | rejected | changes_requested
	uploaderId: number;
	uploaderName: string;
	workspaceId: number;
	createdAt?: ProtoTimestamp;
	updatedAt?: ProtoTimestamp;
	reviewerId?: number;
	reviewerName?: string;
	reviewerNote?: string;
}

export interface DocumentVersion {
	id: number;
	documentId: number;
	version: number;
	hash: string;
	size: number;
	storageKey: string;
	uploaderId: number;
	uploaderName: string;
	createdAt?: ProtoTimestamp;
}

export interface DocumentDetail {
	document: Document;
	versions: DocumentVersion[];
}

export interface DocumentListData {
	documents: Document[];
	total: number;
	totalPages: number;
	page: number;
	perPage: number;
}

// ---------------------------------------------------------------------------
// Document queries
// ---------------------------------------------------------------------------

// getDocuments accepts a single encoded string: "<wsId>?page=1&search=foo&status=bar"
export const getDocuments = query(
	'unchecked',
	async (queryStr: string): Promise<DocumentListData> => {
		const sep = queryStr.indexOf('?');
		const wsId = sep === -1 ? queryStr : queryStr.slice(0, sep);
		const qs = sep === -1 ? '' : queryStr.slice(sep + 1);
		const sp = new URLSearchParams(qs);
		if (!sp.has('page')) sp.set('page', '1');
		if (!sp.has('per_page')) sp.set('per_page', '20');
		const event = getRequestEvent();
		const token = event.cookies.get('session');
		const res = await fetch(`${BACKEND}/workspaces/${wsId}/documents?${sp}`, {
			headers: { Authorization: `Bearer ${token}` }
		});
		if (!res.ok) return { documents: [], total: 0, totalPages: 0, page: 1, perPage: 20 };
		const data = await res.json();
		return {
			documents: (data.documents ?? []) as Document[],
			total: Number(res.headers.get('X-Total-Count') ?? 0),
			totalPages: Number(res.headers.get('X-Total-Pages') ?? 1),
			page: Number(res.headers.get('X-Page') ?? 1),
			perPage: Number(res.headers.get('X-Per-Page') ?? 20)
		};
	}
);

export const getDocumentDetail = query(
	'unchecked',
	async (docId: string): Promise<DocumentDetail | null> => {
		const event = getRequestEvent();
		const token = event.cookies.get('session');
		const res = await fetch(`${BACKEND}/documents/${docId}`, {
			headers: { Authorization: `Bearer ${token}` }
		});
		if (!res.ok) return null;
		const data = await res.json();
		// Response shape: { document: DocumentDetail }
		return data.document as DocumentDetail;
	}
);

// ---------------------------------------------------------------------------
// Audit types
// ---------------------------------------------------------------------------

export interface AuditEntry {
	id: number;
	action: string;
	actorId: number;
	actorName: string;
	targetId?: number;
	targetType: string;
	metadata?: Record<string, unknown>;
	createdAt?: ProtoTimestamp;
}

export interface AuditListData {
	entries: AuditEntry[];
	total: number;
	totalPages: number;
	page: number;
	perPage: number;
}

// ---------------------------------------------------------------------------
// Audit query
// ---------------------------------------------------------------------------

// getAuditLog accepts a single URLSearchParams-encoded string:
// "page=1&action=document.uploaded&target_type=document&from=2026-01-01&to=2026-12-31"
export const getAuditLog = query(
	'unchecked',
	async (queryStr: string = ''): Promise<AuditListData> => {
		const sp = new URLSearchParams(queryStr);
		if (!sp.has('page')) sp.set('page', '1');
		if (!sp.has('per_page')) sp.set('per_page', '25');

		// Convert date-only strings to RFC3339 for the backend
		const from = sp.get('from');
		const to = sp.get('to');
		if (from) sp.set('from', new Date(from).toISOString());
		if (to) {
			const d = new Date(to);
			d.setHours(23, 59, 59, 999);
			sp.set('to', d.toISOString());
		}

		const event = getRequestEvent();
		const token = event.cookies.get('session');
		const res = await fetch(`${BACKEND}/audit?${sp}`, {
			headers: { Authorization: `Bearer ${token}` }
		});
		if (!res.ok) return { entries: [], total: 0, totalPages: 0, page: 1, perPage: 25 };
		const data = await res.json();
		return {
			entries: (data.entries ?? []) as AuditEntry[],
			total: Number(res.headers.get('X-Total-Count') ?? 0),
			totalPages: Number(res.headers.get('X-Total-Pages') ?? 1),
			page: Number(res.headers.get('X-Page') ?? 1),
			perPage: Number(res.headers.get('X-Per-Page') ?? 25)
		};
	}
);
