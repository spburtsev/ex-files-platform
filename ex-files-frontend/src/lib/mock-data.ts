export type UserRole = "manager" | "employee";

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
    email: 'spburtsev@gmail.com',
    role: 'manager',
};

export const MOCK_USERS: MockUser[] = [
    {
        id: 'u1',
        name: 'Alex Johnson',
        email: 'a.johnson@acme.org',
        role: 'employee',
    },
    {
        id: 'u2',
        name: 'Maria Chen',
        email: 'm.chen@acme.org',
        role: 'employee',
    },
    {
        id: 'u3',
        name: 'James Wilson',
        email: 'j.wilson@acme.org',
        role: 'employee',
    },
    {
        id: 'u4',
        name: 'Sofia Martinez',
        email: 's.martinez@acme.org',
        role: 'manager',
    }
];

export const MOCK_ASSIGNMENTS: MockAssignment[] = [
    {
        id: 'a1',
        creatorId: 'u4',
        assigneeId: 'u1',
        title: 'Sorting Algorithms Report',
        description:
            'Implement and benchmark QuickSort, MergeSort, and HeapSort. Analyse worst-case complexity.',
        deadline: '2026-03-15T23:59:00',
        resolved: false,
        commentsCount: 4,
        versionsCount: 2,
    },
    {
        id: 'a2',
        creatorId: 'u4',
        assigneeId: 'u1',
        title: 'Binary Search Trees',
        description: 'Build a BST with insert, delete, and balanced-rotation operations.',
        resolved: true,
        commentsCount: 5,
        versionsCount: 3,
    },
    {
        id: 'a3',
        creatorId: 'u4',
        assigneeId: 'u2',
        title: 'Arrays & Linked Lists',
        description: 'Implement a doubly-linked list and compare performance with dynamic arrays.',
        resolved: true,
        commentsCount: 3,
        versionsCount: 2,
    },
    {
        id: 'a5',
        creatorId: 'u4',
        assigneeId: 'u2',
        title: 'Shell Implementation',
        description: 'Implement a Unix-like shell supporting pipes, redirection, and job control.',
        deadline: '2026-03-22T23:59:00',
        resolved: false,
        commentsCount: 6,
        versionsCount: 4,
    },
    {
        id: 'a6',
        creatorId: 'u4',
        assigneeId: 'u3',
        title: 'Memory Allocator',
        description: 'Build a user-space memory allocator with first-fit and best-fit strategies.',
        deadline: '2026-03-20T23:59:00',
        resolved: false,
        commentsCount: 2,
        versionsCount: 1,
    },
    {
        id: 'a8',
        creatorId: 'u4',
        assigneeId: 'u3',
        title: 'Eigenvalue Analysis',
        description: 'Solve eigenvalue and eigenvector problems for given matrices.',
        deadline: '2026-03-16T23:59:00',
        resolved: true,
        commentsCount: 1,
        versionsCount: 1,
    },
    {
        id: 'a16',
        creatorId: 'u4',
        assigneeId: 'u1',
        title: 'Design Document v1',
        description: 'Create UML class, sequence, and component diagrams for the system design.',
        deadline: '2026-03-18T23:59:00',
        resolved: false,
        commentsCount: 2,
        versionsCount: 1,
    },
    {
        id: 'a17',
        creatorId: 'u4',
        assigneeId: 'u2',
        title: 'Network Protocol Analysis',
        description: 'Capture and analyse TCP/IP traffic. Document findings.',
        resolved: true,
        commentsCount: 0,
        versionsCount: 2,
    },
];
