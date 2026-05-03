import { command, getRequestEvent, requested } from '$app/server';

import { apiOpts } from '$lib/api-client';
import {
	authChangePassword,
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

type ApiResult<T extends { error?: unknown }> = T;
type CommandResult = { ok: true } | { ok: false; error: string };

async function runApi<T extends { error?: unknown }>(
	call: Promise<ApiResult<T>>,
	fallback: string
): Promise<CommandResult> {
	try {
		const r = await call;
		if (r.error) return { ok: false as const, error: errorMessage(r.error, fallback) };
		return { ok: true as const };
	} catch {
		return { ok: false as const, error: NETWORK_ERROR };
	}
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
            console.error('forgot password failed:', r.error);
			return { ok: false as const, error: errorMessage(r.error, 'Request failed') };
		}
		return { ok: true as const };
	} catch (err: unknown) {
        console.error('forgot password failed:', String(err));
		return { ok: false as const, error: NETWORK_ERROR };
	}
});

export const resetPassword = command(
	'unchecked',
	async (data: { token: string; password: string }) =>
		runApi(authResetPassword({ ...apiOpts(), body: data }), 'Reset failed. Token may be invalid or expired.')
);

export const changePassword = command(
	'unchecked',
	async ({ oldPassword, newPassword }: { oldPassword: string; newPassword: string }) =>
		runApi(
			authChangePassword({ ...apiOpts(), body: { oldPassword, newPassword } }),
			'Failed to change password'
		)
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
	async ({ id, name }: { id: string; name: string }) =>
		runApi(
			workspacesUpdate({ ...apiOpts(), path: { id }, body: { name } }),
			'Failed to update workspace'
		)
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
	async ({ workspaceId, userId }: { workspaceId: string; userId: string }) =>
		runApi(
			workspacesAddMember({ ...apiOpts(), path: { id: workspaceId }, body: { userId } }),
			'Failed to add member'
		)
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
	}) =>
		runApi(
			issuesCreate({
				...apiOpts(),
				path: { id: workspaceId },
				body: {
					title,
					description,
					assigneeId,
					deadline: deadline ? new Date(deadline).toISOString() : null
				}
			}),
			'Failed to create issue'
		)
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
	async ({ docId, file }: { docId: string; file: File }) =>
		runApi(
			documentsUploadVersion({ ...apiOpts(), path: { id: docId }, body: { file } }),
			'Upload failed'
		)
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

export const submitDocument = command('unchecked', async (id: string) =>
	runApi(documentsSubmit({ ...apiOpts(), path: { id } }), 'Action failed')
);

export const resubmitDocument = command('unchecked', async (id: string) =>
	runApi(documentsResubmit({ ...apiOpts(), path: { id } }), 'Action failed')
);

export const approveDocument = command('unchecked', async (id: string) =>
	runApi(documentsApprove({ ...apiOpts(), path: { id } }), 'Action failed')
);

export const rejectDocument = command(
	'unchecked',
	async ({ id, note }: { id: string; note: string }) =>
		runApi(documentsReject({ ...apiOpts(), path: { id }, body: { note } }), 'Action failed')
);

export const requestDocumentChanges = command(
	'unchecked',
	async ({ id, note }: { id: string; note: string }) =>
		runApi(
			documentsRequestChanges({ ...apiOpts(), path: { id }, body: { note } }),
			'Action failed'
		)
);

export const assignDocumentReviewer = command(
	'unchecked',
	async ({ id, reviewerId }: { id: string; reviewerId: string }) =>
		runApi(
			documentsAssignReviewer({ ...apiOpts(), path: { id }, body: { reviewerId } }),
			'Failed to assign reviewer'
		)
);

export const updateIssueAssignee = command(
	'unchecked',
	async ({ id, assigneeId }: { id: string; assigneeId: string }) =>
		runApi(
			issuesUpdateAssignee({ ...apiOpts(), path: { id }, body: { assigneeId } }),
			'Failed to change assignee'
		)
);

// ---------------------------------------------------------------------------
// Comments
// ---------------------------------------------------------------------------

function refreshComments() {
	for (const { query } of requested(getComments, 1)) {
		void query.refresh();
	}
}

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
		const r = await runApi(
			commentsCreate({ ...apiOpts(), path: { id: docId }, body: { body, metadata } }),
			'Failed to comment'
		);
		if (r.ok) refreshComments();
		return r;
	}
);

export const deleteComment = command(
	'unchecked',
	async ({ docId, commentId }: { docId: string; commentId: string }) => {
		const r = await runApi(
			commentsDelete({ ...apiOpts(), path: { id: docId, commentId } }),
			'Failed to delete comment'
		);
		if (r.ok) refreshComments();
		return r;
	}
);
