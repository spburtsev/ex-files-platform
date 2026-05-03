<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Label } from '$lib/components/ui/label';
	import Textarea from '$lib/components/ui/textarea/textarea.svelte';
	import { workbenchStore } from '$lib/stores/workbench.svelte';
	import { rejectDocument } from '$lib/commands.remote';
	import { toast } from 'svelte-sonner';

	type Props = {
		target: string | null;
		onSuccess: () => Promise<void>;
	};
	let { target = $bindable(), onSuccess }: Props = $props();

	let note = $state('');
	let busy = $state(false);

	async function handleReject() {
		if (!target) {
			return;
		}
		const doc = workbenchStore.documents.find((d) => d.id === target);
		if (!doc?.serverId) return;
		busy = true;
		try {
			const result = await rejectDocument({ id: doc.serverId, note: note.trim() });
			if (!result.ok) {
				toast.error(result.error ?? m.error_action_failed());
				return;
			}
			workbenchStore.setDocumentReviewStatus(target, 'rejected');
			target = null;
			note = '';
			await onSuccess();
		} finally {
			busy = false;
		}
	}
</script>

<Dialog.Root
	open={target !== null}
	onOpenChange={(open) => {
		if (!open) {
			target = null;
			note = '';
		}
	}}
>
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>{m.doc_reject_title()}</Dialog.Title>
			<Dialog.Description>{m.doc_reject_description()}</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-2 py-2">
			<Label for="reject-note">{m.doc_reject_reason_label()}</Label>
			<Textarea
				id="reject-note"
				bind:value={note}
				placeholder={m.doc_reject_placeholder()}
				rows={4}
			/>
		</div>
		<Dialog.Footer>
			<Button
				variant="outline"
				onclick={() => {
					target = null;
					note = '';
				}}
				disabled={busy}
			>
				{m.common_cancel()}
			</Button>
			<Button variant="destructive" onclick={handleReject} disabled={busy}>
				{busy ? m.doc_rejecting() : m.doc_reject()}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
