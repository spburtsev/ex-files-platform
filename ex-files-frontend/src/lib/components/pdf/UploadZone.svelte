<script lang="ts">
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
		? 'border-blue-500 bg-blue-50'
		: 'border-gray-300 hover:border-gray-400 hover:bg-gray-50'}"
	ondragover={(e) => {
		e.preventDefault();
		dragging = true;
	}}
	ondragleave={() => (dragging = false)}
	ondrop={handleDrop}
	onclick={() => fileInput?.click()}
>
	<svg
		class="mb-3 h-10 w-10 text-gray-400"
		fill="none"
		viewBox="0 0 24 24"
		stroke="currentColor"
		stroke-width="1.5"
	>
		<path
			stroke-linecap="round"
			stroke-linejoin="round"
			d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m6.75 12-3-3m0 0-3 3m3-3v6m-1.5-15H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z"
		/>
	</svg>
	<p class="text-sm text-gray-600">
		<span class="font-semibold text-blue-600">Click to upload</span> or drag and drop
	</p>
	<p class="mt-1 text-xs text-gray-500">PDF files only</p>
</button>

<input
	bind:this={fileInput}
	type="file"
	accept=".pdf,application/pdf"
	class="hidden"
	onchange={handleFileSelect}
/>
