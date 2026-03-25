<script lang="ts">
	import { resolve } from '$app/paths';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	let selectedUserId = $state<string | null>(null);
	let statusFilter = $state<'all' | 'pending' | 'completed'>('pending');

	const filtered = $derived(
		data.assignments.filter((a) => {
			if (selectedUserId && a.assigneeId !== selectedUserId) return false;
			if (statusFilter === 'pending') return !a.resolved;
			if (statusFilter === 'completed') return a.resolved;
			return true;
		})
	);

	const stats = $derived({
		total: data.assignments.length,
		pending: data.assignments.filter((a) => !a.resolved).length,
		completed: data.assignments.filter((a) => a.resolved).length,
		overdue: data.assignments.filter(
			(a) => !a.resolved && a.deadline && new Date(a.deadline) < new Date()
		).length
	});

	const statCards = [
		{ label: 'Total', key: 'total' as const, cls: 'text-gray-700' },
		{ label: 'Pending', key: 'pending' as const, cls: 'text-amber-600' },
		{ label: 'Completed', key: 'completed' as const, cls: 'text-emerald-600' },
		{ label: 'Overdue', key: 'overdue' as const, cls: 'text-red-600' }
	];

	const statusOptions: { val: 'all' | 'pending' | 'completed'; label: string }[] = [
		{ val: 'pending', label: 'Pending' },
		{ val: 'completed', label: 'Completed' },
		{ val: 'all', label: 'All' }
	];

	function userName(userId: string) {
		return data.users.find((u) => u.id === userId)?.name ?? 'Unknown';
	}

	function userInitials(userId: string) {
		const name = userName(userId);
		return name
			.split(' ')
			.map((p) => p[0])
			.join('')
			.toUpperCase();
	}

	function avatarColor(userId: string) {
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
		if (h < 0) return { label: 'Overdue', cls: 'bg-red-50 text-red-600 border-red-200' };
		if (h < 24)
			return { label: `${Math.round(h)}h left`, cls: 'bg-red-50 text-red-600 border-red-200' };
		if (h < 72)
			return {
				label: `${Math.floor(h / 24)}d left`,
				cls: 'bg-amber-50 text-amber-700 border-amber-200'
			};
		return {
			label: new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
			cls: 'bg-gray-50 text-gray-500 border-gray-200'
		};
	}
</script>

<svelte:head>
	<title>ex-files - Dashboard</title>
</svelte:head>

