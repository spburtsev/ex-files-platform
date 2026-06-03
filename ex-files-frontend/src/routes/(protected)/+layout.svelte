<script lang="ts">
	import { page } from '$app/state';
	import { localizeHref, deLocalizeHref } from '$lib/paraglide/runtime';
	import { m } from '$lib/paraglide/messages.js';
	import { logout } from '$lib/commands.remote';
	import { isManager, initials as getInitials, roleLabel } from '$lib/utils';
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
		BadgeCheck
	} from '@lucide/svelte';
	import { extraBreadcrumbs } from '$lib/stores/breadcrumbs.svelte';
	import ErrorBoundary from '$lib/components/custom/ErrorBoundary.svelte';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { toast } from 'svelte-sonner';
	import { getDashboard } from '$lib/queries.remote';

	let { children, data } = $props();

	const me = $derived(data.user);
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
		{
			href: localizeHref('/check'),
			label: m.nav_verify(),
			Icon: BadgeCheck,
			match: (p: string) => p.startsWith('/check')
		}
	]);

	// Global notifications: subscribe to the SSE stream scoped to the current
	// user (the backend derives the userID from the session). Review-feedback
	// events and deadline-reminder events route to toasts; other events
	// (comment.added/.deleted) are filtered by the per-page subscription on
	// the issue/workbench page and ignored here.
	const REVIEW_TOAST_TYPES = new Set([
		'document.approved',
		'document.rejected',
		'document.changes_requested',
		'document.reviewer_assigned'
	]);
	$effect(() => {
		if (!browser || !me) return;
		const es = new EventSource('/api/events');
		es.onmessage = (e) => {
			try {
				const ev = JSON.parse(e.data);
				const t = ev?.type;
				const p = ev?.payload ?? {};
				if (typeof t !== 'string') return;

				if (REVIEW_TOAST_TYPES.has(t)) {
					const name = String(p.name ?? '');
					const wsId = p.workspace_id != null ? String(p.workspace_id) : null;
					const issueId = p.issue_id != null ? String(p.issue_id) : null;
					const action =
						wsId && issueId
							? {
									label: m.notif_view_action(),
									onClick: () =>
										goto(localizeHref(`/workspaces/${wsId}/issues/${issueId}`))
								}
							: undefined;
					switch (t) {
						case 'document.approved':
							toast.success(m.notif_doc_approved({ name }), { action });
							break;
						case 'document.rejected':
							toast.error(m.notif_doc_rejected({ name }), { action });
							break;
						case 'document.changes_requested':
							toast.warning(m.notif_doc_changes_requested({ name }), { action });
							break;
						case 'document.reviewer_assigned':
							toast.info(m.notif_reviewer_assigned({ name }), { action });
							break;
					}
					// Keep the dashboard counts/lists in sync.
					void getDashboard().refresh();
					return;
				}

				if (t === 'deadline.approaching') {
					const title = String(p.title ?? '');
					const wsId = p.workspace_id != null ? String(p.workspace_id) : null;
					const issueId = p.issue_id != null ? String(p.issue_id) : null;
					const action =
						wsId && issueId
							? {
									label: m.notif_view_action(),
									onClick: () =>
										goto(localizeHref(`/workspaces/${wsId}/issues/${issueId}`))
								}
							: undefined;
					switch (p.threshold) {
						case 'overdue':
							toast.error(m.notif_deadline_overdue({ title }), { action });
							break;
						case '1h':
							toast.warning(m.notif_deadline_1h({ title }), { action });
							break;
						case '24h':
							toast.info(m.notif_deadline_24h({ title }), { action });
							break;
					}
					void getDashboard().refresh();
				}
			} catch (err) {
				console.error('global SSE parse error', err);
			}
		};
		return () => es.close();
	});

	const cleanPathname = $derived(deLocalizeHref(page.url.pathname));
	const pageLabel = $derived.by(() => {
		const fromNav = navLinks.find((l) => l.match(cleanPathname))?.label;
		if (fromNav) return fromNav;
		if (cleanPathname === '/profile') return m.nav_profile();

		return 'ex-files';
	});
</script>

<Sidebar.Provider open={data.sidebarOpen}>
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
											<span class="text-muted-foreground">{roleLabel(me?.role)}</span>
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
													>{roleLabel(me?.role)}</Badge
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
					{#if extraBreadcrumbs.segments.length > 0}
						<Breadcrumb.Item>
							<Breadcrumb.Link
								href={navLinks.find((l) => l.match(cleanPathname))?.href ?? localizeHref('/')}
							>
								{pageLabel}
							</Breadcrumb.Link>
						</Breadcrumb.Item>
						{#each extraBreadcrumbs.segments as segment (segment.label)}
							<Breadcrumb.Separator />
							<Breadcrumb.Item class="gap-2">
								{#if segment.href}
									<Breadcrumb.Link href={segment.href}>{segment.label}</Breadcrumb.Link>
								{:else}
									<Breadcrumb.Page>{segment.label}</Breadcrumb.Page>
								{/if}
								{#if segment.badges?.length}
									{#each segment.badges as b (b.label)}
										<Badge variant="secondary" class="text-[10px] {b.cls ?? ''}">
											{b.label}
										</Badge>
									{/each}
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
				<ErrorBoundary {error} />
			{/snippet}
		</svelte:boundary>
	</Sidebar.Inset>
</Sidebar.Provider>
