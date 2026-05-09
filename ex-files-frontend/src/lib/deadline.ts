import { m } from '$lib/paraglide/messages.js';

export interface DeadlineChip {
	label: string;
	cls: string;
}

// Returns a label + tailwind classes describing how close (or past) a deadline
// is. Same tiers used by the issue page sidebar and the dashboard's due-soon
// / overdue lists, so a single change here updates both.
export function deadlineChip(deadline: Date): DeadlineChip {
	const h = (deadline.getTime() - Date.now()) / 3_600_000;
	if (h < 0) {
		return { label: m.workbench_overdue(), cls: 'border-red-200 bg-red-50 text-red-600' };
	}
	if (h < 24) {
		return {
			label: m.workbench_hours_left({ hours: String(Math.round(h)) }),
			cls: 'border-red-200 bg-red-50 text-red-600'
		};
	}
	if (h < 72) {
		return {
			label: m.workbench_days_hours_left({
				days: String(Math.floor(h / 24)),
				hours: String(Math.round(h % 24))
			}),
			cls: 'border-amber-200 bg-amber-50 text-amber-700'
		};
	}
	return {
		label: deadline.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
		cls: 'border-border bg-muted/40 text-muted-foreground'
	};
}
