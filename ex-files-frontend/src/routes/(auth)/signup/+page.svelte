<script lang="ts">
	import { register } from '$lib/commands.remote';
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
				error = result.error ?? 'Registration failed. Please try again.';
			} else {
				window.location.href = '/';
			}
		} catch {
			error = 'Could not reach the server. Please try again.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Sign up — ex-files</title>
</svelte:head>

<div class="flex min-h-svh w-full items-center justify-center p-6 md:p-10">
	<div class="w-full max-w-sm">
		<div class="flex flex-col gap-6">
			<!-- Brand -->
			<a href="/" class="flex items-center justify-center gap-2 self-center">
				<FileCheck2 class="size-6 text-primary" />
				<span class="text-xl font-bold tracking-tight">ex-files</span>
			</a>

			<Card.Root>
				<Card.Header class="text-center">
					<Card.Title class="text-xl">Create an account</Card.Title>
					<Card.Description>Enter your details to get started</Card.Description>
				</Card.Header>
				<Card.Content>
					<form onsubmit={handleSubmit} class="flex flex-col gap-4">
						{#if error}
							<p class="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p>
						{/if}

						<div class="flex flex-col gap-2">
							<Label for="name">Full name</Label>
							<Input
								id="name"
								type="text"
								placeholder="Alex Johnson"
								bind:value={name}
								required
								autocomplete="name"
							/>
						</div>

						<div class="flex flex-col gap-2">
							<Label for="email">Email</Label>
							<Input
								id="email"
								type="email"
								placeholder="you@example.com"
								bind:value={email}
								required
								autocomplete="email"
							/>
						</div>

						<div class="flex flex-col gap-2">
							<Label for="password">Password</Label>
							<Input
								id="password"
								type="password"
								placeholder="••••••••"
								bind:value={password}
								required
								minlength={8}
								autocomplete="new-password"
							/>
							<p class="text-xs text-muted-foreground">Must be at least 8 characters.</p>
						</div>

						<Button type="submit" class="w-full" disabled={loading}>
							{loading ? 'Creating account…' : 'Create account'}
						</Button>
					</form>
				</Card.Content>
				<Card.Footer class="justify-center text-sm text-muted-foreground">
					Already have an account?&nbsp;
					<a href="/login" class="text-foreground underline-offset-4 hover:underline">Log in</a>
				</Card.Footer>
			</Card.Root>
		</div>
	</div>
</div>
