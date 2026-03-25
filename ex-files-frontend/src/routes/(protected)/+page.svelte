<script lang="ts">
	import { getAssignments, getUsers } from '$lib/data.remote';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { MessageSquare, FileText, ArrowRight, CheckCircle2, Clock } from '@lucide/svelte';

	const assignmentsQuery = getAssignments();
	const usersQuery = getUsers();
	const assignments = $derived(assignmentsQuery.current ?? []);
	const users = $derived(usersQuery.current ?? []);

	let selectedUserId = $state<string | null>(null);
	let statusFilter = $state<'all' | 'pending' | 'completed'>('pending');

	const filtered = $derived(
		assignments.filter((a) => {
			if (selectedUserId && a.assigneeId !== selectedUserId) return false;
			if (statusFilter === 'pending') return !a.resolved;
			if (statusFilter === 'completed') return a.resolved;
			return true;
		})
	);

	const stats = $derived({
		total: assignments.length,
		pending: assignments.filter((a) => !a.resolved).length,
		completed: assignments.filter((a) => a.resolved).length,
		overdue: assignments.filter(
			(a) => !a.resolved && a.deadline && new Date(a.deadline) < new Date()
		).length
	});

	const statCards = [
		{ label: 'Total', key: 'total' as const, valueClass: 'text-foreground' },
		{ label: 'Pending', key: 'pending' as const, valueClass: 'text-amber-600' },
		{ label: 'Completed', key: 'completed' as const, valueClass: 'text-emerald-600' },
		{ label: 'Overdue', key: 'overdue' as const, valueClass: 'text-destructive' }
	];

	const statusOptions: { val: 'all' | 'pending' | 'completed'; label: string }[] = [
		{ val: 'pending', label: 'Pending' },
		{ val: 'completed', label: 'Completed' },
		{ val: 'all', label: 'All' }
	];

	function userName(userId: string) {
		return users.find((u) => u.id === userId)?.name ?? 'Unknown';
	}

	function userInitials(userId: string) {
		return userName(userId)
			.split(' ')
			.map((p) => p[0])
			.join('')
			.toUpperCase();
	}

	function avatarColorClass(userId: string) {
		const palette = [
			'bg-blue-500',
			'bg-violet-500',
			'bg-emerald-500',
			'bg-rose-500',
			'bg-amber-500',
			'bg-cyan-500'
		];
		let hash = 0;
		for (const ch of userId) hash = ch.charCodeAt(0) + ((hash << 5) - hash);
		return palette[Math.abs(hash) % palette.length];
	}

	function deadlineInfo(iso: string) {
		const h = (new Date(iso).getTime() - Date.now()) / 3_600_000;
		if (h < 0) return { label: 'Overdue', cls: 'border-red-200 bg-red-50 text-red-600' };
		if (h < 24)
			return { label: `${Math.round(h)}h left`, cls: 'border-red-200 bg-red-50 text-red-600' };
		if (h < 72)
			return {
				label: `${Math.floor(h / 24)}d left`,
				cls: 'border-amber-200 bg-amber-50 text-amber-700'
			};
		return {
			label: new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
			cls: 'border-border bg-muted/40 text-muted-foreground'
		};
	}
</script>

<svelte:head>
	<title>Dashboard — ex-files</title>
</svelte:head>

