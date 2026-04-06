import type { LayoutLoad } from "./$types";
import { getMe } from "$lib/data.remote";

export const load: LayoutLoad = async () => {
    let user;
    try {
        user = await getMe();
    } catch (err: unknown) {
        console.error('Failed to load user data in layout', err);
        user = null;
    }
    return { user };
};

