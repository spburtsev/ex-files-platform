<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';

	type Props = {
		issue: {
			title: string;
			description?: string;
		};
		open: boolean;
	};
	let { issue, open = $bindable() }: Props = $props();
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="max-w-lg">
		<Dialog.Header>
			<Dialog.Title>{issue.title}</Dialog.Title>
			<Dialog.Description>{m.workbench_issue_details()}</Dialog.Description>
		</Dialog.Header>

		<section class="py-2">
			<h3 class="mb-2 text-xs font-semibold text-muted-foreground">
				{m.ws_issue_description_label()}
			</h3>
			{#if issue.description}
				<p class="text-sm leading-relaxed whitespace-pre-line">{issue.description}</p>
			{:else}
				<p class="text-sm text-muted-foreground italic">{m.workbench_no_description()}</p>
			{/if}
		</section>
	</Dialog.Content>
</Dialog.Root>
