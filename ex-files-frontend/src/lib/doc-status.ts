import type { DocumentStatus, ReviewStatus } from '$lib/stores/workbench.svelte';
import { m } from '$lib/paraglide/messages';

export function statusBadgeClass(status: DocumentStatus, reviewStatus?: ReviewStatus): string {
	switch (status) {
		case 'draft':
			return 'bg-amber-100 text-amber-800';
		case 'saving':
			return 'bg-blue-100 text-blue-700';
		case 'error':
			return 'bg-red-100 text-red-700';
		case 'saved':
			switch (reviewStatus) {
				case 'approved':
					return 'bg-emerald-100 text-emerald-700';
				case 'rejected':
					return 'bg-red-100 text-red-700';
				case 'changes_requested':
					return 'bg-amber-100 text-amber-800';
				case 'in_review':
					return 'bg-blue-100 text-blue-700';
				case 'pending':
				default:
					return 'bg-slate-100 text-slate-700';
			}
		default:
			return '';
	}
}

export function statusLabel(status: DocumentStatus, reviewStatus?: ReviewStatus): string {
	switch (status) {
		case 'draft':
			return m.workbench_status_draft();
		case 'saving':
			return m.workbench_saving();
		case 'error':
			return m.workbench_status_error();
		case 'saved':
			switch (reviewStatus) {
				case 'approved':
					return m.workbench_status_approved();
				case 'rejected':
					return m.workbench_status_rejected();
				case 'changes_requested':
					return m.workbench_status_changes_requested();
				case 'in_review':
					return m.workbench_status_awaiting_review();
				case 'pending':
				default:
					return m.workbench_status_saved();
			}
		default:
			return '';
	}
}

export function canActOn(reviewStatus?: ReviewStatus): boolean {
	return (
		reviewStatus === 'pending' ||
		reviewStatus === 'in_review' ||
		reviewStatus === 'changes_requested'
	);
}
