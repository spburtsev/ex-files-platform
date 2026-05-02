<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getWorkspaces } from '$lib/queries.remote';
	import { isManager } from '$lib/utils';
	import { m } from '$lib/paraglide/messages.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { FolderOpen, Search, X } from '@lucide/svelte';
	import WorkspaceCard from './WorkspaceCard.svelte';
	import WorkspaceSkeleton from './WorkspaceSkeleton.svelte';
	import Pagination from '$lib/components/custom/Pagination.svelte';
	import CreateWorkspace from './CreateWorkspace.svelte';

	const { data: pageData } = $props();
	const me = $derived(pageData.user);

	let searchInput = $state('');
	let committedSearch = $state('');
	type StatusFilter = 'all' | 'active' | 'archived';
	let statusFilter = $state<StatusFilter>('active');

	$effect(() => {
		const value = searchInput;
		const t = setTimeout(() => {
			committedSearch = value;
			if (currentPage !== 1) navigatePage(1);
		}, 200);
		return () => clearTimeout(t);
	});

    function handleFilterChange() {
        if (currentPage !== 1) navigatePage(1);
    }

	function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		committedSearch = searchInput;
		if (currentPage !== 1) navigatePage(1);
	}

	function clearSearch() {
		searchInput = '';
		committedSearch = '';
		if (currentPage !== 1) navigatePage(1);
	}

	const currentPage = $derived(Number(page.url.searchParams.get('page') ?? '1'));
	const workspacesQuery = $derived(
		getWorkspaces({ page: currentPage, search: committedSearch, status: statusFilter })
	);
	const data = $derived(workspacesQuery.current);
	const loading = $derived(workspacesQuery.current === undefined);
	const workspaces = $derived(data?.workspaces ?? []);
	const totalPages = $derived(data?.totalPages ?? 1);

	let createOpen = $state(false);

	function navigatePage(p: number) {
		const url = new URL(page.url);
		url.searchParams.set('page', String(p));
		goto(url.toString());
	}

	const displayedFilter = $derived.by(() => {
		if (statusFilter === 'active') return m.ws_status_active();
		if (statusFilter === 'archived') return m.ws_status_archived();

		return m.ws_status_all();
	});
</script>

<svelte:head>
	<title>{m.ws_page_title()}</title>
</svelte:head>

<div class="flex flex-1 flex-col gap-6 p-6">
	<div class="flex items-start justify-between gap-4">
		<div>
			<h1 class="text-lg font-semibold">{m.ws_heading()}</h1>
			<p class="text-sm text-muted-foreground">
				{m.ws_description()}
			</p>
		</div>
		{#if isManager(me?.role)}
			<CreateWorkspace bind:open={createOpen} />
		{/if}
	</div>

	<form onsubmit={handleSubmit} class="flex max-w-xl items-center gap-2" role="search">
		<div class="relative flex-1">
			<Search
				class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
			/>
			<Input
				type="search"
				bind:value={searchInput}
				placeholder={m.common_search()}
				aria-label={m.common_search()}
				class="pr-8 pl-8"
			/>
			{#if searchInput !== ''}
				<Button
					type="button"
					variant="ghost"
					size="icon"
					onclick={clearSearch}
					class="absolute top-1/2 right-1 size-7 -translate-y-1/2"
					aria-label={m.common_clear()}
				>
					<X class="size-3.5" />
				</Button>
			{/if}
		</div>
		<Select.Root bind:value={statusFilter} type="single" onValueChange={handleFilterChange}>
			<Select.Trigger class="w-25">{displayedFilter}</Select.Trigger>
			<Select.Content>
				<Select.Item value="all">{m.ws_status_all()}</Select.Item>
				<Select.Item value="active">{m.ws_status_active()}</Select.Item>
				<Select.Item value="archived">{m.ws_status_archived()}</Select.Item>
			</Select.Content>
		</Select.Root>
	</form>

	{#if loading}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each { length: 6 } as _, i (i)}
				<WorkspaceSkeleton />
			{/each}
		</div>
	{:else if workspaces.length === 0}
		<Card.Root class="flex flex-col items-center justify-center py-16 text-center">
			<Card.Content>
				<FolderOpen class="mx-auto mb-3 size-10 text-muted-foreground/40" />
				{#if committedSearch !== ''}
					<p class="text-sm font-medium">{m.ws_search_no_results()}</p>
				{:else}
					<p class="text-sm font-medium">{m.ws_no_workspaces()}</p>
					<p class="mt-1 text-xs text-muted-foreground">
						{isManager(me?.role) ? m.ws_no_workspaces_manager() : m.ws_no_workspaces_employee()}
					</p>
				{/if}
			</Card.Content>
		</Card.Root>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each workspaces as ws (ws.id)}
				<WorkspaceCard {ws} />
			{/each}
		</div>

		{#if totalPages > 1}
			<Pagination {currentPage} {totalPages} {navigatePage} />
		{/if}
	{/if}
</div>
