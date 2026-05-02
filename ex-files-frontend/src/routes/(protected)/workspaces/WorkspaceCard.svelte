<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { ArrowRight, Users } from '@lucide/svelte';
	import { formatTimestamp } from '$lib/utils';
	import type { Workspace } from '$lib/api';
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { Button } from '$lib/components/ui/button';

    const { ws }: { ws: Workspace } = $props();
</script>

<Card.Root class="flex flex-col transition-shadow hover:shadow-md">
	<Card.Header>
		<div class="flex items-start justify-between gap-2">
			<Card.Title class="text-sm">{ws.name}</Card.Title>
			<Badge variant="secondary" class="shrink-0 text-xs">{m.status_active()}</Badge>
		</div>
	</Card.Header>
	<Card.Content class="pb-3">
		<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
			<Users class="size-3.5" />
			<span>{m.ws_manager_id({ id: ws.managerId })}</span>
		</div>
		<p class="mt-1 text-xs text-muted-foreground">
			{m.ws_created_date({ date: formatTimestamp(ws.createdAt) })}
		</p>
	</Card.Content>
	<Card.Footer class="mt-auto border-t pt-3">
		<Button size="sm" class="w-full gap-1.5" href={localizeHref(`/workspaces/${ws.id}`)}>
			{m.common_open()}
			<ArrowRight class="size-3.5" />
		</Button>
	</Card.Footer>
</Card.Root>
