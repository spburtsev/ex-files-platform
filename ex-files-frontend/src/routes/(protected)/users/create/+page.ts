import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// Root-only page. The backend also enforces this (403), but redirect early for UX.
export const load: PageLoad = async ({ parent }) => {
	const { user } = await parent();
	if (user?.role !== 'root') {
		redirect(302, '/users');
	}
	return {};
};
