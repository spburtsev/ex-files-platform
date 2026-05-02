<script lang="ts">
	import type { ActivityEntry } from '$lib/stores/workbench.svelte';
	import { Upload, MessageSquare, X, Eye } from '@lucide/svelte';
	import { m } from '$lib/paraglide/messages.js';

	interface Props {
		entries: ActivityEntry[];
	}

	let { entries }: Props = $props();

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	const icons: Record<ActivityEntry['action'], any> = {
		upload: Upload,
		comment: MessageSquare,
		delete_comment: X,
		view: Eye
	};

	const colors: Record<ActivityEntry['action'], string> = {
		upload: 'text-emerald-500',
		comment: 'text-primary',
		delete_comment: 'text-destructive',
		view: 'text-muted-foreground'
	};

	function formatTimestamp(date: Date) {
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
	}
</script>

<div class="flex h-full flex-col">
	<div class="flex-1 overflow-y-auto">
		{#if entries.length === 0}
			<div class="px-4 py-8 text-center text-sm text-muted-foreground">{m.pdf_no_activity()}</div>
		{:else}
			<div class="divide-y">
				{#each entries as entry (entry.id)}
					{@const Icon = icons[entry.action]}
					<div class="flex items-start gap-3 px-4 py-3">
						<Icon class="mt-0.5 size-4 shrink-0 {colors[entry.action]}" />
						<div class="min-w-0 flex-1">
							<p class="text-sm">{entry.description}</p>
							<span class="text-xs text-muted-foreground">{formatTimestamp(entry.timestamp)}</span>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>
