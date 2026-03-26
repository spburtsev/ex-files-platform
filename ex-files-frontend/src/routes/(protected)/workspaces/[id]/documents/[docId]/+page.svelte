<script lang="ts">
	import { page } from '$app/state';
	import { goto, invalidateAll } from '$app/navigation';
	import { getDocumentDetail, getMe, getWorkspaceDetail } from '$lib/data.remote';
	import { protoTsToDate, isManager, bid } from '$lib/proto-utils';
	import {
		submitDocument,
		resubmitDocument,
		approveDocument,
		rejectDocument,
		requestDocumentChanges,
		assignDocumentReviewer,
		getDocumentDownloadUrl,
		uploadDocumentVersion,
		deleteDocument
	} from '$lib/commands.remote';
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import UploadZone from '$lib/components/pdf/UploadZone.svelte';
	import {
		Download,
		Trash2,
		Clock,
		Hash,
		FileText,
		Upload,
		User,
		Calendar,
		CheckCircle,
		XCircle,
		MessageSquare,
		Send,
		RotateCcw,
		UserCheck
	} from '@lucide/svelte';
	import { extraBreadcrumbs } from '$lib/stores/breadcrumbs';
	import { onDestroy } from 'svelte';

	const wsId = page.params.id ?? '';
	const docId = page.params.docId ?? '';

	const detailQuery = getDocumentDetail(docId);
	const detail = $derived(detailQuery.current);
	const doc = $derived(detail?.document);
	const versions = $derived(detail?.versions ?? []);

	const meQuery = getMe();
	const me = $derived(meQuery.current?.user);

	const wsQuery = getWorkspaceDetail(wsId);
	const wsDetail = $derived(wsQuery.current);
	const wsName = $derived(wsDetail?.workspace?.name);
	const members = $derived(wsDetail?.members ?? []);

	// Set breadcrumbs: Workspaces > {wsName} > {docName}
	$effect(() => {
		const segments: { label: string; href?: string }[] = [];
		if (wsName) segments.push({ label: wsName, href: `/workspaces/${wsId}` });
		if (doc?.name) segments.push({ label: doc.name });
		if (segments.length > 0) extraBreadcrumbs.set(segments);
	});
	onDestroy(() => extraBreadcrumbs.set([]));

	// Permission flags
	const isUploaderFlag = $derived(doc && me ? bid(me.id) === bid(doc.uploaderId) : false);
	const isManagerFlag = $derived(isManager(me?.role));
	const isAssignedReviewer = $derived(doc && me ? bid(doc.reviewerId) === bid(me.id) : false);
	const canReview = $derived(isManagerFlag || isAssignedReviewer);

	// Which action buttons to show
	const showSubmit = $derived(isUploaderFlag && doc?.status === 'pending');
	const showResubmit = $derived(isUploaderFlag && doc?.status === 'changes_requested');
	const showReviewActions = $derived(canReview && doc?.status === 'in_review');
	const showAssignReviewer = $derived(isManagerFlag && !!doc);

	// Dialog / loading state
	let deleteOpen = $state(false);
	let deleting = $state(false);

	let uploadingVersion = $state(false);
	let uploadVersionError = $state('');
	let downloadingId = $state<bigint | null>(null);

	let rejectOpen = $state(false);
	let rejectNote = $state('');
	let rejecting = $state(false);

	let changesOpen = $state(false);
	let changesNote = $state('');
	let requestingChanges = $state(false);

	let assignReviewerOpen = $state(false);
	let selectedReviewerId = $state<bigint | null>(null);
	let assigningReviewer = $state(false);
	let assignReviewerError = $state('');

	let actionError = $state('');

	function formatDate(ts?: import('@bufbuild/protobuf/wkt').Timestamp): string {
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

	function formatSize(bytes: number | bigint): string {
		const b = Number(bytes);
		if (b < 1024) return `${b} B`;
		if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`;
		return `${(b / (1024 * 1024)).toFixed(1)} MB`;
	}

	function statusVariant(status: string): string {
		switch (status) {
			case 'approved':
				return 'bg-emerald-100 text-emerald-700 hover:bg-emerald-100';
			case 'rejected':
				return 'bg-red-100 text-red-700 hover:bg-red-100';
			case 'in_review':
				return 'bg-blue-100 text-blue-700 hover:bg-blue-100';
			case 'changes_requested':
				return 'bg-amber-100 text-amber-700 hover:bg-amber-100';
			default:
				return '';
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

	async function handleSubmit() {
		actionError = '';
		const result = await submitDocument(docId);
		if (!result.ok) {
			actionError = result.error ?? 'Action failed';
			return;
		}
		await invalidateAll();
	}

	async function handleResubmit() {
		actionError = '';
		const result = await resubmitDocument(docId);
		if (!result.ok) {
			actionError = result.error ?? 'Action failed';
			return;
		}
		await invalidateAll();
	}

	async function handleApprove() {
		actionError = '';
		const result = await approveDocument(docId);
		if (!result.ok) {
			actionError = result.error ?? 'Action failed';
			return;
		}
		await invalidateAll();
	}

	async function handleReject() {
		rejecting = true;
		actionError = '';
		try {
			const result = await rejectDocument({ id: docId, note: rejectNote });
			if (!result.ok) {
				actionError = result.error ?? 'Action failed';
				return;
			}
			rejectOpen = false;
			rejectNote = '';
			await invalidateAll();
		} finally {
			rejecting = false;
		}
	}

	async function handleRequestChanges() {
		requestingChanges = true;
		actionError = '';
		try {
			const result = await requestDocumentChanges({ id: docId, note: changesNote });
			if (!result.ok) {
				actionError = result.error ?? 'Action failed';
				return;
			}
			changesOpen = false;
			changesNote = '';
			await invalidateAll();
		} finally {
			requestingChanges = false;
		}
	}

	async function handleAssignReviewer() {
		if (!selectedReviewerId) return;
		assigningReviewer = true;
		assignReviewerError = '';
		try {
			const result = await assignDocumentReviewer({
				id: docId,
				reviewerId: Number(selectedReviewerId)
			});
			if (!result.ok) {
				assignReviewerError = result.error ?? 'Failed to assign reviewer';
				return;
			}
			assignReviewerOpen = false;
			selectedReviewerId = null;
			await invalidateAll();
		} finally {
			assigningReviewer = false;
		}
	}

	async function handleDownload(versionId: bigint) {
		downloadingId = versionId;
		try {
			const { url } = await getDocumentDownloadUrl({ docId, versionId: Number(versionId) });
			if (!url) return;
			window.open(url, '_blank');
		} catch {
			// ignore
		} finally {
			downloadingId = null;
		}
	}

	async function handleUploadVersion(file: File) {
		uploadingVersion = true;
		uploadVersionError = '';
		try {
			const result = await uploadDocumentVersion({ docId, file });
			if (!result.ok) {
				uploadVersionError = result.error ?? 'Upload failed';
				return;
			}
			await invalidateAll();
		} catch {
			uploadVersionError = 'Network error, please try again';
		} finally {
			uploadingVersion = false;
		}
	}

	async function handleDelete() {
		deleting = true;
		try {
			const result = await deleteDocument(docId);
			if (!result.ok) return;
			goto(`/workspaces/${wsId}`);
		} catch {
			// ignore
		} finally {
			deleting = false;
			deleteOpen = false;
		}
	}
</script>

<svelte:head>
	<title>{doc?.name ?? 'Document'} — ex-files</title>
</svelte:head>

<div class="flex flex-1 flex-col gap-6 p-6">
	{#if !detail}
		<Card.Root class="flex items-center justify-center py-16">
			<Card.Content>
				<p class="text-sm text-muted-foreground">Loading document…</p>
			</Card.Content>
		</Card.Root>
	{:else}
		<!-- Document metadata card -->
		<Card.Root>
			<Card.Header>
				<div class="flex items-start justify-between gap-3">
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-2">
							<FileText class="size-5 shrink-0 text-muted-foreground" />
							<Card.Title class="truncate text-base">{doc?.name}</Card.Title>
						</div>
						<div class="mt-2 flex flex-wrap gap-4 text-xs text-muted-foreground">
							<span class="flex items-center gap-1">
								<User class="size-3.5" />
								{doc?.uploaderName}
							</span>
							<span class="flex items-center gap-1">
								<Calendar class="size-3.5" />
								{formatDate(doc?.createdAt)}
							</span>
							<span class="flex items-center gap-1">
								<FileText class="size-3.5" />
								{doc ? formatSize(doc.size) : '—'} · {doc?.mimeType}
							</span>
							{#if doc?.reviewerName}
								<span class="flex items-center gap-1">
									<UserCheck class="size-3.5" />
									Reviewer: {doc.reviewerName}
								</span>
							{/if}
						</div>
					</div>
					<div class="flex shrink-0 items-center gap-2">
						<Badge variant="secondary" class={`${statusVariant(doc?.status ?? '')}`}>
							{statusLabel(doc?.status ?? 'pending')}
						</Badge>
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
				</div>
			</Card.Header>

			<!-- Reviewer note callout -->
			{#if doc?.reviewerNote && (doc.status === 'rejected' || doc.status === 'changes_requested')}
				<Card.Content class="border-t pt-4">
					<div
						class="rounded-md border p-3 text-sm {doc.status === 'rejected'
							? 'border-red-200 bg-red-50 text-red-800'
							: 'border-amber-200 bg-amber-50 text-amber-800'}"
					>
						<p class="mb-1 font-medium">
							{doc.status === 'rejected' ? 'Rejection reason' : 'Changes requested'}
						</p>
						<p class="text-xs leading-relaxed">{doc.reviewerNote}</p>
					</div>
				</Card.Content>
			{/if}

			<Card.Content class="border-t pt-4">
				<div class="flex items-start gap-2 text-xs">
					<Hash class="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
					<div class="min-w-0">
						<p class="font-medium text-muted-foreground">SHA-256</p>
						<p class="mt-0.5 font-mono text-[11px] break-all">{doc?.hash}</p>
					</div>
				</div>
			</Card.Content>
		</Card.Root>

		<!-- Workflow actions card -->
		{#if showSubmit || showResubmit || showReviewActions || showAssignReviewer}
			<Card.Root>
				<Card.Header class="pb-3">
					<Card.Title class="text-sm">Review Workflow</Card.Title>
				</Card.Header>
				<Card.Content class="flex flex-wrap gap-2">
					{#if showSubmit}
						<Button size="sm" class="gap-1.5" onclick={handleSubmit}>
							<Send class="size-3.5" />
							Submit for Review
						</Button>
					{/if}

					{#if showResubmit}
						<Button size="sm" class="gap-1.5" onclick={handleResubmit}>
							<RotateCcw class="size-3.5" />
							Resubmit
						</Button>
					{/if}

					{#if showReviewActions}
						<Button
							size="sm"
							class="gap-1.5 bg-emerald-600 text-white hover:bg-emerald-700"
							onclick={handleApprove}
						>
							<CheckCircle class="size-3.5" />
							Approve
						</Button>
						<Button
							variant="outline"
							size="sm"
							class="gap-1.5 text-amber-700 hover:text-amber-700"
							onclick={() => (changesOpen = true)}
						>
							<MessageSquare class="size-3.5" />
							Request Changes
						</Button>
						<Button
							variant="outline"
							size="sm"
							class="gap-1.5 text-destructive hover:text-destructive"
							onclick={() => (rejectOpen = true)}
						>
							<XCircle class="size-3.5" />
							Reject
						</Button>
					{/if}

					{#if showAssignReviewer}
						<Button
							variant="outline"
							size="sm"
							class="ml-auto gap-1.5"
							onclick={() => (assignReviewerOpen = true)}
						>
							<UserCheck class="size-3.5" />
							{doc?.reviewerName ? `Reviewer: ${doc.reviewerName}` : 'Assign Reviewer'}
						</Button>
					{/if}

					{#if actionError}
						<p class="w-full text-xs text-destructive">{actionError}</p>
					{/if}
				</Card.Content>
			</Card.Root>
		{/if}

		<!-- Version history -->
		<Card.Root>
			<Card.Header>
				<Card.Title class="text-sm">Version History</Card.Title>
				<Card.Description class="text-xs">
					{versions.length} version{versions.length !== 1 ? 's' : ''}
				</Card.Description>
			</Card.Header>
			<Card.Content>
				{#if versions.length === 0}
					<p class="py-2 text-sm text-muted-foreground">No versions found.</p>
				{:else}
					<ol class="relative border-l border-border">
						{#each [...versions].sort((a, b) => b.version - a.version) as v (v.id)}
							<li class="mb-6 ml-4 last:mb-0">
								<div
									class="absolute -left-1.5 mt-1 h-3 w-3 rounded-full border border-background bg-primary"
								></div>
								<div class="flex items-start justify-between gap-3">
									<div class="min-w-0">
										<p class="text-sm font-semibold">Version {v.version}</p>
										<div
											class="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-muted-foreground"
										>
											<span class="flex items-center gap-1">
												<User class="size-3" />
												{v.uploaderName}
											</span>
											<span class="flex items-center gap-1">
												<Clock class="size-3" />
												{formatDate(v.createdAt)}
											</span>
											<span>{formatSize(v.size)}</span>
										</div>
										<p class="mt-1 font-mono text-[10px] break-all text-muted-foreground">
											{v.hash}
										</p>
									</div>
									<Button
										variant="outline"
										size="sm"
										class="shrink-0 gap-1.5"
										disabled={downloadingId === v.id}
										onclick={() => handleDownload(v.id)}
									>
										<Download class="size-3.5" />
										{downloadingId === v.id ? 'Getting link…' : 'Download'}
									</Button>
								</div>
							</li>
						{/each}
					</ol>
				{/if}
			</Card.Content>
		</Card.Root>

		<!-- Upload new version -->
		<Card.Root>
			<Card.Header class="pb-3">
				<Card.Title class="text-sm">Upload New Version</Card.Title>
				<Card.Description class="text-xs">
					Replace with a revised document while preserving all previous versions.
				</Card.Description>
			</Card.Header>
			<Card.Content>
				{#if uploadingVersion}
					<div class="flex items-center justify-center py-8">
						<Upload class="mr-2 size-5 animate-pulse text-primary" />
						<span class="text-sm text-muted-foreground">Uploading…</span>
					</div>
				{:else}
					<UploadZone onupload={handleUploadVersion} />
				{/if}
				{#if uploadVersionError}
					<p class="mt-2 text-sm text-destructive">{uploadVersionError}</p>
				{/if}
			</Card.Content>
		</Card.Root>
	{/if}
</div>

<!-- Delete confirmation -->
<Dialog.Root bind:open={deleteOpen}>
	<Dialog.Content class="sm:max-w-sm">
		<Dialog.Header>
			<Dialog.Title>Delete Document</Dialog.Title>
			<Dialog.Description>
				Are you sure you want to delete <strong>{doc?.name}</strong>? All versions will be removed.
				This action cannot be undone.
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

<!-- Reject dialog -->
<Dialog.Root bind:open={rejectOpen}>
	<Dialog.Content class="sm:max-w-sm">
		<Dialog.Header>
			<Dialog.Title>Reject Document</Dialog.Title>
			<Dialog.Description>
				Provide a reason so the submitter knows what to address.
			</Dialog.Description>
		</Dialog.Header>
		<div class="grid gap-2 px-6">
			<Label class="text-xs">Reason (optional)</Label>
			<Textarea
				bind:value={rejectNote}
				placeholder="Describe why the document is being rejected…"
				rows={4}
			/>
		</div>
		<Dialog.Footer>
			<Dialog.Close>
				{#snippet child({ props })}
					<Button variant="outline" {...props}>Cancel</Button>
				{/snippet}
			</Dialog.Close>
			<Button variant="destructive" onclick={handleReject} disabled={rejecting}>
				{rejecting ? 'Rejecting…' : 'Reject'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- Request changes dialog -->
<Dialog.Root bind:open={changesOpen}>
	<Dialog.Content class="sm:max-w-sm">
		<Dialog.Header>
			<Dialog.Title>Request Changes</Dialog.Title>
			<Dialog.Description>
				Describe what changes are needed before this document can be approved.
			</Dialog.Description>
		</Dialog.Header>
		<div class="grid gap-2 px-6">
			<Label class="text-xs">Notes (optional)</Label>
			<Textarea bind:value={changesNote} placeholder="Describe the changes required…" rows={4} />
		</div>
		<Dialog.Footer>
			<Dialog.Close>
				{#snippet child({ props })}
					<Button variant="outline" {...props}>Cancel</Button>
				{/snippet}
			</Dialog.Close>
			<Button
				class="bg-amber-600 text-white hover:bg-amber-700"
				onclick={handleRequestChanges}
				disabled={requestingChanges}
			>
				{requestingChanges ? 'Sending…' : 'Request Changes'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- Assign reviewer dialog -->
<Dialog.Root bind:open={assignReviewerOpen}>
	<Dialog.Content class="sm:max-w-sm">
		<Dialog.Header>
			<Dialog.Title>Assign Reviewer</Dialog.Title>
			<Dialog.Description>Choose a workspace member to review this document.</Dialog.Description>
		</Dialog.Header>
		<div class="grid gap-2 px-6">
			<Label class="text-xs">Reviewer</Label>
			<select
				class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none"
				bind:value={selectedReviewerId}
			>
				<option value={null}>Select a member…</option>
				{#each members as m (m.id)}
					<option value={m.id}>{m.name}</option>
				{/each}
			</select>
			{#if assignReviewerError}
				<p class="text-xs text-destructive">{assignReviewerError}</p>
			{/if}
		</div>
		<Dialog.Footer>
			<Dialog.Close>
				{#snippet child({ props })}
					<Button variant="outline" {...props}>Cancel</Button>
				{/snippet}
			</Dialog.Close>
			<Button onclick={handleAssignReviewer} disabled={assigningReviewer || !selectedReviewerId}>
				{assigningReviewer ? 'Assigning…' : 'Assign'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
