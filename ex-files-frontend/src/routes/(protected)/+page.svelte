<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { getDashboard, getMyIssues } from '$lib/queries.remote';
	import { isManager, formatTimestamp } from '$lib/utils';
	import { deadlineChip } from '$lib/deadline';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import {
		CalendarClock,
		AlertTriangle,
		Clock,
		Inbox,
		PencilLine,
		Activity,
		ListChecks,
		Search
	} from '@lucide/svelte';
	import type { Issue } from '$lib/api';

	const { data } = $props();
	const me = $derived(data.user);
	const firstName = $derived(me?.name?.split(' ')[0] ?? '');
	const dashboardQuery = $derived(getDashboard());
	const dash = $derived(dashboardQuery.current);
	const showCreatedCard = $derived(isManager(me?.role));

	const PER_PAGE = 10;
	let searchInput = $state('');
	let searchTerm = $state('');
	let page = $state(1);
	let debounceHandle: ReturnType<typeof setTimeout> | null = null;

	function onSearchInput(value: string) {
		searchInput = value;
		if (debounceHandle) clearTimeout(debounceHandle);
		debounceHandle = setTimeout(() => {
			searchTerm = value.trim();
			page = 1;
		}, 250);
	}

	const myIssuesQuery = $derived(getMyIssues({ page, perPage: PER_PAGE, search: searchTerm }));
	const myIssues = $derived(myIssuesQuery.current?.issues ?? []);
	const myIssuesTotal = $derived(myIssuesQuery.current?.total ?? 0);
	const myIssuesTotalPages = $derived(Math.max(1, myIssuesQuery.current?.totalPages ?? 1));

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

	function myIssueHref(workspaceId: string, issueId: string): string {
		return localizeHref(`/workspaces/${workspaceId}/issues/${issueId}`);
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

	<!-- My current issues (paginated, searchable) -->
	<Card.Root>
		<Card.Header>
			<div class="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
				<div>
					<Card.Title class="flex items-center gap-2 text-base">
						<ListChecks class="size-4 text-muted-foreground" />
						{m.dashboard_my_issues()}
					</Card.Title>
					<Card.Description class="text-xs">{m.dashboard_my_issues_hint()}</Card.Description>
				</div>
				<div class="relative w-full md:w-72">
					<Search
						class="absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
					/>
					<Input
						type="text"
						value={searchInput}
						oninput={(e: Event) => onSearchInput((e.target as HTMLInputElement).value)}
						placeholder={m.dashboard_search_placeholder()}
						class="pl-7 h-9 text-sm"
					/>
				</div>
			</div>
		</Card.Header>
		<Card.Content>
			{#if myIssues.length === 0}
				<p class="py-4 text-center text-sm text-muted-foreground">
					{m.dashboard_no_my_issues()}
				</p>
			{:else}
				<div class="overflow-x-auto">
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b text-xs text-muted-foreground">
								<th class="px-2 py-2 text-left font-medium">{m.dashboard_col_title()}</th>
								<th class="px-2 py-2 text-left font-medium">{m.dashboard_col_workspace()}</th>
								<th class="px-2 py-2 text-left font-medium">{m.dashboard_col_owner()}</th>
								<th class="px-2 py-2 text-left font-medium">{m.dashboard_col_due()}</th>
							</tr>
						</thead>
						<tbody class="divide-y">
							{#each myIssues as issue (issue.id)}
								{@const chip = issue.deadline ? deadlineChip(new Date(issue.deadline)) : null}
								<tr class="hover:bg-muted/40 transition-colors">
									<td class="px-2 py-2 align-middle">
										<a
											href={myIssueHref(issue.workspaceId, issue.id)}
											class="font-medium hover:underline"
										>
											{issue.title}
										</a>
										<span class="ml-2 text-[10px] text-muted-foreground">#{issue.id}</span>
									</td>
									<td class="px-2 py-2 align-middle">
										<a
											href={localizeHref(`/workspaces/${issue.workspaceId}`)}
											class="hover:underline"
										>
											{issue.workspaceName}
										</a>
										<span class="ml-1 text-[10px] text-muted-foreground">#{issue.workspaceId}</span>
									</td>
									<td class="px-2 py-2 align-middle text-muted-foreground">
										{issue.workspaceManagerName}
									</td>
									<td class="px-2 py-2 align-middle">
										{#if chip}
											<Badge variant="outline" class="gap-1 text-[11px] {chip.cls}">
												<Clock class="size-3" />
												{chip.label}
											</Badge>
										{:else}
											<span class="text-[11px] text-muted-foreground">-</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}

			{#if myIssuesTotal > 0}
				<div class="mt-4 flex items-center justify-between text-xs text-muted-foreground">
					<span>
						{m.dashboard_pagination({
							page: String(page),
							total: String(myIssuesTotalPages)
						})}
					</span>
					<div class="flex gap-2">
						<Button
							variant="outline"
							size="sm"
							disabled={page <= 1}
							onclick={() => (page = Math.max(1, page - 1))}
						>
							{m.dashboard_prev()}
						</Button>
						<Button
							variant="outline"
							size="sm"
							disabled={page >= myIssuesTotalPages}
							onclick={() => (page = Math.min(myIssuesTotalPages, page + 1))}
						>
							{m.dashboard_next()}
						</Button>
					</div>
				</div>
			{/if}
		</Card.Content>
	</Card.Root>
</div>
