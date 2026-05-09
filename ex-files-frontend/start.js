const raw = process.env.PUBLIC_MAX_UPLOAD_MB ?? '25';

if (!process.env.BODY_SIZE_LIMIT) {
	if (typeof raw === 'string' && raw.toLowerCase() === 'infinity') {
		process.env.BODY_SIZE_LIMIT = 'Infinity';
	} else {
		const mb = Number(raw);
		if (Number.isFinite(mb) && mb > 0) {
			process.env.BODY_SIZE_LIMIT = String(Math.round(mb * 1024 * 1024));
		}
	}
}

await import('./build/index.js');
