import type { LayoutServerLoad } from './$types';
import { MOCK_ME } from '$lib/mock-data';

export const load: LayoutServerLoad = () => ({ me: MOCK_ME });
