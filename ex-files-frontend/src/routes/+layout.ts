import type { LayoutLoad } from './$types';
import { getMe } from '$lib/queries.remote';
import type { User } from '$lib/api';

export const load: LayoutLoad = async () => {
	let user: User | null = null;
	try {
		user = await getMe().run();
	} catch (err: unknown) {
		console.error('Failed to load user data in layout', err);
	}
	return { user };
};
