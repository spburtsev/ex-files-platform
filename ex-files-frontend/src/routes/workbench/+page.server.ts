import type { PageServerLoad } from './$types';
import { MOCK_ASSIGNMENTS, MOCK_USERS } from '$lib/mock-data';

export const load: PageServerLoad = () => {
	const assignment = MOCK_ASSIGNMENTS[0];
	const user = MOCK_USERS.find((u) => u.id === assignment.assigneeId) ?? null;
	return { assignment, user };
};
