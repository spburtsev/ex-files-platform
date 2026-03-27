<script lang="ts">
	import { page } from '$app/state';
	import { locales, localizeHref, deLocalizeHref, getLocaleForUrl } from '$lib/paraglide/runtime';
	import { m } from '$lib/paraglide/messages.js';
	import '../layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { getMe } from '$lib/data.remote';
	import { logout } from '$lib/commands.remote';
	import { isManager, initials as getInitials } from '$lib/proto-utils';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import * as Avatar from '$lib/components/ui/avatar/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import {
		LayoutDashboard,
		FolderOpen,
		Users,
		ChevronsUpDown,
		LogOut,
		User,
		FileCheck2,
		ScrollText,
		Globe
	} from '@lucide/svelte';
	import { Toaster } from 'svelte-sonner';
	import { extraBreadcrumbs } from '$lib/stores/breadcrumbs';

	let { children } = $props();

	const meQuery = getMe();
	const meLoading = $derived(!meQuery.ready);
	const me = $derived(meQuery.current);
	const meError = $derived(meQuery.error);

	const userInitials = $derived(me?.name ? getInitials(me.name) : '');

	const navLinks = $derived([
		{
			href: localizeHref('/'),
			label: m.nav_dashboard(),
			Icon: LayoutDashboard,
			match: (p: string) => p === '/'
		},
		{
			href: localizeHref('/workspaces'),
			label: m.nav_workspaces(),
			Icon: FolderOpen,
			match: (p: string) => p.startsWith('/workspaces')
		},
		{
			href: localizeHref('/users'),
			label: m.nav_users(),
			Icon: Users,
			match: (p: string) => p.startsWith('/users')
		},
		...(isManager(me?.role)
			? [
					{
						href: localizeHref('/audit'),
						label: m.nav_audit_log(),
						Icon: ScrollText,
						match: (p: string) => p.startsWith('/audit')
					}
				]
			: [])
	]);

	const cleanPathname = $derived(deLocalizeHref(page.url.pathname));
	const pageLabel = $derived(navLinks.find((l) => l.match(cleanPathname))?.label ?? 'ex-files');
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

