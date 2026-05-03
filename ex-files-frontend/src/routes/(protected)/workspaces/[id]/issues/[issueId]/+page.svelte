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
		getDocumentBytes
	} from '$lib/queries.remote';
	import {
		uploadDocument,
		approveDocument,
		rejectDocument,
		requestDocumentChanges,
		updateIssueAssignee
	} from '$lib/commands.remote';
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
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import * as ScrollArea from '$lib/components/ui/scroll-area/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
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
		EllipsisVertical,
		CheckCircle,
		XCircle,
		Pencil
	} from '@lucide/svelte';

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

	const hasComments = $derived(workbenchStore.activeComments.length > 0);

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

	function canShowMenuFor(doc: { serverId?: string }) {
		return canReviewIssue && !!doc.serverId;
	}

	function canActOn(doc: { reviewStatus?: string }) {
		return (
			doc.reviewStatus === 'pending' ||
			doc.reviewStatus === 'in_review' ||
			doc.reviewStatus === 'changes_requested'
		);
	}

	let rejectTarget = $state<string | null>(null);
	let rejectNote = $state('');
	let rejectBusy = $state(false);

	let changesTarget = $state<string | null>(null);
	let changesNote = $state('');
	let changesBusy = $state(false);

	let assigneeDialogOpen = $state(false);
	let assigneeSelection = $state('');
	let assigneeBusy = $state(false);
	const workspaceMembers = $derived(workspaceQuery.current?.members ?? []);

	function openAssigneePicker() {
		if (!issue || issue.resolved) return;
		assigneeSelection = String(issue.assigneeId);
		assigneeDialogOpen = true;
	}

	async function handleChangeAssignee() {
		if (!issue || !assigneeSelection) return;
		if (assigneeSelection === String(issue.assigneeId)) {
			assigneeDialogOpen = false;
			return;
		}
		assigneeBusy = true;
		try {
			const result = await updateIssueAssignee({
				id: String(issue.id),
				assigneeId: assigneeSelection
			});
			if (!result.ok) {
				toast.error(result.error ?? m.error_action_failed());
				return;
			}
			assigneeDialogOpen = false;
			await workbenchQuery.refresh();
		} finally {
			assigneeBusy = false;
		}
	}

	async function handleApprove(localId: string) {
		const doc = workbenchStore.documents.find((d) => d.id === localId);
		if (!doc?.serverId) return;
		const result = await approveDocument(doc.serverId);
		if (!result.ok) {
			toast.error(result.error ?? m.error_action_failed());
			return;
		}
		workbenchStore.setDocumentReviewStatus(localId, 'approved');
		await workbenchQuery.refresh();
	}

	async function handleReject() {
		if (!rejectTarget) return;
		const doc = workbenchStore.documents.find((d) => d.id === rejectTarget);
		if (!doc?.serverId) return;
		rejectBusy = true;
		try {
			const result = await rejectDocument({ id: doc.serverId, note: rejectNote });
			if (!result.ok) {
				toast.error(result.error ?? m.error_action_failed());
				return;
			}
			workbenchStore.setDocumentReviewStatus(rejectTarget, 'rejected');
			rejectTarget = null;
			rejectNote = '';
			await workbenchQuery.refresh();
		} finally {
			rejectBusy = false;
		}
	}

	async function handleRequestChanges() {
		if (!changesTarget) return;
		const doc = workbenchStore.documents.find((d) => d.id === changesTarget);
		if (!doc?.serverId) return;
		changesBusy = true;
		try {
			const result = await requestDocumentChanges({ id: doc.serverId, note: changesNote });
			if (!result.ok) {
				toast.error(result.error ?? m.error_action_failed());
				return;
			}
			workbenchStore.setDocumentReviewStatus(changesTarget, 'changes_requested');
			changesTarget = null;
			changesNote = '';
			await workbenchQuery.refresh();
		} finally {
			changesBusy = false;
		}
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
		if (!workbenchStore.activeDocument?.serverId) return;
		commentDialog = { page, x, y, screenX, screenY };
	}

	function handleCommentSubmit(text: string) {
		if (!commentDialog) return;
		workbenchStore.addComment(
			commentDialog.page,
			commentDialog.x,
			commentDialog.y,
			text,
			'Reviewer'
		);
		commentDialog = null;
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

	function statusBadgeClass(status: string, reviewStatus?: string) {
		// Local upload-lifecycle states win over review status.
		switch (status) {
			case 'draft':
				return 'bg-amber-100 text-amber-800';
			case 'saving':
				return 'bg-blue-100 text-blue-700';
			case 'error':
				return 'bg-red-100 text-red-700';
			case 'saved':
				switch (reviewStatus) {
					case 'approved':
						return 'bg-emerald-100 text-emerald-700';
					case 'rejected':
						return 'bg-red-100 text-red-700';
					case 'changes_requested':
						return 'bg-amber-100 text-amber-800';
					case 'in_review':
						return 'bg-blue-100 text-blue-700';
					case 'pending':
					default:
						return 'bg-slate-100 text-slate-700';
				}
			default:
				return '';
		}
	}

	function statusLabel(status: string, reviewStatus?: string) {
		switch (status) {
			case 'draft':
				return m.workbench_status_draft();
			case 'saving':
				return m.workbench_saving();
			case 'error':
				return m.workbench_status_error();
			case 'saved':
				switch (reviewStatus) {
					case 'approved':
						return m.workbench_status_approved();
					case 'rejected':
						return m.workbench_status_rejected();
					case 'changes_requested':
						return m.workbench_status_changes_requested();
					case 'in_review':
						return m.workbench_status_awaiting_review();
					case 'pending':
					default:
						return m.workbench_status_saved();
				}
			default:
				return '';
		}
	}

	function formatSize(bytes: number) {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
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
							class="size-7"
							title={m.workbench_expand_sidebar()}
							onclick={() => (leftCollapsed = false)}
						>
							<ChevronRight class="size-4" />
						</Button>
						<Button
							variant="outline"
							size="icon"
							class="size-7"
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
									<li>
										<div
											class="group relative flex items-start gap-1 px-3 py-2 transition-colors {workbenchStore.activeDocumentId ===
											doc.id
												? 'bg-primary/8 text-primary'
												: 'text-foreground hover:bg-muted/60'}"
										>
											<div
												role="button"
												tabindex="0"
												class="min-w-0 flex-1 cursor-pointer text-left"
												onclick={() => selectDocument(doc.id)}
												onkeydown={(e) => {
													if (e.key === 'Enter' || e.key === ' ') {
														e.preventDefault();
														selectDocument(doc.id);
													}
												}}
											>
												<p class="truncate text-xs font-medium">{doc.name}</p>
												{#if doc.uploaderName}
													<p class="truncate text-[10px] text-muted-foreground">
														{doc.uploaderName}
													</p>
												{/if}
												<div class="mt-0.5 flex items-center gap-1.5">
													<Badge
														variant="secondary"
														class="h-4 px-1.5 text-[9px] font-semibold {statusBadgeClass(
															doc.status,
															doc.reviewStatus
														)}"
														title={doc.error ?? ''}
													>
														{statusLabel(doc.status, doc.reviewStatus)}
													</Badge>
													<span class="text-[10px] text-muted-foreground">
														{formatSize(doc.size)}
													</span>
												</div>
											</div>

											{#if canShowMenuFor(doc)}
												<DropdownMenu.Root>
													<DropdownMenu.Trigger>
														{#snippet child({ props })}
															<button
																{...props}
																onclick={(e) => e.stopPropagation()}
																disabled={issue?.resolved}
																class="rounded p-1 text-muted-foreground transition hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-muted-foreground"
																aria-label={m.workbench_actions()}
															>
																<EllipsisVertical class="size-3.5" />
															</button>
														{/snippet}
													</DropdownMenu.Trigger>
													<DropdownMenu.Content side="right" align="start" class="w-48">
														<DropdownMenu.Item
															onclick={() => handleApprove(doc.id)}
															disabled={!canActOn(doc)}
														>
															<CheckCircle class="size-3.5" />
															{m.doc_approve()}
														</DropdownMenu.Item>
														<DropdownMenu.Item
															onclick={() => {
																changesTarget = doc.id;
																changesNote = '';
															}}
															disabled={!canActOn(doc)}
														>
															<MessageSquare class="size-3.5" />
															{m.doc_request_changes()}
														</DropdownMenu.Item>
														<DropdownMenu.Separator />
														<DropdownMenu.Item
															onclick={() => {
																rejectTarget = doc.id;
																rejectNote = '';
															}}
															disabled={!canActOn(doc)}
															class="text-red-600 focus:text-red-600"
														>
															<XCircle class="size-3.5" />
															{m.doc_reject()}
														</DropdownMenu.Item>
													</DropdownMenu.Content>
												</DropdownMenu.Root>
											{/if}
										</div>
									</li>
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
								comments={workbenchStore.activeComments}
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
										comments={workbenchStore.activeComments}
										{currentPage}
										ondelete={(id) => workbenchStore.deleteComment(id)}
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
	<Dialog.Root bind:open={detailsOpen}>
		<Dialog.Content class="max-w-lg">
			<Dialog.Header>
				<Dialog.Title>{issue.title}</Dialog.Title>
				<Dialog.Description>{m.workbench_issue_details()}</Dialog.Description>
			</Dialog.Header>

			<section class="py-2">
				<h3 class="mb-2 text-xs font-semibold text-muted-foreground">
					{m.ws_issue_description_label()}
				</h3>
				{#if issue.description}
					<p class="text-sm leading-relaxed whitespace-pre-line">{issue.description}</p>
				{:else}
					<p class="text-sm text-muted-foreground italic">{m.workbench_no_description()}</p>
				{/if}
			</section>
		</Dialog.Content>
	</Dialog.Root>

	<!-- Change Assignee Dialog -->
	<Dialog.Root bind:open={assigneeDialogOpen}>
		<Dialog.Content class="max-w-md">
			<Dialog.Header>
				<Dialog.Title>{m.ws_issue_change_assignee_label()}</Dialog.Title>
			</Dialog.Header>
			<div class="space-y-2 py-2">
				<Select.Root bind:value={assigneeSelection} type="single">
					<Select.Label>{m.ws_issue_assignee_label()}</Select.Label>
					<Select.Trigger class="w-full"
						>{workspaceMembers.find((m) => m.id === assigneeSelection)?.name ||
							m.ws_issue_assignee_select()}</Select.Trigger
					>
					<Select.Content>
						{#each workspaceMembers as member (member.id)}
							<Select.Item value={String(member.id)}>{member.name}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			<Dialog.Footer>
				<Button
					variant="outline"
					onclick={() => (assigneeDialogOpen = false)}
					disabled={assigneeBusy}
				>
					{m.common_cancel()}
				</Button>
				<Button onclick={handleChangeAssignee} disabled={assigneeBusy || !assigneeSelection}>
					{assigneeBusy ? m.common_saving() : m.common_save()}
				</Button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>

	<!-- Reject Dialog -->
	<Dialog.Root
		open={rejectTarget !== null}
		onOpenChange={(open) => {
			if (!open) {
				rejectTarget = null;
				rejectNote = '';
			}
		}}
	>
		<Dialog.Content class="max-w-md">
			<Dialog.Header>
				<Dialog.Title>{m.doc_reject_title()}</Dialog.Title>
				<Dialog.Description>{m.doc_reject_description()}</Dialog.Description>
			</Dialog.Header>
			<div class="space-y-2 py-2">
				<Label for="reject-note">{m.doc_reject_reason_label()}</Label>
				<Textarea
					id="reject-note"
					bind:value={rejectNote}
					placeholder={m.doc_reject_placeholder()}
					rows={4}
				/>
			</div>
			<Dialog.Footer>
				<Button
					variant="outline"
					onclick={() => {
						rejectTarget = null;
						rejectNote = '';
					}}
					disabled={rejectBusy}
				>
					{m.common_cancel()}
				</Button>
				<Button variant="destructive" onclick={handleReject} disabled={rejectBusy}>
					{rejectBusy ? m.doc_rejecting() : m.doc_reject()}
				</Button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>

	<!-- Request Changes Dialog -->
	<Dialog.Root
		open={changesTarget !== null}
		onOpenChange={(open) => {
			if (!open) {
				changesTarget = null;
				changesNote = '';
			}
		}}
	>
		<Dialog.Content class="max-w-md">
			<Dialog.Header>
				<Dialog.Title>{m.doc_changes_title()}</Dialog.Title>
				<Dialog.Description>{m.doc_changes_description()}</Dialog.Description>
			</Dialog.Header>
			<div class="space-y-2 py-2">
				<Label for="changes-note">{m.doc_changes_notes_label()}</Label>
				<Textarea
					id="changes-note"
					bind:value={changesNote}
					placeholder={m.doc_changes_placeholder()}
					rows={4}
				/>
			</div>
			<Dialog.Footer>
				<Button
					variant="outline"
					onclick={() => {
						changesTarget = null;
						changesNote = '';
					}}
					disabled={changesBusy}
				>
					{m.common_cancel()}
				</Button>
				<Button onclick={handleRequestChanges} disabled={changesBusy}>
					{changesBusy ? m.doc_changes_sending() : m.doc_request_changes()}
				</Button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>
{/if}
