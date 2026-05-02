<script lang="ts">
	import { getAuditStats } from '$lib/queries.remote';
	import type { AuditStats } from '$lib/queries.remote';
	import { m } from '$lib/paraglide/messages.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Download } from '@lucide/svelte';
	import { Chart, registerables } from 'chart.js';
	import { onMount } from 'svelte';

	let stats = $state<AuditStats | null>(null);
	let loading = $state(true);

	let actionsCanvas = $state<HTMLCanvasElement>(undefined!);
	let dailyCanvas = $state<HTMLCanvasElement>(undefined!);
	let statusCanvas = $state<HTMLCanvasElement>(undefined!);

	let charts: Chart[] = [];

	onMount(async () => {
		Chart.register(...registerables);
		try {
			stats = await getAuditStats();
		} catch (e) {
			console.error('Failed to load stats', e);
		} finally {
			loading = false;
		}
	});

	$effect(() => {
		if (!stats) return;

		// Destroy previous charts
		charts.forEach((c) => c.destroy());
		charts = [];

		// Actions by Type - bar chart
		if (actionsCanvas && stats.actions_by_type?.length) {
			charts.push(
				new Chart(actionsCanvas, {
					type: 'bar',
					data: {
						labels: stats.actions_by_type.map((a) => a.action.replace('.', ' ')),
						datasets: [
							{
								label: m.analytics_actions_by_type(),
								data: stats.actions_by_type.map((a) => a.count),
								backgroundColor: 'rgba(99, 102, 241, 0.7)',
								borderRadius: 4
							}
						]
					},
					options: {
						responsive: true,
						plugins: { legend: { display: false } },
						scales: { y: { beginAtZero: true, ticks: { stepSize: 1 } } }
					}
				})
			);
		}

		// Daily Activity - line chart
		if (dailyCanvas && stats.daily_activity?.length) {
			charts.push(
				new Chart(dailyCanvas, {
					type: 'line',
					data: {
						labels: stats.daily_activity.map((d) => d.date),
						datasets: [
							{
								label: m.analytics_daily_activity(),
								data: stats.daily_activity.map((d) => d.count),
								borderColor: 'rgba(16, 185, 129, 1)',
								backgroundColor: 'rgba(16, 185, 129, 0.1)',
								fill: true,
								tension: 0.3
							}
						]
					},
					options: {
						responsive: true,
						plugins: { legend: { display: false } },
						scales: { y: { beginAtZero: true, ticks: { stepSize: 1 } } }
					}
				})
			);
		}

		// Documents by Status - doughnut chart
		if (statusCanvas && stats.documents_by_status?.length) {
			const colors = ['#f59e0b', '#3b82f6', '#10b981', '#ef4444', '#8b5cf6'];
			charts.push(
				new Chart(statusCanvas, {
					type: 'doughnut',
					data: {
						labels: stats.documents_by_status.map((s) => s.status),
						datasets: [
							{
								data: stats.documents_by_status.map((s) => s.count),
								backgroundColor: colors.slice(0, stats.documents_by_status.length)
							}
						]
					},
					options: { responsive: true }
				})
			);
		}

		return () => {
			charts.forEach((c) => c.destroy());
			charts = [];
		};
	});

	function downloadCSV() {
		if (!stats) return;
		const rows = [['Category', 'Label', 'Value']];

		for (const a of stats.actions_by_type ?? []) {
			rows.push(['Action', a.action, String(a.count)]);
		}
		for (const d of stats.daily_activity ?? []) {
			rows.push(['Daily Activity', d.date, String(d.count)]);
		}
		for (const s of stats.documents_by_status ?? []) {
			rows.push(['Document Status', s.status, String(s.count)]);
		}
		for (const u of stats.top_actors ?? []) {
			rows.push(['Top Actor', u.actor_name, String(u.count)]);
		}

		const csv = rows.map((r) => r.join(',')).join('\n');
		const blob = new Blob([csv], { type: 'text/csv' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = 'analytics.csv';
		a.click();
		URL.revokeObjectURL(url);
	}
</script>

<svelte:head>
	<title>{m.analytics_page_title()}</title>
</svelte:head>

<div class="flex flex-1 flex-col gap-6 p-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold tracking-tight">{m.analytics_heading()}</h1>
			<p class="text-sm text-muted-foreground">{m.analytics_description()}</p>
		</div>
		{#if stats}
			<Button variant="outline" size="sm" onclick={downloadCSV}>
				<Download class="mr-2 size-4" />
				{m.analytics_download_csv()}
			</Button>
		{/if}
	</div>

	{#if loading}
		<p class="text-sm text-muted-foreground">{m.common_loading()}</p>
	{:else if !stats}
		<p class="text-sm text-muted-foreground">{m.analytics_no_data()}</p>
	{:else}
		<div class="grid gap-6 md:grid-cols-2">
			<!-- Actions by Type -->
			<Card.Root>
				<Card.Header>
					<Card.Title>{m.analytics_actions_by_type()}</Card.Title>
				</Card.Header>
				<Card.Content>
					{#if stats.actions_by_type?.length}
						<canvas bind:this={actionsCanvas}></canvas>
					{:else}
						<p class="text-sm text-muted-foreground">{m.analytics_no_data()}</p>
					{/if}
				</Card.Content>
			</Card.Root>

			<!-- Daily Activity -->
			<Card.Root>
				<Card.Header>
					<Card.Title>{m.analytics_daily_activity()}</Card.Title>
				</Card.Header>
				<Card.Content>
					{#if stats.daily_activity?.length}
						<canvas bind:this={dailyCanvas}></canvas>
					{:else}
						<p class="text-sm text-muted-foreground">{m.analytics_no_data()}</p>
					{/if}
				</Card.Content>
			</Card.Root>

			<!-- Documents by Status -->
			<Card.Root>
				<Card.Header>
					<Card.Title>{m.analytics_documents_by_status()}</Card.Title>
				</Card.Header>
				<Card.Content class="flex justify-center">
					{#if stats.documents_by_status?.length}
						<div class="max-w-xs">
							<canvas bind:this={statusCanvas}></canvas>
						</div>
					{:else}
						<p class="text-sm text-muted-foreground">{m.analytics_no_data()}</p>
					{/if}
				</Card.Content>
			</Card.Root>

			<!-- Top Actors -->
			<Card.Root>
				<Card.Header>
					<Card.Title>{m.analytics_top_actors()}</Card.Title>
				</Card.Header>
				<Card.Content>
					{#if stats.top_actors?.length}
						<div class="space-y-3">
							{#each stats.top_actors as actor, i}
								<div class="flex items-center justify-between">
									<div class="flex items-center gap-2">
										<span
											class="flex size-6 items-center justify-center rounded-full bg-primary/10 text-xs font-medium"
										>
											{i + 1}
										</span>
										<span class="text-sm font-medium">{actor.actor_name}</span>
									</div>
									<span class="text-sm text-muted-foreground">
										{actor.count} actions
									</span>
								</div>
							{/each}
						</div>
					{:else}
						<p class="text-sm text-muted-foreground">{m.analytics_no_data()}</p>
					{/if}
				</Card.Content>
			</Card.Root>
		</div>
	{/if}
</div>
