import { env } from '$env/dynamic/public';

// Single source of truth on the client side. The matching server limit is
// derived in start.js from the same PUBLIC_MAX_UPLOAD_MB env var so the
// client check and the adapter limit always agree.
const raw = env.PUBLIC_MAX_UPLOAD_MB ?? '25';

export const MAX_UPLOAD_MB =
	typeof raw === 'string' && raw.toLowerCase() === 'infinity'
		? Infinity
		: Number(raw) || 25;

export const MAX_UPLOAD_BYTES = MAX_UPLOAD_MB * 1024 * 1024;
