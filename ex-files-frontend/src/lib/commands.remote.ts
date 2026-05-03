import { command, getRequestEvent, requested } from '$app/server';

import { apiOpts } from '$lib/api-client';
import {
	authForgotPassword,
	authLogin,
	authLogout,
	authRegister,
	authResetPassword,
	commentsCreate,
	commentsDelete,
	documentsApprove,
	documentsAssignReviewer,
	documentsDelete,
	documentsGetDownloadUrl,
	documentsReject,
	documentsRequestChanges,
	documentsResubmit,
	documentsSubmit,
	documentsUpload,
	documentsUploadVersion,
	issuesArchive,
	issuesCreate,
	issuesUpdateAssignee,
	workspacesAddMember,
	workspacesArchive,
	workspacesCreate,
	workspacesDelete,
	workspacesRemoveMember,
	workspacesUpdate
} from '$lib/api';
import { getComments, getIssues } from './queries.remote';

const NETWORK_ERROR = 'Unable to reach the server. Please try again later.';

function errorMessage(error: unknown, fallback: string): string {
	if (error && typeof error === 'object'){
        if ('error' in error && typeof error.error === 'string') {
            return error.error;
        }
        if ('error_message' in error && typeof error.error_message === 'string') {
            return error.error_message;
        }
	}
	return fallback;
}

