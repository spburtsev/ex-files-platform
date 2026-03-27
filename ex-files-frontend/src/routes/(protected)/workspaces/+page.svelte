<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getWorkspaces, getMe } from '$lib/data.remote';
	import { createWorkspace } from '$lib/commands.remote';
	import { formatTimestamp, isManager } from '$lib/proto-utils';
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { FolderOpen, Plus, ArrowRight, ChevronLeft, ChevronRight, Users } from '@lucide/svelte';

	const meQuery = getMe();
	const me = $derived(meQuery.current);

	const currentPage = $derived(Number(page.url.searchParams.get('page') ?? '1'));
	const workspacesQuery = $derived(getWorkspaces(currentPage));
	const data = $derived(workspacesQuery.current);
	const loading = $derived(workspacesQuery.current === undefined);
	const workspaces = $derived(data?.workspaces ?? []);
	const totalPages = $derived(data?.totalPages ?? 1);

	let createOpen = $state(false);
	let createName = $state('');
	let creating = $state(false);
	let createError = $state('');

	function navigatePage(p: number) {
		const url = new URL(page.url);
		url.searchParams.set('page', String(p));
		goto(url.toString());
	}

	async function handleCreate() {
		if (!createName.trim()) return;
		creating = true;
		createError = '';
		try {
			const result = await createWorkspace(createName.trim());
			if (!result.ok) {
				createError = result.error ?? m.ws_create_error();
				return;
			}
			createOpen = false;
			createName = '';
			goto(localizeHref(`/workspaces/${result.workspace.id}`));
		} catch {
			createError = m.error_network_retry();
		} finally {
			creating = false;
		}
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
			<Dialog.Root bind:open={createOpen}>
				<Dialog.Trigger>
					{#snippet child({ props })}
						<Button size="sm" class="gap-1.5" {...props}>
							<Plus class="size-4" />
							{m.ws_new()}
						</Button>
					{/snippet}
				</Dialog.Trigger>
				<Dialog.Content class="sm:max-w-md">
					<Dialog.Header>
						<Dialog.Title>{m.ws_create_title()}</Dialog.Title>
						<Dialog.Description>{m.ws_create_description()}</Dialog.Description>
					</Dialog.Header>
					<div class="grid gap-4 py-4">
						<div class="grid gap-2">
							<Label for="ws-name">{m.common_name()}</Label>
							<Input
								id="ws-name"
								placeholder={m.ws_name_placeholder()}
								bind:value={createName}
								onkeydown={(e) => e.key === 'Enter' && handleCreate()}
							/>
						</div>
						{#if createError}
							<p class="text-sm text-destructive">{createError}</p>
						{/if}
					</div>
					<Dialog.Footer>
						<Dialog.Close>
							{#snippet child({ props })}
								<Button variant="outline" {...props}>{m.common_cancel()}</Button>
							{/snippet}
						</Dialog.Close>
						<Button onclick={handleCreate} disabled={creating || !createName.trim()}>
							{creating ? m.common_creating() : m.common_create()}
						</Button>
					</Dialog.Footer>
				</Dialog.Content>
			</Dialog.Root>
		{/if}
	</div>

	{#if loading}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each { length: 6 } as _, i (i)}
				<Card.Root class="flex flex-col">
					<Card.Header>
						<div class="flex items-start justify-between gap-2">
							<Skeleton class="h-4 w-32 rounded" />
							<Skeleton class="h-5 w-14 rounded-full" />
						</div>
					</Card.Header>
					<Card.Content class="pb-3">
						<Skeleton class="h-3 w-28 rounded" />
						<Skeleton class="mt-2 h-3 w-36 rounded" />
					</Card.Content>
					<Card.Footer class="mt-auto border-t pt-3">
						<Skeleton class="h-8 w-full rounded-md" />
					</Card.Footer>
				</Card.Root>
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
				<Card.Root class="flex flex-col transition-shadow hover:shadow-md">
					<Card.Header>
						<div class="flex items-start justify-between gap-2">
							<Card.Title class="text-sm">{ws.name}</Card.Title>
							<Badge variant="secondary" class="shrink-0 text-xs">{m.status_active()}</Badge>
						</div>
					</Card.Header>
					<Card.Content class="pb-3">
						<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
							<Users class="size-3.5" />
							<span>{m.ws_manager_id({ id: ws.managerId })}</span>
						</div>
						<p class="mt-1 text-xs text-muted-foreground">
							{m.ws_created_date({ date: formatTimestamp(ws.createdAt) })}
						</p>
					</Card.Content>
					<Card.Footer class="mt-auto border-t pt-3">
						<Button size="sm" class="w-full gap-1.5" href={localizeHref(`/workspaces/${ws.id}`)}>
							{m.common_open()}
							<ArrowRight class="size-3.5" />
						</Button>
					</Card.Footer>
				</Card.Root>
			{/each}
		</div>

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
				<span class="text-sm text-muted-foreground"
					>{m.common_page_of({ current: String(currentPage), total: String(totalPages) })}</span
				>
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
