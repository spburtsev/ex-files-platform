<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Check } from '@lucide/svelte';

	type Props = {
		members: { id: string; name: string }[];
		selected: string[];
		required: number;
		excludeId?: string;
	};
	let { members, selected = $bindable(), required = $bindable(), excludeId = '' }: Props = $props();

	const available = $derived(members.filter((mm) => String(mm.id) !== String(excludeId)));

	$effect(() => {
		if (excludeId && selected.includes(String(excludeId))) {
			selected = selected.filter((x) => x !== String(excludeId));
		}
	});

	// N is capped at the number of selected reviewers, and never exceeds the
	// number of eligible workspace members.
	const maxRequired = $derived(Math.max(1, Math.min(selected.length, available.length)));
	$effect(() => {
		if (required > maxRequired) required = maxRequired;
		if (required < 1) required = 1;
	});

	function toggle(id: string) {
		selected = selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id];
	}
</script>

<div class="space-y-4">
	<div class="space-y-1.5">
		<Label>{m.review_config_reviewers()}</Label>
		{#if available.length === 0}
			<p class="text-xs text-muted-foreground">{m.review_config_no_members()}</p>
		{:else}
			<div class="max-h-48 overflow-y-auto rounded-md border">
				{#each available as member (member.id)}
					{@const checked = selected.includes(String(member.id))}
					<button
						type="button"
						class="flex w-full items-center justify-between px-3 py-2 text-left text-sm transition-colors hover:bg-muted/60"
						onclick={() => toggle(String(member.id))}
					>
						<span class="truncate">{member.name}</span>
						<span
							class="flex size-4 shrink-0 items-center justify-center rounded border {checked
								? 'border-primary bg-primary text-primary-foreground'
								: 'border-input'}"
						>
							{#if checked}
								<Check class="size-3" />
							{/if}
						</span>
					</button>
				{/each}
			</div>
		{/if}
	</div>

	<div class="space-y-1.5">
		<Label for="required-approvals">{m.review_config_required_approvals()}</Label>
		<Input
			id="required-approvals"
			type="number"
			min="1"
			max={maxRequired}
			bind:value={required}
			disabled={selected.length === 0}
			class="w-24"
		/>
		<p class="text-[11px] text-muted-foreground">
			{m.review_config_required_hint({ max: String(maxRequired) })}
		</p>
	</div>
</div>
