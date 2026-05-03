<script lang="ts">
	import { page } from '$app/state';
	import { onDestroy, tick } from 'svelte';
	import { SvelteMap } from 'svelte/reactivity';
	import { workbenchStore } from '$lib/stores/workbench.svelte';
	import { extraBreadcrumbs } from '$lib/stores/breadcrumbs.svelte';
	import {
		getIssue,
		getWorkspaceDetail,
		getDocuments,
		getDocumentDetail,
		getDocumentBytes,
		getComments
	} from '$lib/queries.remote';
	import { uploadDocument, createComment, deleteComment } from '$lib/commands.remote';
	import { isManager } from '$lib/utils';
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
	import * as ScrollArea from '$lib/components/ui/scroll-area/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
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
		Pencil
	} from '@lucide/svelte';
	import DetailsDialog from './DetailsDialog.svelte';
	import ChangeAssignee from './ChangeAssignee.svelte';
	import RejectDialog from './RejectDialog.svelte';
	import RequestChanges from './RequestChanges.svelte';
	import DocumentItem from './DocumentItem.svelte';

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

	$effect(() => {
		if (!issueId || workbenchStore.hydrated) return;
		const id = issueId;
		(async () => {
			try {
				const res = await getDocuments(id);
				if (workbenchStore.currentIssueId !== id) return;
				workbenchStore.hydrate(
					res.documents.map((d) => ({
						serverId: String(d.id),
						name: d.name,
						size: Number(d.size),
						mimeType: d.mimeType,
						uploaderName: d.uploaderName,
						reviewStatus: d.status
					}))
				);
				const activeId = workbenchStore.activeDocumentId;
				if (activeId) await selectDocument(activeId);
			} catch (err) {
				console.error('Failed to hydrate documents', err);
			}
		})();
	});

	$effect(() => {
		const wsName = workspace?.name;
		const issueTitle = issue?.title;
		if (wsName && issueTitle) {
			extraBreadcrumbs.set([
				{ label: wsName, href: localizeHref(`/workspaces/${wsId}`) },
				{ label: issueTitle }
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
	let rightCollapsed = $state(false);
	let leftCollapsed = $state(false);
	let detailsOpen = $state(false);

	const activeServerId = $derived(workbenchStore.activeDocument?.serverId);
	const commentsQuery = $derived(activeServerId ? getComments(activeServerId) : null);
	const comments = $derived(commentsQuery?.current ?? []);
	const hasComments = $derived(comments.length > 0);

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
	const canReviewIssue = $derived(isManager(me?.role) || isIssueCreator);

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
		let versionId = doc.versionId;
		if (!versionId) {
			const detail = await getDocumentDetail(doc.serverId).run();
			const latest = [...(detail?.versions ?? [])].sort(
				(a, b) => Number(b.version) - Number(a.version)
			)[0];
			if (!latest) return null;
			versionId = latest.id;
		}
		const data = await getDocumentBytes({ docId: doc.serverId, versionId }).run();
		const pdfjsLib = await getPdfjs();
		const probe = await pdfjsLib.getDocument({ data: data.slice() }).promise;
		const numPages = probe.numPages;
		probe.destroy();
		workbenchStore.setDocumentData(localId, data, numPages);
		if (versionId && !doc.versionId) {
			workbenchStore.setDocumentSaved(localId, doc.serverId, versionId);
		}
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
				workbenchStore.setDocumentSaved(docId, result.docId, result.versionId);
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

	function deadlineChip(d: Date) {
		const h = (d.getTime() - Date.now()) / 3_600_000;
		if (h < 0)
			return { label: m.workbench_overdue(), cls: 'border-red-200 bg-red-50 text-red-600' };
		if (h < 24)
			return {
				label: m.workbench_hours_left({ hours: String(Math.round(h)) }),
				cls: 'border-red-200 bg-red-50 text-red-600'
			};
		if (h < 72)
			return {
				label: m.workbench_days_hours_left({
					days: String(Math.floor(h / 24)),
					hours: String(Math.round(h % 24))
				}),
				cls: 'border-amber-200 bg-amber-50 text-amber-700'
			};
		return {
			label: d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
			cls: 'border-border bg-muted/40 text-muted-foreground'
		};
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
					: 'w-64'}"
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
							{#if issue.resolved}
								<Badge
									variant="secondary"
									class="shrink-0 bg-emerald-100 text-[10px] text-emerald-700"
								>
									{m.issue_resolved()}
								</Badge>
							{:else}
								<Badge variant="secondary" class="shrink-0 bg-blue-100 text-[10px] text-blue-700">
									{m.issue_open()}
								</Badge>
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

					<!-- Submissions list header -->
					<div class="shrink-0 space-y-1.5 px-3 pt-3 pb-3">
						<p class="text-xs font-semibold text-muted-foreground">
							{m.workbench_submissions()}
						</p>
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

					<!-- Document list -->
					<ScrollArea.Root class="min-h-0 flex-1">
						{#if workbenchStore.documents.length === 0}
							<p class="px-3 py-2 text-xs text-muted-foreground">{m.workbench_no_submissions()}</p>
						{:else if filteredDocuments.length === 0}
							<p class="px-3 py-2 text-xs text-muted-foreground">{m.ws_no_matches()}</p>
						{:else}
							<ul class="pb-1">
								{#each filteredDocuments as doc, docIdx (docIdx)}
									<DocumentItem
										{doc}
										{issue}
                                        {canReviewIssue}
										onSelect={() => selectDocument(doc.id)}
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
					</ScrollArea.Root>

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
								{/if}
							</div>
						</div>

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

					<!-- Right Panel -->
					<div
						class="relative flex shrink-0 border-l bg-card transition-all duration-200 {rightCollapsed
							? 'w-10'
							: 'w-72'}"
					>
						<!-- Clickable edge -->
						<button
							title={m.workbench_toggle_activity()}
							class="absolute inset-y-0 left-0 z-10 w-1 cursor-col-resize transition-all hover:bg-primary/20"
							onclick={() => (rightCollapsed = !rightCollapsed)}
						></button>

						{#if rightCollapsed}
							<div class="flex w-full flex-col items-center gap-1 pt-2">
								<Button
									variant="outline"
									size="icon"
									title={m.workbench_expand_sidebar()}
									onclick={() => (rightCollapsed = false)}
								>
									<ChevronLeft class="size-4" />
								</Button>
							</div>
						{:else}
							<div class="flex min-h-0 w-full flex-col">
								<CommentPanel
									{comments}
									{currentPage}
									ondelete={handleCommentDelete}
									ongotopage={(p) => (currentPage = p)}
								/>
							</div>
						{/if}
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
	<RejectDialog bind:target={rejectTarget} onSuccess={() => workbenchQuery.refresh()} />
	<!-- Request Changes Dialog -->
	<RequestChanges bind:target={changesTarget} onSuccess={() => workbenchQuery.refresh()} />
{/if}
