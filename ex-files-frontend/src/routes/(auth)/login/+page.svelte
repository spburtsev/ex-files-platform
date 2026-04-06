<script lang="ts">
	import { login } from '$lib/commands.remote';
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { FileCheck2 } from '@lucide/svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';

	let email = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state<string | null>(null);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		loading = true;
		error = null;
		try {
			const result = await login({ email, password });
			if (!result.ok) {
				error = result.error ?? m.login_invalid_credentials();
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
	<title>{m.login_page_title()}</title>
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
					<Card.Title class="text-xl">{m.login_welcome()}</Card.Title>
					<Card.Description>{m.login_description()}</Card.Description>
				</Card.Header>
				<Card.Content>
					<form onsubmit={handleSubmit} class="flex flex-col gap-4">
						{#if error}
							<p class="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p>
						{/if}

						<div class="flex flex-col gap-2">
							<Label for="email">{m.common_email()}</Label>
							<Input
								id="email"
								type="email"
								placeholder={m.login_email_placeholder()}
								bind:value={email}
								required
								autocomplete="email"
							/>
						</div>

						<div class="flex flex-col gap-2">
							<div class="flex items-center justify-between">
								<Label for="password">{m.common_password()}</Label>
								<a
									href={localizeHref('/forgot-password')}
									class="text-xs text-muted-foreground underline-offset-4 hover:underline"
								>
									{m.login_forgot_password()}
								</a>
							</div>
							<Input
								id="password"
								type="password"
								placeholder={m.login_password_placeholder()}
								bind:value={password}
								required
								autocomplete="current-password"
							/>
						</div>

						<Button type="submit" class="w-full" disabled={loading}>
							{loading ? m.login_submitting() : m.login_submit()}
						</Button>
					</form>
				</Card.Content>
				<Card.Footer class="justify-center text-sm text-muted-foreground">
					{m.login_no_account()}&nbsp;
					<a
						href={localizeHref('/signup')}
						class="text-foreground underline-offset-4 hover:underline">{m.login_signup_link()}</a
					>
				</Card.Footer>
			</Card.Root>
		</div>
	</div>
</div>