<Sidebar.Provider>
	<Sidebar.Root collapsible="icon">
		<!-- Header: brand -->
		<Sidebar.Header>
			<a href={localizeHref('/')} class="flex items-center gap-2 overflow-hidden px-2 py-1">
				<FileCheck2 class="size-5 shrink-0 text-primary" />
				<span
					class="truncate text-sm font-semibold tracking-wide group-data-[collapsible=icon]:hidden"
				>
					ex-files
				</span>
			</a>
		</Sidebar.Header>

		<!-- Nav links -->
		<Sidebar.Content>
			<Sidebar.Group>
				<Sidebar.GroupLabel>{m.nav_platform()}</Sidebar.GroupLabel>
				<Sidebar.Menu>
					{#each navLinks as link (link.href)}
						<Sidebar.MenuItem>
							<Sidebar.MenuButton isActive={link.match(cleanPathname)} tooltipContent={link.label}>
								{#snippet child({ props })}
									<a href={link.href} {...props}>
										<link.Icon />
										<span>{link.label}</span>
									</a>
								{/snippet}
							</Sidebar.MenuButton>
						</Sidebar.MenuItem>
					{/each}
				</Sidebar.Menu>
			</Sidebar.Group>
		</Sidebar.Content>

		<!-- Footer: user menu -->
		<Sidebar.Footer>
			<Sidebar.Menu>
				<Sidebar.MenuItem>
					{#if meLoading}
						<div class="flex h-12 items-center gap-2 rounded-md px-2">
							<Skeleton class="h-8 w-8 shrink-0 rounded-lg" />
							<div class="grid flex-1 gap-1.5 group-data-[collapsible=icon]:hidden">
								<Skeleton class="h-3.5 w-24 rounded" />
								<Skeleton class="h-3 w-16 rounded" />
							</div>
						</div>
					{:else if meError}
						<div class="flex h-12 items-center gap-2 rounded-md px-2 text-destructive">
							<div
								class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-destructive/10 text-sm"
							>
								!
							</div>
							<span class="truncate text-xs group-data-[collapsible=icon]:hidden"
								>{m.nav_offline()}</span
							>
						</div>
					{:else}
						<DropdownMenu.Root>
							<DropdownMenu.Trigger>
								{#snippet child({ props })}
									<Sidebar.MenuButton size="lg" tooltipContent={me?.name ?? ''} {...props}>
										<Avatar.Root class="h-8 w-8 rounded-lg">
											<Avatar.Fallback
												class="rounded-lg bg-primary text-xs font-semibold text-primary-foreground"
											>
												{userInitials}
											</Avatar.Fallback>
										</Avatar.Root>
										<div class="grid flex-1 text-left text-xs leading-tight">
											<span class="truncate font-semibold">{me?.name ?? ''}</span>
											{#if isManager(me?.role)}
												<span class="text-muted-foreground">{m.role_manager()}</span>
											{:else}
												<span class="truncate text-muted-foreground">{me?.email ?? ''}</span>
											{/if}
										</div>
										<ChevronsUpDown class="ml-auto size-4" />
									</Sidebar.MenuButton>
								{/snippet}
							</DropdownMenu.Trigger>
							<DropdownMenu.Content
								class="w-[--bits-dropdown-menu-anchor-width] min-w-56 rounded-lg"
								side="bottom"
								align="end"
								sideOffset={4}
							>
								<DropdownMenu.Label class="p-0 font-normal">
									<div class="flex items-center gap-2 px-1 py-1.5 text-left text-xs">
										<Avatar.Root class="h-8 w-8 rounded-lg">
											<Avatar.Fallback
												class="rounded-lg bg-primary text-xs font-semibold text-primary-foreground"
											>
												{userInitials}
											</Avatar.Fallback>
										</Avatar.Root>
										<div class="grid flex-1 text-left leading-tight">
											<div class="flex items-center gap-1.5">
												<span class="truncate font-semibold">{me?.name}</span>
												{#if isManager(me?.role)}
													<Badge variant="secondary" class="h-4 px-1 text-[10px] text-violet-700"
														>{m.role_manager()}</Badge
													>
												{/if}
											</div>
											<span class="truncate text-muted-foreground">{me?.email}</span>
										</div>
									</div>
								</DropdownMenu.Label>
								<DropdownMenu.Separator />
								<DropdownMenu.Item>
									{#snippet child({ props })}
										<a href={localizeHref('/profile')} {...props}>
											<User class="size-4" />
											{m.nav_profile()}
										</a>
									{/snippet}
								</DropdownMenu.Item>
								<DropdownMenu.Item
									class="text-destructive focus:text-destructive"
									onclick={async () => {
										await logout();
										window.location.href = localizeHref('/login');
									}}
								>
									<LogOut class="size-4" />
									{m.nav_logout()}
								</DropdownMenu.Item>
							</DropdownMenu.Content>
						</DropdownMenu.Root>
					{/if}
				</Sidebar.MenuItem>
			</Sidebar.Menu>
		</Sidebar.Footer>

		<Sidebar.Rail />
	</Sidebar.Root>

	<!-- Main content area -->
	<Sidebar.Inset>
		<header class="flex h-12 shrink-0 items-center gap-2 px-4">
			<Sidebar.Trigger class="-ms-1" />
			<Separator orientation="vertical" class="me-2 data-[orientation=vertical]:h-4" />
			<Breadcrumb.Root>
				<Breadcrumb.List>
					{#if $extraBreadcrumbs.length > 0}
						<Breadcrumb.Item>
							<Breadcrumb.Link
								href={navLinks.find((l) => l.match(cleanPathname))?.href ?? localizeHref('/')}
							>
								{pageLabel}
							</Breadcrumb.Link>
						</Breadcrumb.Item>
						{#each $extraBreadcrumbs as segment (segment.label)}
							<Breadcrumb.Separator />
							<Breadcrumb.Item>
								{#if segment.href}
									<Breadcrumb.Link href={segment.href}>{segment.label}</Breadcrumb.Link>
								{:else}
									<Breadcrumb.Page>{segment.label}</Breadcrumb.Page>
								{/if}
							</Breadcrumb.Item>
						{/each}
					{:else}
						<Breadcrumb.Item>
							<Breadcrumb.Page>{pageLabel}</Breadcrumb.Page>
						</Breadcrumb.Item>
					{/if}
				</Breadcrumb.List>
			</Breadcrumb.Root>
		</header>

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
	</Sidebar.Inset>
</Sidebar.Provider>

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
