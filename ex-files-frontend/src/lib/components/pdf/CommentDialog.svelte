<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';

	interface Props {
		page: number;
		x: number;
		y: number;
		screenX: number;
		screenY: number;
		onsubmit: (text: string) => void;
		oncancel: () => void;
	}

	let { page, x, y, screenX, screenY, onsubmit, oncancel }: Props = $props();

	let text = $state('');

	// Clamp so the dialog doesn't overflow the viewport
	const DIALOG_W = 288; // ~w-72
	const DIALOG_H = 160; // approximate height
	const left = $derived(Math.min(screenX, window.innerWidth - DIALOG_W - 8));
	const top = $derived(Math.min(screenY, window.innerHeight - DIALOG_H - 8));

	function handleSubmit(e: Event) {
		e.preventDefault();
		if (!text.trim()) return;
		onsubmit(text.trim());
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			oncancel();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="fixed inset-0 z-50" onclick={oncancel}>
	<div
		class="absolute w-72 rounded-xl border bg-card p-4 shadow-xl ring-2 ring-primary/20"
		style="left: {left}px; top: {top}px;"
		onclick={(e) => e.stopPropagation()}
	>
		<h4 class="mb-1 text-sm font-semibold">Add Comment</h4>
		<p class="mb-3 text-xs text-muted-foreground">
			Page {page + 1} &middot; Position ({Math.round(x)}%, {Math.round(y)}%)
		</p>

		<form onsubmit={handleSubmit} class="flex flex-col gap-3">
			<Textarea
				autofocus
				bind:value={text}
				placeholder="Write your comment..."
				rows={3}
				class="resize-none text-sm"
			/>
			<div class="flex justify-end gap-2">
				<Button type="button" variant="ghost" size="sm" onclick={oncancel}>Cancel</Button>
				<Button type="submit" size="sm" disabled={!text.trim()}>Add Comment</Button>
			</div>
		</form>
	</div>
</div>
