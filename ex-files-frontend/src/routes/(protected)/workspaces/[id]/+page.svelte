<script lang="ts">
	import { page } from '$app/state';
	import { goto, invalidateAll } from '$app/navigation';
	import {
		getWorkspaceDetail,
		getAssignableMembers,
		getIssues
	} from '$lib/queries.remote';
	import { formatTimestamp, roleName, isManager, initials } from '$lib/utils';
	import {
		updateWorkspace,
		deleteWorkspace,
		addWorkspaceMember,
		removeWorkspaceMember,
		createIssue
	} from '$lib/commands.remote';
	import { toast } from 'svelte-sonner';
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as Avatar from '$lib/components/ui/avatar/index.js';
	import * as Tabs from '$lib/components/ui/tabs/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import {
		Pencil,
		Trash2,
		UserPlus,
		UserMinus,
		Calendar,
		Crown,
		FileText,
		ArrowRight,
		Plus
	} from '@lucide/svelte';
	import { extraBreadcrumbs } from '$lib/stores/breadcrumbs.svelte';
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

    const { data } = $props();
	const me = $derived(data.user);

	const detailQuery = getWorkspaceDetail(wsId);
	const detail = $derived(detailQuery.current);
	const ws = $derived(detail?.workspace);
	const manager = $derived(detail?.manager);
	const members = $derived(detail?.members ?? []);

	// Issues
	const issuesQuery = getIssues(wsId);
	const issuesList = $derived(issuesQuery.current ?? []);

	// Users assignable as members (employees not already in workspace)
	const assignableQuery = getAssignableMembers(wsId);
	const nonMembers = $derived(assignableQuery.current ?? []);

	const isOwner = $derived(manager && me && manager.id === me.id);

	// Edit dialog
	let editOpen = $state(false);
	let editName = $state('');
	let editing = $state(false);
	let editError = $state('');

	// Delete dialog
	let deleteOpen = $state(false);
	let deleting = $state(false);

	// New issue dialog
	let newIssueOpen = $state(false);
	let newIssueTitle = $state('');
	let newIssueDesc = $state('');
	let newIssueAssignee = $state('');
	let newIssueDeadline = $state('');
	let creatingIssue = $state(false);
	let newIssueError = $state('');

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
				editError = result.error ?? m.ws_update_error();
				return;
			}
			editOpen = false;
			await invalidateAll();
		} catch {
			editError = m.error_network_retry();
		} finally {
			editing = false;
		}
	}

	async function handleDelete() {
		deleting = true;
		try {
			const result = await deleteWorkspace(wsId);
			if (!result.ok) {
				toast.error(m.error_delete_workspace());
				return;
			}
			goto(localizeHref('/workspaces'));
		} catch {
			toast.error(m.error_delete_workspace());
		} finally {
			deleting = false;
			deleteOpen = false;
		}
	}

	async function handleAddMember(userId: string) {
		addingId = userId;
		addError = '';
		try {
			const result = await addWorkspaceMember({ workspaceId: wsId, userId });
			if (!result.ok) {
				addError = result.error ?? m.ws_add_member_error();
				return;
			}
			await invalidateAll();
			memberSearch = '';
		} catch {
			addError = m.error_network_retry();
		} finally {
			addingId = null;
		}
	}

	async function handleCreateIssue() {
		if (!newIssueTitle.trim() || !newIssueAssignee) return;
		creatingIssue = true;
		newIssueError = '';
		try {
			const result = await createIssue({
				workspaceId: wsId,
				title: newIssueTitle.trim(),
				description: newIssueDesc.trim() || undefined,
				assigneeId: newIssueAssignee,
				deadline: newIssueDeadline || undefined
			});
			if (!result.ok) {
				newIssueError = result.error ?? m.ws_issue_create_error();
				return;
			}
			newIssueOpen = false;
			newIssueTitle = '';
			newIssueDesc = '';
			newIssueAssignee = '';
			newIssueDeadline = '';
			await invalidateAll();
		} catch {
			newIssueError = m.error_network_retry();
		} finally {
			creatingIssue = false;
		}
	}

	async function handleRemoveMember(userId: string) {
		try {
			const result = await removeWorkspaceMember({ workspaceId: wsId, userId });
			if (!result.ok) {
				toast.error(m.ws_remove_member_error());
				return;
			}
			await invalidateAll();
		} catch {
			toast.error(m.ws_remove_member_error());
		}
	}
</script>

