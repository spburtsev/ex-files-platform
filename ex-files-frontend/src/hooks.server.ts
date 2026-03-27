import { sequence } from '@sveltejs/kit/hooks';
import { redirect, type Handle, type HandleFetch } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import {
	cookieName,
	cookieMaxAge,
	deLocalizeUrl,
	getTextDirection,
	localizeHref
} from '$lib/paraglide/runtime';
import { paraglideMiddleware } from '$lib/paraglide/server';

const BACKEND = env.BACKEND_URL ?? 'http://localhost:8080';

const PUBLIC_PATHS = ['/login', '/signup'];

const handleAuth: Handle = ({ event, resolve }) => {
	const pathname = deLocalizeUrl(event.url).pathname;
	const isPublic = PUBLIC_PATHS.some((p) => pathname.startsWith(p));
	if (!isPublic && !event.cookies.get('session')) {
		redirect(303, localizeHref('/login'));
	}
	return resolve(event);
};

const handleParaglide: Handle = async ({ event, resolve }) => {
	let detectedLocale: string | undefined;
	const response = await paraglideMiddleware(event.request, ({ request, locale }) => {
		event.request = request;
		detectedLocale = locale;

		return resolve(event, {
			transformPageChunk: ({ html }) =>
				html
					.replace('%paraglide.lang%', locale)
					.replace('%paraglide.dir%', getTextDirection(locale))
		});
	});
	if (detectedLocale) {
		response.headers.append(
			'set-cookie',
			`${cookieName}=${detectedLocale}; Path=/; Max-Age=${cookieMaxAge}; SameSite=Lax`
		);
	}
	return response;
};

export const handle: Handle = sequence(handleAuth, handleParaglide);

const AUTH_PASSTHROUGH = ['/auth/login', '/auth/register', '/auth/logout'];

export const handleFetch: HandleFetch = async ({ event, request, fetch }) => {
	// Forward session cookie to backend (cross-origin in Docker)
	if (request.url.startsWith(BACKEND)) {
		const session = event.cookies.get('session');
		if (session) {
			request.headers.set('cookie', `session=${session}`);
		}
	}

	const response = await fetch(request);
	if (
		response.status === 401 &&
		!AUTH_PASSTHROUGH.some((p) => request.url.includes(p))
	) {
		event.cookies.delete('session', { path: '/' });
		redirect(303, localizeHref('/login'));
	}
	return response;
};
