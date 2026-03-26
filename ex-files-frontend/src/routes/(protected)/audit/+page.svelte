<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getAuditLog } from '$lib/data.remote';
	import { m } from '$lib/paraglide/messages.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { ChevronLeft, ChevronRight, Filter, X } from '@lucide/svelte';
	import { protoTsToDate } from '$lib/proto-utils';
	import type { Timestamp } from '@bufbuild/protobuf/wkt';

	// Read all filters from URL (reactive)
	const currentPage = $derived(Number(page.url.searchParams.get('page') ?? '1'));
	const filterAction = $derived(page.url.searchParams.get('action') ?? '');
	const filterTargetType = $derived(page.url.searchParams.get('target_type') ?? '');
	const filterFrom = $derived(page.url.searchParams.get('from') ?? '');
	const filterTo = $derived(page.url.searchParams.get('to') ?? '');

	// Encode all params into a single query string for the unchecked query
	const auditQueryStr = $derived.by(() => {
		const parts: Record<string, string> = { page: String(currentPage) };
		if (filterAction) parts.action = filterAction;
		if (filterTargetType) parts.target_type = filterTargetType;
		if (filterFrom) parts.from = filterFrom;
		if (filterTo) parts.to = filterTo;
		return Object.entries(parts)
			.map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
			.join('&');
	});

	const logQuery = $derived(getAuditLog(auditQueryStr));
	const loading = $derived(logQuery.current === undefined);
	const logData = $derived(logQuery.current);
	const entries = $derived(logData?.entries ?? []);
	const totalPages = $derived(logData?.totalPages ?? 1);
	const total = $derived(logData?.total ?? 0);

	// Local filter form state
	let formAction = $state(page.url.searchParams.get('action') ?? '');
	let formTargetType = $state(page.url.searchParams.get('target_type') ?? '');
	let formFrom = $state(page.url.searchParams.get('from') ?? '');
	let formTo = $state(page.url.searchParams.get('to') ?? '');

	const hasFilters = $derived(filterAction || filterTargetType || filterFrom || filterTo);

	const KNOWN_ACTIONS = [
		'user.registered',
		'user.logged_in',
		'workspace.created',
		'workspace.updated',
		'workspace.deleted',
		'workspace.member_added',
		'workspace.member_removed',
		'document.uploaded',
		'document.version_created',
		'document.approved',
		'document.rejected',
		'document.comment_added',
		'user.role_changed'
	];

	const TARGET_TYPES = ['', 'user', 'workspace', 'document'];

	function actionCategory(action: string): string {
		if (action.startsWith('user.')) return 'user';
		if (action.startsWith('workspace.')) return 'workspace';
		if (action.startsWith('document.')) return 'document';
		return 'other';
	}

	function actionBadgeClass(action: string): string {
		const cat = actionCategory(action);
		switch (cat) {
			case 'user':
				return 'bg-violet-100 text-violet-700';
			case 'workspace':
				return 'bg-blue-100 text-blue-700';
			case 'document':
				return 'bg-emerald-100 text-emerald-700';
			default:
				return 'bg-muted text-muted-foreground';
		}
	}

	function formatDate(ts?: Timestamp): string {
		const d = protoTsToDate(ts);
		if (!d) return '—';
		return d.toLocaleString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function applyFilters() {
		const url = new URL(page.url);
		url.searchParams.set('page', '1');
		if (formAction) url.searchParams.set('action', formAction);
		else url.searchParams.delete('action');
		if (formTargetType) url.searchParams.set('target_type', formTargetType);
		else url.searchParams.delete('target_type');
		if (formFrom) url.searchParams.set('from', formFrom);
		else url.searchParams.delete('from');
		if (formTo) url.searchParams.set('to', formTo);
		else url.searchParams.delete('to');
		goto(url.toString());
	}

	function clearFilters() {
		formAction = '';
		formTargetType = '';
		formFrom = '';
		formTo = '';
		const url = new URL(page.url);
		url.searchParams.delete('action');
		url.searchParams.delete('target_type');
		url.searchParams.delete('from');
		url.searchParams.delete('to');
		url.searchParams.set('page', '1');
		goto(url.toString());
	}

	function navigatePage(p: number) {
		const url = new URL(page.url);
		url.searchParams.set('page', String(p));
		goto(url.toString());
	}
</script>

<svelte:head>
	<title>{m.audit_page_title()}</title>
</svelte:head>

<div class="flex flex-1 flex-col gap-6 p-6">
	<div class="flex items-start justify-between gap-4">
		<div>
			<h1 class="text-lg font-semibold">{m.audit_heading()}</h1>
			<p class="text-sm text-muted-foreground">
				{m.audit_description()}
				{#if total > 0}
					<span class="font-medium text-foreground">{m.audit_entries_count({ count: total.toLocaleString() })}</span>.
				{/if}
			</p>
		</div>
	</div>

	<!-- Filters -->
	<Card.Root>
		<Card.Header class="pb-3">
			<div class="flex items-center gap-1.5">
				<Filter class="size-3.5 text-muted-foreground" />
				<Card.Title class="text-sm">{m.audit_filters()}</Card.Title>
				{#if hasFilters}
					<Button
						variant="ghost"
						size="sm"
						class="ml-auto h-6 gap-1 px-2 text-xs text-muted-foreground"
						onclick={clearFilters}
					>
						<X class="size-3" />
						{m.common_clear()}
					</Button>
				{/if}
			</div>
		</Card.Header>
		<Card.Content>
			<form
				class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4"
				onsubmit={(e) => {
					e.preventDefault();
					applyFilters();
				}}
			>
				<!-- Action filter -->
				<div class="grid gap-1.5">
					<Label class="text-xs">{m.audit_action_label()}</Label>
					<select
						class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none"
						bind:value={formAction}
					>
						<option value="">{m.audit_all_actions()}</option>
						{#each KNOWN_ACTIONS as a (a)}
							<option value={a}>{a}</option>
						{/each}
					</select>
				</div>

				<!-- Target type filter -->
				<div class="grid gap-1.5">
					<Label class="text-xs">{m.audit_target_type()}</Label>
					<select
						class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none"
						bind:value={formTargetType}
					>
						{#each TARGET_TYPES as t (t)}
							<option value={t}>{t || m.audit_all_types()}</option>
						{/each}
					</select>
				</div>

				<!-- Date range -->
				<div class="grid gap-1.5">
					<Label class="text-xs">{m.audit_from()}</Label>
					<Input type="date" bind:value={formFrom} />
				</div>
				<div class="grid gap-1.5">
					<Label class="text-xs">{m.audit_to()}</Label>
					<Input type="date" bind:value={formTo} />
				</div>

				<div class="flex items-end sm:col-span-2 lg:col-span-4">
					<Button type="submit" size="sm">{m.audit_apply_filters()}</Button>
				</div>
			</form>
		</Card.Content>
	</Card.Root>

	<!-- Entries -->
	{#if loading}
		<Card.Root>
			<Card.Content class="p-0">
				<div class="divide-y">
					{#each { length: 8 } as _, i (i)}
						<div class="flex items-center gap-3 px-4 py-3">
							<Skeleton class="h-3 w-36 shrink-0 rounded" />
							<Separator orientation="vertical" class="h-6 self-stretch" />
							<Skeleton class="h-5 w-40 shrink-0 rounded-full" />
							<div class="min-w-0 flex-1">
								<Skeleton class="h-4 w-24 rounded" />
								<Skeleton class="mt-1 h-3 w-16 rounded" />
							</div>
						</div>
					{/each}
				</div>
			</Card.Content>
		</Card.Root>
	{:else if entries.length === 0}
		<Card.Root class="flex flex-col items-center justify-center py-16 text-center">
			<Card.Content>
				<p class="text-sm font-medium">{m.audit_no_entries()}</p>
				<p class="mt-1 text-xs text-muted-foreground">
					{hasFilters ? m.audit_adjust_filters() : m.audit_empty()}
				</p>
			</Card.Content>
		</Card.Root>
	{:else}
		<Card.Root>
			<Card.Content class="p-0">
				<div class="divide-y">
					{#each entries as entry (entry.id)}
						<div class="flex items-start gap-3 px-4 py-3">
							<!-- Timestamp -->
							<div class="w-36 shrink-0 text-xs text-muted-foreground">
								{formatDate(entry.createdAt)}
							</div>

							<Separator orientation="vertical" class="h-auto self-stretch" />

							<!-- Action badge -->
							<div class="w-56 shrink-0">
								<Badge
									variant="secondary"
									class="font-mono text-[10px] {actionBadgeClass(entry.action)}"
								>
									{entry.action}
								</Badge>
							</div>

							<!-- Actor -->
							<div class="min-w-0 flex-1">
								<p class="text-sm font-medium">{entry.actorName}</p>
								<p class="text-xs text-muted-foreground">
									{#if entry.targetType}
										{entry.targetType}
										{#if entry.targetId}
											#{entry.targetId}
										{/if}
									{/if}
								</p>
							</div>

							<!-- Metadata -->
							{#if entry.metadata && Object.keys(entry.metadata).length > 0}
								<div class="hidden max-w-xs shrink-0 xl:block">
									<p class="truncate font-mono text-[10px] text-muted-foreground">
										{JSON.stringify(entry.metadata)}
									</p>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</Card.Content>
		</Card.Root>

		<!-- Pagination -->
		{#if totalPages > 1}
			<div class="flex items-center justify-center gap-2">
				<Button
					variant="outline"
					size="sm"
					class="gap-1"
					disabled={currentPage <= 1}
					onclick={() => navigatePage(currentPage - 1)}
				>
					<ChevronLeft class="size-4" />
					{m.common_prev()}
				</Button>
				<span class="text-sm text-muted-foreground">{m.common_page_of({ current: String(currentPage), total: String(totalPages) })}</span>
				<Button
					variant="outline"
					size="sm"
					class="gap-1"
					disabled={currentPage >= totalPages}
					onclick={() => navigatePage(currentPage + 1)}
				>
					{m.common_next()}
					<ChevronRight class="size-4" />
				</Button>
			</div>
		{/if}
	{/if}
</div>
