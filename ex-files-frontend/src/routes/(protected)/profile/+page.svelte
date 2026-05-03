<script lang="ts">
	import { changePassword, forgotPassword } from '$lib/commands.remote';
	import { m } from '$lib/paraglide/messages.js';
	import { toast } from 'svelte-sonner';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';

	const { data } = $props();
	const me = $derived(data.user);

	let oldPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let saving = $state(false);

	let resetSending = $state(false);
	let resetCooldown = $state(0);

	async function handleChangePassword(e: Event) {
		e.preventDefault();
		if (newPassword.length < 8) {
			toast.error(m.profile_password_too_short());
			return;
		}
		if (newPassword !== confirmPassword) {
			toast.error(m.profile_password_mismatch());
			return;
		}
		if (newPassword === oldPassword) {
			toast.error(m.profile_password_unchanged());
			return;
		}
		saving = true;
		try {
			const r = await changePassword({ oldPassword, newPassword });
			if (!r.ok) {
				toast.error(r.error ?? m.error_action_failed());
				return;
			}
			toast.success(m.profile_password_updated());
			oldPassword = '';
			newPassword = '';
			confirmPassword = '';
		} finally {
			saving = false;
		}
	}

	async function handleSendReset() {
		if (!me?.email || resetCooldown > 0) return;
		resetSending = true;
		try {
			await forgotPassword(me.email);
			toast.success(m.profile_reset_sent());
			resetCooldown = 30;
			const interval = setInterval(() => {
				resetCooldown -= 1;
				if (resetCooldown <= 0) clearInterval(interval);
			}, 1000);
		} finally {
			resetSending = false;
		}
	}
</script>

<svelte:head>
	<title>{m.profile_page_title()}</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-xl flex-col gap-6 p-6">
	<header class="flex flex-col gap-1">
		<h1 class="text-2xl font-semibold tracking-tight">{m.profile_heading()}</h1>
		{#if me}
			<p class="text-sm text-muted-foreground">{me.name} - {me.email}</p>
		{/if}
	</header>

	<Card.Root>
		<Card.Header>
			<Card.Title>{m.profile_change_password_heading()}</Card.Title>
			<Card.Description>{m.profile_change_password_desc()}</Card.Description>
		</Card.Header>
		<Card.Content>
			<form onsubmit={handleChangePassword} class="flex flex-col gap-4">
				<div class="flex flex-col gap-2">
					<Label for="old-password">{m.profile_old_password_label()}</Label>
					<Input
						id="old-password"
						type="password"
						bind:value={oldPassword}
						required
						autocomplete="current-password"
					/>
				</div>

				<div class="flex flex-col gap-2">
					<Label for="new-password">{m.profile_new_password_label()}</Label>
					<Input
						id="new-password"
						type="password"
						bind:value={newPassword}
						required
						minlength={8}
						autocomplete="new-password"
					/>
				</div>

				<div class="flex flex-col gap-2">
					<Label for="confirm-password">{m.profile_confirm_password_label()}</Label>
					<Input
						id="confirm-password"
						type="password"
						bind:value={confirmPassword}
						required
						minlength={8}
						autocomplete="new-password"
					/>
				</div>

				<Button type="submit" disabled={saving} class="self-start">
					{saving ? m.profile_submitting() : m.profile_submit()}
				</Button>
			</form>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header>
			<Card.Title>{m.profile_reset_heading()}</Card.Title>
			<Card.Description>{m.profile_reset_desc()}</Card.Description>
		</Card.Header>
		<Card.Content>
			<Button
				variant="outline"
				onclick={handleSendReset}
				disabled={resetSending || resetCooldown > 0 || !me?.email}
			>
				{#if resetCooldown > 0}
					{m.profile_reset_button()} ({resetCooldown}s)
				{:else}
					{m.profile_reset_button()}
				{/if}
			</Button>
		</Card.Content>
	</Card.Root>
</div>
