<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { updateIssueAssignee } from '$lib/commands.remote';
	import { toast } from 'svelte-sonner';

	type Props = {
        issueId: string;
		open: boolean;
        currentAssigneeId: string;
		workspaceMembers: { id: string; name: string }[];
		onSuccess: () => Promise<void>;
	};
	let {
		open = $bindable(),
        issueId,
		workspaceMembers,
        currentAssigneeId,
		onSuccess,
	}: Props = $props();

	let selection = $state('');
	let busy = $state(false);

	async function handleChangeAssignee() {
		if (selection === currentAssigneeId) {
			open = false;
			return;
		}

		try {
		    busy = true;
			const result = await updateIssueAssignee({
				id: issueId,
				assigneeId: selection,
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
			<Dialog.Title>{m.ws_issue_change_assignee_label()}</Dialog.Title>
		</Dialog.Header>
		<div class="space-y-2 py-2">
			<Select.Root bind:value={selection} type="single">
				<Select.Label>{m.ws_issue_assignee_label()}</Select.Label>
				<Select.Trigger class="w-full"
					>{workspaceMembers.find((mem) => mem.id === selection)?.name ||
						m.ws_issue_assignee_select()}</Select.Trigger
				>
				<Select.Content>
					{#each workspaceMembers as member (member.id)}
						<Select.Item value={String(member.id)}>{member.name}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (open = false)} disabled={busy}>
				{m.common_cancel()}
			</Button>
			<Button onclick={() => handleChangeAssignee()} disabled={busy || !selection}>
				{busy ? m.common_saving() : m.common_save()}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