function setSessionCookie(token: string) {
	const event = getRequestEvent();
	event.cookies.set('session', token, {
		path: '/',
		httpOnly: true,
		maxAge: 8 * 60 * 60,
		sameSite: 'lax'
	});
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

export const login = command(
	'unchecked',
	async (credentials: { email: string; password: string }) => {
		try {
			const r = await authLogin({ ...apiOpts(), body: credentials });
			if (r.error || !r.data) {
				return {
					ok: false as const,
					error: errorMessage(r.error, 'Invalid email or password')
				};
			}
			setSessionCookie(r.data.token);
			return { ok: true as const };
		} catch (err) {
			console.error('login failed', err);
			return { ok: false as const, error: NETWORK_ERROR };
		}
	}
);

export const register = command(
	'unchecked',
	async (details: { name: string; email: string; password: string }) => {
		try {
			const r = await authRegister({ ...apiOpts(), body: details });
			if (r.error || !r.data) {
				return {
					ok: false as const,
					error: errorMessage(r.error, 'Registration failed. Please try again.')
				};
			}
			setSessionCookie(r.data.token);
			return { ok: true as const };
		} catch {
			return { ok: false as const, error: NETWORK_ERROR };
		}
	}
);

export const forgotPassword = command('unchecked', async (email: string) => {
	try {
		const r = await authForgotPassword({ ...apiOpts(), body: { email } });
		if (r.error) {
			return { ok: false as const, error: errorMessage(r.error, 'Request failed') };
		}
		return { ok: true as const };
	} catch {
		return { ok: false as const, error: NETWORK_ERROR };
	}
});

export const resetPassword = command(
	'unchecked',
	async (data: { token: string; password: string }) => {
		try {
			const r = await authResetPassword({ ...apiOpts(), body: data });
			if (r.error) {
				return {
					ok: false as const,
					error: errorMessage(r.error, 'Reset failed. Token may be invalid or expired.')
				};
			}
			return { ok: true as const };
		} catch {
			return { ok: false as const, error: NETWORK_ERROR };
		}
	}
);

export const logout = command(async () => {
	try {
		await authLogout(apiOpts());
	} catch {
		// Even if backend is down, clear the local session cookie below.
	}
	const event = getRequestEvent();
	event.cookies.delete('session', { path: '/' });
});

// ---------------------------------------------------------------------------
// Workspaces
// ---------------------------------------------------------------------------

export const createWorkspace = command('unchecked', async (name: string) => {
	const r = await workspacesCreate({ ...apiOpts(), body: { name } });
	if (r.error || !r.data) {
		return { ok: false as const, error: errorMessage(r.error, 'Failed to create workspace') };
	}
	return { ok: true as const, workspace: { id: r.data.workspace.id } };
});

export const updateWorkspace = command(
	'unchecked',
	async ({ id, name }: { id: string; name: string }) => {
		const r = await workspacesUpdate({ ...apiOpts(), path: { id }, body: { name } });
		if (r.error) {
			return { ok: false as const, error: errorMessage(r.error, 'Failed to update workspace') };
		}
		return { ok: true as const };
	}
);

export const deleteWorkspace = command('unchecked', async (id: string) => {
	const r = await workspacesDelete({ ...apiOpts(), path: { id } });
	return { ok: !r.error };
});

export const archiveWorkspace = command('unchecked', async (id: string) => {
	const r = await workspacesArchive({ ...apiOpts(), path: { id } });
	return { ok: !r.error };
});

export const archiveIssue = command(
	'unchecked',
	async ({ issueId, archived }: { issueId: string; archived: boolean }) => {
		const r = await issuesArchive({ ...apiOpts(), path: { id: issueId }, body: { archived } });

        for (const { query } of requested(getIssues, 2)) {
			void query.refresh();
		}
        return { ok: !r.error };
	}
);

export const addWorkspaceMember = command(
	'unchecked',
	async ({ workspaceId, userId }: { workspaceId: string; userId: string }) => {
		const r = await workspacesAddMember({
			...apiOpts(),
			path: { id: workspaceId },
			body: { userId }
		});
		if (r.error) {
			return { ok: false as const, error: errorMessage(r.error, 'Failed to add member') };
		}
		return { ok: true as const };
	}
);

export const removeWorkspaceMember = command(
	'unchecked',
	async ({ workspaceId, userId }: { workspaceId: string; userId: string }) => {
		const r = await workspacesRemoveMember({
			...apiOpts(),
			path: { id: workspaceId, userId }
		});
		return { ok: !r.error };
	}
);

// ---------------------------------------------------------------------------
// Issues
// ---------------------------------------------------------------------------

export const createIssue = command(
	'unchecked',
	async ({
		workspaceId,
		title,
		description,
		assigneeId,
		deadline
	}: {
		workspaceId: string;
		title: string;
		description?: string;
		assigneeId: string;
		deadline?: string;
	}) => {
		const r = await issuesCreate({
			...apiOpts(),
			path: { id: workspaceId },
			body: {
				title,
				description,
				assigneeId,
				deadline: deadline ? new Date(deadline).toISOString() : null
			}
		});
		if (r.error) {
			return { ok: false as const, error: errorMessage(r.error, 'Failed to create issue') };
		}
		return { ok: true as const };
	}
);

// ---------------------------------------------------------------------------
// Documents
// ---------------------------------------------------------------------------

export const uploadDocument = command(
	'unchecked',
	async ({
		issueId,
		name,
		mimeType,
		data
	}: {
		issueId: string;
		name: string;
		mimeType: string;
		data: Uint8Array;
	}) => {
		const file = new File([data.slice()], name, { type: mimeType });
		const r = await documentsUpload({
			...apiOpts(),
			path: { id: issueId },
			body: { file }
		});
		if (r.error || !r.data) {
			return { ok: false as const, error: errorMessage(r.error, 'Upload failed') };
		}
		return {
			ok: true as const,
			docId: r.data.document.id,
			versionId: r.data.version.id
		};
	}
);

export const deleteDocument = command('unchecked', async (id: string) => {
	const r = await documentsDelete({ ...apiOpts(), path: { id } });
	return { ok: !r.error };
});

export const uploadDocumentVersion = command(
	'unchecked',
	async ({ docId, file }: { docId: string; file: File }) => {
		const r = await documentsUploadVersion({
			...apiOpts(),
			path: { id: docId },
			body: { file }
		});
		if (r.error) {
			return { ok: false as const, error: errorMessage(r.error, 'Upload failed') };
		}
		return { ok: true as const };
	}
);

export const getDocumentDownloadUrl = command(
	'unchecked',
	async ({ docId, versionId }: { docId: string; versionId: string }) => {
		const r = await documentsGetDownloadUrl({
			...apiOpts(),
			path: { id: docId, versionId }
		});
		if (r.error || !r.data) return { url: null };
		return { url: r.data.url };
	}
);

// ---------------------------------------------------------------------------
// Document workflow
// ---------------------------------------------------------------------------

export const submitDocument = command('unchecked', async (id: string) => {
	const r = await documentsSubmit({ ...apiOpts(), path: { id } });
	if (r.error) return { ok: false as const, error: errorMessage(r.error, 'Action failed') };
	return { ok: true as const };
});

export const resubmitDocument = command('unchecked', async (id: string) => {
	const r = await documentsResubmit({ ...apiOpts(), path: { id } });
	if (r.error) return { ok: false as const, error: errorMessage(r.error, 'Action failed') };
	return { ok: true as const };
});

export const approveDocument = command('unchecked', async (id: string) => {
	const r = await documentsApprove({ ...apiOpts(), path: { id } });
	if (r.error) return { ok: false as const, error: errorMessage(r.error, 'Action failed') };
	return { ok: true as const };
});

export const rejectDocument = command(
	'unchecked',
	async ({ id, note }: { id: string; note: string }) => {
		const r = await documentsReject({ ...apiOpts(), path: { id }, body: { note } });
		if (r.error) return { ok: false as const, error: errorMessage(r.error, 'Action failed') };
		return { ok: true as const };
	}
);

export const requestDocumentChanges = command(
	'unchecked',
	async ({ id, note }: { id: string; note: string }) => {
		const r = await documentsRequestChanges({ ...apiOpts(), path: { id }, body: { note } });
		if (r.error) return { ok: false as const, error: errorMessage(r.error, 'Action failed') };
		return { ok: true as const };
	}
);

export const assignDocumentReviewer = command(
	'unchecked',
	async ({ id, reviewerId }: { id: string; reviewerId: string }) => {
		const r = await documentsAssignReviewer({
			...apiOpts(),
			path: { id },
			body: { reviewerId }
		});
		if (r.error) {
			return { ok: false as const, error: errorMessage(r.error, 'Failed to assign reviewer') };
		}
		return { ok: true as const };
	}
);

export const updateIssueAssignee = command(
	'unchecked',
	async ({ id, assigneeId }: { id: string; assigneeId: string }) => {
		const r = await issuesUpdateAssignee({
			...apiOpts(),
			path: { id },
			body: { assigneeId }
		});
		if (r.error) {
			return { ok: false as const, error: errorMessage(r.error, 'Failed to change assignee') };
		}
		return { ok: true as const };
	}
);

// ---------------------------------------------------------------------------
// Comments
// ---------------------------------------------------------------------------

export const createComment = command(
	'unchecked',
	async ({
		docId,
		body,
		metadata
	}: {
		docId: string;
		body: string;
		metadata: { page: number; x: number; y: number };
	}) => {
		const r = await commentsCreate({
			...apiOpts(),
			path: { id: docId },
			body: { body, metadata }
		});
		if (r.error) {
            return { 
                ok: false as const, 
                error: errorMessage(r.error, 'Failed to comment'),
            };
        }
        for (const { query } of requested(getComments, 1)) {
			void query.refresh();
		}
		return { ok: true as const };
	}
);

export const deleteComment = command(
	'unchecked',
	async ({ docId, commentId }: { docId: string; commentId: string }) => {
		const r = await commentsDelete({
			...apiOpts(),
			path: { id: docId, commentId }
		});
		if (r.error) {
			return { 
                ok: false as const, 
                error: errorMessage(r.error, 'Failed to delete comment'),
            };
        }
        for (const { query } of requested(getComments, 1)) {
			void query.refresh();
		}
		return { ok: true as const };
	}
);
