import type { Role } from '$lib/api';

/** True for manager-level roles (manager or root). */
export function isManager(role?: Role | string): boolean {
	return role === 'manager' || role === 'root';
}

/** Pass-through helper kept for callers that previously mapped proto enum to string. */
export function roleName(role?: Role | string): string {
	return role ?? 'unknown';
}

/** Format an ISO 8601 date-time string for display. */
export function formatTimestamp(
	iso?: string,
	opts?: { withTime?: boolean }
): string {
	if (!iso) return '-';
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return '-';
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

/** Extract initials from a full name. */
export function initials(name: string): string {
	return name
		.split(' ')
		.map((p) => p[0])
		.join('')
		.toUpperCase()
		.slice(0, 2);
}
