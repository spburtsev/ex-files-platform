<script lang="ts">
	import { forgotPassword } from '$lib/commands.remote';
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { FileCheck2 } from '@lucide/svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';

	let email = $state('');
	let loading = $state(false);
	let error = $state<string | null>(null);
	let success = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		loading = true;
		error = null;
		try {
			const result = await forgotPassword(email);
			if (!result.ok) {
				error = result.error;
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
	<title>{m.forgot_page_title()}</title>
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
					<Card.Title class="text-xl">{m.forgot_heading()}</Card.Title>
					<Card.Description>{m.forgot_description()}</Card.Description>
				</Card.Header>
				<Card.Content>
					{#if success}
						<p class="rounded-md bg-green-500/10 px-3 py-2 text-sm text-green-700 dark:text-green-400">
							{m.forgot_success()}
						</p>
						<div class="mt-4 text-center">
							<a
								href={localizeHref('/reset-password')}
								class="text-sm text-foreground underline-offset-4 hover:underline"
							>
								{m.reset_heading()}
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

							<Button type="submit" class="w-full" disabled={loading}>
								{loading ? m.forgot_submitting() : m.forgot_submit()}
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
