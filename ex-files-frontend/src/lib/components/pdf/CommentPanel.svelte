<script lang="ts">
	import type { Comment } from '$lib/stores/workbench.svelte';

	interface Props {
		comments: Comment[];
		currentPage: number;
		ondelete: (id: string) => void;
		ongotopage: (page: number) => void;
	}

	let { comments, currentPage, ondelete, ongotopage }: Props = $props();

	let filter = $state<'page' | 'all'>('page');

	const visibleComments = $derived(
		filter === 'page' ? comments.filter((c) => c.page === currentPage) : [...comments]
	);

	function commentNumber(comment: Comment): number {
		const pageComments = comments.filter((c) => c.page === comment.page);
		return pageComments.indexOf(comment) + 1;
	}

	function formatTime(date: Date) {
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}
</script>

<div class="flex h-full flex-col">
	<div class="flex items-center justify-between border-b px-4 py-3">
		<h3 class="text-sm font-semibold text-gray-800">Comments ({comments.length})</h3>
		<div class="flex gap-1 rounded-md bg-gray-100 p-0.5 text-xs">
			<button
				class="rounded px-2 py-1 {filter === 'page'
					? 'bg-white font-medium shadow-sm'
					: 'text-gray-500 hover:text-gray-700'}"
				onclick={() => (filter = 'page')}
			>
				This page
			</button>
			<button
				class="rounded px-2 py-1 {filter === 'all'
					? 'bg-white font-medium shadow-sm'
					: 'text-gray-500 hover:text-gray-700'}"
				onclick={() => (filter = 'all')}
			>
				All
			</button>
		</div>
	</div>

	<div class="flex-1 overflow-y-auto">
		{#if visibleComments.length === 0}
			<div class="px-4 py-8 text-center text-sm text-gray-400">
				{filter === 'page' ? 'No comments on this page' : 'No comments yet'}
				<p class="mt-1 text-xs">Click on the PDF to add one</p>
			</div>
		{:else}
			<div class="divide-y">
				{#each visibleComments as comment (comment.id)}
					<div class="group px-4 py-3">
						<div class="flex items-start justify-between">
							<div class="flex items-center gap-2">
								<span
									class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-amber-400 text-[10px] font-bold text-white"
								>
									{commentNumber(comment)}
								</span>
								<span class="text-sm font-medium text-gray-800">{comment.author}</span>
							</div>
							<button
								class="text-gray-300 opacity-0 transition-opacity group-hover:opacity-100 hover:text-red-500"
								onclick={() => ondelete(comment.id)}
								title="Delete comment"
							>
								<svg
									class="h-4 w-4"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2"
								>
									<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
								</svg>
							</button>
						</div>
						<p class="mt-1 text-sm text-gray-600">{comment.text}</p>
						<div class="mt-1.5 flex items-center gap-2 text-xs text-gray-400">
							<span>{formatTime(comment.createdAt)}</span>
							{#if filter === 'all'}
								<button
									class="text-blue-500 hover:underline"
									onclick={() => ongotopage(comment.page)}
								>
									Page {comment.page + 1}
								</button>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>
