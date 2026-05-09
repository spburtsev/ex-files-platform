<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { FileCheck2 } from '@lucide/svelte';
	import VerifyCard from '$lib/components/custom/VerifyCard.svelte';
</script>

<svelte:head>
	<title>{m.verify_page_title()}</title>
</svelte:head>

<div class="flex min-h-svh w-full items-center justify-center p-6 md:p-10">
	<div class="w-full max-w-md">
		<div class="flex flex-col gap-6">
			<a href={localizeHref('/')} class="flex items-center justify-center gap-2 self-center">
				<FileCheck2 class="size-6 text-primary" />
				<span class="text-xl font-bold tracking-tight">ex-files</span>
			</a>

			<VerifyCard />
		</div>
	</div>
</div>
