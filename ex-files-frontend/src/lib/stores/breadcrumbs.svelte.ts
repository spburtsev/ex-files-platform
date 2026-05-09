export interface BreadcrumbBadge {
	label: string;
	cls?: string;
}

export interface BreadcrumbSegment {
	label: string;
	href?: string;
	/** Optional badges rendered inline after the segment label. */
	badges?: BreadcrumbBadge[];
}

function createExtraBreadcrumbs() {
	let segments = $state<BreadcrumbSegment[]>([]);
	return {
		get segments() {
			return segments;
		},
		set(next: BreadcrumbSegment[]) {
			segments = next;
		}
	};
}

export const extraBreadcrumbs = createExtraBreadcrumbs();
