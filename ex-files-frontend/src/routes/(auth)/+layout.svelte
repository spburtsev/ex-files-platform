<script lang="ts">
	import { page } from '$app/state';
	import { locales, localizeHref, getLocaleForUrl } from '$lib/paraglide/runtime';
	import '../layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { Globe } from '@lucide/svelte';

	let { children } = $props();

	const currentLocale = $derived((() => { try { return getLocaleForUrl(page.url.href); } catch { return 'en'; } })());
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

{@render children()}

<div class="fixed bottom-4 right-4 z-50">
	<div class="flex items-center gap-1 rounded-full border bg-card px-2 py-1 shadow-md">
		<Globe class="size-3.5 text-muted-foreground" />
		{#each locales as locale (locale)}
			{#if currentLocale === locale}
				<span class="rounded-md px-2 py-0.5 text-xs font-medium bg-primary text-primary-foreground">
					{locale.toUpperCase()}
				</span>
			{:else}
				<a
					href={localizeHref(page.url.pathname, { locale })}
					data-sveltekit-reload
					class="rounded-md px-2 py-0.5 text-xs font-medium transition-colors text-muted-foreground hover:bg-muted"
				>
					{locale.toUpperCase()}
				</a>
			{/if}
		{/each}
	</div>
</div>
