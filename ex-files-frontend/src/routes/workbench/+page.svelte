<script lang="ts">
	import { workbenchStore } from '$lib/stores/workbench.svelte';
	import type { PageData } from './$types';
	import UploadZone from '$lib/components/pdf/UploadZone.svelte';
	import PdfViewer from '$lib/components/pdf/PdfViewer.svelte';
	import CommentPanel from '$lib/components/pdf/CommentPanel.svelte';
	import CommentDialog from '$lib/components/pdf/CommentDialog.svelte';
	import ActivityLog from '$lib/components/pdf/ActivityLog.svelte';

	let { data }: { data: PageData } = $props();

	const assignment = $derived(data.assignment);
	const user = $derived(data.user);

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

	function deadlineChip(iso: string) {
		const h = (new Date(iso).getTime() - Date.now()) / 3_600_000;
		if (h < 0) return { label: 'Overdue', cls: 'bg-red-50 text-red-600 border-red-200' };
		if (h < 24)
			return { label: `${Math.round(h)}h left`, cls: 'bg-red-50 text-red-600 border-red-200' };
		if (h < 72)
			return {
				label: `${Math.floor(h / 24)}d ${Math.round(h % 24)}h left`,
				cls: 'bg-amber-50 text-amber-700 border-amber-200'
			};
		return {
			label: new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
			cls: 'bg-gray-50 text-gray-500 border-gray-200'
		};
	}

	const dl = $derived(
		!assignment.resolved && assignment.deadline ? deadlineChip(assignment.deadline) : null
	);
</script>

<svelte:head>
	<title>ex-files - Document Review</title>
</svelte:head>

