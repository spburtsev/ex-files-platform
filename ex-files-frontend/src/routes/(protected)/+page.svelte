<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { getDashboard } from '$lib/queries.remote';
	import { isManager, formatTimestamp } from '$lib/utils';
	import { deadlineChip } from '$lib/deadline';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { CalendarClock, AlertTriangle, Clock, Inbox, PencilLine, Activity } from '@lucide/svelte';
	import type { Issue } from '$lib/api';

	const { data } = $props();
	const me = $derived(data.user);
	const firstName = $derived(me?.name?.split(' ')[0] ?? '');
	const dashboardQuery = $derived(getDashboard());
	const dash = $derived(dashboardQuery.current);
	const showCreatedCard = $derived(isManager(me?.role));

	function relativeFromNow(iso?: string): string {
		if (!iso) return '';
		const ms = new Date(iso).getTime() - Date.now();
		const abs = Math.abs(ms);
		const minute = 60_000;
		const hour = 60 * minute;
		const day = 24 * hour;
		if (abs < hour) return Math.max(1, Math.round(abs / minute)) + 'm ago';
		if (abs < day) return Math.round(abs / hour) + 'h ago';
		if (abs < 7 * day) return Math.round(abs / day) + 'd ago';
		return formatTimestamp(iso, { withTime: false });
	}

	function issueHref(issue: Issue): string {
		return localizeHref(`/workspaces/${issue.workspaceId}/issues/${issue.id}`);
	}
</script>

<svelte:head>
	<title>{m.dashboard_page_title()}</title>
</svelte:head>

<div class="flex flex-1 flex-col gap-6 p-6">
	<h1 class="text-2xl font-semibold">{m.dashboard_greeting({ name: firstName })}</h1>

	<!-- Stat cards -->
	<div class="grid grid-cols-1 gap-4 md:grid-cols-3">
		<Card.Root>
			<Card.Header class="flex flex-row items-center justify-between pb-2">
				<Card.Title class="text-sm font-medium text-muted-foreground"
					>{m.dashboard_assigned_open()}</Card.Title
				>
				<Inbox class="size-4 text-muted-foreground" />
			</Card.Header>
			<Card.Content>
				<p class="text-3xl font-bold">{dash?.assignedOpenCount ?? 0}</p>
				<p class="mt-1 text-xs text-muted-foreground">{m.dashboard_open_issues()}</p>
			</Card.Content>
		</Card.Root>

		{#if showCreatedCard}
			<Card.Root>
				<Card.Header class="flex flex-row items-center justify-between pb-2">
					<Card.Title class="text-sm font-medium text-muted-foreground"
						>{m.dashboard_created_open()}</Card.Title
					>
					<PencilLine class="size-4 text-muted-foreground" />
				</Card.Header>
				<Card.Content>
					<p class="text-3xl font-bold">{dash?.createdOpenCount ?? 0}</p>
					<p class="mt-1 text-xs text-muted-foreground">{m.dashboard_open_issues()}</p>
				</Card.Content>
			</Card.Root>
		{/if}

		<Card.Root>
			<Card.Header class="flex flex-row items-center justify-between pb-2">
				<Card.Title class="text-sm font-medium text-muted-foreground"
					>{m.dashboard_overdue_count()}</Card.Title
				>
				<AlertTriangle class="size-4 text-red-500" />
			</Card.Header>
			<Card.Content>
				<p class="text-3xl font-bold {(dash?.overdue.length ?? 0) > 0 ? 'text-red-600' : ''}">
					{dash?.overdue.length ?? 0}
				</p>
				<p class="mt-1 text-xs text-muted-foreground">{m.dashboard_open_issues()}</p>
			</Card.Content>
		</Card.Root>
	</div>

	<!-- Due soon + Overdue -->
	<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
		<Card.Root>
			<Card.Header>
				<Card.Title class="flex items-center gap-2 text-base">
					<CalendarClock class="size-4 text-amber-600" />
					{m.dashboard_due_soon()}
				</Card.Title>
				<Card.Description class="text-xs">{m.dashboard_due_soon_hint()}</Card.Description>
			</Card.Header>
			<Card.Content>
				{#if !dash || dash.dueSoon.length === 0}
					<p class="py-2 text-sm text-muted-foreground">{m.dashboard_no_due_soon()}</p>
				{:else}
					<ul class="divide-y">
						{#each dash.dueSoon as issue (issue.id)}
							{@const chip = issue.deadline ? deadlineChip(new Date(issue.deadline)) : null}
							<li>
								<a
									href={issueHref(issue)}
									class="flex items-start justify-between gap-3 py-2 hover:bg-muted/40 rounded px-2 -mx-2 transition-colors"
								>
									<p class="min-w-0 flex-1 truncate text-sm font-medium">{issue.title}</p>
									{#if chip}
										<Badge variant="outline" class="shrink-0 gap-1 text-[11px] {chip.cls}">
											<Clock class="size-3" />
											{chip.label}
										</Badge>
									{/if}
								</a>
							</li>
						{/each}
					</ul>
				{/if}
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Header>
				<Card.Title class="flex items-center gap-2 text-base">
					<AlertTriangle class="size-4 text-red-500" />
					{m.dashboard_overdue()}
				</Card.Title>
				<Card.Description class="text-xs">{m.dashboard_overdue_hint()}</Card.Description>
			</Card.Header>
			<Card.Content>
				{#if !dash || dash.overdue.length === 0}
					<p class="py-2 text-sm text-muted-foreground">{m.dashboard_no_overdue()}</p>
				{:else}
					<ul class="divide-y">
						{#each dash.overdue as issue (issue.id)}
							{@const chip = issue.deadline ? deadlineChip(new Date(issue.deadline)) : null}
							<li>
								<a
									href={issueHref(issue)}
									class="flex items-start justify-between gap-3 py-2 hover:bg-muted/40 rounded px-2 -mx-2 transition-colors"
								>
									<p class="min-w-0 flex-1 truncate text-sm font-medium">{issue.title}</p>
									{#if chip}
										<Badge variant="outline" class="shrink-0 gap-1 text-[11px] {chip.cls}">
											<Clock class="size-3" />
											{chip.label}
										</Badge>
									{/if}
								</a>
							</li>
						{/each}
					</ul>
				{/if}
			</Card.Content>
		</Card.Root>
	</div>

	<!-- Recent activity -->
	<Card.Root>
		<Card.Header>
			<Card.Title class="flex items-center gap-2 text-base">
				<Activity class="size-4 text-muted-foreground" />
				{m.dashboard_recent()}
			</Card.Title>
			<Card.Description class="text-xs">{m.dashboard_recent_hint()}</Card.Description>
		</Card.Header>
		<Card.Content>
			{#if !dash || dash.recent.length === 0}
				<p class="py-2 text-sm text-muted-foreground">{m.dashboard_no_recent()}</p>
			{:else}
				<ul class="divide-y">
					{#each dash.recent as issue (issue.id)}
						<li>
							<a
								href={issueHref(issue)}
								class="flex items-center justify-between gap-3 py-2 hover:bg-muted/40 rounded px-2 -mx-2 transition-colors"
							>
								<p class="min-w-0 flex-1 truncate text-sm font-medium">{issue.title}</p>
								<span class="shrink-0 text-xs text-muted-foreground">
									{m.dashboard_last_activity({
										when: relativeFromNow(issue.lastActivityAt ?? issue.updatedAt)
									})}
								</span>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</Card.Content>
	</Card.Root>
</div>
