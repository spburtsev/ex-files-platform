export type DocumentStatus = 'draft' | 'saving' | 'saved' | 'error';

export interface Document {
	id: string;
	serverId?: string;
	versionId?: number;
	name: string;
	size: number;
	data: Uint8Array | null;
	uploadedAt: Date;
	pageCount: number;
	status: DocumentStatus;
	error?: string;
	mimeType: string;
}

export interface HydratedDocument {
	serverId: string;
	name: string;
	size: number;
	mimeType: string;
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

interface IssueSlot {
	documents: Document[];
	comments: Comment[];
	activityLog: ActivityEntry[];
	activeDocumentId: string | null;
	hydrated: boolean;
}

function emptySlot(): IssueSlot {
	return {
		documents: [],
		comments: [],
		activityLog: [],
		activeDocumentId: null,
		hydrated: false
	};
}

function createWorkbenchStore() {
	const slots = $state<Record<string, IssueSlot>>({});
	let currentIssueId = $state<string | null>(null);

	const slot = $derived<IssueSlot>(
		currentIssueId && slots[currentIssueId] ? slots[currentIssueId] : emptySlot()
	);
	const activeDocument = $derived(
		slot.documents.find((d) => d.id === slot.activeDocumentId) ?? null
	);
	const activeComments = $derived(
		slot.comments.filter((c) => c.documentId === slot.activeDocumentId)
	);

	function setIssue(issueId: string) {
		if (!slots[issueId]) {
			slots[issueId] = emptySlot();
		}
		currentIssueId = issueId;
	}

	function addActivity(action: ActivityEntry['action'], description: string) {
		if (!currentIssueId) return;
		slots[currentIssueId].activityLog.unshift({
			id: crypto.randomUUID(),
			action,
			description,
			timestamp: new Date()
		});
	}

	function uploadDocument(file: File, data: Uint8Array, pageCount: number) {
		if (!currentIssueId) return;
		const doc: Document = {
			id: crypto.randomUUID(),
			name: file.name,
			size: file.size,
			data,
			uploadedAt: new Date(),
			pageCount,
			status: 'draft',
			mimeType: file.type || 'application/pdf'
		};
		slots[currentIssueId].documents.push(doc);
		slots[currentIssueId].activeDocumentId = doc.id;
		addActivity('upload', `Added draft "${file.name}" (${pageCount} pages)`);
		return doc;
	}

	function setDocumentStatus(id: string, status: DocumentStatus, error?: string) {
		if (!currentIssueId) return;
		const doc = slots[currentIssueId].documents.find((d) => d.id === id);
		if (!doc) return;
		doc.status = status;
		doc.error = error;
	}

	function setDocumentSaved(id: string, serverId: string, versionId: number) {
		if (!currentIssueId) return;
		const doc = slots[currentIssueId].documents.find((d) => d.id === id);
		if (!doc) return;
		doc.status = 'saved';
		doc.error = undefined;
		doc.serverId = serverId;
		doc.versionId = versionId;
	}

	function setDocumentData(id: string, data: Uint8Array, pageCount: number) {
		if (!currentIssueId) return;
		const doc = slots[currentIssueId].documents.find((d) => d.id === id);
		if (!doc) return;
		doc.data = data;
		doc.pageCount = pageCount;
	}

	function hydrate(docs: HydratedDocument[]) {
		if (!currentIssueId) return;
		const s = slots[currentIssueId];
		const existingServerIds = new Set(
			s.documents.map((d) => d.serverId).filter((x): x is string => !!x)
		);
		for (const d of docs) {
			if (existingServerIds.has(d.serverId)) continue;
			s.documents.push({
				id: crypto.randomUUID(),
				serverId: d.serverId,
				name: d.name,
				size: d.size,
				data: null,
				uploadedAt: new Date(),
				pageCount: 0,
				status: 'saved',
				mimeType: d.mimeType
			});
		}
		s.hydrated = true;
		if (!s.activeDocumentId && s.documents.length > 0) {
			s.activeDocumentId = s.documents[0].id;
		}
	}

	function setActiveDocument(id: string) {
		if (!currentIssueId) return;
		slots[currentIssueId].activeDocumentId = id;
		const doc = slots[currentIssueId].documents.find((d) => d.id === id);
		if (doc) {
			addActivity('view', `Opened "${doc.name}"`);
		}
	}

	function addComment(page: number, x: number, y: number, text: string, author: string) {
		if (!currentIssueId) return;
		const activeId = slots[currentIssueId].activeDocumentId;
		if (!activeId) return;
		const comment: Comment = {
			id: crypto.randomUUID(),
			documentId: activeId,
			page,
			x,
			y,
			text,
			author,
			createdAt: new Date()
		};
		slots[currentIssueId].comments.push(comment);
		addActivity(
			'comment',
			`${author} commented on page ${page + 1}: "${text.slice(0, 50)}${text.length > 50 ? '...' : ''}"`
		);
		return comment;
	}

	function deleteComment(id: string) {
		if (!currentIssueId) return;
		const list = slots[currentIssueId].comments;
		const idx = list.findIndex((c) => c.id === id);
		if (idx === -1) return;
		const comment = list[idx];
		list.splice(idx, 1);
		addActivity('delete_comment', `Deleted comment on page ${comment.page + 1}`);
	}

	function discardDocument(id: string) {
		if (!currentIssueId) return;
		const s = slots[currentIssueId];
		const idx = s.documents.findIndex((d) => d.id === id);
		if (idx === -1) return;
		const doc = s.documents[idx];
		s.documents.splice(idx, 1);
		for (let i = s.comments.length - 1; i >= 0; i--) {
			if (s.comments[i].documentId === id) s.comments.splice(i, 1);
		}
		if (s.activeDocumentId === id) {
			s.activeDocumentId = s.documents[0]?.id ?? null;
		}
		addActivity('delete_comment', `Discarded draft "${doc.name}"`);
	}

	return {
		get currentIssueId() {
			return currentIssueId;
		},
		get documents() {
			return slot.documents;
		},
		get comments() {
			return slot.comments;
		},
		get activityLog() {
			return slot.activityLog;
		},
		get activeDocument() {
			return activeDocument;
		},
		get activeDocumentId() {
			return slot.activeDocumentId;
		},
		get activeComments() {
			return activeComments;
		},
		get hydrated() {
			return slot.hydrated;
		},
		setIssue,
		uploadDocument,
		setDocumentStatus,
		setDocumentSaved,
		setDocumentData,
		hydrate,
		setActiveDocument,
		addComment,
		deleteComment,
		discardDocument
	};
}

export const workbenchStore = createWorkbenchStore();
