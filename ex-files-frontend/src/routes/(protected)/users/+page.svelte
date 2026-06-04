<script lang="ts">
	import { getUsers } from '$lib/queries.remote';
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { Button } from '$lib/components/ui/button/index.js';
	import { UserPlus } from '@lucide/svelte';
	import UserSkeleton from './UserSkeleton.svelte';
	import UserCard from './UserCard.svelte';

	const { data } = $props();
	const me = $derived(data.user);

	const usersQuery = getUsers();
	const users = $derived(usersQuery.current ?? []);
</script>

<svelte:head>
	<title>{m.users_page_title()}</title>
</svelte:head>

<div class="flex flex-1 flex-col gap-6 p-6">
	<div class="flex items-start justify-between gap-4">
		<div>
			<h1 class="text-lg font-semibold">{m.users_heading()}</h1>
			<p class="text-sm text-muted-foreground">{m.users_description()}</p>
		</div>
		{#if me?.role === 'root'}
			<Button href={localizeHref('/users/create')} class="shrink-0 gap-1.5">
				<UserPlus class="size-4" />
				{m.users_new()}
			</Button>
		{/if}
	</div>

	{#if !usersQuery.ready}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each { length: 8 } as _, i (i)}
				<UserSkeleton />
			{/each}
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#if users.length === 0}
				<p class="text-sm text-muted-foreground italic">No users found</p>
			{:else}
				{#each users as user (user.id)}
					<UserCard user={user} />
				{/each}
			{/if}
		</div>
	{/if}
</div>