<svelte:head>
	<title>{ws?.name ?? m.nav_workspaces()} - ex-files</title>
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
								{m.ws_manager_label({ name: manager?.name ?? '-' })}
							</span>
							<span class="flex items-center gap-1">
								<Calendar class="size-3.5" />
								{m.ws_created_date({ date: formatTimestamp(ws?.createdAt) })}
							</span>
						</Card.Description>
					</div>
					{#if isOwner}
						<div class="flex shrink-0 gap-2">
							<Button variant="outline" size="sm" class="gap-1.5" onclick={openEdit}>
								<Pencil class="size-3.5" />
								{m.common_edit()}
							</Button>
							<Button
								variant="outline"
								size="sm"
								class="gap-1.5 text-destructive hover:text-destructive"
								onclick={() => (deleteOpen = true)}
							>
								<Trash2 class="size-3.5" />
								{m.common_delete()}
							</Button>
						</div>
					{/if}
				</div>
			</Card.Header>
		</Card.Root>

		<!-- Tabbed content: Issues / Members -->
		<Tabs.Root value="issues">
			<div class="flex items-center justify-between">
				<Tabs.List>
					<Tabs.Trigger value="issues">
						<FileText class="mr-1.5 size-3.5" />
						{m.ws_issues_tab()}
						{#if issuesList.length > 0}
							<span class="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-semibold">
								{issuesList.length}
							</span>
						{/if}
					</Tabs.Trigger>
					<Tabs.Trigger value="members">
						<UserPlus class="mr-1.5 size-3.5" />
						{m.ws_members_tab()}
						{#if members.length > 0}
							<span class="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-semibold">
								{members.length}
							</span>
						{/if}
					</Tabs.Trigger>
				</Tabs.List>
				{#if isManager(me?.role)}
					<Button size="sm" class="gap-1.5" onclick={() => (newIssueOpen = true)}>
						<Plus class="size-4" />
						{m.ws_new_issue()}
					</Button>
				{/if}
			</div>

			<!-- Issues tab -->
			<Tabs.Content value="issues" class="mt-4 flex flex-col gap-4">
				{#if issuesList.length === 0}
					<Card.Root class="flex flex-col items-center justify-center py-12 text-center">
						<Card.Content>
							<FileText class="mx-auto mb-3 size-8 text-muted-foreground/40" />
							<p class="text-sm font-medium">{m.ws_no_issues()}</p>
							<p class="mt-1 text-xs text-muted-foreground">{m.ws_no_issues_hint()}</p>
						</Card.Content>
					</Card.Root>
				{:else}
					<div class="flex flex-col gap-2">
						{#each issuesList as issue (issue.id)}
							<Card.Root class="transition-shadow hover:shadow-sm">
								<Card.Content class="flex items-center gap-3 py-3">
									<FileText class="size-8 shrink-0 text-muted-foreground/60" />
									<div class="min-w-0 flex-1">
										<p class="truncate text-sm font-medium">{issue.title}</p>
										{#if issue.description}
											<p class="mt-0.5 truncate text-xs text-muted-foreground">
												{issue.description}
											</p>
										{/if}
									</div>
									<Badge
										variant="secondary"
										class="shrink-0 text-xs {issue.resolved
											? 'bg-emerald-100 text-emerald-700'
											: 'bg-blue-100 text-blue-700'}"
									>
										{issue.resolved ? m.issue_resolved() : m.issue_open()}
									</Badge>
									<Button
										variant="ghost"
										size="sm"
										class="shrink-0 gap-1"
										href={localizeHref(`/workspaces/${wsId}/issues/${issue.id}`)}
									>
										{m.common_view()}
										<ArrowRight class="size-3.5" />
									</Button>
								</Card.Content>
							</Card.Root>
						{/each}
					</div>
				{/if}
			</Tabs.Content>

			<!-- Members tab -->
			<Tabs.Content value="members" class="mt-4">
				<Card.Root>
					<Card.Header>
						<div class="flex items-center justify-between gap-2">
							<div>
								<Card.Title class="text-sm">{m.ws_members_heading()}</Card.Title>
								<Card.Description class="text-xs">
									{members.length === 1
										? m.ws_member_count({ count: String(members.length) })
										: m.ws_members_count({ count: String(members.length) })}
								</Card.Description>
							</div>
							{#if isOwner}
								<Dialog.Root bind:open={addOpen}>
									<Dialog.Trigger>
										{#snippet child({ props })}
											<Button size="sm" class="gap-1.5" {...props}>
												<UserPlus class="size-4" />
												{m.ws_add_member()}
											</Button>
										{/snippet}
									</Dialog.Trigger>
									<Dialog.Content class="sm:max-w-md">
										<Dialog.Header>
											<Dialog.Title>{m.ws_add_member_title()}</Dialog.Title>
											<Dialog.Description>
												{m.ws_add_member_description()}
											</Dialog.Description>
										</Dialog.Header>
										<div class="grid gap-3 py-4">
											<div class="grid gap-2">
												<Label for="member-search">{m.ws_member_search_label()}</Label>
												<Input
													id="member-search"
													placeholder={m.ws_member_search_placeholder()}
													bind:value={memberSearch}
												/>
											</div>
											{#if addError}
												<p class="text-sm text-destructive">{addError}</p>
											{/if}
											<div class="max-h-60 overflow-y-auto rounded-md border">
												{#if filteredNonMembers.length === 0}
													<p class="p-4 text-center text-xs text-muted-foreground">
														{nonMembers.length === 0 ? m.ws_all_members() : m.ws_no_matches()}
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
																<span class="text-xs text-muted-foreground">{m.ws_adding()}</span>
															{/if}
														</button>
													{/each}
												{/if}
											</div>
										</div>
										<Dialog.Footer>
											<Dialog.Close>
												{#snippet child({ props })}
													<Button variant="outline" {...props}>{m.common_close()}</Button>
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
								{m.ws_no_members()}{isOwner ? m.ws_no_members_hint() : ''}
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
												{m.role_manager()}
											</Badge>
										{:else}
											<Badge variant="outline" class="shrink-0 text-[10px] text-muted-foreground">
												{m.role_employee()}
											</Badge>
										{/if}
										{#if isOwner}
											<Button
												variant="ghost"
												size="sm"
												class="h-7 w-7 shrink-0 p-0 text-muted-foreground hover:text-destructive"
												onclick={() => handleRemoveMember(member.id)}
												title={m.ws_remove_member()}
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
			<Dialog.Title>{m.ws_edit_title()}</Dialog.Title>
		</Dialog.Header>
		<div class="grid gap-4 py-4">
			<div class="grid gap-2">
				<Label for="edit-name">{m.common_name()}</Label>
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
					<Button variant="outline" {...props}>{m.common_cancel()}</Button>
				{/snippet}
			</Dialog.Close>
			<Button onclick={handleEdit} disabled={editing || !editName.trim()}>
				{editing ? m.common_saving() : m.common_save()}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- New issue dialog -->
<Dialog.Root bind:open={newIssueOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>{m.ws_new_issue_title()}</Dialog.Title>
			<Dialog.Description>{m.ws_new_issue_description()}</Dialog.Description>
		</Dialog.Header>
		<div class="grid gap-4 py-4">
			<div class="grid gap-2">
				<Label for="issue-title">{m.ws_issue_title_label()}</Label>
				<Input
					id="issue-title"
					placeholder={m.ws_issue_title_placeholder()}
					bind:value={newIssueTitle}
				/>
			</div>
			<div class="grid gap-2">
				<Label for="issue-desc">{m.ws_issue_description_label()}</Label>
				<Textarea
					id="issue-desc"
					placeholder={m.ws_issue_description_placeholder()}
					bind:value={newIssueDesc}
					rows={3}
				/>
			</div>
			<div class="grid gap-2">
				<Label for="issue-assignee">{m.ws_issue_assignee_label()}</Label>
				<select
					id="issue-assignee"
					class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none"
					bind:value={newIssueAssignee}
				>
					<option value="" disabled>{m.ws_issue_assignee_select()}</option>
					{#each members as member (member.id)}
						<option value={String(member.id)}>{member.name}</option>
					{/each}
				</select>
			</div>
			<div class="grid gap-2">
				<Label for="issue-deadline">{m.ws_issue_deadline_label()}</Label>
				<Input id="issue-deadline" type="date" bind:value={newIssueDeadline} />
			</div>
			{#if newIssueError}
				<p class="text-sm text-destructive">{newIssueError}</p>
			{/if}
		</div>
		<Dialog.Footer>
			<Dialog.Close>
				{#snippet child({ props })}
					<Button variant="outline" {...props}>{m.common_cancel()}</Button>
				{/snippet}
			</Dialog.Close>
			<Button
				onclick={handleCreateIssue}
				disabled={creatingIssue || !newIssueTitle.trim() || !newIssueAssignee}
			>
				{creatingIssue ? m.common_creating() : m.common_create()}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- Delete confirmation dialog -->
<Dialog.Root bind:open={deleteOpen}>
	<Dialog.Content class="sm:max-w-sm">
		<Dialog.Header>
			<Dialog.Title>{m.ws_delete_title()}</Dialog.Title>
			<Dialog.Description>
				{m.ws_delete_confirm({ name: ws?.name ?? '' })}
			</Dialog.Description>
		</Dialog.Header>
		<Dialog.Footer>
			<Dialog.Close>
				{#snippet child({ props })}
					<Button variant="outline" {...props}>{m.common_cancel()}</Button>
				{/snippet}
			</Dialog.Close>
			<Button variant="destructive" onclick={handleDelete} disabled={deleting}>
				{deleting ? m.common_deleting() : m.common_delete()}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
