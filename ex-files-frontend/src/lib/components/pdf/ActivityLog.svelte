<script lang="ts">
	import type { ActivityEntry } from '$lib/stores/workbench.svelte';

	interface Props {
		entries: ActivityEntry[];
	}

	let { entries }: Props = $props();

	const icons: Record<ActivityEntry['action'], string> = {
		upload:
			'M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5m-13.5-9L12 3m0 0 4.5 4.5M12 3v13.5',
		comment:
			'M7.5 8.25h9m-9 3H12m-9.75 1.51c0 1.6 1.123 2.994 2.707 3.227 1.129.166 2.27.293 3.423.379.35.026.67.21.865.501L12 21l2.755-4.133a1.14 1.14 0 0 1 .865-.501 48.172 48.172 0 0 0 3.423-.379c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0 0 12 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018Z',
		delete_comment: 'M6 18L18 6M6 6l12 12',
		view: 'M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z'
	};

	const colors: Record<ActivityEntry['action'], string> = {
		upload: 'text-green-500',
		comment: 'text-blue-500',
		delete_comment: 'text-red-400',
		view: 'text-gray-400'
	};

	function formatTimestamp(date: Date) {
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
	}
</script>

<div class="flex h-full flex-col">
	<div class="border-b px-4 py-3">
		<h3 class="text-sm font-semibold text-gray-800">Activity Log ({entries.length})</h3>
	</div>

	<div class="flex-1 overflow-y-auto">
		{#if entries.length === 0}
			<div class="px-4 py-8 text-center text-sm text-gray-400">No activity yet</div>
		{:else}
			<div class="divide-y">
				{#each entries as entry (entry.id)}
					<div class="flex items-start gap-3 px-4 py-3">
						<svg
							class="mt-0.5 h-4 w-4 shrink-0 {colors[entry.action]}"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.5"
						>
							<path stroke-linecap="round" stroke-linejoin="round" d={icons[entry.action]} />
						</svg>
						<div class="min-w-0 flex-1">
							<p class="text-sm text-gray-700">{entry.description}</p>
							<span class="text-xs text-gray-400">{formatTimestamp(entry.timestamp)}</span>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>
