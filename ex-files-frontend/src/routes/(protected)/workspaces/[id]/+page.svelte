<script lang="ts">
	import { page } from '$app/state';
	import { goto, invalidateAll } from '$app/navigation';
	import { getWorkspaceDetail, getMe, getSystemUsers, getDocuments } from '$lib/data.remote';
	import { protoTsToDate, roleName } from '$lib/proto-utils';
	import {
		updateWorkspace,
		deleteWorkspace,
		addWorkspaceMember,
		removeWorkspaceMember
	} from '$lib/commands.remote';
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as Avatar from '$lib/components/ui/avatar/index.js';
	import * as Tabs from '$lib/components/ui/tabs/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import {
		Pencil,
		Trash2,
		UserPlus,
		UserMinus,
		Calendar,
		Crown,
		FileText,
		Search,
		ChevronLeft,
		ChevronRight,
		ArrowRight
	} from '@lucide/svelte';
	import type { Timestamp } from '@bufbuild/protobuf/wkt';
	import { extraBreadcrumbs } from '$lib/stores/breadcrumbs';
	import { onDestroy } from 'svelte';

	const wsId = page.params.id ?? '';

	// Set breadcrumb when workspace name is available
	$effect(() => {
		const name = ws?.name;
		if (name) {
			extraBreadcrumbs.set([{ label: name }]);
		}
	});
	onDestroy(() => extraBreadcrumbs.set([]));

	const meQuery = getMe();
	const me = $derived(meQuery.current?.user);

	const detailQuery = getWorkspaceDetail(wsId);
	const detail = $derived(detailQuery.current);
	const ws = $derived(detail?.workspace);
	const manager = $derived(detail?.manager);
	const members = $derived(detail?.members ?? []);

	// Documents — encode all params into a single query string for the unchecked query
	const docPage = Number(page.url.searchParams.get('doc_page') ?? '1');
	const docSearch = page.url.searchParams.get('doc_search') ?? '';
	const docStatus = page.url.searchParams.get('doc_status') ?? '';
	function buildQS(parts: Record<string, string>): string {
		return Object.entries(parts)
			.map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
			.join('&');
	}
	const docParts: Record<string, string> = { page: String(docPage) };
	if (docSearch) docParts.search = docSearch;
	if (docStatus) docParts.status = docStatus;
	const documentsQuery = getDocuments(`${wsId}?${buildQS(docParts)}`);
	const docsData = $derived(documentsQuery.current);
	const documents = $derived(docsData?.documents ?? []);
	const docTotalPages = $derived(docsData?.totalPages ?? 1);

	// Users (for add-member picker)
	const usersQuery = getSystemUsers();
	const allUsers = $derived(usersQuery.current ?? []);
	const memberIds = $derived(new Set(members.map((m) => String(m.id))));
	const nonMembers = $derived(allUsers.filter((u) => !memberIds.has(String(u.id))));

	const isOwner = $derived(manager && me && manager.id === me.id);

	// Edit dialog
	let editOpen = $state(false);
	let editName = $state('');
	let editing = $state(false);
	let editError = $state('');

	// Delete dialog
	let deleteOpen = $state(false);
	let deleting = $state(false);

	// Add member dialog
	let addOpen = $state(false);
	let memberSearch = $state('');
	let addingId = $state<string | null>(null);
	let addError = $state('');

	const filteredNonMembers = $derived(
		nonMembers.filter(
			(u) =>
				!memberSearch ||
				u.name.toLowerCase().includes(memberSearch.toLowerCase()) ||
				u.email.toLowerCase().includes(memberSearch.toLowerCase())
		)
	);

	// Document search state (local – triggers URL navigation on submit)
	let searchInput = $state(docSearch);
	let uploading = $state(false);
	let uploadError = $state('');

	function formatDate(ts?: Timestamp): string {
		const d = protoTsToDate(ts);
		if (!d) return '—';
		return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
	}

	function formatSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

	function statusVariant(status: string): string {
		switch (status) {
			case 'approved':
				return 'bg-emerald-100 text-emerald-700';
			case 'rejected':
				return 'bg-red-100 text-red-700';
			case 'in_review':
				return 'bg-blue-100 text-blue-700';
			case 'changes_requested':
				return 'bg-amber-100 text-amber-700';
			default:
				return 'bg-muted text-muted-foreground';
		}
	}

	function statusLabel(status: string): string {
		switch (status) {
			case 'in_review':
				return 'In Review';
			case 'changes_requested':
				return 'Changes Requested';
			default:
				return status.charAt(0).toUpperCase() + status.slice(1);
		}
	}

	function initials(name: string): string {
		return name
			.split(' ')
			.map((p) => p[0])
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}

	function navigateDocPage(p: number) {
		const url = new URL(page.url);
		url.searchParams.set('doc_page', String(p));
		goto(url.toString());
	}

	function applyDocSearch() {
		const url = new URL(page.url);
		url.searchParams.set('doc_page', '1');
		if (searchInput.trim()) {
			url.searchParams.set('doc_search', searchInput.trim());
		} else {
			url.searchParams.delete('doc_search');
		}
		goto(url.toString());
	}

	function applyStatusFilter(status: string) {
		const url = new URL(page.url);
		url.searchParams.set('doc_page', '1');
		if (status) {
			url.searchParams.set('doc_status', status);
		} else {
			url.searchParams.delete('doc_status');
		}
		goto(url.toString());
	}

	function openEdit() {
		editName = ws?.name ?? '';
		editError = '';
		editOpen = true;
	}

	async function handleEdit() {
		if (!editName.trim()) return;
		editing = true;
		editError = '';
		try {
			const result = await updateWorkspace({ id: wsId, name: editName.trim() });
			if (!result.ok) {
				editError = result.error ?? 'Failed to update workspace';
				return;
			}
			editOpen = false;
			await invalidateAll();
		} catch {
			editError = 'Network error, please try again';
		} finally {
			editing = false;
		}
	}

	async function handleDelete() {
		deleting = true;
		try {
			const result = await deleteWorkspace(wsId);
			if (!result.ok) return;
			goto('/workspaces');
		} catch {
			// ignore
		} finally {
			deleting = false;
			deleteOpen = false;
		}
	}

	async function handleAddMember(userId: string) {
		addingId = userId;
		addError = '';
		try {
			const result = await addWorkspaceMember({ workspaceId: wsId, userId: Number(userId) });
			if (!result.ok) {
				addError = result.error ?? 'Failed to add member';
				return;
			}
			await invalidateAll();
			memberSearch = '';
		} catch {
			addError = 'Network error, please try again';
		} finally {
			addingId = null;
		}
	}

	async function handleRemoveMember(userId: bigint) {
		try {
			const result = await removeWorkspaceMember({ workspaceId: wsId, userId });
			if (!result.ok) return;
			await invalidateAll();
		} catch {
			// ignore
		}
	}
