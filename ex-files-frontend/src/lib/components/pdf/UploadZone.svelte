<script lang="ts">
	import { UploadCloud } from '@lucide/svelte';

	interface Props {
		onupload: (file: File) => void;
	}

	let { onupload }: Props = $props();
	let dragging = $state(false);
	let fileInput = $state<HTMLInputElement>();

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		dragging = false;
		const file = e.dataTransfer?.files[0];
		if (file?.type === 'application/pdf') {
			onupload(file);
		}
	}

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (file) {
			onupload(file);
			input.value = '';
		}
	}
</script>

<button
	type="button"
	class="flex w-full flex-col items-center justify-center rounded-lg border-2 border-dashed p-12 transition-colors
		{dragging
		? 'border-primary bg-primary/5'
		: 'border-border hover:border-muted-foreground/50 hover:bg-muted/40'}"
	ondragover={(e) => {
		e.preventDefault();
		dragging = true;
	}}
	ondragleave={() => (dragging = false)}
	ondrop={handleDrop}
	onclick={() => fileInput?.click()}
>
	<UploadCloud class="mb-3 size-10 text-muted-foreground" />
	<p class="text-sm text-muted-foreground">
		<span class="font-semibold text-primary">Click to upload</span> or drag and drop
	</p>
	<p class="mt-1 text-xs text-muted-foreground">PDF files only</p>
</button>

<input
	bind:this={fileInput}
	type="file"
	accept=".pdf,application/pdf"
	class="hidden"
	onchange={handleFileSelect}
/>
