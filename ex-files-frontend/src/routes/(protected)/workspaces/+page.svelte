<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getWorkspaces, getMe, protoTsToDate } from '$lib/data.remote';
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { FolderOpen, Plus, ArrowRight, ChevronLeft, ChevronRight, Users } from '@lucide/svelte';

	const meQuery = getMe();
	const me = $derived(meQuery.current);

	const currentPage = Number(page.url.searchParams.get('page') ?? '1');
	const workspacesQuery = getWorkspaces(currentPage);
	const data = $derived(workspacesQuery.current);
	const workspaces = $derived(data?.workspaces ?? []);
	const totalPages = $derived(data?.totalPages ?? 1);

	let createOpen = $state(false);
	let createName = $state('');
	let creating = $state(false);
	let createError = $state('');

	function formatDate(ts?: { seconds: number }): string {
		const d = protoTsToDate(ts);
		if (!d) return '—';
		return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
	}

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
			const res = await fetch('/api/workspaces', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ name: createName.trim() })
			});
			if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				createError = err.error ?? 'Failed to create workspace';
				return;
			}
			const { workspace } = await res.json();
			createOpen = false;
			createName = '';
			goto(`/workspaces/${workspace.id}`);
		} catch {
			createError = 'Network error, please try again';
		} finally {
			creating = false;
		}
	}
</script>

<svelte:head>
	<title>Workspaces — ex-files</title>
</svelte:head>

<div class="flex flex-1 flex-col gap-6 p-6">
	<div class="flex items-start justify-between gap-4">
		<div>
			<h1 class="text-lg font-semibold">Workspaces</h1>
			<p class="text-sm text-muted-foreground">
				Organize related documents into workspaces for streamlined review.
			</p>
		</div>
		{#if me?.role === 'manager'}
			<Dialog.Root bind:open={createOpen}>
				<Dialog.Trigger>
					{#snippet child({ props })}
						<Button size="sm" class="gap-1.5" {...props}>
							<Plus class="size-4" />
							New Workspace
						</Button>
					{/snippet}
				</Dialog.Trigger>
				<Dialog.Content class="sm:max-w-md">
					<Dialog.Header>
						<Dialog.Title>Create Workspace</Dialog.Title>
						<Dialog.Description>Give your workspace a clear, descriptive name.</Dialog.Description>
					</Dialog.Header>
					<div class="grid gap-4 py-4">
						<div class="grid gap-2">
							<Label for="ws-name">Name</Label>
							<Input
								id="ws-name"
								placeholder="e.g. Q2 2026 Contracts"
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
								<Button variant="outline" {...props}>Cancel</Button>
							{/snippet}
						</Dialog.Close>
						<Button onclick={handleCreate} disabled={creating || !createName.trim()}>
							{creating ? 'Creating…' : 'Create'}
						</Button>
					</Dialog.Footer>
				</Dialog.Content>
			</Dialog.Root>
		{/if}
	</div>

	{#if workspaces.length === 0}
		<Card.Root class="flex flex-col items-center justify-center py-16 text-center">
			<Card.Content>
				<FolderOpen class="mx-auto mb-3 size-10 text-muted-foreground/40" />
				<p class="text-sm font-medium">No workspaces yet</p>
				<p class="mt-1 text-xs text-muted-foreground">
					{me?.role === 'manager'
						? 'Create a workspace to get started.'
						: 'You have not been added to any workspaces.'}
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
							<Badge variant="secondary" class="shrink-0 text-xs">Active</Badge>
						</div>
					</Card.Header>
					<Card.Content class="pb-3">
						<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
							<Users class="size-3.5" />
							<span>Manager ID: {ws.managerId}</span>
						</div>
						<p class="mt-1 text-xs text-muted-foreground">Created {formatDate(ws.createdAt)}</p>
					</Card.Content>
					<Card.Footer class="mt-auto border-t pt-3">
						<Button size="sm" class="w-full gap-1.5" href="/workspaces/{ws.id}">
							Open
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
					Prev
				</Button>
				<span class="text-sm text-muted-foreground">Page {currentPage} of {totalPages}</span>
				<Button
					variant="outline"
					size="sm"
					class="gap-1"
					disabled={currentPage >= totalPages}
					onclick={() => navigatePage(currentPage + 1)}
				>
					Next
					<ChevronRight class="size-4" />
				</Button>
			</div>
		{/if}
	{/if}
</div>
