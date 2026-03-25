<script lang="ts">
	import { page } from '$app/state';
	import { locales, localizeHref } from '$lib/paraglide/runtime';
	import '../layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { getMe } from '$lib/data.remote';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import * as Avatar from '$lib/components/ui/avatar/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { LayoutDashboard, FolderOpen, Users, ChevronsUpDown, LogOut, User, FileCheck2 } from '@lucide/svelte';

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

	const navLinks = [
		{
			href: '/',
			label: 'Dashboard',
			Icon: LayoutDashboard,
			match: (p: string) => p === '/'
		},
		{
			href: '/workspaces',
			label: 'Workspaces',
			Icon: FolderOpen,
			match: (p: string) => p.startsWith('/workspaces')
		},
		{
			href: '/users',
			label: 'Users',
			Icon: Users,
			match: (p: string) => p.startsWith('/users')
		}
	];

	const pageLabel = $derived(navLinks.find((l) => l.match(page.url.pathname))?.label ?? 'ex-files');
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<Sidebar.Provider>
	<Sidebar.Root collapsible="icon">
		<!-- Header: brand -->
		<Sidebar.Header>
			<a href="/" class="flex items-center gap-2 overflow-hidden px-2 py-1">
				<FileCheck2 class="size-5 shrink-0 text-primary" />
				<span class="truncate text-sm font-semibold tracking-wide group-data-[collapsible=icon]:hidden">
					ex-files
				</span>
			</a>
		</Sidebar.Header>

		<!-- Nav links -->
		<Sidebar.Content>
			<Sidebar.Group>
				<Sidebar.GroupLabel>Platform</Sidebar.GroupLabel>
				<Sidebar.Menu>
					{#each navLinks as link (link.href)}
						<Sidebar.MenuItem>
							<Sidebar.MenuButton
								isActive={link.match(page.url.pathname)}
								tooltipContent={link.label}
							>
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
					<DropdownMenu.Root>
						<DropdownMenu.Trigger>
							{#snippet child({ props })}
								<Sidebar.MenuButton size="lg" tooltipContent={me?.name ?? ''} {...props}>
									<Avatar.Root class="h-8 w-8 rounded-lg">
										<Avatar.Fallback
											class="rounded-lg bg-primary text-xs font-semibold text-primary-foreground"
										>
											{initials}
										</Avatar.Fallback>
									</Avatar.Root>
									<div class="grid flex-1 text-left text-xs leading-tight">
										<span class="truncate font-semibold">{me?.name ?? ''}</span>
										{#if me?.role === 'manager'}
											<span class="text-muted-foreground">Manager</span>
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
											{initials}
										</Avatar.Fallback>
									</Avatar.Root>
									<div class="grid flex-1 text-left leading-tight">
										<div class="flex items-center gap-1.5">
											<span class="truncate font-semibold">{me?.name}</span>
											{#if me?.role === 'manager'}
												<Badge variant="secondary" class="h-4 px-1 text-[10px] text-violet-700"
													>Manager</Badge
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
									<a href="/profile" {...props}>
										<User class="size-4" />
										Profile
									</a>
								{/snippet}
							</DropdownMenu.Item>
							<DropdownMenu.Item
								class="text-destructive focus:text-destructive"
								onclick={async () => {
									await fetch('/api/auth/logout', { method: 'POST' });
									window.location.href = '/login';
								}}
							>
								<LogOut class="size-4" />
								Log out
							</DropdownMenu.Item>
						</DropdownMenu.Content>
					</DropdownMenu.Root>
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
					<Breadcrumb.Item>
						<Breadcrumb.Page>{pageLabel}</Breadcrumb.Page>
					</Breadcrumb.Item>
				</Breadcrumb.List>
			</Breadcrumb.Root>
		</header>

		{@render children()}
	</Sidebar.Inset>
</Sidebar.Provider>

<div style="display:none">
	{#each locales as locale (locale)}
		<a href={localizeHref(page.url.pathname, { locale })}>{locale}</a>
	{/each}
</div>
