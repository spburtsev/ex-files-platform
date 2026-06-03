<script lang="ts">
	import type { PDFDocumentProxy } from 'pdfjs-dist';
	import type { Attachment } from 'svelte/attachments';
	import { getPdfjs } from '$lib/pdf/pdfjs';
	import type { Comment } from '$lib/api';
	import { avatarColorClass, initials } from '$lib/utils';
	import { deriveClusters } from '$lib/pdf/rendering';

	interface Props {
		comments: Comment[];
		currentPage: number;
		showMarkers: boolean;
		scale?: number;
		onpageclick: (page: number, x: number, y: number, screenX: number, screenY: number) => void;
		onpagecount: (count: number) => void;
	}

	let {
		comments,
		currentPage,
		showMarkers,
		scale = $bindable(1),
		onpageclick,
		onpagecount
	}: Props = $props();

	let pdfDoc = $state<PDFDocumentProxy | null>(null);
	let error = $state<string | null>(null);
	let canvasRef: HTMLCanvasElement | null = null;
	let loadToken = 0;
	let hoveredClusterId = $state<string | null>(null);
	let pinnedClusterId = $state<string | null>(null);

	export async function load(data: Uint8Array) {
		const myToken = ++loadToken;
		error = null;
		try {
			const pdfjsLib = await getPdfjs();
			const doc = await pdfjsLib.getDocument({ data: data.slice() }).promise;
			if (myToken !== loadToken) {
				doc.destroy();
				return;
			}
			pdfDoc?.destroy();
			pdfDoc = doc;
			onpagecount(doc.numPages);
		} catch (e) {
			if (myToken === loadToken) {
				error = e instanceof Error ? e.message : 'Failed to load PDF';
			}
		}
	}

	$effect(() => {
		return () => {
			pdfDoc?.destroy();
			pdfDoc = null;
		};
	});

	const pageComments = $derived(comments.filter((c) => c.metadata.page === currentPage + 1));
	const clusteredComments = $derived(
		pageComments.map((c) => ({ ...c, x: c.metadata.x, y: c.metadata.y }))
	);

	// Markers within 3% of canvas size (marker diameter at default zoom) collapse
	// into a single cluster, displayed at the centroid.
	const CLUSTER_THRESHOLD = 0.03;

	const clusters = $derived(deriveClusters(clusteredComments, CLUSTER_THRESHOLD));

	// Drop any pinned feed when navigating between pages.
	$effect(() => {
		void currentPage;
		pinnedClusterId = null;
	});

	function renderAttachment(doc: PDFDocumentProxy): Attachment<HTMLCanvasElement> {
		return (canvas) => {
			canvasRef = canvas;
			$effect(() => {
				const page = currentPage;
				const s = scale;
				let cancelled = false;
				(async () => {
					try {
						const p = await doc.getPage(page + 1);
						const viewport = p.getViewport({ scale: s });
						if (cancelled) return;
						canvas.width = viewport.width;
						canvas.height = viewport.height;
						await p.render({ canvas, viewport }).promise;
						if (!cancelled) error = null;
					} catch (e) {
						const isCancelledRender =
							e instanceof Error && e.name === 'RenderingCancelledException';
						if (!cancelled && !isCancelledRender) {
							error = e instanceof Error ? e.message : 'Failed to render page';
						}
					}
				})();
				return () => {
					cancelled = true;
				};
			});
			return () => {
				canvasRef = null;
			};
		};
	}

	function handleCanvasClick(e: MouseEvent) {
		// A pinned feed is dismissed by any canvas click outside the marker/popup.
		// Don't open the new-comment dialog on the same click that dismisses it.
		if (pinnedClusterId !== null) {
			pinnedClusterId = null;
			return;
		}
		if (!canvasRef) return;
		const rect = canvasRef.getBoundingClientRect();
		const x = (e.clientX - rect.left) / rect.width;
		const y = (e.clientY - rect.top) / rect.height;
		onpageclick(currentPage, x, y, e.clientX, e.clientY);
	}

	function toggleClusterPin(e: MouseEvent, clusterId: string) {
		e.stopPropagation();
		pinnedClusterId = pinnedClusterId === clusterId ? null : clusterId;
	}

	function formatTime(iso: string) {
		return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}
