import type { PageServerLoad } from './$types';
import { MOCK_ASSIGNMENTS, MOCK_USERS } from '$lib/mock-data';

export const load: PageServerLoad = () => ({
	assignments: MOCK_ASSIGNMENTS,
	users: MOCK_USERS,
});
