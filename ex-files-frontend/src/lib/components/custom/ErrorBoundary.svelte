<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';

	const { error }: { error: unknown } = $props();
</script>

<div class="flex flex-1 flex-col items-center justify-center gap-2 p-6 text-center">
	<p class="text-4xl font-bold text-muted-foreground">!</p>
	<p class="text-sm text-destructive">
		{(error as Error).message ?? m.error_action_failed()}
	</p>
</div>
