<script lang="ts">
	import type { Comment } from '$lib/api';
	import { Button } from '$lib/components/ui/button/index.js';
	import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
	import { Trash2 } from '@lucide/svelte';
	import { m } from '$lib/paraglide/messages.js';

	interface Props {
		comment: Comment;
		commentNumber: number;
		ondelete: (id: string) => void;
		ongotopage: (page: number) => void;
	}
	let { comment, commentNumber, ondelete, ongotopage }: Props = $props();

	let confirmOpen = $state(false);

	function formatTime(iso: string) {
		return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}
</script>

<div class="px-4 py-3">
	<div class="flex items-start justify-between">
		<div class="flex items-center gap-2">
			<span
				class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-amber-400 text-[10px] font-bold text-white"
			>
				{commentNumber}
			</span>
			<p class="text-sm">{comment.body}</p>
		</div>
		<Button
			variant="ghost"
			size="icon"
			class="size-6 hover:text-destructive"
			onclick={() => (confirmOpen = true)}
		>
			<Trash2 class="size-3" />
		</Button>
	</div>
	<div class="mt-1.5 flex items-center gap-2 text-[10px] text-muted-foreground">
		<span>{comment.authorName}, {formatTime(comment.createdAt)}</span>
		<button class="text-primary hover:underline" onclick={() => ongotopage(comment.metadata.page - 1)}>
			{m.pdf_page_label({ page: String(comment.metadata.page) })}
		</button>
	</div>
</div>

<AlertDialog.Root bind:open={confirmOpen}>
	<AlertDialog.Content class="sm:max-w-sm">
		<AlertDialog.Header>
			<AlertDialog.Title>{m.comment_delete_confirm_title()}</AlertDialog.Title>
			<AlertDialog.Description>{m.comment_delete_confirm_desc()}</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>{m.common_cancel()}</AlertDialog.Cancel>
			<AlertDialog.Action onclick={() => ondelete(comment.id)}>
				{m.common_delete()}
			</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
