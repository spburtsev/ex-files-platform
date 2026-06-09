const raw = process.env.PUBLIC_MAX_UPLOAD_MB ?? '25';

if (!process.env.BODY_SIZE_LIMIT) {
	if (typeof raw === 'string' && raw.toLowerCase() === 'infinity') {
		process.env.BODY_SIZE_LIMIT = 'Infinity';
	} else {
		const mb = Number(raw);
		if (Number.isFinite(mb) && mb > 0) {
			// Uploads travel as a Uint8Array argument to a remote command, which
			// devalue serializes as base64 (~4/3 inflation) inside a JSON envelope.
			// The adapter limit must cover the encoded size of a max-size file,
			// not the raw size, or files in the top quarter of the allowed range
			// get rejected with 413. 4/3 plus 1 MiB headroom for the envelope.
			const rawBytes = mb * 1024 * 1024;
			process.env.BODY_SIZE_LIMIT = String(Math.ceil((rawBytes * 4) / 3) + 1024 * 1024);
		}
	}
}

await import('./build/index.js');
