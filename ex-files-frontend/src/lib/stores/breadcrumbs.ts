import { writable } from 'svelte/store';

export interface BreadcrumbSegment {
	label: string;
	href?: string;
}

/**
 * Extra breadcrumb segments to append after the primary nav label.
 * Child pages set this on mount and clear it on destroy.
 */
export const extraBreadcrumbs = writable<BreadcrumbSegment[]>([]);
