<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
</script>

<Card.Root>
	<Card.Header class="flex flex-row items-center gap-3 pb-2">
		<Skeleton class="h-10 w-10 rounded-full" />
		<div class="min-w-0 flex-1">
			<Skeleton class="h-4 w-24 rounded" />
			<Skeleton class="mt-1.5 h-3 w-32 rounded" />
		</div>
	</Card.Header>
	<Card.Content>
		<Skeleton class="h-5 w-16 rounded-full" />
	</Card.Content>
</Card.Root>
