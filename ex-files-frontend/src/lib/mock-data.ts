export type UserRole = 'manager' | 'employee';

export interface MockUser {
	id: string;
	name: string;
	email: string;
	role: UserRole;
}

export interface MockAssignment {
	id: string;
	creatorId: string;
	assigneeId: string;
	title: string;
	description: string;
	deadline?: string;
	resolved: boolean;
	commentsCount: number;
	versionsCount: number;
}

export const MOCK_ME: MockUser = {
	id: 'u99999',
	name: 'Sergei Burtsev',
	email: 'sergei.p.burtsev@gmail.com',
	role: 'manager'
};