<div class="flex flex-1 flex-col gap-6 p-6">
	<!-- Stat cards -->
	<div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
		{#each statCards as card (card.key)}
			<Card.Root>
				<Card.Header class="pb-1">
					<Card.Description>{card.label}</Card.Description>
				</Card.Header>
				<Card.Content>
					<p class="text-3xl font-bold {card.valueClass}">{stats[card.key]}</p>
				</Card.Content>
			</Card.Root>
		{/each}
	</div>

	<!-- Filters -->
	<div class="flex flex-wrap items-center gap-3">
		<div class="flex rounded-lg border bg-card p-0.5 shadow-sm">
			{#each statusOptions as opt (opt.val)}
				<Button
					variant={statusFilter === opt.val ? 'default' : 'ghost'}
					size="sm"
					class="h-7 rounded-md px-3 text-xs"
					onclick={() => (statusFilter = opt.val)}
				>
					{opt.label}
				</Button>
			{/each}
		</div>

		<div class="flex flex-wrap gap-2">
			<Button
				variant={selectedUserId === null ? 'secondary' : 'outline'}
				size="sm"
				class="h-7 rounded-full text-xs"
				onclick={() => (selectedUserId = null)}
			>
				All assignees
			</Button>
			{#each users as user (user.id)}
				<Button
					variant={selectedUserId === user.id ? 'secondary' : 'outline'}
					size="sm"
					class="h-7 gap-1.5 rounded-full text-xs"
					onclick={() => (selectedUserId = selectedUserId === user.id ? null : user.id)}
				>
					<span
						class="flex h-4 w-4 shrink-0 items-center justify-center rounded-full text-[9px] font-bold text-white {avatarColorClass(
							user.id
						)}"
					>
						{userInitials(user.id)}
					</span>
					{user.name.split(' ')[0]}
				</Button>
			{/each}
		</div>
	</div>

	<!-- Assignment cards -->
	{#if filtered.length === 0}
		<Card.Root class="flex flex-col items-center justify-center py-16 text-center">
			<Card.Content>
				<p class="text-sm text-muted-foreground">No assignments match the current filter</p>
			</Card.Content>
		</Card.Root>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each filtered as assignment (assignment.id)}
				{@const dl =
					assignment.deadline && !assignment.resolved ? deadlineInfo(assignment.deadline) : null}
				<Card.Root class="flex flex-col transition-shadow hover:shadow-md">
					<Card.Header class="pb-2">
						<div class="flex items-start justify-between gap-3">
							<div class="min-w-0">
								<Card.Title class="truncate text-sm">{assignment.title}</Card.Title>
								<Card.Description class="mt-1 line-clamp-2 text-xs leading-relaxed">
									{assignment.description}
								</Card.Description>
							</div>
							{#if assignment.resolved}
								<Badge
									variant="secondary"
									class="shrink-0 gap-1 bg-emerald-100 text-emerald-700 hover:bg-emerald-100"
								>
									<CheckCircle2 class="size-3" />
									Done
								</Badge>
							{:else}
								<Badge
									variant="secondary"
									class="shrink-0 gap-1 bg-amber-100 text-amber-700 hover:bg-amber-100"
								>
									<Clock class="size-3" />
									Pending
								</Badge>
							{/if}
						</div>
					</Card.Header>

					<Card.Content class="pb-3">
						<div class="flex flex-wrap items-center gap-2">
							<span
								class="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground"
							>
								<span
									class="flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-full text-[8px] font-bold text-white {avatarColorClass(
										assignment.assigneeId
									)}"
								>
									{userInitials(assignment.assigneeId)}
								</span>
								{userName(assignment.assigneeId)}
							</span>
							{#if dl}
								<Badge variant="outline" class="text-[11px] {dl.cls}">{dl.label}</Badge>
							{/if}
						</div>
					</Card.Content>

					<Card.Footer class="mt-auto border-t pt-3">
						<div class="flex w-full items-center justify-between">
							<div class="flex items-center gap-3 text-[11px] text-muted-foreground">
								<span class="flex items-center gap-1">
									<MessageSquare class="size-3.5" />
									{assignment.commentsCount}
								</span>
								<span class="flex items-center gap-1">
									<FileText class="size-3.5" />
									{assignment.versionsCount}
								</span>
							</div>
							<Button size="sm" href="/workbench" class="h-7 gap-1 text-xs">
								Review
								<ArrowRight class="size-3" />
							</Button>
						</div>
					</Card.Footer>
				</Card.Root>
			{/each}
		</div>
	{/if}
</div>
