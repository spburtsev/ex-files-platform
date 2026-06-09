<script lang="ts">
	import { verifyDocumentHash } from '$lib/commands.remote';
	import { m } from '$lib/paraglide/messages.js';
	import { localizeHref } from '$lib/paraglide/runtime';
	import { formatTimestamp } from '$lib/utils';
	import { UploadCloud, X, CheckCircle2, XCircle } from '@lucide/svelte';
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';

	interface Props {
		showLoginLink?: boolean;
	}
	let { showLoginLink = true }: Props = $props();

	type VerifyResult = {
		verified: boolean;
		documentName?: string;
		status?: string;
		notarizedAt?: string;
		hash: string;
	};

	let stagedFile = $state<File | null>(null);
	let hashInput = $state('');
	let dragging = $state(false);
	let fileInput = $state<HTMLInputElement>();

	let inlineError = $state<string | null>(null);
	let phase = $state<'idle' | 'hashing' | 'verifying'>('idle');

	let resultOpen = $state(false);
	let result = $state<VerifyResult | null>(null);

	const HASH_REGEX = /^[0-9a-f]{64}$/;

	const canSubmit = $derived(
		phase === 'idle' && (stagedFile !== null || hashInput.trim().length > 0)
	);

	function selectFile(file: File | undefined | null) {
		if (!file) return;
		stagedFile = file;
		hashInput = '';
		inlineError = null;
	}

	function clearFile() {
		stagedFile = null;
		if (fileInput) fileInput.value = '';
	}

	function onHashInput() {
		if (stagedFile) clearFile();
		inlineError = null;
	}

	async function sha256Hex(file: File): Promise<string> {
		const buf = await file.arrayBuffer();
		const digest = await crypto.subtle.digest('SHA-256', buf);
		return [...new Uint8Array(digest)].map((b) => b.toString(16).padStart(2, '0')).join('');
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (phase !== 'idle') return;
		inlineError = null;

		let hash: string;
		if (stagedFile) {
			phase = 'hashing';
			try {
				hash = await sha256Hex(stagedFile);
			} catch (err) {
				console.error('hash failed', err);
				inlineError = m.verify_invalid_hash();
				phase = 'idle';
				return;
			}
		} else {
			hash = hashInput.trim().toLowerCase();
			if (!HASH_REGEX.test(hash)) {
				inlineError = m.verify_invalid_hash();
				return;
			}
		}

		phase = 'verifying';
		let r: Awaited<ReturnType<typeof verifyDocumentHash>>;
		try {
			r = await verifyDocumentHash(hash);
		} catch (err) {
			console.error('verify failed', err);
			inlineError = m.error_network_retry();
			return;
		} finally {
			// Always leave the verifying state, or the form stays dead after a
			// transport failure until a full reload.
			phase = 'idle';
		}

		if (!r.ok) {
			inlineError = r.error;
			return;
		}

		result = {
			verified: r.result.verified,
			documentName: r.result.documentName,
			status: r.result.status,
			notarizedAt: r.result.notarizedAt,
			hash: r.result.hash ?? hash
		};
		resultOpen = true;
	}

	function statusBadgeClass(status?: string): string {
		switch (status) {
			case 'approved':
				return 'bg-emerald-100 text-emerald-700';
			case 'rejected':
				return 'bg-red-100 text-red-700';
			case 'in_review':
				return 'bg-blue-100 text-blue-700';
			case 'changes_requested':
				return 'bg-amber-100 text-amber-800';
			case 'pending':
			default:
				return 'bg-slate-100 text-slate-700';
		}
	}
</script>

