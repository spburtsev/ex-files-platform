import type { DocumentStatus as ApiDocumentStatus } from '$lib/api';

export type DocumentStatus = 'draft' | 'saving' | 'saved' | 'error';
export type ReviewStatus = ApiDocumentStatus;

export interface Document {
	id: string;
	serverId?: string;
	versionId?: string;
	name: string;
	size: number;
	data: Uint8Array | null;
	uploadedAt: Date;
	pageCount: number;
	status: DocumentStatus;
	error?: string;
	mimeType: string;
	uploaderName?: string;
	reviewStatus?: ReviewStatus;
}

export interface HydratedDocument {
	serverId: string;
	name: string;
	size: number;
	mimeType: string;
	uploaderName?: string;
	reviewStatus?: ReviewStatus;
}

interface IssueSlot {
	documents: Document[];
	activeDocumentId: string | null;
	hydrated: boolean;
}

function emptySlot(): IssueSlot {
	return {
		documents: [],
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

	function setIssue(issueId: string) {
		if (!slots[issueId]) {
			slots[issueId] = emptySlot();
		}
		currentIssueId = issueId;
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
		return doc;
	}

	function setDocumentStatus(id: string, status: DocumentStatus, error?: string) {
		if (!currentIssueId) return;
		const doc = slots[currentIssueId].documents.find((d) => d.id === id);
		if (!doc) return;
		doc.status = status;
		doc.error = error;
	}

	function setDocumentSaved(id: string, serverId: string, versionId: string) {
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

	function setDocumentReviewStatus(id: string, reviewStatus: ReviewStatus) {
		if (!currentIssueId) return;
		const doc = slots[currentIssueId].documents.find((d) => d.id === id);
		if (!doc) return;
		doc.reviewStatus = reviewStatus;
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
				mimeType: d.mimeType,
				uploaderName: d.uploaderName,
				reviewStatus: d.reviewStatus
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
	}

	function discardDocument(id: string) {
		if (!currentIssueId) return;
		const s = slots[currentIssueId];
		const idx = s.documents.findIndex((d) => d.id === id);
		if (idx === -1) return;
		s.documents.splice(idx, 1);
		if (s.activeDocumentId === id) {
			s.activeDocumentId = s.documents[0]?.id ?? null;
		}
	}

	return {
		get currentIssueId() {
			return currentIssueId;
		},
		get documents() {
			return slot.documents;
		},
		get activeDocument() {
			return activeDocument;
		},
		get activeDocumentId() {
			return slot.activeDocumentId;
		},
		get hydrated() {
			return slot.hydrated;
		},
		setIssue,
		uploadDocument,
		setDocumentStatus,
		setDocumentSaved,
		setDocumentData,
		setDocumentReviewStatus,
		hydrate,
		setActiveDocument,
		discardDocument
	};
}

export const workbenchStore = createWorkbenchStore();
