<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
</script>

<Card.Root class="flex flex-col">
	<Card.Header>
		<div class="flex items-start justify-between gap-2">
			<Skeleton class="h-4 w-32 rounded" />
			<Skeleton class="h-5 w-14 rounded-full" />
		</div>
	</Card.Header>
	<Card.Content class="pb-3">
		<Skeleton class="h-3 w-28 rounded" />
		<Skeleton class="mt-2 h-3 w-36 rounded" />
	</Card.Content>
	<Card.Footer class="mt-auto border-t pt-3">
		<Skeleton class="h-8 w-full rounded-md" />
	</Card.Footer>
</Card.Root>