<Card.Root>
	<Card.Header class="text-center">
		<Card.Title class="text-xl">{m.verify_heading()}</Card.Title>
		<Card.Description>{m.verify_subtitle()}</Card.Description>
	</Card.Header>
	<Card.Content>
		<form onsubmit={handleSubmit} class="flex flex-col gap-4">
			{#if inlineError}
				<p class="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
					{inlineError}
				</p>
			{/if}

			<div class="flex flex-col gap-2">
				<Label>{m.verify_file_label()}</Label>
				{#if stagedFile}
					<div class="flex items-center justify-between rounded-lg border bg-muted/40 px-3 py-2">
						<p class="truncate text-sm">
							{m.verify_file_selected({ name: stagedFile.name })}
						</p>
						<Button type="button" variant="ghost" size="sm" class="gap-1.5" onclick={clearFile}>
							<X class="size-3.5" />
							{m.verify_file_clear()}
						</Button>
					</div>
				{:else}
					<button
						type="button"
						class="flex w-full flex-col items-center justify-center rounded-lg border-2 border-dashed p-6 text-center transition-colors {dragging
							? 'border-primary bg-primary/5'
							: 'border-border hover:border-muted-foreground/50 hover:bg-muted/40'}"
						ondragover={(e) => {
							e.preventDefault();
							dragging = true;
						}}
						ondragleave={() => (dragging = false)}
						ondrop={(e) => {
							e.preventDefault();
							dragging = false;
							selectFile(e.dataTransfer?.files[0]);
						}}
						onclick={() => fileInput?.click()}
					>
						<UploadCloud class="mb-2 size-8 text-muted-foreground" />
						<p class="text-xs text-muted-foreground">{m.verify_file_hint()}</p>
					</button>
					<input
						bind:this={fileInput}
						type="file"
						class="hidden"
						onchange={(e) => {
							const f = (e.target as HTMLInputElement).files?.[0];
							selectFile(f);
						}}
					/>
				{/if}
			</div>

			<div class="flex items-center gap-3 text-xs text-muted-foreground uppercase">
				<span class="h-px flex-1 bg-border"></span>
				{m.verify_or()}
				<span class="h-px flex-1 bg-border"></span>
			</div>

			<div class="flex flex-col gap-2">
				<Label for="hash">{m.verify_hash_label()}</Label>
				<Input
					id="hash"
					type="text"
					placeholder={m.verify_hash_placeholder()}
					bind:value={hashInput}
					oninput={onHashInput}
					maxlength={64}
					autocomplete="off"
					spellcheck={false}
					class="font-mono text-xs"
				/>
			</div>

			<Button type="submit" class="w-full" disabled={!canSubmit} aria-busy={phase !== 'idle'}>
				{#if phase === 'hashing'}
					{m.verify_hashing()}
				{:else if phase === 'verifying'}
					{m.verify_verifying()}
				{:else}
					{m.verify_submit()}
				{/if}
			</Button>
		</form>
	</Card.Content>
	{#if showLoginLink}
		<Card.Footer class="justify-center text-sm text-muted-foreground">
			<a href={localizeHref('/login')} class="text-foreground underline-offset-4 hover:underline">
				{m.signup_login_link()}
			</a>
		</Card.Footer>
	{/if}
</Card.Root>

<Dialog.Root bind:open={resultOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				{#if result?.verified}
					<CheckCircle2 class="size-5 text-emerald-600" />
					{m.verify_result_title_verified()}
				{:else}
					<XCircle class="size-5 text-red-600" />
					{m.verify_result_title_not_found()}
				{/if}
			</Dialog.Title>
			<Dialog.Description>
				{#if result?.verified}
					{m.verify_result_verified_description()}
				{:else}
					{m.verify_result_not_found_description()}
				{/if}
			</Dialog.Description>
		</Dialog.Header>

		{#if result}
			<dl class="grid grid-cols-[7rem_1fr] gap-y-2 text-sm">
				{#if result.verified && result.documentName}
					<dt class="text-muted-foreground">{m.verify_result_doc_name()}</dt>
					<dd class="truncate font-medium">{result.documentName}</dd>
				{/if}
				{#if result.verified && result.status}
					<dt class="text-muted-foreground">{m.verify_result_status()}</dt>
					<dd>
						<Badge variant="secondary" class="text-[10px] {statusBadgeClass(result.status)}">
							{result.status}
						</Badge>
					</dd>
				{/if}
				{#if result.verified && result.notarizedAt}
					<dt class="text-muted-foreground">{m.verify_result_notarized_at()}</dt>
					<dd>{formatTimestamp(result.notarizedAt, { withTime: true })}</dd>
				{/if}
				<dt class="text-muted-foreground">{m.verify_result_hash()}</dt>
				<dd class="font-mono text-[11px] break-all">{result.hash}</dd>
			</dl>
		{/if}

		<Dialog.Footer>
			<Dialog.Close>
				{#snippet child({ props })}
					<Button variant="outline" {...props}>{m.verify_result_close()}</Button>
				{/snippet}
			</Dialog.Close>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
