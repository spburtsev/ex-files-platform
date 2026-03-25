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
