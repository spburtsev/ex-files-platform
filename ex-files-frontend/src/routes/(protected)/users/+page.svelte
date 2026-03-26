<script lang="ts">
	import { getUsers } from '$lib/data.remote';
	import { Role } from '$lib/gen/assignments/v1/assignments_pb';
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Avatar from '$lib/components/ui/avatar/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';

	const usersQuery = getUsers();
	const loading = $derived(usersQuery.current === undefined);
	const users = $derived(usersQuery.current ?? []);

	function initials(name: string) {
		return name
			.split(' ')
			.map((p) => p[0])
			.join('')
			.toUpperCase();
	}

	function avatarColorClass(id: string) {
		const palette = [
			'bg-blue-500',
			'bg-violet-500',
			'bg-emerald-500',
			'bg-rose-500',
			'bg-amber-500',
			'bg-cyan-500'
		];
		let hash = 0;
		for (const ch of id) hash = ch.charCodeAt(0) + ((hash << 5) - hash);
		return palette[Math.abs(hash) % palette.length];
	}
</script>

<svelte:head>
	<title>Users — ex-files</title>
</svelte:head>

<div class="flex flex-1 flex-col gap-6 p-6">
	<div>
		<h1 class="text-lg font-semibold">Users</h1>
		<p class="text-sm text-muted-foreground">Manage platform members and their roles.</p>
	</div>

	{#if loading}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each { length: 8 } as _, i (i)}
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
			{/each}
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each users as user (user.id)}
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
						{#if user.role === Role.MANAGER}
							<Badge variant="secondary" class="text-violet-700">Manager</Badge>
						{:else}
							<Badge variant="outline">Employee</Badge>
						{/if}
					</Card.Content>
				</Card.Root>
			{/each}
		</div>
	{/if}
</div>