<div class="flex flex-1 overflow-hidden bg-gray-50">
	<!-- Left Sidebar -->
	<aside
		class="relative flex shrink-0 flex-col border-r bg-white transition-all duration-200 {leftCollapsed
			? 'w-10'
			: 'w-64'}"
	>
		<!-- Clickable edge -->
		<button
			title="Toggle Sidebar"
			class="absolute inset-y-0 right-0 z-10 w-1 cursor-col-resize transition-all hover:bg-blue-400/40 hover:shadow-[4px_0_12px_rgba(96,165,250,0.5)]"
			onclick={() => (leftCollapsed = !leftCollapsed)}
		></button>

		{#if leftCollapsed}
			<!-- Collapsed strip -->
			<div class="flex w-full flex-col items-center gap-1 pt-2">
				<button
					title="Expand sidebar"
					class="rounded-md border border-blue-200 bg-blue-50 p-1.5 text-blue-500 transition-colors hover:border-blue-300 hover:bg-blue-100 hover:text-blue-700"
					onclick={() => (leftCollapsed = false)}
				>
					<svg
						class="h-4 w-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="1.5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
					</svg>
				</button>
				<span title="Upload document">
					<button
						class="rounded-md border border-blue-200 bg-blue-50 p-1.5 text-blue-400 transition-colors hover:border-blue-300 hover:bg-blue-100 hover:text-blue-600"
						title="Toggle Upload Document"
						onclick={() => {
							leftCollapsed = false;
							showUpload = true;
						}}
					>
						<svg
							class="h-4 w-4"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M9 8.25H7.5a2.25 2.25 0 0 0-2.25 2.25v9a2.25 2.25 0 0 0 2.25 2.25h9a2.25 2.25 0 0 0 2.25-2.25v-9a2.25 2.25 0 0 0-2.25-2.25H15m0-3-3-3m0 0-3 3m3-3V15"
							/>
						</svg>
					</button>
				</span>
			</div>
		{:else}
			<!-- Expanded -->
			<!-- Assignment info -->
			<div class="border-b px-3 py-3">
				<div class="space-y-1.5 rounded-lg border border-blue-200 bg-blue-50/40 px-3 py-2.5">
					{#if user}
						<p class="truncate text-[10px] font-medium text-gray-500">{user.name}</p>
					{/if}
					<h2 class="line-clamp-2 text-sm leading-snug font-semibold text-gray-900">
						{assignment.title}
					</h2>
					<p class="line-clamp-3 text-xs leading-relaxed text-gray-700">{assignment.description}</p>
				</div>
			</div>

			<!-- Controls: deadline pill + details button -->
			<div class="shrink-0 space-y-2 px-3 pt-4 pb-1">
				<div class="flex flex-wrap items-center gap-2">
					{#if dl}
						<span
							class="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium {dl.cls}"
						>
							<svg
								class="h-3 w-3 shrink-0"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="2"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"
								/>
							</svg>
							{dl.label}
						</span>
					{/if}
					<button
						onclick={() => (detailsOpen = true)}
						class="inline-flex items-center gap-1 rounded-full border border-blue-200 bg-blue-50 px-2 py-0.5 text-[11px] font-medium text-blue-600 transition-colors hover:border-blue-300 hover:bg-blue-100"
					>
						<svg
							class="h-3 w-3 shrink-0"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="m11.25 11.25.041-.02a.75.75 0 0 1 1.063.852l-.708 2.836a.75.75 0 0 0 1.063.853l.041-.021M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-3.75h.008v.008H12V8.25Z"
							/>
						</svg>
						Details
					</button>
				</div>
				<p class="text-[10px] font-semibold text-gray-400">Submissions</p>
			</div>

			<!-- Document list -->
			<div class="min-h-0 flex-1 overflow-y-auto">
				{#if workbenchStore.documents.length === 0}
					<p class="px-3 py-2 text-xs text-gray-400">No submissions yet</p>
				{:else}
					<ul class="pb-1">
						{#each workbenchStore.documents as doc, docIdx (docIdx)}
							<li>
								<button
									class="w-full px-3 py-2 text-left transition-colors {workbenchStore.activeDocumentId ===
									doc.id
										? 'bg-blue-50 text-blue-700'
										: 'text-gray-700 hover:bg-gray-50'}"
									onclick={() => {
										workbenchStore.setActiveDocument(doc.id);
										currentPage = 0;
									}}
								>
									<p class="truncate text-xs font-medium">{doc.name}</p>
									<p class="mt-0.5 text-[10px] text-gray-400">{formatSize(doc.size)}</p>
								</button>
							</li>
						{/each}
					</ul>
				{/if}
			</div>

			<!-- Upload -->
			<div class="shrink-0 border-t p-3">
				{#if showUpload}
					<div class="flex flex-col gap-2">
						<UploadZone onupload={handleUpload} />
						<button
							class="text-xs text-gray-500 hover:text-gray-700"
							onclick={() => (showUpload = false)}
						>
							Cancel
						</button>
					</div>
				{:else}
					<button
						class="inline-flex w-full items-center justify-center gap-1.5 rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs text-gray-600 shadow-sm hover:bg-gray-50"
						onclick={() => (showUpload = true)}
					>
						<svg
							class="h-3.5 w-3.5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M9 8.25H7.5a2.25 2.25 0 0 0-2.25 2.25v9a2.25 2.25 0 0 0 2.25 2.25h9a2.25 2.25 0 0 0 2.25-2.25v-9a2.25 2.25 0 0 0-2.25-2.25H15m0-3-3-3m0 0-3 3m3-3V15"
							/>
						</svg>
						Upload submission
					</button>
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
					<h2 class="text-xl font-semibold text-gray-800">{assignment.title}</h2>
					<p class="mt-1 text-sm text-gray-500">{assignment.description}</p>
				</div>
				<div class="w-full max-w-lg">
					<UploadZone onupload={handleUpload} />
				</div>
				{#if workbenchStore.activityLog.length > 0}
					<div class="mt-4 w-full max-w-lg rounded-lg border bg-white shadow-sm">
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
				class="relative flex shrink-0 border-l bg-white transition-all duration-200 {rightCollapsed
					? 'w-10'
					: 'w-72'}"
			>
				<!-- Clickable edge -->
				<button
					title="Toggle Activity Sidebar"
					class="absolute inset-y-0 left-0 z-10 w-1 cursor-col-resize transition-all hover:bg-blue-400/40 hover:shadow-[-4px_0_12px_rgba(96,165,250,0.5)]"
					onclick={() => (rightCollapsed = !rightCollapsed)}
				></button>

				{#if rightCollapsed}
					<div class="flex w-full flex-col items-center gap-1 pt-2">
						<button
							title="Expand Activity Sidebar"
							class="rounded-md border border-blue-200 bg-blue-50 p-1.5 text-blue-500 transition-colors hover:border-blue-300 hover:bg-blue-100 hover:text-blue-700"
							onclick={() => (rightCollapsed = false)}
						>
							<svg
								class="h-4 w-4"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="1.5"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M15.75 19.5 8.25 12l7.5-7.5"
								/>
							</svg>
						</button>
						<span
							title={!hasComments
								? 'No markers to show'
								: showMarkers
									? 'Hide markers'
									: 'Show markers'}
						>
							<button
								disabled={!hasComments}
								title="Toggle Show Comments"
								class="rounded-md border p-1.5 transition-colors disabled:cursor-not-allowed disabled:opacity-30 {showMarkers &&
								hasComments
									? 'border-amber-300 bg-amber-50 text-amber-500 hover:bg-amber-100'
									: 'border-blue-200 bg-blue-50 text-blue-400 hover:border-blue-300 hover:bg-blue-100 hover:text-blue-600'}"
								onclick={() => (showMarkers = !showMarkers)}
							>
								<svg
									class="h-4 w-4"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="1.5"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M7.5 8.25h9m-9 3H12m-9.75 1.51c0 1.6 1.123 2.994 2.707 3.227 1.129.166 2.27.293 3.423.379.35.026.67.21.865.501L12 21l2.755-4.133a1.14 1.14 0 0 1 .865-.501 48.172 48.172 0 0 0 3.423-.379c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0 0 12 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018Z"
									/>
								</svg>
							</button>
						</span>
					</div>
				{:else}
					<div class="flex min-h-0 w-full flex-col">
						<div class="shrink-0 border-b px-3 py-2">
							<button
								disabled={!hasComments}
								class="inline-flex w-full items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs shadow-sm transition-colors disabled:cursor-not-allowed disabled:opacity-40 {showMarkers
									? 'border-amber-300 bg-amber-50 text-amber-700'
									: 'border-gray-300 bg-white text-gray-600'} hover:enabled:bg-gray-50"
								onclick={() => (showMarkers = !showMarkers)}
							>
								<svg
									class="h-3.5 w-3.5 shrink-0"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="1.5"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M7.5 8.25h9m-9 3H12m-9.75 1.51c0 1.6 1.123 2.994 2.707 3.227 1.129.166 2.27.293 3.423.379.35.026.67.21.865.501L12 21l2.755-4.133a1.14 1.14 0 0 1 .865-.501 48.172 48.172 0 0 0 3.423-.379c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0 0 12 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018Z"
									/>
								</svg>
								{showMarkers ? 'Hide markers' : 'Show markers'}
							</button>
						</div>

						<div class="flex shrink-0 border-b">
							<button
								class="flex-1 px-4 py-2.5 text-sm font-medium transition-colors {sidePanel ===
								'comments'
									? 'border-b-2 border-blue-600 text-blue-600'
									: 'text-gray-500 hover:text-gray-700'}"
								onclick={() => (sidePanel = 'comments')}
							>
								Comments
							</button>
							<button
								class="flex-1 px-4 py-2.5 text-sm font-medium transition-colors {sidePanel ===
								'activity'
									? 'border-b-2 border-blue-600 text-blue-600'
									: 'text-gray-500 hover:text-gray-700'}"
								onclick={() => (sidePanel = 'activity')}
							>
								Activity
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
{#if detailsOpen}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<div class="absolute inset-0 bg-black/50" onclick={() => (detailsOpen = false)}></div>

		<div
			class="relative flex max-h-[80vh] w-full max-w-lg flex-col rounded-xl border border-gray-200 bg-white shadow-2xl"
		>
			<!-- Header -->
			<div class="flex shrink-0 items-start justify-between border-b px-5 py-4">
				<div>
					<h2 class="text-base font-semibold text-gray-900">{assignment.title}</h2>
					<p class="mt-0.5 text-xs text-gray-500">Assignment details</p>
				</div>
				<button
					onclick={() => (detailsOpen = false)}
					class="rounded-md p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
					title="Toggle Details"
				>
					<svg
						class="h-4 w-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<!-- Body -->
			<div class="flex-1 space-y-5 overflow-y-auto px-5 py-4">
				<section>
					<h3 class="mb-2 text-xs font-semibold text-gray-500">Instructions</h3>
					<p class="text-sm leading-relaxed text-gray-700">{assignment.description}</p>
					<p class="mt-2 text-sm leading-relaxed text-gray-700">
						Please ensure your submission follows the formatting guidelines. Your work must be
						original and properly cited. Late submissions will be penalised unless an extension has
						been granted in advance.
					</p>
				</section>
			</div>

			<!-- Footer -->
			<div class="flex shrink-0 justify-end border-t px-5 py-3">
				<button
					class="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
				>
					<svg
						class="h-3.5 w-3.5 shrink-0"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3"
						/>
					</svg>
					Download materials
				</button>
			</div>
		</div>
	</div>
{/if}
