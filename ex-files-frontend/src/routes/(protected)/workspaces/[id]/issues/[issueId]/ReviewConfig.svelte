<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import ReviewerPicker from '$lib/components/ReviewerPicker.svelte';
	import { updateReviewConfig } from '$lib/commands.remote';
	import { toast } from 'svelte-sonner';

	type Props = {
		open: boolean;
		issueId: string;
		workspaceMembers: { id: string; name: string }[];
		currentReviewerIds: string[];
		currentRequiredApprovals: number;
		/** The issue assignee — excluded from the reviewer pool. */
		assigneeId?: string;
		onSuccess: () => Promise<void>;
	};
	let {
		open = $bindable(),
		issueId,
		workspaceMembers,
		currentReviewerIds,
		currentRequiredApprovals,
		assigneeId = '',
		onSuccess
	}: Props = $props();

	let selected = $state<string[]>([]);
	let required = $state(1);
	let busy = $state(false);

	// Re-seed the form each time the dialog opens.
	$effect(() => {
		if (open) {
			selected = [...currentReviewerIds];
			required = currentRequiredApprovals;
		}
	});

	async function handleSave() {
		try {
			busy = true;
			const result = await updateReviewConfig({
				id: issueId,
				reviewerIds: selected,
				requiredApprovals: selected.length === 0 ? 1 : required
			});
			if (!result.ok) {
				toast.error(result.error ?? m.error_action_failed());
				return;
			}
			open = false;
			await onSuccess();
		} finally {
			busy = false;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>{m.review_config_title()}</Dialog.Title>
			<Dialog.Description>{m.review_config_description()}</Dialog.Description>
		</Dialog.Header>

		<div class="py-2">
			<ReviewerPicker
				members={workspaceMembers}
				excludeId={assigneeId}
				bind:selected
				bind:required
			/>
		</div>

		<Dialog.Footer>
			<Button variant="outline" onclick={() => (open = false)} disabled={busy}>
				{m.common_cancel()}
			</Button>
			<Button onclick={handleSave} disabled={busy}>
				{busy ? m.common_saving() : m.common_save()}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
