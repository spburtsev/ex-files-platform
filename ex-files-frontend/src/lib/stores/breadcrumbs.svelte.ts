export interface BreadcrumbSegment {
	label: string;
	href?: string;
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
