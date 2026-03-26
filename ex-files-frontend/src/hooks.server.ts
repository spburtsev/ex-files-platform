import { sequence } from '@sveltejs/kit/hooks';
import { redirect, type Handle } from '@sveltejs/kit';
import { getTextDirection } from '$lib/paraglide/runtime';
import { paraglideMiddleware } from '$lib/paraglide/server';

const PUBLIC_PATHS = ['/login', '/signup'];

const handleAuth: Handle = ({ event, resolve }) => {
	const { pathname } = event.url;
	const isPublic = PUBLIC_PATHS.some((p) => pathname.startsWith(p));
	if (!isPublic && !event.cookies.get('session')) {
		redirect(303, '/login');
	}
	return resolve(event);
};

const handleParaglide: Handle = ({ event, resolve }) =>
	paraglideMiddleware(event.request, ({ request, locale }) => {
		event.request = request;

		return resolve(event, {
			transformPageChunk: ({ html }) =>
				html
					.replace('%paraglide.lang%', locale)
					.replace('%paraglide.dir%', getTextDirection(locale))
		});
	});

export const handle: Handle = sequence(handleAuth, handleParaglide);
