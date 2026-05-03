<script lang="ts">
	import type { Comment } from '$lib/api';
	import { m } from '$lib/paraglide/messages.js';
	import CommentItem from './CommentItem.svelte';

	interface Props {
		comments: Comment[];
		currentPage: number;
		ondelete: (id: string) => void;
		ongotopage: (page: number) => void;
	}

	let { comments, currentPage, ondelete, ongotopage }: Props = $props();

	let filter = $state<'page' | 'all'>('page');

	const visibleComments = $derived(
		filter === 'page' ? comments.filter((c) => c.metadata.page === currentPage + 1) : [...comments]
	);

	function commentNumber(comment: Comment): number {
		const pageComments = comments.filter((c) => c.metadata.page === comment.metadata.page);
		return pageComments.indexOf(comment) + 1;
	}
</script>

<div class="flex h-full flex-col">
	<div class="flex items-center justify-between border-b px-3 py-2">
		<div class="flex shrink-0 items-center gap-1.5 px-1 text-xs font-medium text-muted-foreground">
			{m.workbench_comments()}
			{#if comments.length > 0}
				<span class="rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-semibold">
					{comments.length}
				</span>
			{/if}
		</div>
		<div class="flex gap-1 rounded-md bg-muted p-0.5 text-xs">
			<button
				class="rounded px-2 py-1 {filter === 'page'
					? 'bg-background font-medium shadow-sm'
					: 'text-muted-foreground hover:text-foreground'}"
				onclick={() => (filter = 'page')}
			>
				{m.pdf_this_page()}
			</button>
			<button
				class="rounded px-2 py-1 {filter === 'all'
					? 'bg-background font-medium shadow-sm'
					: 'text-muted-foreground hover:text-foreground'}"
				onclick={() => (filter = 'all')}
			>
				{m.pdf_all()}
			</button>
		</div>
	</div>

	<div class="flex-1 overflow-y-auto">
		{#if visibleComments.length === 0}
			<div class="px-4 py-8 text-center text-sm text-muted-foreground">
				{filter === 'page' ? m.pdf_no_comments_page() : m.pdf_no_comments()}
				<p class="mt-1 text-xs">{m.pdf_click_to_add()}</p>
			</div>
		{:else}
			<div class="divide-y">
				{#each visibleComments as comment (comment.id)}
					<CommentItem {comment} commentNumber={commentNumber(comment)} {ondelete} {ongotopage} />
				{/each}
			</div>
		{/if}
	</div>
</div>