</script>

<div class="relative flex flex-col">
	{#if error}
		<div class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<div class="relative flex justify-center overflow-auto bg-gray-100 p-6">
		{#if pdfDoc}
			<!-- svelte-ignore a11y_click_events_have_key_events -->
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div class="relative inline-block cursor-crosshair shadow-lg" onclick={handleCanvasClick}>
				<canvas {@attach renderAttachment(pdfDoc)}></canvas>

				{#if showMarkers}
					{#each clusters as cluster (cluster.id)}
						{@const isOpen = pinnedClusterId === cluster.id || hoveredClusterId === cluster.id}
						{@const isPinned = pinnedClusterId === cluster.id}
						{@const showBelow = cluster.y < 0.25}
						{@const isFeed = cluster.items.length > 1}
						<div
							class="absolute"
							style="left: {cluster.x * 100}%; top: {cluster.y * 100}%"
							onmouseenter={() => (hoveredClusterId = cluster.id)}
							onmouseleave={() => (hoveredClusterId = null)}
						>
							<!-- svelte-ignore a11y_click_events_have_key_events -->
							<!-- svelte-ignore a11y_no_static_element_interactions -->
							<div
								class="flex h-6 w-6 -translate-x-1/2 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full bg-amber-400 text-xs font-bold text-white shadow-md ring-2 transition-transform hover:scale-125 {isPinned
									? 'ring-amber-600'
									: 'ring-white'}"
								onclick={(e) => toggleClusterPin(e, cluster.id)}
							>
								{cluster.items.length}
							</div>

							{#if isOpen}
								<!-- svelte-ignore a11y_click_events_have_key_events -->
								<!-- svelte-ignore a11y_no_static_element_interactions -->
								<div
									class="absolute left-1/2 z-20 -translate-x-1/2 rounded-lg border bg-card shadow-xl {isFeed
										? 'w-72'
										: 'w-56 p-3'} {showBelow ? 'top-full mt-2' : 'bottom-full mb-2'}"
									onclick={(e) => e.stopPropagation()}
								>
									{#if isFeed}
										<div class="max-h-64 divide-y overflow-y-auto">
											{#each cluster.items as c (c.id)}
												<div class="px-3 py-2">
													<div class="flex items-center gap-2">
														<span
															class="flex h-5 w-5 shrink-0 items-center justify-center rounded-md text-[10px] font-bold text-white {avatarColorClass(
																c.authorId
															)}"
															title={c.authorName}
														>
															{initials(c.authorName)}
														</span>
														<div class="min-w-0 flex-1">
															<p class="truncate text-xs font-medium">{c.authorName}</p>
															<p class="text-[10px] text-muted-foreground">
																{formatTime(c.createdAt)}
															</p>
														</div>
													</div>
													<p class="mt-1.5 text-sm leading-snug text-muted-foreground">
														{c.body}
													</p>
												</div>
											{/each}
										</div>
									{:else}
										{@const c = cluster.items[0]}
										<div class="flex items-center gap-2">
											<span
												class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-xs font-bold text-white {avatarColorClass(
													c.authorId
												)}"
												title={c.authorName}
											>
												{initials(c.authorName)}
											</span>
											<div class="min-w-0">
												<p class="truncate text-sm font-medium">{c.authorName}</p>
												<p class="text-xs text-muted-foreground">{formatTime(c.createdAt)}</p>
											</div>
										</div>
										<p class="mt-2 text-sm leading-snug text-muted-foreground">{c.body}</p>
									{/if}
									<!-- caret -->
									{#if showBelow}
										<div
											class="absolute bottom-full left-1/2 -translate-x-1/2 border-4 border-transparent border-b-border"
										></div>
									{:else}
										<div
											class="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-border"
										></div>
									{/if}
								</div>
							{/if}
						</div>
					{/each}
				{/if}
			</div>
		{/if}
	</div>
</div>
