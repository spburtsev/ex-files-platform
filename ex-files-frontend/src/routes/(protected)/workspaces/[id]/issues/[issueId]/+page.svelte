<script lang="ts">
	import { page } from '$app/state';
	import { browser } from '$app/environment';
	import { onDestroy, tick, untrack } from 'svelte';
	import { SvelteMap } from 'svelte/reactivity';
	import { workbenchStore } from '$lib/stores/workbench.svelte';
	import { extraBreadcrumbs } from '$lib/stores/breadcrumbs.svelte';
	import {
		getIssue,
		getWorkspaceDetail,
		getDocuments,
		getDocumentBytes,
		getComments
	} from '$lib/queries.remote';
	import {
		uploadDocument,
		createComment,
		deleteComment,
		approveDocument
	} from '$lib/commands.remote';
	import { MAX_UPLOAD_BYTES, MAX_UPLOAD_MB } from '$lib/upload';
	import { isManager, initials, avatarColorClass } from '$lib/utils';
	import { deadlineChip } from '$lib/deadline';
	import { statusBadgeClass, statusLabel, canActOn } from '$lib/doc-status';
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { toast } from 'svelte-sonner';
	import { getPdfjs } from '$lib/pdf/pdfjs';
	import UploadZone from '$lib/components/pdf/UploadZone.svelte';
	import PdfViewer from '$lib/components/pdf/PdfViewer.svelte';
	import CommentPanel from '$lib/components/pdf/CommentPanel.svelte';
	import CommentDialog from '$lib/components/pdf/CommentDialog.svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Toggle } from '$lib/components/ui/toggle/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import * as Sheet from '$lib/components/ui/sheet/index.js';
	import {
		ChevronRight,
		ChevronLeft,
		Upload,
		MessageSquare,
		Clock,
		Info,
		Save,
		LoaderCircle,
		Minus,
		Plus,
		Trash2,
		Pencil,
		CheckCircle,
		XCircle,
		Users
	} from '@lucide/svelte';
	import DetailsDialog from './DetailsDialog.svelte';
	import ChangeAssignee from './ChangeAssignee.svelte';
	import RejectDialog from './RejectDialog.svelte';
	import RequestChanges from './RequestChanges.svelte';
	import DocumentItem from './DocumentItem.svelte';
	import ReviewConfig from './ReviewConfig.svelte';

	const { data } = $props();
	const me = $derived(data.user);

	const wsId = $derived(page.params.id ?? '');
	const issueId = $derived(page.params.issueId ?? '');

	const workbenchQuery = $derived(getIssue(issueId));
	const issue = $derived(workbenchQuery.current?.issue);
	const user = $derived(workbenchQuery.current?.user);

	const workspaceQuery = $derived(getWorkspaceDetail(wsId));
	const workspace = $derived(workspaceQuery.current?.workspace);

	$effect(() => {
		if (issueId) workbenchStore.setIssue(issueId);
	});

	async function syncDocumentsFromServer(imperative = false) {
		if (!issueId) return;
		const id = issueId;
		try {
			const res = imperative ? await getDocuments(id).run() : await getDocuments(id);
			if (workbenchStore.currentIssueId !== id) return;
			workbenchStore.hydrate(
				res.documents.map((d) => ({
					serverId: String(d.id),
					name: d.name,
					size: Number(d.size),
					mimeType: d.mimeType,
					uploaderName: d.uploaderName,
					reviewStatus: d.status,
					reviewerNote: d.reviewerNote,
					approvals: d.approvals,
					approvalCount: d.approvalCount,
					requiredApprovals: d.requiredApprovals
				}))
			);
		} catch (err) {
			console.error('Failed to sync documents', err);
		}
	}

	$effect(() => {
		const id = issueId;
		if (!id) return;
		untrack(() => {
			void (async () => {
				if (!workbenchStore.hydrated) await syncDocumentsFromServer();
				if (workbenchStore.currentIssueId !== id) return;
				const activeId = workbenchStore.activeDocumentId;
				if (activeId) await selectDocument(activeId);
			})();
		});
	});

	$effect(() => {
		const wsName = workspace?.name;
		const issueTitle = issue?.title;
		if (wsName && issueTitle) {
			const statusBadge = issue.resolved
				? { label: m.issue_resolved(), cls: 'bg-emerald-100 text-emerald-700' }
				: { label: m.issue_open(), cls: 'bg-blue-100 text-blue-700' };
			extraBreadcrumbs.set([
				{ label: wsName, href: localizeHref(`/workspaces/${wsId}`) },
				{ label: issueTitle, badges: [statusBadge] }
			]);
		}
	});
	onDestroy(() => extraBreadcrumbs.set([]));

	let currentPage = $state(0);
	let pageCount = $state(0);
	let scale = $state(1);
	let commentDialog = $state<{
		page: number;
		x: number;
		y: number;
		screenX: number;
		screenY: number;
	} | null>(null);
	let showUpload = $state(false);
	let showMarkers = $state(true);
	let leftCollapsed = $state(false);
	let detailsOpen = $state(false);
	let versionsOpen = $state(false);
	let reviewConfigOpen = $state(false);

	const activeServerId = $derived(workbenchStore.activeDocument?.serverId);
	const commentsQuery = $derived(activeServerId ? getComments(activeServerId) : null);
	const comments = $derived(commentsQuery?.current ?? []);
	const hasComments = $derived(comments.length > 0);

	const COMMENT_EVENTS = new Set(['comment.added', 'comment.deleted']);
	const REVIEW_EVENTS = new Set([
		'document.approved',
		'document.rejected',
		'document.changes_requested',
		'document.reviewer_assigned',
		'document.approval_added'
	]);
	$effect(() => {
		if (!browser) return;
		const docId = activeServerId;
		if (!docId) return;
		const es = new EventSource(`/api/events?documentId=${docId}`);
		es.onmessage = (e) => {
			try {
				const ev = JSON.parse(e.data);
				const t = ev?.type;
				if (typeof t !== 'string') return;
				if (COMMENT_EVENTS.has(t)) commentsQuery?.refresh();
				else if (REVIEW_EVENTS.has(t)) {
					workbenchQuery.refresh();
					void syncDocumentsFromServer(true);
				}
			} catch (err) {
				console.error('SSE parse error', err);
			}
		};
		return () => es.close();
	});

	type SubmissionFilter = 'all' | 'approved' | 'changes_requested' | 'rejected';
	let submissionFilter = $state<SubmissionFilter>('all');
	const filteredDocuments = $derived(
		submissionFilter === 'all'
			? workbenchStore.documents
			: workbenchStore.documents.filter((d) => d.reviewStatus === submissionFilter)
	);
	const displayedSubmissionFilter = $derived.by(() => {
		switch (submissionFilter) {
			case 'approved':
				return m.workbench_status_approved();
			case 'changes_requested':
				return m.workbench_status_changes_requested();
			case 'rejected':
				return m.workbench_status_rejected();
			default:
				return m.ws_status_all();
		}
	});

	let pdfViewer = $state<ReturnType<typeof PdfViewer> | undefined>();
	const pageByDoc = new SvelteMap<string, number>();

	function rememberPageOf(localId: string | null) {
		if (localId) pageByDoc.set(localId, currentPage);
	}

	const isIssueCreator = $derived(me && issue ? Number(me.id) === Number(issue.creatorId) : false);
	// Reviewer panel: with a non-empty panel only its members may review; an
	// empty panel falls back to managers / the issue creator (legacy behavior).
	const reviewerIds = $derived(new Set((issue?.reviewers ?? []).map((r) => String(r.id))));
	const panelEmpty = $derived(reviewerIds.size === 0);
	const isPanelReviewer = $derived(me ? reviewerIds.has(String(me.id)) : false);
	const canReviewIssue = $derived(
		panelEmpty ? isManager(me?.role) || isIssueCreator : isPanelReviewer
	);
	// Whether the toolbar should expose review actions for the currently shown
	// document: a saved (serverId) doc, the viewer can review, issue still open.
	const canReviewActive = $derived(
		canReviewIssue && !!workbenchStore.activeDocument?.serverId && !issue?.resolved
	);
	// Only the issue creator or workspace owner may configure the panel + N.
	const canConfigureReview = $derived(
		isIssueCreator || (!!workspace && !!me && String(workspace.managerId) === String(me.id))
	);
	// True once the current user has already approved the active version.
	const alreadyApprovedActive = $derived(
		!!me &&
			!!workbenchStore.activeDocument?.approvals?.some(
				(a) => String(a.reviewerId) === String(me.id)
			)
	);

	let rejectTarget = $state<string | null>(null);
	let changesTarget = $state<string | null>(null);

	let assigneeDialogOpen = $state(false);
	const workspaceMembers = $derived(workspaceQuery.current?.members ?? []);

	function openAssigneePicker() {
		if (!issue || issue.resolved) return;
		assigneeDialogOpen = true;
	}

	async function ensureBytes(localId: string): Promise<Uint8Array | null> {
		const doc = workbenchStore.documents.find((d) => d.id === localId);
		if (!doc) return null;
		if (doc.data) return doc.data;
		if (!doc.serverId) return null;
		const data = await getDocumentBytes(doc.serverId).run();
		const pdfjsLib = await getPdfjs();
		const probe = await pdfjsLib.getDocument({ data: data.slice() }).promise;
		const numPages = probe.numPages;
		probe.destroy();
		workbenchStore.setDocumentData(localId, data, numPages);
		return data;
	}

	async function selectDocument(localId: string) {
		const prev = workbenchStore.activeDocumentId;
		if (prev && prev !== localId) rememberPageOf(prev);
		workbenchStore.setActiveDocument(localId);
		currentPage = pageByDoc.get(localId) ?? 0;
		let data: Uint8Array | null;
		try {
			data = await ensureBytes(localId);
		} catch (err) {
			console.error('Failed to load document binary', err);
			toast.error(m.error_action_failed());
			return;
		}
		if (!data) return;
		if (workbenchStore.activeDocumentId !== localId) return;
		await tick();
		await pdfViewer?.load(data);
	}

	async function handleUpload(file: File) {
		if (file.size > MAX_UPLOAD_BYTES) {
			toast.error(m.workbench_file_too_large({ mb: String(MAX_UPLOAD_MB) }));
			return;
		}
		rememberPageOf(workbenchStore.activeDocumentId);
		const pdfjsLib = await getPdfjs();
		const buffer = await file.arrayBuffer();
		const data = new Uint8Array(buffer);
		const doc = await pdfjsLib.getDocument({ data: data.slice() }).promise;
		const uploaded = workbenchStore.uploadDocument(file, data, doc.numPages);
		doc.destroy();
		currentPage = 0;
		showUpload = false;
		if (uploaded) {
			await tick();
			await pdfViewer?.load(data);
		}
	}

	function handlePageClick(page: number, x: number, y: number, screenX: number, screenY: number) {
		if (!workbenchStore.activeDocument?.serverId) {
			return;
		}
		commentDialog = { page, x, y, screenX, screenY };
		console.log('Opening comment dialog at', { page, x, y, screenX, screenY });
	}

	async function handleCommentSubmit(text: string) {
		if (!commentDialog) return;
		const docId = workbenchStore.activeDocument?.serverId;
		if (!docId || !commentsQuery) {
			commentDialog = null;
			return;
		}
		const meta = {
			page: commentDialog.page + 1,
			x: commentDialog.x,
			y: commentDialog.y
		};
		commentDialog = null;
		const r = await createComment({ docId, body: text, metadata: meta }).updates(commentsQuery);
		if (!r.ok) {
			toast.error(r.error ?? m.error_action_failed());
		}
	}

	async function handleCommentDelete(commentId: string) {
		const docId = workbenchStore.activeDocument?.serverId;
		if (!docId || !commentsQuery) return;
		const r = await deleteComment({ docId, commentId }).updates(commentsQuery);
		if (!r.ok) toast.error(r.error ?? m.error_action_failed());
	}

	// Approve the currently shown document from the toolbar (mirrors the
	// per-version approve action that also lives in DocumentItem).
	async function handleApproveActive() {
		const doc = workbenchStore.activeDocument;
		if (!doc?.serverId) return;
		const r = await approveDocument(doc.serverId);
		if (!r.ok) {
			toast.error(r.error ?? m.error_action_failed());
			return;
		}
		workbenchStore.setDocumentReviewStatus(doc.id, 'approved');
		await workbenchQuery.refresh();
	}

	function handleDiscard(docId: string) {
		const doc = workbenchStore.documents.find((d) => d.id === docId);
		if (!doc) return;
		pageByDoc.delete(docId);
		workbenchStore.discardDocument(docId);
		toast.success(m.workbench_discard_success({ name: doc.name }));
	}

	async function handleSave(docId: string) {
		const doc = workbenchStore.documents.find((d) => d.id === docId);
		if (!doc || !doc.data || doc.status === 'saving' || doc.status === 'saved') return;
		workbenchStore.setDocumentStatus(docId, 'saving');
		try {
			const result = await uploadDocument({
				issueId,
				name: doc.name,
				mimeType: doc.mimeType,
				data: doc.data.slice()
			});
			if (result.ok) {
				workbenchStore.setDocumentSaved(docId, result.docId);
				toast.success(m.workbench_save_success({ name: doc.name }));
			} else {
				workbenchStore.setDocumentStatus(docId, 'error', result.error);
				toast.error(result.error ?? m.workbench_save_error());
			}
		} catch (err: unknown) {
			console.error('Error uploading document', err);
			workbenchStore.setDocumentStatus(docId, 'error', m.error_network_retry());
			toast.error(m.error_network_retry());
		}
	}

	const dl = $derived.by(() => {
		if (!issue || issue.resolved || !issue.deadline) return null;
		const d = issue.deadline ? new Date(issue.deadline) : null;
		return d ? deadlineChip(d) : null;
	});
