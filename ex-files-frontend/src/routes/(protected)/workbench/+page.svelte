<script lang="ts">
	import { workbenchStore } from '$lib/stores/workbench.svelte';
	import { getAssignment } from '$lib/data.remote';
	import { protoTsToDate } from '$lib/proto-utils';
	import { m } from '$lib/paraglide/messages.js';
	import UploadZone from '$lib/components/pdf/UploadZone.svelte';
	import PdfViewer from '$lib/components/pdf/PdfViewer.svelte';
	import CommentPanel from '$lib/components/pdf/CommentPanel.svelte';
	import CommentDialog from '$lib/components/pdf/CommentDialog.svelte';
	import ActivityLog from '$lib/components/pdf/ActivityLog.svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as ScrollArea from '$lib/components/ui/scroll-area/index.js';
	import { ChevronRight, ChevronLeft, Upload, MessageSquare, Clock, Info } from '@lucide/svelte';

	const workbenchQuery = getAssignment('1');
	const assignment = $derived(workbenchQuery.current?.assignment);
	const user = $derived(workbenchQuery.current?.user);

	let currentPage = $state(0);
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

	function formatSize(bytes: number) {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

	function deadlineChip(d: Date) {
		const h = (d.getTime() - Date.now()) / 3_600_000;
		if (h < 0) return { label: m.workbench_overdue(), cls: 'border-red-200 bg-red-50 text-red-600' };
		if (h < 24)
			return { label: m.workbench_hours_left({ hours: String(Math.round(h)) }), cls: 'border-red-200 bg-red-50 text-red-600' };
		if (h < 72)
			return {
				label: m.workbench_days_hours_left({ days: String(Math.floor(h / 24)), hours: String(Math.round(h % 24)) }),
				cls: 'border-amber-200 bg-amber-50 text-amber-700'
			};
		return {
			label: d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
			cls: 'border-border bg-muted/40 text-muted-foreground'
		};
	}

	const dl = $derived.by(() => {
		if (!assignment || assignment.resolved || !assignment.deadline) return null;
		const d = protoTsToDate(assignment.deadline);
		return d ? deadlineChip(d) : null;
	});
</script>

<svelte:head>
	<title>{m.workbench_page_title()}</title>
</svelte:head>

{#if assignment}
	<div class="flex flex-1 overflow-hidden">
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
				<!-- Assignment info -->
				<div class="border-b px-3 py-3">
					<div class="space-y-1.5 rounded-lg border bg-muted/40 px-3 py-2.5">
						{#if user}
							<p class="truncate text-[10px] font-medium text-muted-foreground">{user.name}</p>
						{/if}
						<h2 class="line-clamp-2 text-sm leading-snug font-semibold">
							{assignment.title}
						</h2>
						<p class="line-clamp-3 text-xs leading-relaxed text-muted-foreground">
							{assignment.description}
						</p>
					</div>
				</div>

				<!-- Controls: deadline pill + details button -->
				<div class="shrink-0 space-y-2 px-3 pt-4 pb-1">
					<div class="flex flex-wrap items-center gap-2">
						{#if dl}
							<Badge variant="outline" class="gap-1 text-[11px] {dl.cls}">
								<Clock class="size-3 shrink-0" />
								{dl.label}
							</Badge>
						{/if}
						<Button
							variant="outline"
							size="sm"
							class="h-6 gap-1 rounded-full text-[11px]"
							onclick={() => (detailsOpen = true)}
						>
							<Info class="size-3 shrink-0" />
							{m.workbench_details()}
						</Button>
					</div>
					<p class="text-[10px] font-semibold text-muted-foreground">{m.workbench_submissions()}</p>
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
										<p class="mt-0.5 text-[10px] text-muted-foreground">{formatSize(doc.size)}</p>
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
					<div class="text-center">
						<h2 class="text-xl font-semibold">{assignment.title}</h2>
						<p class="mt-1 text-sm text-muted-foreground">{assignment.description}</p>
					</div>
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
					<div class="flex-1 overflow-auto">
						<PdfViewer
							data={workbenchStore.activeDocument.data}
							comments={workbenchStore.activeComments}
							{currentPage}
							{showMarkers}
							onpageclick={handlePageClick}
							onpagechange={(p) => (currentPage = p)}
							onpagecount={() => {}}
						/>
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

							<div class="flex min-h-0 flex-1 flex-col">
								<div class="flex shrink-0 border-b bg-card">
									<button
										class="flex-1 px-4 py-2.5 text-sm font-medium transition-colors {sidePanel ===
										'comments'
											? 'border-b-2 border-primary text-primary'
											: 'text-muted-foreground hover:text-foreground'}"
										onclick={() => (sidePanel = 'comments')}
									>
										{m.workbench_comments()}
									</button>
									<button
										class="flex-1 px-4 py-2.5 text-sm font-medium transition-colors {sidePanel ===
										'activity'
											? 'border-b-2 border-primary text-primary'
											: 'text-muted-foreground hover:text-foreground'}"
										onclick={() => (sidePanel = 'activity')}
									>
										{m.workbench_activity()}
									</button>
								</div>
								<div class="min-h-0 flex-1 overflow-hidden">
									{#if sidePanel === 'comments'}
										<CommentPanel
											comments={workbenchStore.activeComments}
											{currentPage}
											ondelete={(id) => workbenchStore.deleteComment(id)}
											ongotopage={(p) => (currentPage = p)}
										/>
									{:else}
										<ActivityLog entries={workbenchStore.activityLog} />
									{/if}
								</div>
							</div>
						</div>
					{/if}
				</div>
			{/if}
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
				<Dialog.Title>{assignment.title}</Dialog.Title>
				<Dialog.Description>{m.workbench_assignment_details()}</Dialog.Description>
			</Dialog.Header>

			<div class="space-y-4 py-2">
				<section>
					<h3 class="mb-2 text-xs font-semibold text-muted-foreground">{m.workbench_instructions()}</h3>
					<p class="text-sm leading-relaxed">{assignment.description}</p>
					<p class="mt-2 text-sm leading-relaxed text-muted-foreground">
						{m.workbench_instructions_text()}
					</p>
				</section>
			</div>
		</Dialog.Content>
	</Dialog.Root>
{/if}
