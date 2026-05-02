<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getWorkspaces } from '$lib/queries.remote';
	import { isManager } from '$lib/utils';
	import { m } from '$lib/paraglide/messages.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { FolderOpen } from '@lucide/svelte';
	import WorkspaceCard from './WorkspaceCard.svelte';
	import WorkspaceSkeleton from './WorkspaceSkeleton.svelte';
	import Pagination from '$lib/components/custom/Pagination.svelte';
	import CreateWorkspace from './CreateWorkspace.svelte';

	const { data: pageData } = $props();
	const me = $derived(pageData.user);

	const currentPage = $derived(Number(page.url.searchParams.get('page') ?? '1'));
	const workspacesQuery = $derived(getWorkspaces(currentPage));
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
				<p class="text-sm font-medium">{m.ws_no_workspaces()}</p>
				<p class="mt-1 text-xs text-muted-foreground">
					{isManager(me?.role) ? m.ws_no_workspaces_manager() : m.ws_no_workspaces_employee()}
				</p>
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
