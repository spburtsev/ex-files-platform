<script lang="ts">
	import { getMe } from '$lib/data.remote';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { TriangleAlert } from '@lucide/svelte';

	const meQuery = getMe();
	const meLoading = $derived(meQuery.current === undefined);
	const me = $derived(meQuery.current?.user);
	const meError = $derived(meQuery.current?.error);
	const firstName = $derived(me?.name?.split(' ')[0] ?? '');
</script>

<svelte:head>
	<title>Dashboard — ex-files</title>
</svelte:head>

<div class="flex flex-1 flex-col p-6">
	{#if meLoading}
		<Skeleton class="h-8 w-48 rounded" />
	{:else if meError}
		<div class="flex items-center gap-2 text-destructive">
			<TriangleAlert class="size-5" />
			<p class="text-sm">{meError}</p>
		</div>
	{:else}
		<h1 class="text-2xl font-semibold">Hello, {firstName}!</h1>
	{/if}
</div>
