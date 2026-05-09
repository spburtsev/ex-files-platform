import { redirect } from '@sveltejs/kit';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ parent, data }) => {
    const { user } = await parent();
    if (!user) {
        redirect(302, '/login');
    }
    return { ...data, user };
};
