export interface Document {
	id: string;
	name: string;
	size: number;
	data: Uint8Array;
	uploadedAt: Date;
	pageCount: number;
}

export interface Comment {
	id: string;
	documentId: string;
	page: number;
	x: number;
	y: number;
	text: string;
	author: string;
	createdAt: Date;
}

export interface ActivityEntry {
	id: string;
	action: 'upload' | 'comment' | 'delete_comment' | 'view';
	description: string;
	timestamp: Date;
}

function createWorkbenchStore() {
	const documents = $state<Document[]>([]);
	const comments = $state<Comment[]>([]);
	const activityLog = $state<ActivityEntry[]>([]);
	let activeDocumentId = $state<string | null>(null);

	const activeDocument = $derived(documents.find((d) => d.id === activeDocumentId) ?? null);
	const activeComments = $derived(comments.filter((c) => c.documentId === activeDocumentId));

	function addActivity(action: ActivityEntry['action'], description: string) {
		activityLog.unshift({
			id: crypto.randomUUID(),
			action,
			description,
			timestamp: new Date()
		});
	}

	function uploadDocument(file: File, data: Uint8Array, pageCount: number) {
		const doc: Document = {
			id: crypto.randomUUID(),
			name: file.name,
			size: file.size,
			data,
			uploadedAt: new Date(),
			pageCount
		};
		documents.push(doc);
		activeDocumentId = doc.id;
		addActivity('upload', `Uploaded "${file.name}" (${pageCount} pages)`);
		return doc;
	}

	function setActiveDocument(id: string) {
		activeDocumentId = id;
		const doc = documents.find((d) => d.id === id);
		if (doc) {
			addActivity('view', `Opened "${doc.name}"`);
		}
	}

	function addComment(page: number, x: number, y: number, text: string, author: string) {
		if (!activeDocumentId) return;
		const comment: Comment = {
			id: crypto.randomUUID(),
			documentId: activeDocumentId,
			page,
			x,
			y,
			text,
			author,
			createdAt: new Date()
		};
		comments.push(comment);
		addActivity(
			'comment',
			`${author} commented on page ${page + 1}: "${text.slice(0, 50)}${text.length > 50 ? '...' : ''}"`
		);
		return comment;
	}

	function deleteComment(id: string) {
		const idx = comments.findIndex((c) => c.id === id);
		if (idx === -1) return;
		const comment = comments[idx];
		comments.splice(idx, 1);
		addActivity('delete_comment', `Deleted comment on page ${comment.page + 1}`);
	}

	return {
		get documents() {
			return documents;
		},
		get comments() {
			return comments;
		},
		get activityLog() {
			return activityLog;
		},
		get activeDocument() {
			return activeDocument;
		},
		get activeDocumentId() {
			return activeDocumentId;
		},
		get activeComments() {
			return activeComments;
		},
		uploadDocument,
		setActiveDocument,
		addComment,
		deleteComment
	};
}

export const workbenchStore = createWorkbenchStore();
