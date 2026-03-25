<script lang="ts">
	import { page } from '$app/state';
	import { locales, localizeHref } from '$lib/paraglide/runtime';
	import { resolve } from '$app/paths';
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="flex min-h-screen flex-col">
	<nav class="border-b bg-white">
		<div class="mx-auto flex max-w-6xl items-center gap-6 px-6 py-3">
			<a href={resolve('/')} class="text-sm font-semibold tracking-wide text-gray-900">ex-files</a>
			<div class="flex items-center gap-1">
				<a
					href={resolve('/users')}
					class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors {page.url.pathname.startsWith(
						'/users'
					)
						? 'bg-blue-50 text-blue-700'
						: 'text-gray-500 hover:bg-gray-100 hover:text-gray-800'}"
				>
					Users
				</a>
				<a
					href={resolve('/assignments')}
					class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors {page.url.pathname.startsWith(
						'/assignments'
					)
						? 'bg-blue-50 text-blue-700'
						: 'text-gray-500 hover:bg-gray-100 hover:text-gray-800'}"
				>
					Assignments
				</a>
			</div>
		</div>
	</nav>

	{@render children()}
</div>

<div style="display:none">
	{#each locales as locale (locale)}
		<a href={localizeHref(page.url.pathname, { locale })}>{locale}</a>
	{/each}
</div>
