import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { Role } from '$lib/gen/auth/v1/auth_pb';

export function protoTsToDate(ts?: Timestamp): Date | null {
	if (!ts) return null;
	return new Date(Number(ts.seconds) * 1000 + Math.floor(ts.nanos / 1_000_000));
}

export function tsToIso(ts?: Timestamp): string | undefined {
	const d = protoTsToDate(ts);
	return d ? d.toISOString().slice(0, 19) : undefined;
}

/** Check if a proto auth Role is a manager-level role (manager or root) */
export function isManager(role?: Role): boolean {
	return role === Role.MANAGER || role === Role.ROOT;
}

/** Map proto auth Role to a display string */
export function roleName(role?: Role): string {
	switch (role) {
		case Role.ROOT:
			return 'root';
		case Role.MANAGER:
			return 'manager';
		case Role.EMPLOYEE:
			return 'employee';
		default:
			return 'unknown';
	}
}

/** Format a protobuf Timestamp for display */
export function formatTimestamp(ts?: Timestamp, opts?: { withTime?: boolean }): string {
	const d = protoTsToDate(ts);
	if (!d) return '-';
	if (opts?.withTime) {
		return d.toLocaleString(undefined, {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}
	return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

/** Extract initials from a full name */
export function initials(name: string): string {
	return name
		.split(' ')
		.map((p) => p[0])
		.join('')
		.toUpperCase()
		.slice(0, 2);
}
