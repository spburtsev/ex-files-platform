<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import VerifyCard from '$lib/components/custom/VerifyCard.svelte';
</script>

<svelte:head>
	<title>{m.verify_page_title()}</title>
</svelte:head>

<div class="flex flex-1 justify-center p-6">
	<div class="w-full max-w-md">
		<VerifyCard showLoginLink={false} />
	</div>
</div>
