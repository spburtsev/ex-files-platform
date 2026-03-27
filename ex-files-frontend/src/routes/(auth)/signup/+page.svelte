<script lang="ts">
	import { register } from '$lib/commands.remote';
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { FileCheck2 } from '@lucide/svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';

	let name = $state('');
	let email = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state<string | null>(null);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		loading = true;
		error = null;
		try {
			const result = await register({ name, email, password });
			if (!result.ok) {
				error = result.error ?? m.signup_error();
			} else {
				window.location.href = localizeHref('/');
			}
		} catch {
			error = m.error_network();
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>{m.signup_page_title()}</title>
</svelte:head>

<div class="flex min-h-svh w-full items-center justify-center p-6 md:p-10">
	<div class="w-full max-w-sm">
		<div class="flex flex-col gap-6">
			<!-- Brand -->
			<a href={localizeHref('/')} class="flex items-center justify-center gap-2 self-center">
				<FileCheck2 class="size-6 text-primary" />
				<span class="text-xl font-bold tracking-tight">ex-files</span>
			</a>

			<Card.Root>
				<Card.Header class="text-center">
					<Card.Title class="text-xl">{m.signup_heading()}</Card.Title>
					<Card.Description>{m.signup_description()}</Card.Description>
				</Card.Header>
				<Card.Content>
					<form onsubmit={handleSubmit} class="flex flex-col gap-4">
						{#if error}
							<p class="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p>
						{/if}

						<div class="flex flex-col gap-2">
							<Label for="name">{m.signup_full_name()}</Label>
							<Input
								id="name"
								type="text"
								placeholder={m.signup_name_placeholder()}
								bind:value={name}
								required
								autocomplete="name"
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
								autocomplete="email"
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

						<Button type="submit" class="w-full" disabled={loading}>
							{loading ? m.signup_submitting() : m.signup_submit()}
						</Button>
					</form>
				</Card.Content>
				<Card.Footer class="justify-center text-sm text-muted-foreground">
					{m.signup_has_account()}&nbsp;
					<a href={localizeHref('/login')} class="text-foreground underline-offset-4 hover:underline">{m.signup_login_link()}</a>
				</Card.Footer>
			</Card.Root>
		</div>
	</div>
</div>
