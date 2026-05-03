<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { MessageSquare, EllipsisVertical, CheckCircle, XCircle } from '@lucide/svelte';
	import type { Issue } from '$lib/api';
	import type { Document } from '$lib/stores/workbench.svelte';
	import { workbenchStore } from '$lib/stores/workbench.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { approveDocument } from '$lib/commands.remote';
	import { toast } from 'svelte-sonner';
	import { m } from '$lib/paraglide/messages';

	type Props = {
		doc: Document;
		issue: Issue;
		canReviewIssue: boolean;
		onSelect: (doc: Document) => void;
		onApproved: () => Promise<void>;
		onRequestChangesClick: (doc: Document) => void;
		onRejectClick: (doc: Document) => void;
	};
	let {
		doc,
		issue,
		canReviewIssue,
		onSelect,
		onApproved,
		onRequestChangesClick,
		onRejectClick
	}: Props = $props();

	function canShowMenuFor(doc: { serverId?: string }) {
		return canReviewIssue && !!doc.serverId;
	}

	function canActOn(doc: { reviewStatus?: string }) {
		return (
			doc.reviewStatus === 'pending' ||
			doc.reviewStatus === 'in_review' ||
			doc.reviewStatus === 'changes_requested'
		);
	}

	function statusBadgeClass(status: string, reviewStatus?: string) {
		// Local upload-lifecycle states win over review status.
		switch (status) {
			case 'draft':
				return 'bg-amber-100 text-amber-800';
			case 'saving':
				return 'bg-blue-100 text-blue-700';
			case 'error':
				return 'bg-red-100 text-red-700';
			case 'saved':
				switch (reviewStatus) {
					case 'approved':
						return 'bg-emerald-100 text-emerald-700';
					case 'rejected':
						return 'bg-red-100 text-red-700';
					case 'changes_requested':
						return 'bg-amber-100 text-amber-800';
					case 'in_review':
						return 'bg-blue-100 text-blue-700';
					case 'pending':
					default:
						return 'bg-slate-100 text-slate-700';
				}
			default:
				return '';
		}
	}

	function statusLabel(status: string, reviewStatus?: string) {
		switch (status) {
			case 'draft':
				return m.workbench_status_draft();
			case 'saving':
				return m.workbench_saving();
			case 'error':
				return m.workbench_status_error();
			case 'saved':
				switch (reviewStatus) {
					case 'approved':
						return m.workbench_status_approved();
					case 'rejected':
						return m.workbench_status_rejected();
					case 'changes_requested':
						return m.workbench_status_changes_requested();
					case 'in_review':
						return m.workbench_status_awaiting_review();
					case 'pending':
					default:
						return m.workbench_status_saved();
				}
			default:
				return '';
		}
	}

	function formatSize(bytes: number) {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

	async function handleApprove(localId: string) {
		const doc = workbenchStore.documents.find((d) => d.id === localId);
		if (!doc?.serverId) return;
		const result = await approveDocument(doc.serverId);
		if (!result.ok) {
			toast.error(result.error ?? m.error_action_failed());
			return;
		}
		workbenchStore.setDocumentReviewStatus(localId, 'approved');
		await onApproved();
	}
</script>

<li>
	<div
		class="group relative flex items-start gap-1 px-3 py-2 transition-colors {workbenchStore.activeDocumentId ===
		doc.id
			? 'bg-primary/8 text-primary'
			: 'text-foreground hover:bg-muted/60'}"
	>
		<div
			role="button"
			tabindex="0"
			class="min-w-0 flex-1 cursor-pointer text-left"
			onclick={() => onSelect(doc)}
			onkeydown={(e) => {
				if (e.key === 'Enter' || e.key === ' ') {
					e.preventDefault();
					onSelect(doc);
				}
			}}
		>
			<p class="truncate text-xs font-medium">{doc.name}</p>
			{#if doc.uploaderName}
				<p class="truncate text-[10px] text-muted-foreground">
					{doc.uploaderName}
				</p>
			{/if}
			<div class="mt-0.5 flex items-center gap-1.5">
				<Badge
					variant="secondary"
					class="h-4 px-1.5 text-[9px] font-semibold {statusBadgeClass(
						doc.status,
						doc.reviewStatus
					)}"
					title={doc.error ?? ''}
				>
					{statusLabel(doc.status, doc.reviewStatus)}
				</Badge>
				<span class="text-[10px] text-muted-foreground">
					{formatSize(doc.size)}
				</span>
			</div>
		</div>

		{#if canShowMenuFor(doc)}
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<button
							{...props}
							onclick={(e) => e.stopPropagation()}
							disabled={issue?.resolved}
							class="rounded p-1 text-muted-foreground transition hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-muted-foreground"
							aria-label={m.workbench_actions()}
						>
							<EllipsisVertical class="size-3.5" />
						</button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content side="right" align="start" class="w-48">
					<DropdownMenu.Item onclick={() => handleApprove(doc.id)} disabled={!canActOn(doc)}>
						<CheckCircle class="size-3.5" />
						{m.doc_approve()}
					</DropdownMenu.Item>
					<DropdownMenu.Item
						onclick={() => {
							onRequestChangesClick(doc);
						}}
						disabled={!canActOn(doc)}
					>
						<MessageSquare class="size-3.5" />
						{m.doc_request_changes()}
					</DropdownMenu.Item>
					<DropdownMenu.Separator />
					<DropdownMenu.Item
						onclick={() => {
							onRejectClick(doc);
						}}
						disabled={!canActOn(doc)}
						class="text-red-600 focus:text-red-600"
					>
						<XCircle class="size-3.5" />
						{m.doc_reject()}
					</DropdownMenu.Item>
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		{/if}
	</div>
</li>