</script>

<svelte:head>
	<title>{issue?.title ?? m.workbench_page_title()} - ex-files</title>
</svelte:head>

{#if issue}
	<div class="flex h-[calc(100svh-3rem)] flex-col overflow-hidden border-t">
		<!-- Workbench tri-pane -->
		<div class="flex min-h-0 flex-1 overflow-hidden">
			<!-- Left Sidebar -->
			<aside
				class="relative flex shrink-0 flex-col border-r bg-card transition-all duration-200 {leftCollapsed
					? 'w-10'
					: 'w-80'}"
			>
				<!-- Clickable edge -->
				<button
					title={m.workbench_toggle_sidebar()}
					class="absolute inset-y-0 right-0 z-10 w-1 cursor-col-resize transition-all hover:bg-primary/20"
					onclick={() => (leftCollapsed = !leftCollapsed)}
				></button>

				{#if leftCollapsed}
					<!-- Collapsed strip -->
					<div class="flex w-full flex-col items-center gap-1 pt-2">
						<Button
							variant="outline"
							size="icon"
							title={m.workbench_expand_sidebar()}
							onclick={() => (leftCollapsed = false)}
						>
							<ChevronRight class="size-4" />
						</Button>
						<Button
							variant="outline"
							size="icon"
							title={m.workbench_upload_document()}
							onclick={() => {
								leftCollapsed = false;
								showUpload = true;
							}}
						>
							<Upload class="size-4" />
						</Button>
					</div>
				{:else}
					<!-- Issue info: assignee + status + deadline + details -->
					<div class="shrink-0 space-y-2 border-b px-3 py-3">
						<div class="flex items-center gap-2">
							{#if user}
								<p class="min-w-0 flex-1 truncate text-xs font-medium">
									<span class="text-muted-foreground">{m.ws_issue_assignee_label()}:</span>
									{user.name}
								</p>
								{#if canReviewIssue && !issue.resolved}
									<button
										type="button"
										onclick={openAssigneePicker}
										class="rounded p-1 text-muted-foreground transition hover:bg-muted hover:text-foreground"
										aria-label={m.ws_issue_change_assignee_label()}
										title={m.ws_issue_change_assignee_label()}
									>
										<Pencil class="size-3" />
									</button>
								{/if}
							{/if}
						</div>
						{#if dl}
							<Badge variant="outline" class="w-full justify-center gap-1 text-[11px] {dl.cls}">
								<Clock class="size-3 shrink-0" />
								{dl.label}
							</Badge>
						{/if}
						<Button
							variant="outline"
							size="sm"
							class="w-full gap-1.5 text-xs"
							onclick={() => (detailsOpen = true)}
						>
							<Info class="size-3.5" />
							{m.workbench_details()}
						</Button>
					</div>

					<!-- Current version + older-versions drawer trigger -->
					<div class="shrink-0 space-y-2 border-b px-3 py-3">
						<p class="text-xs font-semibold text-muted-foreground">
							{m.workbench_current_version()}
						</p>
						{#if workbenchStore.activeDocument}
							{@const ad = workbenchStore.activeDocument}
							<div class="flex items-center gap-2">
								<p class="min-w-0 flex-1 truncate text-xs font-medium">{ad.name}</p>
								<Badge
									variant="secondary"
									class="h-4 shrink-0 px-1.5 text-[9px] font-semibold {statusBadgeClass(
										ad.status,
										ad.reviewStatus
									)}"
								>
									{statusLabel(ad.status, ad.reviewStatus)}
								</Badge>
							</div>
							{#if !panelEmpty && ad.serverId}
								<div class="flex items-center gap-2 text-[11px] text-muted-foreground">
									<span>
										{m.workbench_approvals_progress({
											count: String(ad.approvalCount ?? 0),
											required: String(ad.requiredApprovals ?? 1)
										})}
									</span>
									{#if ad.approvals && ad.approvals.length > 0}
										<div class="flex -space-x-1">
											{#each ad.approvals as a (a.reviewerId)}
												<span
													class="flex size-4 items-center justify-center rounded-full text-[8px] font-bold text-white ring-1 ring-card {avatarColorClass(
														a.reviewerId
													)}"
													title={a.reviewerName}
												>
													{initials(a.reviewerName)}
												</span>
											{/each}
										</div>
									{/if}
								</div>
							{/if}
						{:else}
							<p class="text-xs text-muted-foreground">{m.workbench_no_submissions()}</p>
						{/if}
						{#if !panelEmpty}
							<p
								class="truncate text-[11px] text-muted-foreground"
								title={(issue?.reviewers ?? []).map((r) => r.name).join(', ')}
							>
								<span class="font-medium">{m.review_config_reviewers()}:</span>
								{(issue?.reviewers ?? []).map((r) => r.name).join(', ')}
							</p>
						{/if}
						<Button
							variant="ghost"
							size="sm"
							class="w-full justify-start gap-1.5 text-xs"
							onclick={() => (versionsOpen = true)}
						>
							<Clock class="size-3.5" />
							{m.workbench_show_older_versions()}
						</Button>
						{#if canConfigureReview && !issue?.resolved}
							<Button
								variant="ghost"
								size="sm"
								class="w-full justify-start gap-1.5 text-xs"
								onclick={() => (reviewConfigOpen = true)}
							>
								<Users class="size-3.5" />
								{m.review_config_title()}
							</Button>
						{/if}
					</div>

					<!-- Comments -->
					<div class="flex min-h-0 flex-1 flex-col">
						<CommentPanel
							{comments}
							{currentPage}
							currentUserId={me?.id}
							ondelete={handleCommentDelete}
							ongotopage={(p) => (currentPage = p)}
						/>
					</div>

					<!-- Upload -->
					<div class="shrink-0 border-t p-3">
						{#if issue?.resolved}
							<p class="text-center text-[11px] text-muted-foreground">
								{m.workbench_resolved_no_uploads()}
							</p>
						{:else if showUpload}
							<div class="flex flex-col gap-2">
								<UploadZone onupload={handleUpload} />
								<Button
									variant="ghost"
									size="sm"
									class="text-xs"
									onclick={() => (showUpload = false)}
								>
									{m.common_cancel()}
								</Button>
							</div>
						{:else}
							<Button
								variant="outline"
								size="sm"
								class="w-full gap-1.5 text-xs"
								onclick={() => (showUpload = true)}
							>
								<Upload class="size-3.5" />
								{m.workbench_upload_submission()}
							</Button>
						{/if}
					</div>
				{/if}
			</aside>

			<!-- Main Content -->
			<div class="flex min-w-0 flex-1">
				{#if !workbenchStore.activeDocument}
					<!-- Upload State -->
					<div class="flex flex-1 flex-col items-center justify-center gap-6 p-8">
						<div class="w-full max-w-lg">
							<UploadZone onupload={handleUpload} />
						</div>
					</div>
				{:else}
					<!-- PDF Viewer -->
					<div class="flex min-w-0 flex-1 flex-col overflow-hidden">
						<!-- Document toolbar: zoom + pages + save -->
						<div class="flex shrink-0 items-center gap-3 border-b bg-card px-3">
							<!-- Zoom -->
							<div class="flex flex-1 items-center">
								<Button
									variant="ghost"
									size="icon-xs"
									disabled={scale <= 0.5}
									aria-label={m.pdf_zoom_out()}
									onclick={() => (scale = Math.max(0.5, scale - 0.25))}
								>
									<Minus class="size-3.5" />
								</Button>
								<span class="text-center text-[10px] tabular-nums">
									{Math.round(scale * 100)}%
								</span>
								<Button
									variant="ghost"
									size="icon-xs"
									disabled={scale >= 3}
									aria-label={m.pdf_zoom_in()}
									onclick={() => (scale = Math.min(3, scale + 0.25))}
								>
									<Plus class="size-3.5" />
								</Button>
							</div>

							<!-- Pages -->
							<div class="flex items-center gap-1">
								<Button
									variant="ghost"
									size="icon-xs"
									disabled={currentPage <= 0}
									aria-label={m.pdf_page_back()}
									onclick={() => (currentPage = Math.max(0, currentPage - 1))}
								>
									<ChevronLeft class="size-3.5" />
								</Button>
								<span class="text-xs tabular-nums">
									{currentPage + 1} / {pageCount || '...'}
								</span>
								<Button
									variant="ghost"
									size="icon-xs"
									disabled={pageCount === 0 || currentPage >= pageCount - 1}
									aria-label={m.pdf_page_forward()}
									onclick={() => (currentPage = Math.min(pageCount - 1, currentPage + 1))}
								>
									<ChevronRight class="size-3.5" />
								</Button>
							</div>

							<!-- Toggle comments + Save / discard draft -->
							<div class="flex flex-1 items-center justify-end gap-2">
								<Toggle
									bind:pressed={showMarkers}
									size="sm"
									disabled={!hasComments}
									class="data-[state=on]:bg-transparent data-[state=on]:*:[svg]:fill-blue-500 data-[state=on]:*:[svg]:stroke-blue-500"
									title={!hasComments
										? m.workbench_no_markers()
										: showMarkers
											? m.workbench_hide_comments()
											: m.workbench_show_comments()}
								>
									<MessageSquare class="size-3.5 shrink-0" />
									{showMarkers ? m.workbench_hide_comments() : m.workbench_show_comments()}
								</Toggle>
								{#if workbenchStore.activeDocument.status !== 'saved'}
									{@const ad = workbenchStore.activeDocument}
									<Button
										variant="outline"
										size="xs"
										class="gap-1.5"
										disabled={ad.status === 'saving'}
										onclick={() => handleDiscard(ad.id)}
									>
										<Trash2 class="size-3.5" />
										{m.workbench_discard_button()}
									</Button>
									<Button
										size="xs"
										class="gap-1.5"
										disabled={ad.status === 'saving'}
										onclick={() => handleSave(ad.id)}
									>
										{#if ad.status === 'saving'}
											<LoaderCircle class="size-3.5 animate-spin" />
											{m.workbench_saving()}
										{:else}
											<Save class="size-3.5" />
											{m.workbench_save_button()}
										{/if}
									</Button>
								{:else if canReviewActive}
									{@const ad = workbenchStore.activeDocument}
									<Button
										size="xs"
										class="gap-1.5"
										disabled={!canActOn(ad.reviewStatus) || alreadyApprovedActive}
										title={alreadyApprovedActive ? m.workbench_already_approved() : undefined}
										onclick={handleApproveActive}
									>
										<CheckCircle class="size-3.5" />
										{m.doc_approve()}
									</Button>
									<Button
										variant="outline"
										size="xs"
										class="gap-1.5"
										disabled={!canActOn(ad.reviewStatus)}
										onclick={() => (changesTarget = ad.id)}
									>
										<MessageSquare class="size-3.5" />
										{m.doc_request_changes()}
									</Button>
									<Button
										variant="outline"
										size="xs"
										class="gap-1.5 text-red-600 hover:text-red-600"
										disabled={!canActOn(ad.reviewStatus)}
										onclick={() => (rejectTarget = ad.id)}
									>
										<XCircle class="size-3.5" />
										{m.doc_reject()}
									</Button>
								{/if}
							</div>
						</div>

						{#if (workbenchStore.activeDocument?.reviewStatus === 'rejected' || workbenchStore.activeDocument?.reviewStatus === 'changes_requested') && workbenchStore.activeDocument?.reviewerNote}
							<div
								class="shrink-0 border-b px-4 py-3 text-sm {workbenchStore.activeDocument
									.reviewStatus === 'rejected'
									? 'border-red-200 bg-red-50 text-red-800'
									: 'border-amber-200 bg-amber-50 text-amber-800'}"
							>
								<p class="font-medium">
									{workbenchStore.activeDocument.reviewStatus === 'rejected'
										? m.doc_rejection_reason()
										: m.doc_changes_requested_label()}
								</p>
								<p class="mt-1 text-xs leading-relaxed whitespace-pre-line">
									{workbenchStore.activeDocument.reviewerNote}
								</p>
							</div>
						{/if}

						<div class="relative flex-1 overflow-auto">
							<PdfViewer
								{comments}
								{currentPage}
								{showMarkers}
								bind:scale
								bind:this={pdfViewer}
								onpageclick={handlePageClick}
								onpagecount={(c) => (pageCount = c)}
							/>
							{#if !workbenchStore.activeDocument.data}
								<div class="absolute inset-0 flex items-center justify-center bg-gray-100/80">
									<LoaderCircle class="size-6 animate-spin text-muted-foreground" />
								</div>
							{/if}
						</div>
					</div>
				{/if}
			</div>
		</div>
	</div>

	<!-- Comment Dialog -->
	{#if commentDialog}
		<CommentDialog
			page={commentDialog.page}
			x={commentDialog.x}
			y={commentDialog.y}
			screenX={commentDialog.screenX}
			screenY={commentDialog.screenY}
			onsubmit={handleCommentSubmit}
			oncancel={() => (commentDialog = null)}
		/>
	{/if}

	<!-- Older versions drawer -->
	<Sheet.Root bind:open={versionsOpen}>
		<Sheet.Content side="right" class="w-full gap-0 p-0 sm:max-w-md">
			<Sheet.Header class="border-b px-4 py-3">
				<Sheet.Title>{m.workbench_all_versions()}</Sheet.Title>
			</Sheet.Header>
			<div class="shrink-0 border-b px-4 py-3">
				<Select.Root bind:value={submissionFilter} type="single">
					<Select.Trigger class="w-full">{displayedSubmissionFilter}</Select.Trigger>
					<Select.Content>
						<Select.Item value="all">{m.ws_status_all()}</Select.Item>
						<Select.Item value="approved">{m.workbench_status_approved()}</Select.Item>
						<Select.Item value="changes_requested"
							>{m.workbench_status_changes_requested()}</Select.Item
						>
						<Select.Item value="rejected">{m.workbench_status_rejected()}</Select.Item>
					</Select.Content>
				</Select.Root>
			</div>
			<div class="min-h-0 flex-1 overflow-y-auto">
				{#if workbenchStore.documents.length === 0}
					<p class="px-4 py-3 text-xs text-muted-foreground">{m.workbench_no_submissions()}</p>
				{:else if filteredDocuments.length === 0}
					<p class="px-4 py-3 text-xs text-muted-foreground">{m.ws_no_matches()}</p>
				{:else}
					<ul class="pb-1">
						{#each filteredDocuments as doc, docIdx (docIdx)}
							<DocumentItem
								{doc}
								{issue}
								{canReviewIssue}
								onSelect={() => {
									selectDocument(doc.id);
									versionsOpen = false;
								}}
								onApproved={() => workbenchQuery.refresh()}
								onRequestChangesClick={(doc) => {
									changesTarget = doc.id;
								}}
								onRejectClick={(doc) => {
									rejectTarget = doc.id;
								}}
							/>
						{/each}
					</ul>
				{/if}
			</div>
		</Sheet.Content>
	</Sheet.Root>

	<!-- Details Dialog -->
	<DetailsDialog bind:open={detailsOpen} {issue} />

	<!-- Change Assignee Dialog -->
	<ChangeAssignee
		bind:open={assigneeDialogOpen}
		{workspaceMembers}
		currentAssigneeId={issue?.assigneeId ?? ''}
		{issueId}
		onSuccess={() => workbenchQuery.refresh()}
	/>
	<!-- Reject Dialog -->
	<RejectDialog
		bind:target={rejectTarget}
		onSuccess={async () => {
			await workbenchQuery.refresh();
			await syncDocumentsFromServer(true);
		}}
	/>
	<!-- Request Changes Dialog -->
	<RequestChanges
		bind:target={changesTarget}
		onSuccess={async () => {
			await workbenchQuery.refresh();
			await syncDocumentsFromServer(true);
		}}
	/>
	<!-- Review Config Dialog -->
	<ReviewConfig
		bind:open={reviewConfigOpen}
		{issueId}
		{workspaceMembers}
		assigneeId={issue?.assigneeId ?? ''}
		currentReviewerIds={(issue?.reviewers ?? []).map((r) => String(r.id))}
		currentRequiredApprovals={issue?.requiredApprovals ?? 1}
		onSuccess={async () => {
			await workbenchQuery.refresh();
			await syncDocumentsFromServer(true);
		}}
	/>
{/if}
