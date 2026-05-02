<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { m } from '$lib/paraglide/messages.js';
	import { ChevronLeft, ChevronRight } from '@lucide/svelte';

    type Props = {
        currentPage: number;
        totalPages: number;
        navigatePage: (page: number) => void;
    }
    const { currentPage, totalPages, navigatePage }: Props = $props();
</script>

<div class="flex items-center justify-center gap-2">
	<Button
		variant="outline"
		size="sm"
		class="gap-1"
		disabled={currentPage <= 1}
		onclick={() => navigatePage(currentPage - 1)}
	>
		<ChevronLeft class="size-4" />
		{m.common_prev()}
	</Button>
	<span class="text-sm text-muted-foreground"
		>{m.common_page_of({ current: String(currentPage), total: String(totalPages) })}</span
	>
	<Button
		variant="outline"
		size="sm"
		class="gap-1"
		disabled={currentPage >= totalPages}
		onclick={() => navigatePage(currentPage + 1)}
	>
		{m.common_next()}
		<ChevronRight class="size-4" />
	</Button>
</div>
