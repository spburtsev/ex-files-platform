<script lang="ts">
	import { page } from '$app/state';
	import { locales, localizeHref } from '$lib/paraglide/runtime';
	import { resolve } from '$app/paths';
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
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<svelte:window onclick={(e) => { if (menuOpen && !(e.target as Element).closest('[data-avatar-menu]')) closeMenu(); }} />

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
				{#if me?.role === 'manager'}
					<span class="rounded-full bg-violet-100 px-2.5 py-0.5 text-xs font-medium text-violet-700">
						Manager
					</span>
				{/if}

				<div class="relative" data-avatar-menu>
					<button
						onclick={toggleMenu}
						title="{me?.name ?? ''} ({me?.email ?? ''})"
						class="flex h-8 w-8 select-none items-center justify-center rounded-full bg-blue-600 text-xs font-semibold text-white transition-opacity hover:opacity-90"
					>
						{initials}
					</button>

					{#if menuOpen}
						<div
							class="absolute right-0 top-full z-50 mt-2 w-48 overflow-hidden rounded-lg border border-gray-400 bg-white shadow-lg"
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
		</div>
	</nav>

	{@render children()}
</div>

<div style="display:none">
	{#each locales as locale (locale)}
		<a href={localizeHref(page.url.pathname, { locale })}>{locale}</a>
	{/each}
</div>
