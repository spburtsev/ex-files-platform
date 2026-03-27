<script lang="ts">
	import { getMe } from '$lib/data.remote';
	import { m } from '$lib/paraglide/messages.js';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';

	const meQuery = getMe();
	const me = $derived(meQuery.current);
	const firstName = $derived(me?.name?.split(' ')[0] ?? '');
</script>

<svelte:head>
	<title>{m.dashboard_page_title()}</title>
</svelte:head>

<div class="flex flex-1 flex-col p-6">
	{#if !meQuery.ready}
		<Skeleton class="h-8 w-48 rounded" />
	{:else}
		<h1 class="text-2xl font-semibold">{m.dashboard_greeting({ name: firstName })}</h1>
	{/if}
</div>