<div class="bg-gray-50">
	<main class="mx-auto max-w-6xl space-y-6 px-6 py-8">
		<!-- Stats -->
		<div class="grid grid-cols-4 gap-4">
			{#each statCards as card (card.key)}
				<div class="rounded-xl border bg-white px-5 py-4 shadow-sm">
					<p class="text-xs font-medium text-gray-400">{card.label}</p>
					<p class="mt-1 text-2xl font-bold {card.cls}">{stats[card.key]}</p>
				</div>
			{/each}
		</div>

		<!-- Filters row -->
		<div class="flex flex-wrap items-center gap-3">
			<!-- Status filter -->
			<div class="flex rounded-lg border bg-white p-0.5 shadow-sm">
				{#each statusOptions as opt (opt.val)}
					<button
						class="rounded-md px-3 py-1.5 text-xs font-medium transition-colors {statusFilter ===
						opt.val
							? 'bg-blue-600 text-white shadow-sm'
							: 'text-gray-500 hover:text-gray-800'}"
						onclick={() => (statusFilter = opt.val)}
					>
						{opt.label}
					</button>
				{/each}
			</div>

			<!-- User chips -->
			<div class="flex flex-wrap gap-2">
				<button
					class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors {selectedUserId ===
					null
						? 'border-blue-300 bg-blue-50 text-blue-700'
						: 'border-gray-200 bg-white text-gray-500 hover:border-gray-300 hover:text-gray-700'}"
					onclick={() => (selectedUserId = null)}
				>
					All assignees
				</button>
				{#each data.users as user (user.id)}
					<button
						class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors {selectedUserId ===
						user.id
							? 'border-blue-300 bg-blue-50 text-blue-700'
							: 'border-gray-200 bg-white text-gray-500 hover:border-gray-300 hover:text-gray-700'}"
						onclick={() => (selectedUserId = selectedUserId === user.id ? null : user.id)}
					>
						<span
							class="flex h-4 w-4 shrink-0 items-center justify-center rounded-full text-[9px] font-bold text-white {avatarColor(
								user.id
							)}"
						>
							{userInitials(user.id)}
						</span>
						{user.name.split(' ')[0]}
					</button>
				{/each}
			</div>
		</div>

		<!-- Assignment cards -->
		{#if filtered.length === 0}
			<div
				class="flex flex-col items-center justify-center rounded-xl border bg-white py-16 text-center shadow-sm"
			>
				<p class="text-sm font-medium text-gray-500">No assignments match the current filter</p>
			</div>
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each filtered as assignment (assignment.id)}
					{@const dl =
						assignment.deadline && !assignment.resolved ? deadlineInfo(assignment.deadline) : null}
					<div
						class="flex flex-col rounded-xl border bg-white shadow-sm transition-shadow hover:shadow-md"
					>
						<!-- Card header -->
						<div class="flex items-start justify-between gap-3 px-5 pt-5 pb-3">
							<div class="min-w-0">
								<h3 class="truncate text-sm font-semibold text-gray-900">{assignment.title}</h3>
								<p class="mt-1 line-clamp-2 text-xs leading-relaxed text-gray-500">
									{assignment.description}
								</p>
							</div>
							{#if assignment.resolved}
								<span
									class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-100"
								>
									<svg
										class="h-3 w-3 text-emerald-600"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2.5"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="m4.5 12.75 6 6 9-13.5"
										/>
									</svg>
								</span>
							{:else}
								<span
									class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-amber-100"
								>
									<svg
										class="h-3 w-3 text-amber-600"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"
										/>
									</svg>
								</span>
							{/if}
						</div>

						<!-- Meta row -->
						<div class="flex flex-wrap items-center gap-2 px-5 pb-4">
							<span
								class="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 text-[11px] font-medium text-gray-600"
							>
								<span
									class="flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-full text-[8px] font-bold text-white {avatarColor(
										assignment.assigneeId
									)}"
								>
									{userInitials(assignment.assigneeId)}
								</span>
								{userName(assignment.assigneeId)}
							</span>

							{#if dl}
								<span
									class="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium {dl.cls}"
								>
									{dl.label}
								</span>
							{/if}

						</div>

						<!-- Footer -->
						<div class="mt-auto flex items-center justify-between border-t px-5 py-3">
							<div class="flex items-center gap-3 text-[11px] text-gray-400">
								<span class="flex items-center gap-1">
									<svg
										class="h-3.5 w-3.5"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="1.5"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M7.5 8.25h9m-9 3H12m-9.75 1.51c0 1.6 1.123 2.994 2.707 3.227 1.129.166 2.27.293 3.423.379.35.026.67.21.865.501L12 21l2.755-4.133a1.14 1.14 0 0 1 .865-.501 48.172 48.172 0 0 0 3.423-.379c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0 0 12 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018Z"
										/>
									</svg>
									{assignment.commentsCount}
								</span>
								<span class="flex items-center gap-1">
									<svg
										class="h-3.5 w-3.5"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="1.5"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z"
										/>
									</svg>
									{assignment.versionsCount}
								</span>
							</div>
							<a
								href={resolve('/workbench')}
								class="inline-flex items-center gap-1 rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-blue-700"
							>
								Review
								<svg
									class="h-3 w-3"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M13.5 4.5 21 12m0 0-7.5 7.5M21 12H3"
									/>
								</svg>
							</a>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</main>
</div>
