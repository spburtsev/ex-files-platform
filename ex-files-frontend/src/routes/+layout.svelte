<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/state';
	import { locales, localizeHref, getLocaleForUrl, type Locale } from '$lib/paraglide/runtime';
	import { Globe } from '@lucide/svelte';
	import { Toaster } from 'svelte-sonner';
	import ErrorBoundary from '$lib/components/custom/ErrorBoundary.svelte';
	import * as Select from '$lib/components/ui/select/index.js';

	let { children } = $props();

	const currentLocale = $derived.by(() => {
		try {
			return getLocaleForUrl(page.url.href);
		} catch {
			return 'en';
		}
	});

	function changeLocale(next: string) {
		if (next === currentLocale) return;
		window.location.assign(localizeHref(page.url.pathname, { locale: next as Locale }));
	}
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<svelte:boundary>
	{@render children()}
	{#snippet failed(error)}
        <ErrorBoundary {error} />
	{/snippet}
</svelte:boundary>

<div class="fixed right-4 bottom-4 z-50 flex items-center gap-1.5">
	<Globe class="size-4 text-muted-foreground" />
	<Select.Root type="single" value={currentLocale} onValueChange={changeLocale}>
		<Select.Trigger size="sm" class="min-w-14" aria-label="Language">
			{currentLocale.toUpperCase()}
		</Select.Trigger>
		<Select.Content>
			{#each locales as locale (locale)}
				<Select.Item value={locale}>{locale.toUpperCase()}</Select.Item>
			{/each}
		</Select.Content>
	</Select.Root>
</div>

<Toaster richColors closeButton position="top-right" />
