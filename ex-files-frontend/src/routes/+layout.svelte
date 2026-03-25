<script lang="ts">
	import { page } from '$app/state';
	import { locales, localizeHref } from '$lib/paraglide/runtime';
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { getMe } from '$lib/data.remote';

	let { children } = $props();

	const meQuery = getMe();
	const me = $derived(meQuery.current);

	const initials = $derived(
		me?.name
			.split(' ')
			.map((p) => p[0])
			.join('')
			.toUpperCase() ?? ''
	);

	let menuOpen = $state(false);

	function toggleMenu() {
		menuOpen = !menuOpen;
	}

	function closeMenu() {
		menuOpen = false;
	}

	const navLinks = [
		{
			href: '/',
			label: 'Dashboard',
			icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6',
			match: (p: string) => p === '/'
		},
		{
			href: '/workspaces',
			label: 'Workspaces',
			icon: 'M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V7z',
			match: (p: string) => p.startsWith('/workspaces')
		},
		{
			href: '/users',
			label: 'Users',
			icon: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z',
			match: (p: string) => p.startsWith('/users')
		}
	];
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<svelte:window
	onclick={(e) => {
		if (menuOpen && !(e.target as Element).closest('[data-avatar-menu]')) closeMenu();
	}}
/>

<div class="flex min-h-screen">
	<!-- Sidebar -->
	<aside class="flex w-56 shrink-0 flex-col border-r bg-white">
		<!-- Logo -->
		<div class="border-b px-5 py-4">
			<a href="/" class="text-sm font-semibold tracking-wide text-gray-900">ex-files</a>
		</div>

		<!-- Nav links -->
		<nav class="flex flex-1 flex-col gap-0.5 px-3 py-3">
			{#each navLinks as link}
				{@const active = link.match(page.url.pathname)}
				<a
					href={link.href}
					class="flex items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium transition-colors {active
						? 'bg-blue-50 text-blue-700'
						: 'text-gray-500 hover:bg-gray-100 hover:text-gray-800'}"
				>
					<svg
						class="h-4 w-4 shrink-0"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="1.5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d={link.icon} />
					</svg>
					{link.label}
				</a>
			{/each}
		</nav>

		<!-- User section -->
		<div class="border-t px-3 py-3">
			<div class="relative" data-avatar-menu>
				<button
					onclick={toggleMenu}
					class="flex w-full items-center gap-2.5 rounded-md px-2 py-2 transition-colors hover:bg-gray-100"
				>
					<div
						class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-blue-600 text-xs font-semibold text-white"
					>
						{initials}
					</div>
					<div class="min-w-0 flex-1 text-left">
						<p class="truncate text-sm font-medium text-gray-900">{me?.name ?? ''}</p>
						{#if me?.role === 'manager'}
							<span class="text-xs font-medium text-violet-600">Manager</span>
						{:else}
							<p class="truncate text-xs text-gray-500">{me?.email ?? ''}</p>
						{/if}
					</div>
				</button>

				{#if menuOpen}
					<div
						class="absolute bottom-full left-0 z-50 mb-2 w-48 overflow-hidden rounded-lg border border-gray-400 bg-white shadow-lg"
					>
						<div class="border-b px-4 py-3">
							<p class="truncate text-sm font-medium text-gray-900">{me?.name}</p>
							<p class="truncate text-xs text-gray-500">{me?.email}</p>
						</div>
						<div class="py-1">
							<a
								href="/profile"
								onclick={closeMenu}
								class="flex items-center gap-2.5 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
							>
								<svg
									class="h-4 w-4 shrink-0 text-gray-400"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="1.5"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z"
									/>
								</svg>
								Profile
							</a>
							<button
								onclick={closeMenu}
								class="flex w-full items-center gap-2.5 px-4 py-2 text-sm text-red-600 hover:bg-red-50"
							>
								<svg
									class="h-4 w-4 shrink-0"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="1.5"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M8.25 9V5.25A2.25 2.25 0 0 1 10.5 3h6a2.25 2.25 0 0 1 2.25 2.25v13.5A2.25 2.25 0 0 1 16.5 21h-6a2.25 2.25 0 0 1-2.25-2.25V15m-3 0-3-3m0 0 3-3m-3 3H15"
									/>
								</svg>
								Logout
							</button>
						</div>
					</div>
				{/if}
			</div>
		</div>
	</aside>

	<!-- Main content -->
	<main class="flex min-w-0 flex-1 flex-col">
		{@render children()}
	</main>
</div>

<div style="display:none">
	{#each locales as locale (locale)}
		<a href={localizeHref(page.url.pathname, { locale })}>{locale}</a>
	{/each}
</div>
