<script lang="ts">
	import { onMount } from 'svelte';
	import type { PDFDocumentProxy } from 'pdfjs-dist';
	import type { Comment } from '$lib/stores/workbench.svelte';

	interface Props {
		data: Uint8Array;
		comments: Comment[];
		currentPage: number;
		showMarkers: boolean;
		scale?: number;
		onpageclick: (page: number, x: number, y: number, screenX: number, screenY: number) => void;
		onpagecount: (count: number) => void;
	}

	let {
		data,
		comments,
		currentPage,
		showMarkers,
		scale = $bindable(1),
		onpageclick,
		onpagecount
	}: Props = $props();

	let canvasEl = $state<HTMLCanvasElement>();
	let containerEl = $state<HTMLDivElement>();
	let pdfDoc = $state<PDFDocumentProxy | null>(null);
	let error = $state<string | null>(null);
	let renderedKey = $state('');
	let hoveredCommentId = $state<string | null>(null);

	const pageComments = $derived(comments.filter((c) => c.page === currentPage));

	onMount(() => {
		// Polyfill Map.prototype.getOrInsertComputed for pdfjs-dist v5
		// if (!("getOrInsertComputed" in Map.prototype)) {
		// 	Map.prototype.getOrInsertComputed = function (key, callbackFn) {
		// 		if (this.has(key)) return this.get(key);
		// 		const value = callbackFn(key);
		// 		this.set(key, value);
		// 		return value;
		// 	};
		// }

		loadPdf();

		return () => {
			pdfDoc?.destroy();
		};
	});

	async function loadPdf() {
		try {
			const pdfjsLib = await import('pdfjs-dist');
			pdfjsLib.GlobalWorkerOptions.workerSrc = new URL(
				'pdfjs-dist/build/pdf.worker.mjs',
				import.meta.url
			).href;
			const doc = await pdfjsLib.getDocument({ data: data.slice() }).promise;
			pdfDoc = doc;
			onpagecount(doc.numPages);
		} catch (e) {
			console.error('Failed to load PDF:', e);
			error = e instanceof Error ? e.message : 'Failed to load PDF';
		}
	}

	$effect(() => {
		const key = `${currentPage}-${scale}`;
		if (pdfDoc && canvasEl && key !== renderedKey) {
			renderPage(pdfDoc, currentPage, scale, canvasEl);
		}
	});

	async function renderPage(
		doc: PDFDocumentProxy,
		pageNum: number,
		renderScale: number,
		canvas: HTMLCanvasElement
	) {
		const key = `${pageNum}-${renderScale}`;
		try {
			const page = await doc.getPage(pageNum + 1);
			const viewport = page.getViewport({ scale: renderScale });
			canvas.width = viewport.width;
			canvas.height = viewport.height;

			await page.render({ canvas, viewport }).promise;
			renderedKey = key;
		} catch (e) {
			console.error('Failed to render page:', e);
			error = e instanceof Error ? e.message : 'Failed to render page';
		}
	}

	function handleCanvasClick(e: MouseEvent) {
		if (!canvasEl) return;
		const rect = canvasEl.getBoundingClientRect();
		const x = ((e.clientX - rect.left) / rect.width) * 100;
		const y = ((e.clientY - rect.top) / rect.height) * 100;
		onpageclick(currentPage, x, y, e.clientX, e.clientY);
	}

	function formatTime(date: Date) {
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	function avatarColor(name: string) {
		const colors = [
			'bg-blue-500',
			'bg-emerald-500',
			'bg-violet-500',
			'bg-rose-500',
			'bg-amber-500',
			'bg-cyan-500'
		];
		let hash = 0;
		for (const ch of name) hash = ch.charCodeAt(0) + ((hash << 5) - hash);
		return colors[Math.abs(hash) % colors.length];
	}
</script>

<div class="relative flex flex-col">
	{#if error}
		<div class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<div bind:this={containerEl} class="relative flex justify-center overflow-auto bg-gray-100 p-6">
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="relative inline-block cursor-crosshair shadow-lg" onclick={handleCanvasClick}>
			<canvas bind:this={canvasEl}></canvas>

			{#if showMarkers}
				{#each pageComments as comment, i (comment.id)}
					<div
						class="absolute"
						style="left: {comment.x}%; top: {comment.y}%"
						onmouseenter={() => (hoveredCommentId = comment.id)}
						onmouseleave={() => (hoveredCommentId = null)}
					>
						<div
							class="flex h-6 w-6 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full bg-amber-400 text-xs font-bold text-white shadow-md ring-2 ring-white transition-transform hover:scale-125"
						>
							{i + 1}
						</div>

						{#if hoveredCommentId === comment.id}
							{@const showBelow = comment.y < 25}
							<div
								class="absolute left-1/2 z-20 w-56 -translate-x-1/2 rounded-lg border bg-card p-3 shadow-xl {showBelow
									? 'top-full mt-2'
									: 'bottom-full mb-2'}"
							>
								<div class="flex items-center gap-2">
									<div
										class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white {avatarColor(
											comment.author
										)}"
									>
										{comment.author.charAt(0).toUpperCase()}
									</div>
									<div class="min-w-0">
										<p class="truncate text-sm font-medium">{comment.author}</p>
										<p class="text-xs text-muted-foreground">{formatTime(comment.createdAt)}</p>
									</div>
								</div>
								<p class="mt-2 text-sm leading-snug text-muted-foreground">{comment.text}</p>
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
	</div>
</div>
