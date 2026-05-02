<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/state';
	import { locales, localizeHref, getLocaleForUrl } from '$lib/paraglide/runtime';
	import { Globe } from '@lucide/svelte';
	import { Toaster } from 'svelte-sonner';
	import { m } from '$lib/paraglide/messages';

	let { children } = $props();

	const currentLocale = $derived(
		(() => {
			try {
				return getLocaleForUrl(page.url.href);
			} catch {
				return 'en';
			}
		})()
	);
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<svelte:boundary>
	{@render children()}
	{#snippet failed(error)}
		<div class="flex flex-1 flex-col items-center justify-center gap-2 p-6 text-center">
			<p class="text-4xl font-bold text-muted-foreground">!</p>
			<p class="text-sm text-destructive">
				{(error as Error).message ?? m.error_action_failed()}
			</p>
		</div>
	{/snippet}
</svelte:boundary>

<div class="fixed right-4 bottom-4 z-50">
	<div class="flex items-center gap-1 rounded-full border bg-card px-2 py-1 shadow-md">
		<Globe class="size-3.5 text-muted-foreground" />
		{#each locales as locale (locale)}
			{#if currentLocale === locale}
				<span class="rounded-md bg-primary px-2 py-0.5 text-xs font-medium text-primary-foreground">
					{locale.toUpperCase()}
				</span>
			{:else}
				<a
					href={localizeHref(page.url.pathname, { locale })}
					data-sveltekit-reload
					class="rounded-md px-2 py-0.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted"
				>
					{locale.toUpperCase()}
				</a>
			{/if}
		{/each}
	</div>
</div>

<Toaster richColors closeButton position="top-right" />
