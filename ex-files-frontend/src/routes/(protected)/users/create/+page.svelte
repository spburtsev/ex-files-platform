<script lang="ts">
	import { createUser } from '$lib/commands.remote';
	import { m } from '$lib/paraglide/messages.js';
	import { goto } from '$app/navigation';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { toast } from 'svelte-sonner';
	import { ArrowLeft, UserPlus } from '@lucide/svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { roleLabel } from '$lib/utils';

	type CreatableRole = 'employee' | 'manager' | 'root';
	const roles: CreatableRole[] = ['employee', 'manager', 'root'];

	let name = $state('');
	let email = $state('');
	let password = $state('');
	let role = $state<CreatableRole>('employee');
	let loading = $state(false);
	let error = $state<string | null>(null);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		loading = true;
		error = null;
		try {
			const result = await createUser({ name, email, password, role });
			if (!result.ok) {
				error = result.error ?? m.users_create_error();
				return;
			}
			toast.success(m.users_create_success());
			goto(localizeHref('/users'));
		} catch {
			error = m.error_network();
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>{m.users_create_page_title()}</title>
</svelte:head>

<div class="flex flex-1 flex-col gap-6 p-6">
	<a
		href={localizeHref('/users')}
		class="flex items-center gap-1.5 self-start text-sm text-muted-foreground hover:text-foreground"
	>
		<ArrowLeft class="size-4" />
		{m.users_create_back()}
	</a>

	<Card.Root class="max-w-lg">
		<Card.Header>
			<Card.Title>{m.users_create_heading()}</Card.Title>
			<Card.Description>{m.users_create_description()}</Card.Description>
		</Card.Header>
		<Card.Content>
			<form onsubmit={handleSubmit} class="flex flex-col gap-4">
				{#if error}
					<p class="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p>
				{/if}

				<div class="flex flex-col gap-2">
					<Label for="name">{m.common_name()}</Label>
					<Input
						id="name"
						type="text"
						placeholder={m.signup_name_placeholder()}
						bind:value={name}
						required
						autocomplete="off"
					/>
				</div>

				<div class="flex flex-col gap-2">
					<Label for="email">{m.common_email()}</Label>
					<Input
						id="email"
						type="email"
						placeholder={m.signup_email_placeholder()}
						bind:value={email}
						required
						autocomplete="off"
					/>
				</div>

				<div class="flex flex-col gap-2">
					<Label for="password">{m.common_password()}</Label>
					<Input
						id="password"
						type="password"
						placeholder={m.signup_password_placeholder()}
						bind:value={password}
						required
						minlength={8}
						autocomplete="new-password"
					/>
					<p class="text-xs text-muted-foreground">{m.signup_password_hint()}</p>
				</div>

				<div class="flex flex-col gap-2">
					<Label for="role">{m.users_role()}</Label>
					<Select.Root type="single" value={role} onValueChange={(v) => (role = v as CreatableRole)}>
						<Select.Trigger id="role" class="w-full">{roleLabel(role)}</Select.Trigger>
						<Select.Content>
							{#each roles as r (r)}
								<Select.Item value={r} label={roleLabel(r)}>{roleLabel(r)}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>

				<Button type="submit" class="w-full gap-1.5" disabled={loading}>
					<UserPlus class="size-4" />
					{loading ? m.users_create_submitting() : m.users_create_submit()}
				</Button>
			</form>
		</Card.Content>
	</Card.Root>
</div>
