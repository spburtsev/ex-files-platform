<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Avatar from '$lib/components/ui/avatar/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import type { User } from '$lib/api';
	import { m } from '$lib/paraglide/messages.js';
	import { avatarColorClass, initials } from '$lib/utils';

	const { user }: { user: User } = $props();
</script>

<Card.Root>
	<Card.Header class="flex flex-row items-center gap-3 pb-2">
		<Avatar.Root class="h-10 w-10">
			<Avatar.Fallback class="text-sm font-semibold text-white {avatarColorClass(user.id)}">
				{initials(user.name)}
			</Avatar.Fallback>
		</Avatar.Root>
		<div class="min-w-0">
			<Card.Title class="truncate text-sm">{user.name}</Card.Title>
			<Card.Description class="truncate text-xs">{user.email}</Card.Description>
		</div>
	</Card.Header>
	<Card.Content>
		{#if user.role === 'manager' || user.role === 'root'}
			<Badge variant="secondary" class="text-violet-700">{m.role_manager()}</Badge>
		{:else}
			<Badge variant="outline">{m.role_employee()}</Badge>
		{/if}
	</Card.Content>
</Card.Root>
