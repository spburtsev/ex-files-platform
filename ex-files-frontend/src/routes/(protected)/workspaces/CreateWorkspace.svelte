<script lang="ts">
	import { createWorkspace } from '$lib/commands.remote';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Plus } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { m } from '$lib/paraglide/messages.js';

	type Props = {
		open: boolean;
	};

	let { open = $bindable() }: Props = $props();

	let createName = $state('');
	let creating = $state(false);
	let createError = $state('');

	async function handleCreate() {
		if (!createName.trim()) return;
		creating = true;
		createError = '';
		try {
			const result = await createWorkspace(createName.trim());
			if (!result.ok) {
				createError = result.error ?? m.ws_create_error();
				return;
			}
			open = false;
			createName = '';
			goto(localizeHref(`/workspaces/${result.workspace.id}`));
		} catch {
			createError = m.error_network_retry();
		} finally {
			creating = false;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Trigger>
		{#snippet child({ props })}
			<Button class="gap-1.5" {...props}>
				<Plus class="size-4" />
				{m.ws_new()}
			</Button>
		{/snippet}
	</Dialog.Trigger>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>{m.ws_create_title()}</Dialog.Title>
			<Dialog.Description>{m.ws_create_description()}</Dialog.Description>
		</Dialog.Header>
		<div class="grid gap-4 py-4">
			<div class="grid gap-2">
				<Label for="ws-name">{m.common_name()}</Label>
				<Input
					id="ws-name"
					placeholder={m.ws_name_placeholder()}
					bind:value={createName}
					onkeydown={(e) => e.key === 'Enter' && handleCreate()}
				/>
			</div>
			{#if createError}
				<p class="text-sm text-destructive">{createError}</p>
			{/if}
		</div>
		<Dialog.Footer>
			<Dialog.Close>
				{#snippet child({ props })}
					<Button variant="outline" {...props}>{m.common_cancel()}</Button>
				{/snippet}
			</Dialog.Close>
			<Button onclick={handleCreate} disabled={creating || !createName.trim()}>
				{creating ? m.common_creating() : m.common_create()}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
