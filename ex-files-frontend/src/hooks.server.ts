import { sequence } from '@sveltejs/kit/hooks';
import { redirect, type Handle } from '@sveltejs/kit';
import {
	cookieName,
	cookieMaxAge,
	deLocalizeUrl,
	getTextDirection,
	localizeHref
} from '$lib/paraglide/runtime';
import { paraglideMiddleware } from '$lib/paraglide/server';

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
