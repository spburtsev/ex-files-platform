<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { FileText, ArrowRight } from '@lucide/svelte';

	const workspaces = [
		{
			id: 'ws1',
			name: 'Q1 2026 Contracts',
			description: 'Vendor and client contracts for the first quarter.',
			documentCount: 12,
			lastActivity: '2 days ago'
		},
		{
			id: 'ws2',
			name: 'HR Compliance Pack',
			description: 'Employment agreements, NDAs, and policy acknowledgements.',
			documentCount: 5,
			lastActivity: '5 days ago'
		},
		{
			id: 'ws3',
			name: 'Vendor Agreements',
			description: 'Supplier contracts and service-level agreements.',
			documentCount: 8,
			lastActivity: 'Today'
		}
	];
</script>

<svelte:head>
	<title>Workspaces — ex-files</title>
</svelte:head>

<div class="flex flex-1 flex-col gap-6 p-6">
	<div>
		<h1 class="text-lg font-semibold">Workspaces</h1>
		<p class="text-sm text-muted-foreground">
			Organize related documents into workspaces for streamlined review.
		</p>
	</div>

	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
		{#each workspaces as ws (ws.id)}
			<Card.Root class="flex flex-col">
				<Card.Header>
					<div class="flex items-start justify-between gap-2">
						<Card.Title class="text-sm">{ws.name}</Card.Title>
						<Badge variant="secondary" class="shrink-0 text-xs">Active</Badge>
					</div>
					<Card.Description class="text-xs">{ws.description}</Card.Description>
				</Card.Header>
				<Card.Content class="pb-3">
					<div class="flex items-center gap-1.5 text-sm text-muted-foreground">
						<FileText class="size-4" />
						<span>{ws.documentCount} documents</span>
					</div>
					<p class="mt-1 text-xs text-muted-foreground">Last activity: {ws.lastActivity}</p>
				</Card.Content>
				<Card.Footer class="mt-auto border-t pt-3">
					<Button size="sm" class="w-full gap-1.5">
						Open
						<ArrowRight class="size-3.5" />
					</Button>
				</Card.Footer>
			</Card.Root>
		{/each}
	</div>
</div>
