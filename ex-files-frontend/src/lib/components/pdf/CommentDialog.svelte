<script lang="ts">
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
	let textInput = $state<HTMLTextAreaElement>();

	// Clamp so the dialog doesn't overflow the viewport
	const DIALOG_W = 288; // ~w-72
	const DIALOG_H = 160; // approximate height
	const left = $derived(Math.min(screenX, window.innerWidth - DIALOG_W - 8));
	const top = $derived(Math.min(screenY, window.innerHeight - DIALOG_H - 8));

	$effect(() => {
		textInput?.focus();
	});

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
		class="absolute w-72 rounded-xl bg-white p-4 shadow-[0_8px_32px_rgba(0,0,0,0.22),0_0_0_1.5px_rgba(0,0,0,0.08)] ring-2 ring-blue-500/30"
		style="left: {left}px; top: {top}px;"
		onclick={(e) => e.stopPropagation()}
	>
		<h4 class="mb-1 text-sm font-semibold text-gray-800">Add Comment</h4>
		<p class="mb-3 text-xs text-gray-400">
			Page {page + 1} &middot; Position ({Math.round(x)}%, {Math.round(y)}%)
		</p>

		<form onsubmit={handleSubmit} class="flex flex-col gap-3">
			<textarea
				bind:this={textInput}
				bind:value={text}
				placeholder="Write your comment..."
				rows={3}
				class="resize-none rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
			></textarea>
			<div class="flex justify-end gap-2">
				<button
					type="button"
					class="rounded-lg px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100"
					onclick={oncancel}
				>
					Cancel
				</button>
				<button
					type="submit"
					class="rounded-lg bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-40"
					disabled={!text.trim()}
				>
					Add Comment
				</button>
			</div>
		</form>
	</div>
</div>
