<script lang="ts">
	import { page } from '$app/state';
	import { onDestroy } from 'svelte';
	import { workbenchStore } from '$lib/stores/workbench.svelte';
	import { extraBreadcrumbs } from '$lib/stores/breadcrumbs';
	import {
		getIssue,
		getWorkspaceDetail,
		getDocuments,
		getDocumentDetail,
		getDocumentBytes
	} from '$lib/data.remote';
	import { uploadDocument } from '$lib/commands.remote';
	import { protoTsToDate } from '$lib/proto-utils';
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { toast } from 'svelte-sonner';
	import UploadZone from '$lib/components/pdf/UploadZone.svelte';
	import PdfViewer from '$lib/components/pdf/PdfViewer.svelte';
	import CommentPanel from '$lib/components/pdf/CommentPanel.svelte';
	import CommentDialog from '$lib/components/pdf/CommentDialog.svelte';
	import ActivityLog from '$lib/components/pdf/ActivityLog.svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as ScrollArea from '$lib/components/ui/scroll-area/index.js';
	import * as Tabs from '$lib/components/ui/tabs/index.js';
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
		Trash2
	} from '@lucide/svelte';

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
						mimeType: d.mimeType
					}))
				);
			} catch (err) {
				console.error('Failed to hydrate documents', err);
			}
		})();
	});

	$effect(() => {
		const ad = workbenchStore.activeDocument;
		if (!ad || ad.data || !ad.serverId) return;
		const localId = ad.id;
		const serverId = ad.serverId;
		(async () => {
			try {
				let versionId = ad.versionId;
				if (!versionId) {
					const detail = await getDocumentDetail(serverId);
					const latest = [...(detail?.versions ?? [])].sort(
						(a, b) => Number(b.version) - Number(a.version)
					)[0];
					if (!latest) return;
					versionId = Number(latest.id);
				}
				const data = await getDocumentBytes({ docId: serverId, versionId });
				const pdfjsLib = await getPdfjs();
				const doc = await pdfjsLib.getDocument({ data: data.slice() }).promise;
				const numPages = doc.numPages;
				doc.destroy();
				if (workbenchStore.activeDocument?.id !== localId) return;
				workbenchStore.setDocumentData(localId, data, numPages);
				if (versionId && !ad.versionId) {
					workbenchStore.setDocumentSaved(localId, serverId, versionId);
				}
			} catch (err) {
				console.error('Failed to load document binary', err);
				toast.error(m.error_action_failed());
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
	let sidePanel = $state<'comments' | 'activity'>('comments');
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

	async function getPdfjs() {
		const pdfjsLib = await import('pdfjs-dist');
		pdfjsLib.GlobalWorkerOptions.workerSrc = new URL(
			'pdfjs-dist/build/pdf.worker.mjs',
			import.meta.url
		).href;
		return pdfjsLib;
	}

	async function handleUpload(file: File) {
		const pdfjsLib = await getPdfjs();
		const buffer = await file.arrayBuffer();
		const data = new Uint8Array(buffer);
		const doc = await pdfjsLib.getDocument({ data: data.slice() }).promise;
		workbenchStore.uploadDocument(file, data, doc.numPages);
		doc.destroy();
		currentPage = 0;
		showUpload = false;
	}

	function handlePageClick(page: number, x: number, y: number, screenX: number, screenY: number) {
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

	function statusBadgeClass(status: string) {
		switch (status) {
			case 'draft':
				return 'bg-amber-100 text-amber-800';
			case 'saving':
				return 'bg-blue-100 text-blue-700';
			case 'saved':
				return 'bg-emerald-100 text-emerald-700';
			case 'error':
				return 'bg-red-100 text-red-700';
			default:
				return '';
		}
	}

	function statusLabel(status: string) {
		switch (status) {
			case 'draft':
				return m.workbench_status_draft();
			case 'saving':
				return m.workbench_saving();
			case 'saved':
				return m.workbench_status_saved();
			case 'error':
				return m.workbench_status_error();
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
		const d = protoTsToDate(issue.deadline);
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
								<p class="min-w-0 flex-1 truncate text-xs font-medium">{user.name}</p>
							{/if}
							{#if issue.resolved}
								<Badge
									variant="secondary"
									class="shrink-0 bg-emerald-100 text-[10px] text-emerald-700"
								>
									{m.issue_resolved()}
								</Badge>
							{:else}
								<Badge
									variant="secondary"
									class="shrink-0 bg-blue-100 text-[10px] text-blue-700"
								>
									{m.issue_open()}
								</Badge>
							{/if}
						</div>
						{#if dl}
							<Badge
								variant="outline"
								class="w-full justify-center gap-1 text-[11px] {dl.cls}"
							>
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
					<div class="shrink-0 px-3 pt-3 pb-1">
						<p class="text-[10px] font-semibold text-muted-foreground">
							{m.workbench_submissions()}
						</p>
					</div>

					<!-- Document list -->
					<ScrollArea.Root class="min-h-0 flex-1">
						{#if workbenchStore.documents.length === 0}
							<p class="px-3 py-2 text-xs text-muted-foreground">{m.workbench_no_submissions()}</p>
						{:else}
							<ul class="pb-1">
								{#each workbenchStore.documents as doc, docIdx (docIdx)}
									<li>
										<button
											class="w-full px-3 py-2 text-left transition-colors {workbenchStore.activeDocumentId ===
											doc.id
												? 'bg-primary/8 text-primary'
												: 'text-foreground hover:bg-muted/60'}"
											onclick={() => {
												workbenchStore.setActiveDocument(doc.id);
												currentPage = 0;
											}}
										>
											<p class="truncate text-xs font-medium">{doc.name}</p>
											<div class="mt-0.5 flex items-center gap-1.5">
												<Badge
													variant="secondary"
													class="h-4 px-1.5 text-[9px] font-semibold {statusBadgeClass(doc.status)}"
													title={doc.error ?? ''}
												>
													{statusLabel(doc.status)}
												</Badge>
												<span class="text-[10px] text-muted-foreground">{formatSize(doc.size)}</span
												>
											</div>
										</button>
									</li>
								{/each}
							</ul>
						{/if}
					</ScrollArea.Root>

					<!-- Upload -->
					<div class="shrink-0 border-t p-3">
						{#if showUpload}
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
						{#if workbenchStore.activityLog.length > 0}
							<div class="mt-4 w-full max-w-lg rounded-lg border bg-card shadow-sm">
								<ActivityLog entries={workbenchStore.activityLog} />
							</div>
						{/if}
					</div>
				{:else}
					<!-- PDF Viewer -->
					<div class="flex min-w-0 flex-1 flex-col overflow-hidden">
						<!-- Document toolbar: zoom + pages + save -->
						<div class="flex shrink-0 items-center gap-3 border-b bg-card px-3">
							<!-- Zoom -->
							<div class="flex flex-1 items-center gap-1">
								<Button
									variant="ghost"
									size="icon"
									class="size-7"
									disabled={scale <= 0.5}
									aria-label={m.pdf_zoom_out()}
									onclick={() => (scale = Math.max(0.5, scale - 0.25))}
								>
									<Minus class="size-3.5" />
								</Button>
								<span class="w-10 text-center text-xs tabular-nums">
									{Math.round(scale * 100)}%
								</span>
								<Button
									variant="ghost"
									size="icon"
									class="size-7"
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
									size="icon"
									class="size-7"
									disabled={currentPage <= 0}
									aria-label={m.pdf_page_back()}
									onclick={() => (currentPage = Math.max(0, currentPage - 1))}
								>
									<ChevronLeft class="size-3.5" />
								</Button>
								<span class="text-xs tabular-nums">
									{currentPage + 1} / {pageCount || '…'}
								</span>
								<Button
									variant="ghost"
									size="icon"
									class="size-7"
									disabled={pageCount === 0 || currentPage >= pageCount - 1}
									aria-label={m.pdf_page_forward()}
									onclick={() => (currentPage = Math.min(pageCount - 1, currentPage + 1))}
								>
									<ChevronRight class="size-3.5" />
								</Button>
							</div>

							<!-- Save / discard draft -->
							<div class="flex flex-1 justify-end gap-2">
								{#if workbenchStore.activeDocument.status !== 'saved'}
									{@const ad = workbenchStore.activeDocument}
									<Button
										variant="outline"
										size="sm"
										class="gap-1.5"
										disabled={ad.status === 'saving'}
										onclick={() => handleDiscard(ad.id)}
									>
										<Trash2 class="size-3.5" />
										{m.workbench_discard_button()}
									</Button>
									<Button
										size="sm"
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

						<div class="flex-1 overflow-auto">
							{#if workbenchStore.activeDocument.data}
								<PdfViewer
									data={workbenchStore.activeDocument.data}
									comments={workbenchStore.activeComments}
									{currentPage}
									{showMarkers}
									bind:scale
									onpageclick={handlePageClick}
									onpagecount={(c) => (pageCount = c)}
								/>
							{:else}
								<div class="flex h-full items-center justify-center">
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
									class="size-7"
									title={m.workbench_expand_sidebar()}
									onclick={() => (rightCollapsed = false)}
								>
									<ChevronLeft class="size-4" />
								</Button>
								<Button
									variant="outline"
									size="icon"
									class="size-7 {showMarkers && hasComments
										? 'border-amber-300 bg-amber-50 text-amber-500'
										: ''}"
									disabled={!hasComments}
									title={!hasComments
										? m.workbench_no_markers()
										: showMarkers
											? m.workbench_hide_markers()
											: m.workbench_show_markers()}
									onclick={() => (showMarkers = !showMarkers)}
								>
									<MessageSquare class="size-4" />
								</Button>
							</div>
						{:else}
							<div class="flex min-h-0 w-full flex-col">
								<div class="shrink-0 border-b px-3 py-2">
									<Button
										variant={showMarkers ? 'secondary' : 'outline'}
										size="sm"
										class="w-full gap-1.5 text-xs {showMarkers
											? 'border-amber-300 bg-amber-50 text-amber-700 hover:bg-amber-100'
											: ''}"
										disabled={!hasComments}
										onclick={() => (showMarkers = !showMarkers)}
									>
										<MessageSquare class="size-3.5 shrink-0" />
										{showMarkers ? m.workbench_hide_markers() : m.workbench_show_markers()}
									</Button>
								</div>

								<Tabs.Root
									bind:value={sidePanel}
									class="flex min-h-0 flex-1 flex-col gap-0"
								>
									<Tabs.List class="mx-2 mt-2 w-auto shrink-0">
										<Tabs.Trigger value="comments" class="flex-1">
											{m.workbench_comments()}
											{#if workbenchStore.comments.length > 0}
												<span
													class="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-semibold"
												>
													{workbenchStore.comments.length}
												</span>
											{/if}
										</Tabs.Trigger>
										<Tabs.Trigger value="activity" class="flex-1">
											{m.workbench_activity()}
										</Tabs.Trigger>
									</Tabs.List>
									<Tabs.Content
										value="comments"
										class="mt-2 min-h-0 flex-1 overflow-hidden"
									>
										<CommentPanel
											comments={workbenchStore.activeComments}
											{currentPage}
											ondelete={(id) => workbenchStore.deleteComment(id)}
											ongotopage={(p) => (currentPage = p)}
										/>
									</Tabs.Content>
									<Tabs.Content
										value="activity"
										class="mt-2 min-h-0 flex-1 overflow-hidden"
									>
										<ActivityLog entries={workbenchStore.activityLog} />
									</Tabs.Content>
								</Tabs.Root>
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
					<p class="text-sm italic text-muted-foreground">{m.workbench_no_description()}</p>
				{/if}
			</section>
		</Dialog.Content>
	</Dialog.Root>
{/if}