</script>

<svelte:head>
	<title>{ws?.name ?? 'Workspace'} — ex-files</title>
</svelte:head>

<div class="flex flex-1 flex-col gap-6 p-6">
	{#if !detail}
		<!-- Loading skeleton -->
		<Card.Root>
			<Card.Header>
				<div class="flex items-start gap-3">
					<div class="min-w-0 flex-1 space-y-2">
						<Skeleton class="h-5 w-48" />
						<div class="flex gap-3">
							<Skeleton class="h-4 w-32" />
							<Skeleton class="h-4 w-36" />
						</div>
					</div>
				</div>
			</Card.Header>
		</Card.Root>
	{:else}
		<!-- Workspace info card -->
		<Card.Root>
			<Card.Header>
				<div class="flex items-start gap-3">
					<div class="min-w-0 flex-1">
						<Card.Title class="text-base">{ws?.name}</Card.Title>
						<Card.Description class="mt-1 flex flex-wrap gap-3 text-xs">
							<span class="flex items-center gap-1">
								<Crown class="size-3.5" />
								Manager: {manager?.name ?? '—'}
							</span>
							<span class="flex items-center gap-1">
								<Calendar class="size-3.5" />
								Created {formatDate(ws?.createdAt)}
							</span>
						</Card.Description>
					</div>
					{#if isOwner}
						<div class="flex shrink-0 gap-2">
							<Button variant="outline" size="sm" class="gap-1.5" onclick={openEdit}>
								<Pencil class="size-3.5" />
								Edit
							</Button>
							<Button
								variant="outline"
								size="sm"
								class="gap-1.5 text-destructive hover:text-destructive"
								onclick={() => (deleteOpen = true)}
							>
								<Trash2 class="size-3.5" />
								Delete
							</Button>
						</div>
					{/if}
				</div>
			</Card.Header>
		</Card.Root>

		<!-- Tabbed content: Documents / Members -->
		<Tabs.Root value="documents">
			<Tabs.List>
				<Tabs.Trigger value="documents">
					<FileText class="mr-1.5 size-3.5" />
					Documents
					{#if docsData && docsData.total > 0}
						<span class="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-semibold">
							{docsData.total}
						</span>
					{/if}
				</Tabs.Trigger>
				<Tabs.Trigger value="members">
					<UserPlus class="mr-1.5 size-3.5" />
					Members
					{#if members.length > 0}
						<span class="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-semibold">
							{members.length}
						</span>
					{/if}
				</Tabs.Trigger>
			</Tabs.List>

			<!-- Documents tab -->
			<Tabs.Content value="documents" class="mt-4 flex flex-col gap-4">
				<!-- Search + filter -->
				<div class="flex flex-wrap items-center gap-2">
					<form
						class="flex flex-1 items-center gap-2"
						onsubmit={(e) => {
							e.preventDefault();
							applyDocSearch();
						}}
					>
						<div class="relative min-w-48 flex-1">
							<Search
								class="absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground"
							/>
							<Input placeholder="Search documents…" class="pl-8" bind:value={searchInput} />
						</div>
						<Button type="submit" size="sm" variant="secondary">Search</Button>
						{#if docSearch}
							<Button
								type="button"
								size="sm"
								variant="ghost"
								onclick={() => {
									searchInput = '';
									applyDocSearch();
								}}
							>
								Clear
							</Button>
						{/if}
					</form>
					<!-- Status filter -->
					<div class="flex gap-1">
						{#each [['', 'All'], ['pending', 'Pending'], ['in_review', 'In Review'], ['approved', 'Approved'], ['rejected', 'Rejected']] as [val, label] (val)}
							<Button
								variant={docStatus === val ? 'default' : 'outline'}
								size="sm"
								class="h-8 text-xs"
								onclick={() => applyStatusFilter(val)}
							>
								{label}
							</Button>
						{/each}
					</div>
				</div>

				<!-- Document list -->
				{#if documents.length === 0}
					<Card.Root class="flex flex-col items-center justify-center py-12 text-center">
						<Card.Content>
							<FileText class="mx-auto mb-3 size-8 text-muted-foreground/40" />
							<p class="text-sm font-medium">No documents yet</p>
							<p class="mt-1 text-xs text-muted-foreground">Upload a PDF to get started.</p>
						</Card.Content>
					</Card.Root>
				{:else}
					<div class="flex flex-col gap-2">
						{#each documents as doc (doc.id)}
							<Card.Root class="transition-shadow hover:shadow-sm">
								<Card.Content class="flex items-center gap-3 py-3">
									<FileText class="size-8 shrink-0 text-muted-foreground/60" />
									<div class="min-w-0 flex-1">
										<p class="truncate text-sm font-medium">{doc.name}</p>
										<div
											class="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-muted-foreground"
										>
											<span>{formatSize(Number(doc.size))}</span>
											<span>·</span>
											<span>{doc.uploaderName}</span>
											<span>·</span>
											<span>{formatDate(doc.createdAt)}</span>
										</div>
									</div>
									<Badge variant="secondary" class="shrink-0 text-xs {statusVariant(doc.status)}">
										{statusLabel(doc.status)}
									</Badge>
									<Button
										variant="ghost"
										size="sm"
										class="shrink-0 gap-1"
										href="/workspaces/{wsId}/documents/{doc.id}"
									>
										View
										<ArrowRight class="size-3.5" />
									</Button>
								</Card.Content>
							</Card.Root>
						{/each}
					</div>

					<!-- Upload zone (below documents) -->
					{#if uploading}
						<p class="text-center text-sm text-muted-foreground">Uploading…</p>
					{/if}
					{#if uploadError}
						<p class="text-center text-sm text-destructive">{uploadError}</p>
					{/if}

					{#if docTotalPages > 1}
						<div class="flex items-center justify-center gap-2">
							<Button
								variant="outline"
								size="sm"
								class="gap-1"
								disabled={docPage <= 1}
								onclick={() => navigateDocPage(docPage - 1)}
							>
								<ChevronLeft class="size-4" />
								Prev
							</Button>
							<span class="text-sm text-muted-foreground">
								Page {docPage} of {docTotalPages}
							</span>
							<Button
								variant="outline"
								size="sm"
								class="gap-1"
								disabled={docPage >= docTotalPages}
								onclick={() => navigateDocPage(docPage + 1)}
							>
								Next
								<ChevronRight class="size-4" />
							</Button>
						</div>
					{/if}
				{/if}
			</Tabs.Content>

			<!-- Members tab -->
			<Tabs.Content value="members" class="mt-4">
				<Card.Root>
					<Card.Header>
						<div class="flex items-center justify-between gap-2">
							<div>
								<Card.Title class="text-sm">Members</Card.Title>
								<Card.Description class="text-xs">
									{members.length}
									{members.length === 1 ? 'member' : 'members'}
								</Card.Description>
							</div>
							{#if isOwner}
								<Dialog.Root bind:open={addOpen}>
									<Dialog.Trigger>
										{#snippet child({ props })}
											<Button size="sm" class="gap-1.5" {...props}>
												<UserPlus class="size-4" />
												Add Member
											</Button>
										{/snippet}
									</Dialog.Trigger>
									<Dialog.Content class="sm:max-w-md">
										<Dialog.Header>
											<Dialog.Title>Add Member</Dialog.Title>
											<Dialog.Description>
												Search for a user to add to this workspace.
											</Dialog.Description>
										</Dialog.Header>
										<div class="grid gap-3 py-4">
											<div class="grid gap-2">
												<Label for="member-search">Search</Label>
												<Input
													id="member-search"
													placeholder="Name or email…"
													bind:value={memberSearch}
												/>
											</div>
											{#if addError}
												<p class="text-sm text-destructive">{addError}</p>
											{/if}
											<div class="max-h-60 overflow-y-auto rounded-md border">
												{#if filteredNonMembers.length === 0}
													<p class="p-4 text-center text-xs text-muted-foreground">
														{nonMembers.length === 0
															? 'All users are already members.'
															: 'No matches.'}
													</p>
												{:else}
													{#each filteredNonMembers as u (u.id)}
														<button
															class="flex w-full items-center gap-3 px-3 py-2 text-left text-sm transition-colors hover:bg-muted disabled:opacity-50"
															disabled={addingId === String(u.id)}
															onclick={() => handleAddMember(String(u.id))}
														>
															<Avatar.Root class="h-7 w-7 shrink-0">
																<Avatar.Fallback
																	class="bg-primary text-[10px] font-semibold text-primary-foreground"
																>
																	{initials(u.name)}
																</Avatar.Fallback>
															</Avatar.Root>
															<div class="min-w-0 flex-1">
																<p class="truncate font-medium">{u.name}</p>
																<p class="truncate text-xs text-muted-foreground">{u.email}</p>
															</div>
															{#if addingId === String(u.id)}
																<span class="text-xs text-muted-foreground">Adding…</span>
															{/if}
														</button>
													{/each}
												{/if}
											</div>
										</div>
										<Dialog.Footer>
											<Dialog.Close>
												{#snippet child({ props })}
													<Button variant="outline" {...props}>Close</Button>
												{/snippet}
											</Dialog.Close>
										</Dialog.Footer>
									</Dialog.Content>
								</Dialog.Root>
							{/if}
						</div>
					</Card.Header>
					<Card.Content>
						{#if members.length === 0}
							<p class="py-4 text-center text-sm text-muted-foreground">
								No members yet.{isOwner ? ' Use "Add Member" to invite someone.' : ''}
							</p>
						{:else}
							<ul class="divide-y">
								{#each members as member (member.id)}
									{@const role = roleName(member.role)}
									<li class="flex items-center gap-3 py-2.5 first:pt-0 last:pb-0">
										<Avatar.Root class="h-8 w-8 shrink-0">
											<Avatar.Fallback
												class="bg-primary text-xs font-semibold text-primary-foreground"
											>
												{initials(member.name)}
											</Avatar.Fallback>
										</Avatar.Root>
										<div class="min-w-0 flex-1">
											<p class="truncate text-sm font-medium">{member.name}</p>
											<p class="truncate text-xs text-muted-foreground">{member.email}</p>
										</div>
										{#if role === 'manager'}
											<Badge variant="secondary" class="shrink-0 text-[10px] text-violet-700">
												Manager
											</Badge>
										{:else}
											<Badge variant="outline" class="shrink-0 text-[10px] text-muted-foreground">
												Employee
											</Badge>
										{/if}
										{#if isOwner}
											<Button
												variant="ghost"
												size="sm"
												class="h-7 w-7 shrink-0 p-0 text-muted-foreground hover:text-destructive"
												onclick={() => handleRemoveMember(member.id)}
												title="Remove member"
											>
												<UserMinus class="size-4" />
											</Button>
										{/if}
									</li>
								{/each}
							</ul>
						{/if}
					</Card.Content>
				</Card.Root>
			</Tabs.Content>
		</Tabs.Root>
	{/if}
</div>

<!-- Edit dialog -->
<Dialog.Root bind:open={editOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Edit Workspace</Dialog.Title>
		</Dialog.Header>
		<div class="grid gap-4 py-4">
			<div class="grid gap-2">
				<Label for="edit-name">Name</Label>
				<Input
					id="edit-name"
					bind:value={editName}
					onkeydown={(e) => e.key === 'Enter' && handleEdit()}
				/>
			</div>
			{#if editError}
				<p class="text-sm text-destructive">{editError}</p>
			{/if}
		</div>
		<Dialog.Footer>
			<Dialog.Close>
				{#snippet child({ props })}
					<Button variant="outline" {...props}>Cancel</Button>
				{/snippet}
			</Dialog.Close>
			<Button onclick={handleEdit} disabled={editing || !editName.trim()}>
				{editing ? 'Saving…' : 'Save'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- Delete confirmation dialog -->
<Dialog.Root bind:open={deleteOpen}>
	<Dialog.Content class="sm:max-w-sm">
		<Dialog.Header>
			<Dialog.Title>Delete Workspace</Dialog.Title>
			<Dialog.Description>
				Are you sure you want to delete <strong>{ws?.name}</strong>? This action cannot be undone.
			</Dialog.Description>
		</Dialog.Header>
		<Dialog.Footer>
			<Dialog.Close>
				{#snippet child({ props })}
					<Button variant="outline" {...props}>Cancel</Button>
				{/snippet}
			</Dialog.Close>
			<Button variant="destructive" onclick={handleDelete} disabled={deleting}>
				{deleting ? 'Deleting…' : 'Delete'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
