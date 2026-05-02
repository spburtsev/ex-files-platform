<script lang="ts">
	import { resetPassword } from '$lib/commands.remote';
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { FileCheck2 } from '@lucide/svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';

	let token = $state('');
	let password = $state('');
	let loading = $state(false);
	let error = $state<string | null>(null);
	let success = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		loading = true;
		error = null;
		try {
			const result = await resetPassword({ token, password });
			if (!result.ok) {
				error = result.error ?? m.reset_invalid_token();
			} else {
				success = true;
			}
		} catch {
			error = m.error_network();
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>{m.reset_page_title()}</title>
</svelte:head>

<div class="flex min-h-svh w-full items-center justify-center p-6 md:p-10">
	<div class="w-full max-w-sm">
		<div class="flex flex-col gap-6">
			<a href={localizeHref('/')} class="flex items-center justify-center gap-2 self-center">
				<FileCheck2 class="size-6 text-primary" />
				<span class="text-xl font-bold tracking-tight">ex-files</span>
			</a>

			<Card.Root>
				<Card.Header class="text-center">
					<Card.Title class="text-xl">{m.reset_heading()}</Card.Title>
					<Card.Description>{m.reset_description()}</Card.Description>
				</Card.Header>
				<Card.Content>
					{#if success}
						<p class="rounded-md bg-green-500/10 px-3 py-2 text-sm text-green-700 dark:text-green-400">
							{m.reset_success()}
						</p>
						<div class="mt-4 text-center">
							<a
								href={localizeHref('/login')}
								class="text-sm text-foreground underline-offset-4 hover:underline"
							>
								{m.login_submit()}
							</a>
						</div>
					{:else}
						<form onsubmit={handleSubmit} class="flex flex-col gap-4">
							{#if error}
								<p class="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
									{error}
								</p>
							{/if}

							<div class="flex flex-col gap-2">
								<Label for="token">{m.reset_token_label()}</Label>
								<Input
									id="token"
									type="text"
									placeholder={m.reset_token_placeholder()}
									bind:value={token}
									required
								/>
							</div>

							<div class="flex flex-col gap-2">
								<Label for="password">{m.reset_new_password()}</Label>
								<Input
									id="password"
									type="password"
									placeholder={m.reset_password_placeholder()}
									bind:value={password}
									required
									minlength={8}
									autocomplete="new-password"
								/>
							</div>

							<Button type="submit" class="w-full" disabled={loading}>
								{loading ? m.reset_submitting() : m.reset_submit()}
							</Button>
						</form>
					{/if}
				</Card.Content>
				<Card.Footer class="justify-center text-sm text-muted-foreground">
					<a
						href={localizeHref('/login')}
						class="text-foreground underline-offset-4 hover:underline"
					>
						{m.forgot_back_to_login()}
					</a>
				</Card.Footer>
			</Card.Root>
		</div>
	</div>
</div>
