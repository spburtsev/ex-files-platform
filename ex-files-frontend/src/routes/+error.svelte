<script lang="ts">
	import { page } from '$app/state';
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { Button } from '$lib/components/ui/button/index.js';
	import { FileCheck2 } from '@lucide/svelte';
</script>

<div class="flex min-h-svh w-full flex-col items-center justify-center gap-4 p-6">
	<FileCheck2 class="size-10 text-muted-foreground" />
	<div class="text-center">
		<h1 class="text-4xl font-bold">{page.status}</h1>
		<p class="mt-2 text-muted-foreground">{page.error?.message ?? m.error_action_failed()}</p>
	</div>
	<Button href={localizeHref('/')} variant="outline" class="mt-4">
		{m.nav_dashboard()}
	</Button>
</div>
