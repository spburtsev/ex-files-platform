import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';
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

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

export function initials(name: string) {
    return name
        .split(' ')
        .map((p) => p[0])
        .join('')
        .toUpperCase();
}

const palette = [
    'bg-blue-500',
    'bg-violet-500',
    'bg-emerald-500',
    'bg-rose-500',
    'bg-amber-500',
    'bg-cyan-500'
];

export function avatarColorClass(id: string) {
    let hash = 0;
    for (const ch of id) {
        hash = ch.charCodeAt(0) + ((hash << 5) - hash);
    }
    return palette[Math.abs(hash) % palette.length];
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChild<T> = T extends { child?: any } ? Omit<T, 'child'> : T;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChildren<T> = T extends { children?: any } ? Omit<T, 'children'> : T;
export type WithoutChildrenOrChild<T> = WithoutChildren<WithoutChild<T>>;
export type WithElementRef<T, U extends HTMLElement = HTMLElement> = T & { ref?: U | null };
