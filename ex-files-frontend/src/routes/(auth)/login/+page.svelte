<script lang="ts">
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
			const res = await fetch('/api/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email, password })
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({}));
				error = body.error ?? 'Invalid email or password.';
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
	<title>Log in — ex-files</title>
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
					<Card.Title class="text-xl">Welcome back</Card.Title>
					<Card.Description>Log in to your ex-files account</Card.Description>
				</Card.Header>
				<Card.Content>
					<form onsubmit={handleSubmit} class="flex flex-col gap-4">
						{#if error}
							<p class="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p>
						{/if}

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
							<div class="flex items-center justify-between">
								<Label for="password">Password</Label>
								<a
									href="/forgot-password"
									class="text-xs text-muted-foreground underline-offset-4 hover:underline"
								>
									Forgot password?
								</a>
							</div>
							<Input
								id="password"
								type="password"
								placeholder="••••••••"
								bind:value={password}
								required
								autocomplete="current-password"
							/>
						</div>

						<Button type="submit" class="w-full" disabled={loading}>
							{loading ? 'Logging in…' : 'Log in'}
						</Button>
					</form>
				</Card.Content>
				<Card.Footer class="justify-center text-sm text-muted-foreground">
					Don't have an account?&nbsp;
					<a href="/signup" class="text-foreground underline-offset-4 hover:underline">Sign up</a>
				</Card.Footer>
			</Card.Root>
		</div>
	</div>
</div>
