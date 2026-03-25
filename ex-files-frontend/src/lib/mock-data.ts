export type CompletionType = 'percentage' | 'points' | 'status';

export interface MockUser {
	id: string;
	name: string;
	email: string;
}

export interface MockAssignment {
	id: string;
	userId: string;
	title: string;
	description: string;
	deadline?: string;
	completionType: CompletionType;
	completed: boolean;
	grade?: string;
	commentCount: number;
	submissionCount: number;
}

export const MOCK_USERS: MockUser[] = [
	{
		id: 'u1',
		name: 'Alex Johnson',
		email: 'a.johnson@acme.org'
	},
	{
		id: 'u2',
		name: 'Maria Chen',
		email: 'm.chen@acme.org'
	},
	{
		id: 'u3',
		name: 'James Wilson',
		email: 'j.wilson@acme.org'
	},
	{
		id: 'u4',
		name: 'Sofia Martinez',
		email: 's.martinez@acme.org'
	}
];

export const MOCK_ASSIGNMENTS: MockAssignment[] = [
	{
		id: 'a1',
		userId: 'u1',
		title: 'Sorting Algorithms Report',
		description:
			'Implement and benchmark QuickSort, MergeSort, and HeapSort. Analyse worst-case complexity.',
		deadline: '2026-03-15T23:59:00',
		completionType: 'percentage',
		completed: false,
		commentCount: 4,
		submissionCount: 2
	},
	{
		id: 'a2',
		userId: 'u1',
		title: 'Binary Search Trees',
		description: 'Build a BST with insert, delete, and balanced-rotation operations.',
		completionType: 'percentage',
		completed: true,
		grade: '92%',
		commentCount: 5,
		submissionCount: 3
	},
	{
		id: 'a3',
		userId: 'u2',
		title: 'Arrays & Linked Lists',
		description: 'Implement a doubly-linked list and compare performance with dynamic arrays.',
		completionType: 'status',
		completed: true,
		grade: 'Passed',
		commentCount: 3,
		submissionCount: 2
	},
	{
		id: 'a5',
		userId: 'u2',
		title: 'Shell Implementation',
		description: 'Implement a Unix-like shell supporting pipes, redirection, and job control.',
		completionType: 'status',
		completed: false,
		deadline: '2026-03-22T23:59:00',
		commentCount: 6,
		submissionCount: 4
	},
	{
		id: 'a6',
		userId: 'u3',
		title: 'Memory Allocator',
		description: 'Build a user-space memory allocator with first-fit and best-fit strategies.',
		deadline: '2026-03-20T23:59:00',
		completionType: 'points',
		completed: false,
		commentCount: 2,
		submissionCount: 1
	},
	{
		id: 'a8',
		userId: 'u3',
		title: 'Eigenvalue Analysis',
		description: 'Solve eigenvalue and eigenvector problems for given matrices.',
		deadline: '2026-03-16T23:59:00',
		completionType: 'points',
		completed: true,
		grade: '87/100',
		commentCount: 1,
		submissionCount: 1
	},
	{
		id: 'a16',
		userId: 'u4',
		title: 'Design Document v1',
		description: 'Create UML class, sequence, and component diagrams for the system design.',
		deadline: '2026-03-18T23:59:00',
		completionType: 'percentage',
		completed: false,
		commentCount: 2,
		submissionCount: 1
	},
	{
		id: 'a17',
		userId: 'u4',
		title: 'Network Protocol Analysis',
		description: 'Capture and analyse TCP/IP traffic. Document findings.',
		completionType: 'status',
		completed: true,
		grade: 'Passed',
		commentCount: 0,
		submissionCount: 2
	}
];
