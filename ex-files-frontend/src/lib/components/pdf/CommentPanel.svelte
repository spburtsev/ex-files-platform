<script lang="ts">
	import type { Comment } from '$lib/stores/workbench.svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import { X } from '@lucide/svelte';
	import { m } from '$lib/paraglide/messages.js';

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
		<h3 class="text-sm font-semibold">{m.pdf_comments_count({ count: String(comments.length) })}</h3>
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
					<div class="group px-4 py-3">
						<div class="flex items-start justify-between">
							<div class="flex items-center gap-2">
								<span
									class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-amber-400 text-[10px] font-bold text-white"
								>
									{commentNumber(comment)}
								</span>
								<span class="text-sm font-medium">{comment.author}</span>
							</div>
							<Button
								variant="ghost"
								size="icon"
								class="size-6 opacity-0 transition-opacity group-hover:opacity-100 hover:text-destructive"
								onclick={() => ondelete(comment.id)}
							>
								<X class="size-3" />
							</Button>
						</div>
						<p class="mt-1 text-sm text-muted-foreground">{comment.text}</p>
						<div class="mt-1.5 flex items-center gap-2 text-xs text-muted-foreground">
							<span>{formatTime(comment.createdAt)}</span>
							{#if filter === 'all'}
								<button
									class="text-primary hover:underline"
									onclick={() => ongotopage(comment.page)}
								>
									{m.pdf_page_label({ page: String(comment.page + 1) })}
								</button>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>
