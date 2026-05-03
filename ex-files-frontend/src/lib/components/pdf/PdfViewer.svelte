<script lang="ts">
	import type { PDFDocumentProxy } from 'pdfjs-dist';
	import type { Attachment } from 'svelte/attachments';
	import { getPdfjs } from '$lib/pdf/pdfjs';
	import type { Comment } from '$lib/api';

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
	let hoveredCommentId = $state<string | null>(null);

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

	const pageComments = $derived(comments.filter((c) => c.metadata.page === currentPage+1));

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
					} catch (e) {
						if (!cancelled) {
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
		if (!canvasRef) return;
		const rect = canvasRef.getBoundingClientRect();
		const x = (e.clientX - rect.left) / rect.width;
		const y = (e.clientY - rect.top) / rect.height;
		onpageclick(currentPage, x, y, e.clientX, e.clientY);
	}

	function formatTime(iso: string) {
		return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
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

	<div class="relative flex justify-center overflow-auto bg-gray-100 p-6">
		{#if pdfDoc}
			<!-- svelte-ignore a11y_click_events_have_key_events -->
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div class="relative inline-block cursor-crosshair shadow-lg" onclick={handleCanvasClick}>
				<canvas {@attach renderAttachment(pdfDoc)}></canvas>

				{#if showMarkers}
					{#each pageComments as comment, i (comment.id)}
						<div
							class="absolute"
							style="left: {comment.metadata.x * 100}%; top: {comment.metadata.y * 100}%"
							onmouseenter={() => (hoveredCommentId = comment.id)}
							onmouseleave={() => (hoveredCommentId = null)}
						>
							<div
								class="flex h-6 w-6 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full bg-amber-400 text-xs font-bold text-white shadow-md ring-2 ring-white transition-transform hover:scale-125"
							>
								{i + 1}
							</div>

							{#if hoveredCommentId === comment.id}
								{@const showBelow = comment.metadata.y < 0.25}
								<div
									class="absolute left-1/2 z-20 w-56 -translate-x-1/2 rounded-lg border bg-card p-3 shadow-xl {showBelow
										? 'top-full mt-2'
										: 'bottom-full mb-2'}"
								>
									<div class="flex items-center gap-2">
										<div
											class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white {avatarColor(
												comment.authorName
											)}"
										>
											{comment.authorName.charAt(0).toUpperCase()}
										</div>
										<div class="min-w-0">
											<p class="truncate text-sm font-medium">{comment.authorName}</p>
											<p class="text-xs text-muted-foreground">{formatTime(comment.createdAt)}</p>
										</div>
									</div>
									<p class="mt-2 text-sm leading-snug text-muted-foreground">{comment.body}</p>
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
