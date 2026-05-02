<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';

	const { data } = $props();
	const me = $derived(data.user);
	const firstName = $derived(me?.name?.split(' ')[0] ?? '');
</script>

<svelte:head>
	<title>{m.dashboard_page_title()}</title>
</svelte:head>

<div class="flex flex-1 flex-col p-6">
	<h1 class="text-2xl font-semibold">{m.dashboard_greeting({ name: firstName })}</h1>
</div>
