<script lang="ts">
	import { page } from '$app/state';
	import { locales, localizeHref } from '$lib/paraglide/runtime';
	import { resolve } from '$app/paths';
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import type { LayoutData } from './$types';

	let { data, children }: { data: LayoutData; children: any } = $props();

	const initials = $derived(
		data.me.name
			.split(' ')
			.map((p: string) => p[0])
			.join('')
			.toUpperCase()
	);
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

			<div class="ml-auto flex items-center gap-2">
				{#if data.me.role === 'manager'}
					<span class="rounded-full bg-violet-100 px-2.5 py-0.5 text-xs font-medium text-violet-700">
						Manager
					</span>
				{/if}
				<div
					title="{data.me.name} ({data.me.email})"
					class="flex h-8 w-8 select-none items-center justify-center rounded-full bg-blue-600 text-xs font-semibold text-white"
				>
					{initials}
				</div>
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
