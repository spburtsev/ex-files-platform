package handlers

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
)

func documentToOAPI(d *models.Document) oapi.Document {
	out := oapi.Document{
		ID:           formatID(d.ID),
		Name:         d.Name,
		MimeType:     d.MimeType,
		Size:         d.Size,
		Hash:         d.Hash,
		Status:       oapi.DocumentStatus(d.Status),
		UploaderId:   formatID(d.UploaderID),
		UploaderName: d.Uploader.Name,
		IssueId:      formatID(d.IssueID),
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
	if d.ReviewerID != nil {
		out.ReviewerId = oapi.NewOptNilString(formatID(*d.ReviewerID))
		out.ReviewerName = oapi.NewOptNilString(d.Reviewer.Name)
	}
	if d.ReviewerNote != "" {
		out.ReviewerNote = oapi.NewOptString(d.ReviewerNote)
	}
	return out
}

func versionToOAPI(v *models.DocumentVersion) oapi.DocumentVersion {
	return oapi.DocumentVersion{
		ID:           formatID(v.ID),
		DocumentId:   formatID(v.DocumentID),
		Version:      int32(v.Version),
		Hash:         v.Hash,
		Size:         v.Size,
		StorageKey:   v.StorageKey,
		UploaderId:   formatID(v.UploaderID),
		UploaderName: v.Uploader.Name,
		CreatedAt:    v.CreatedAt,
	}
}

// DocumentsList implements GET /issues/{id}/documents.
func (s *Server) DocumentsList(ctx context.Context, params oapi.DocumentsListParams) (oapi.DocumentsListRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.DocumentsListUnauthorized{Error: "unauthorized"}, nil
	}
	issueID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsListInternalServerError{Error: "invalid issue id"}, nil
	}

	page, perPage, offset := resolvePagination(params.Page, params.PerPage)
	search := params.Search.Or("")
	status := ""
	if v, ok := params.Status.Get(); ok {
		status = string(v)
	}

	docs, total, err := s.DocumentRepo.ListByIssue(issueID, search, status, perPage, offset)
	if err != nil {
		logErr("documents.list", err)
		return &oapi.DocumentsListInternalServerError{Error: "failed to fetch documents"}, nil
	}

	out := make([]oapi.Document, len(docs))
	for i := range docs {
		out[i] = documentToOAPI(&docs[i])
	}
	return &oapi.ListDocumentsResponseHeaders{
		XPage:       optInt32(page),
		XPerPage:    optInt32(perPage),
		XTotalCount: optInt64(total),
		XTotalPages: optInt32(totalPages(total, perPage)),
		Response:    oapi.ListDocumentsResponse{Documents: out},
	}, nil
}

// DocumentsUpload implements POST /issues/{id}/documents.
func (s *Server) DocumentsUpload(ctx context.Context, req *oapi.DocumentsUploadReq, params oapi.DocumentsUploadParams) (oapi.DocumentsUploadRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.DocumentsUploadUnauthorized{Error: "unauthorized"}, nil
	}
	issueID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsUploadBadRequest{Error: "invalid issue id"}, nil
	}

	issue, err := s.IssueRepo.FindByID(issueID)
	if err != nil {
		return &oapi.DocumentsUploadBadRequest{Error: "issue not found"}, nil
	}
	if issue.Resolved {
		return &oapi.DocumentsUploadUnprocessableEntity{
			Error: "issue is resolved; no more documents may be uploaded",
		}, nil
	}

	mp := req.File

	hasher := sha256.New()
	if _, err := io.Copy(hasher, mp.File); err != nil {
		logErr("documents.upload.hash", err)
		return &oapi.DocumentsUploadInternalServerError{Error: "failed to hash file"}, nil
	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	if existing, err := s.DocumentRepo.FindByIssueAndHash(issueID, hash); err == nil && existing != nil {
		return &oapi.DocumentsUploadConflict{
			Error: fmt.Sprintf("issue already has a document with the same content: %q", existing.Name),
		}, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logErr("documents.upload.lookup", err)
		return &oapi.DocumentsUploadInternalServerError{Error: "failed to check existing documents"}, nil
	}

	seeker, ok := mp.File.(io.Seeker)
	if !ok {
		return &oapi.DocumentsUploadInternalServerError{Error: "uploaded file is not seekable"}, nil
	}
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		logErr("documents.upload.seek", err)
		return &oapi.DocumentsUploadInternalServerError{Error: "failed to read file"}, nil
	}

	mimeType := mp.Header.Get("Content-Type")
	doc := models.Document{
		Name:       mp.Name,
		MimeType:   mimeType,
		Size:       mp.Size,
		Hash:       hash,
		Status:     models.DocumentStatusPending,
		UploaderID: uid,
		IssueID:    issueID,
	}
	if err := s.DocumentRepo.Create(&doc); err != nil {
		logErr("documents.upload.create", err)
		return &oapi.DocumentsUploadInternalServerError{Error: "failed to create document"}, nil
	}

	storageKey := fmt.Sprintf("issues/%d/documents/%d/v1/%s", issueID, doc.ID, mp.Name)
	if err := s.Storage.Upload(ctx, storageKey, mp.File, mp.Size, mimeType); err != nil {
		logErr("documents.upload.storage", err)
		return &oapi.DocumentsUploadInternalServerError{Error: "failed to upload file"}, nil
	}

	version := models.DocumentVersion{
		DocumentID: doc.ID,
		Version:    1,
		Hash:       hash,
		Size:       mp.Size,
		StorageKey: storageKey,
		UploaderID: uid,
	}
	if err := s.DocumentRepo.CreateVersion(&version); err != nil {
		logErr("documents.upload.create_version", err)
		return &oapi.DocumentsUploadInternalServerError{Error: "failed to create version"}, nil
	}

	logAudit(s.Audit, models.AuditActionDocumentUploaded, uid, uintPtr(doc.ID), "document", map[string]any{
		"name":     doc.Name,
		"hash":     hash,
		"issue_id": issueID,
	})

	loaded, err := s.DocumentRepo.FindByID(doc.ID)
	if err != nil {
		logErr("documents.upload.reload", err)
		return &oapi.DocumentsUploadInternalServerError{Error: "failed to reload document"}, nil
	}

	return &oapi.UploadDocumentResponse{
		Document: documentToOAPI(loaded),
		Version:  versionToOAPI(&version),
	}, nil
}

// DocumentsUploadVersion implements POST /documents/{id}/versions.
func (s *Server) DocumentsUploadVersion(ctx context.Context, req *oapi.DocumentsUploadVersionReq, params oapi.DocumentsUploadVersionParams) (oapi.DocumentsUploadVersionRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.DocumentsUploadVersionUnauthorized{Error: "unauthorized"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsUploadVersionBadRequest{Error: "invalid document id"}, nil
	}
	doc, err := s.DocumentRepo.FindByID(docID)
	if err != nil {
		return &oapi.DocumentsUploadVersionNotFound{Error: "document not found"}, nil
	}

	mp := req.File

	hasher := sha256.New()
	if _, err := io.Copy(hasher, mp.File); err != nil {
		logErr("documents.upload_version.hash", err)
		return &oapi.DocumentsUploadVersionInternalServerError{Error: "failed to hash file"}, nil
	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	seeker, ok := mp.File.(io.Seeker)
	if !ok {
		return &oapi.DocumentsUploadVersionInternalServerError{Error: "uploaded file is not seekable"}, nil
	}
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		logErr("documents.upload_version.seek", err)
		return &oapi.DocumentsUploadVersionInternalServerError{Error: "failed to read file"}, nil
	}

	latest, err := s.DocumentRepo.LatestVersionNumber(docID)
	if err != nil {
		logErr("documents.upload_version.latest", err)
		return &oapi.DocumentsUploadVersionInternalServerError{Error: "failed to get version info"}, nil
	}
	newVersion := latest + 1

	storageKey := fmt.Sprintf("issues/%d/documents/%d/v%d/%s", doc.IssueID, doc.ID, newVersion, mp.Name)
	mimeType := mp.Header.Get("Content-Type")
	if err := s.Storage.Upload(ctx, storageKey, mp.File, mp.Size, mimeType); err != nil {
		logErr("documents.upload_version.storage", err)
		return &oapi.DocumentsUploadVersionInternalServerError{Error: "failed to upload file"}, nil
	}

	version := models.DocumentVersion{
		DocumentID: docID,
		Version:    newVersion,
		Hash:       hash,
		Size:       mp.Size,
		StorageKey: storageKey,
		UploaderID: uid,
	}
	if err := s.DocumentRepo.CreateVersion(&version); err != nil {
		logErr("documents.upload_version.create", err)
		return &oapi.DocumentsUploadVersionInternalServerError{Error: "failed to create version"}, nil
	}

	logAudit(s.Audit, models.AuditActionVersionCreated, uid, uintPtr(doc.ID), "document", map[string]any{
		"version": newVersion,
		"hash":    hash,
	})

	return &oapi.UploadDocumentResponse{
		Document: documentToOAPI(doc),
		Version:  versionToOAPI(&version),
	}, nil
}

// DocumentsGet implements GET /documents/{id}.
func (s *Server) DocumentsGet(ctx context.Context, params oapi.DocumentsGetParams) (oapi.DocumentsGetRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.DocumentsGetUnauthorized{Error: "unauthorized"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsGetNotFound{Error: "document not found"}, nil
	}
	doc, err := s.DocumentRepo.FindByID(docID)
	if err != nil {
		return &oapi.DocumentsGetNotFound{Error: "document not found"}, nil
	}
	versions, err := s.DocumentRepo.GetVersions(docID)
	if err != nil {
		logErr("documents.get.versions", err)
		return &oapi.DocumentsGetInternalServerError{Error: "failed to fetch versions"}, nil
	}
	out := make([]oapi.DocumentVersion, len(versions))
	for i := range versions {
		out[i] = versionToOAPI(&versions[i])
	}
	return &oapi.GetDocumentResponse{
		Document: oapi.DocumentDetail{
			Document: documentToOAPI(doc),
			Versions: out,
		},
	}, nil
}

// DocumentsGetFile implements GET /documents/{id}/versions/{versionId}/file.
func (s *Server) DocumentsGetFile(ctx context.Context, params oapi.DocumentsGetFileParams) (oapi.DocumentsGetFileRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.DocumentsGetFileUnauthorized{Error: "unauthorized"}, nil
	}
	versionID, ok := parseUintID(params.VersionId)
	if !ok {
		return &oapi.DocumentsGetFileNotFound{Error: "invalid version id"}, nil
	}
	version, err := s.DocumentRepo.GetVersion(versionID)
	if err != nil {
		return &oapi.DocumentsGetFileNotFound{Error: "version not found"}, nil
	}
	reader, err := s.Storage.Get(ctx, version.StorageKey)
	if err != nil {
		logErr("documents.file.read", err)
		return &oapi.DocumentsGetFileInternalServerError{Error: "failed to read file"}, nil
	}
	out := oapi.DocumentsGetFileOKHeaders{
		Response: oapi.DocumentsGetFileOK{Data: reader},
	}
	if version.Size > 0 {
		out.ContentLength = optInt64(version.Size)
	}
	return &out, nil
}

// DocumentsGetDownloadUrl implements GET /documents/{id}/versions/{versionId}/download.
func (s *Server) DocumentsGetDownloadUrl(ctx context.Context, params oapi.DocumentsGetDownloadUrlParams) (oapi.DocumentsGetDownloadUrlRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.DocumentsGetDownloadUrlUnauthorized{Error: "unauthorized"}, nil
	}
	versionID, ok := parseUintID(params.VersionId)
	if !ok {
		return &oapi.DocumentsGetDownloadUrlNotFound{Error: "invalid version id"}, nil
	}
	version, err := s.DocumentRepo.GetVersion(versionID)
	if err != nil {
		return &oapi.DocumentsGetDownloadUrlNotFound{Error: "version not found"}, nil
	}
	signedURL, err := s.Storage.PresignedURL(ctx, version.StorageKey, 15*time.Minute)
	if err != nil {
		logErr("documents.download.presign", err)
		return &oapi.DocumentsGetDownloadUrlInternalServerError{Error: "failed to generate download URL"}, nil
	}
	parsed, err := url.Parse(signedURL)
	if err != nil {
		logErr("documents.download.parse", err)
		return &oapi.DocumentsGetDownloadUrlInternalServerError{Error: "failed to parse download URL"}, nil
	}
	return &oapi.GetDownloadUrlResponse{URL: *parsed}, nil
}

// DocumentsSubmit implements POST /documents/{id}/submit.
func (s *Server) DocumentsSubmit(ctx context.Context, params oapi.DocumentsSubmitParams) (oapi.DocumentsSubmitRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.DocumentsSubmitUnauthorized{Error: "unauthorized"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsSubmitNotFound{Error: "document not found"}, nil
	}
	doc, err := s.DocumentRepo.FindByID(docID)
	if err != nil {
		return &oapi.DocumentsSubmitNotFound{Error: "document not found"}, nil
	}
	if doc.UploaderID != uid {
		return &oapi.DocumentsSubmitForbidden{Error: "only the uploader may submit this document"}, nil
	}
	if !doc.CanTransitionTo(models.DocumentStatusInReview) {
		return &oapi.DocumentsSubmitUnprocessableEntity{Error: "document cannot be submitted from its current status"}, nil
	}
	doc.Status = models.DocumentStatusInReview
	doc.ReviewerNote = ""
	if err := s.DocumentRepo.Update(doc); err != nil {
		logErr("documents.submit.update", err)
		return &oapi.DocumentsSubmitInternalServerError{Error: "failed to update document"}, nil
	}
	logAudit(s.Audit, models.AuditActionDocumentSubmitted, uid, uintPtr(doc.ID), "document", map[string]any{
		"document_id": doc.ID,
	})
	return &oapi.UpdateDocumentResponse{Document: documentToOAPI(doc)}, nil
}

// DocumentsResubmit implements POST /documents/{id}/resubmit.
func (s *Server) DocumentsResubmit(ctx context.Context, params oapi.DocumentsResubmitParams) (oapi.DocumentsResubmitRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.DocumentsResubmitUnauthorized{Error: "unauthorized"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsResubmitNotFound{Error: "document not found"}, nil
	}
	doc, err := s.DocumentRepo.FindByID(docID)
	if err != nil {
		return &oapi.DocumentsResubmitNotFound{Error: "document not found"}, nil
	}
	if doc.UploaderID != uid {
		return &oapi.DocumentsResubmitForbidden{Error: "only the uploader may resubmit this document"}, nil
	}
	if !doc.CanTransitionTo(models.DocumentStatusInReview) {
		return &oapi.DocumentsResubmitUnprocessableEntity{Error: "document cannot be resubmitted from its current status"}, nil
	}
	doc.Status = models.DocumentStatusInReview
	doc.ReviewerNote = ""
	if err := s.DocumentRepo.Update(doc); err != nil {
		logErr("documents.resubmit.update", err)
		return &oapi.DocumentsResubmitInternalServerError{Error: "failed to update document"}, nil
	}
	logAudit(s.Audit, models.AuditActionDocumentSubmitted, uid, uintPtr(doc.ID), "document", map[string]any{
		"document_id": doc.ID,
		"resubmit":    true,
	})
	return &oapi.UpdateDocumentResponse{Document: documentToOAPI(doc)}, nil
}

func canReview(doc *models.Document, callerID uint, role models.Role) bool {
	if role.CanManageWorkspaces() {
		return true
	}
	return doc.ReviewerID != nil && *doc.ReviewerID == callerID
}

// DocumentsApprove implements POST /documents/{id}/approve.
func (s *Server) DocumentsApprove(ctx context.Context, params oapi.DocumentsApproveParams) (oapi.DocumentsApproveRes, error) {
	uid, role, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.DocumentsApproveUnauthorized{Error: "unauthorized"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsApproveNotFound{Error: "document not found"}, nil
	}
	doc, err := s.DocumentRepo.FindByID(docID)
	if err != nil {
		return &oapi.DocumentsApproveNotFound{Error: "document not found"}, nil
	}
	if !canReview(doc, uid, role) {
		return &oapi.DocumentsApproveForbidden{Error: "not authorized to review this document"}, nil
	}
	if !doc.CanTransitionTo(models.DocumentStatusApproved) {
		return &oapi.DocumentsApproveUnprocessableEntity{Error: "document cannot be approved from its current status"}, nil
	}
	doc.Status = models.DocumentStatusApproved
	if err := s.DocumentRepo.Update(doc); err != nil {
		logErr("documents.approve.update", err)
		return &oapi.DocumentsApproveInternalServerError{Error: "failed to update document"}, nil
	}

	if issue, err := s.IssueRepo.FindByID(doc.IssueID); err != nil {
		logErr("documents.approve.issue_lookup", err)
	} else if !issue.Resolved {
		issue.Resolved = true
		if err := s.IssueRepo.Update(issue); err != nil {
			logErr("documents.approve.issue_resolve", err)
		}
	}

	logAudit(s.Audit, models.AuditActionDocumentApproved, uid, uintPtr(doc.ID), "document", map[string]any{
		"document_id": doc.ID,
	})
	notifyDocumentEvent(s.Email, s.UserRepo, s.Hub, doc, "document.approved",
		fmt.Sprintf("Document approved: %s", doc.Name),
		fmt.Sprintf("<p>Your document <strong>%s</strong> has been approved.</p>", doc.Name),
	)
	return &oapi.UpdateDocumentResponse{Document: documentToOAPI(doc)}, nil
}

// DocumentsReject implements POST /documents/{id}/reject.
func (s *Server) DocumentsReject(ctx context.Context, req oapi.OptReviewNoteRequest, params oapi.DocumentsRejectParams) (oapi.DocumentsRejectRes, error) {
	uid, role, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.DocumentsRejectUnauthorized{Error: "unauthorized"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsRejectNotFound{Error: "document not found"}, nil
	}
	doc, err := s.DocumentRepo.FindByID(docID)
	if err != nil {
		return &oapi.DocumentsRejectNotFound{Error: "document not found"}, nil
	}
	if !canReview(doc, uid, role) {
		return &oapi.DocumentsRejectForbidden{Error: "not authorized to review this document"}, nil
	}
	if !doc.CanTransitionTo(models.DocumentStatusRejected) {
		return &oapi.DocumentsRejectUnprocessableEntity{Error: "document cannot be rejected from its current status"}, nil
	}
	note := ""
	if v, ok := req.Get(); ok {
		note = v.Note.Or("")
	}
	doc.Status = models.DocumentStatusRejected
	doc.ReviewerNote = note
	if err := s.DocumentRepo.Update(doc); err != nil {
		logErr("documents.reject.update", err)
		return &oapi.DocumentsRejectInternalServerError{Error: "failed to update document"}, nil
	}
	logAudit(s.Audit, models.AuditActionDocumentRejected, uid, uintPtr(doc.ID), "document", map[string]any{
		"document_id": doc.ID,
		"note":        note,
	})
	notifyDocumentEvent(s.Email, s.UserRepo, s.Hub, doc, "document.rejected",
		fmt.Sprintf("Document rejected: %s", doc.Name),
		fmt.Sprintf("<p>Your document <strong>%s</strong> has been rejected.</p><p>Reason: %s</p>", doc.Name, note),
	)
	return &oapi.UpdateDocumentResponse{Document: documentToOAPI(doc)}, nil
}

// DocumentsRequestChanges implements POST /documents/{id}/request-changes.
func (s *Server) DocumentsRequestChanges(ctx context.Context, req oapi.OptReviewNoteRequest, params oapi.DocumentsRequestChangesParams) (oapi.DocumentsRequestChangesRes, error) {
	uid, role, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.DocumentsRequestChangesUnauthorized{Error: "unauthorized"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsRequestChangesNotFound{Error: "document not found"}, nil
	}
	doc, err := s.DocumentRepo.FindByID(docID)
	if err != nil {
		return &oapi.DocumentsRequestChangesNotFound{Error: "document not found"}, nil
	}
	if !canReview(doc, uid, role) {
		return &oapi.DocumentsRequestChangesForbidden{Error: "not authorized to review this document"}, nil
	}
	if !doc.CanTransitionTo(models.DocumentStatusChangesRequested) {
		return &oapi.DocumentsRequestChangesUnprocessableEntity{Error: "document cannot have changes requested from its current status"}, nil
	}
	note := ""
	if v, ok := req.Get(); ok {
		note = v.Note.Or("")
	}
	doc.Status = models.DocumentStatusChangesRequested
	doc.ReviewerNote = note
	if err := s.DocumentRepo.Update(doc); err != nil {
		logErr("documents.request_changes.update", err)
		return &oapi.DocumentsRequestChangesInternalServerError{Error: "failed to update document"}, nil
	}
	logAudit(s.Audit, models.AuditActionDocumentChangesRequested, uid, uintPtr(doc.ID), "document", map[string]any{
		"document_id": doc.ID,
		"note":        note,
	})
	notifyDocumentEvent(s.Email, s.UserRepo, s.Hub, doc, "document.changes_requested",
		fmt.Sprintf("Changes requested: %s", doc.Name),
		fmt.Sprintf("<p>Changes have been requested for your document <strong>%s</strong>.</p><p>Note: %s</p>", doc.Name, note),
	)
	return &oapi.UpdateDocumentResponse{Document: documentToOAPI(doc)}, nil
}

// DocumentsAssignReviewer implements PUT /documents/{id}/reviewer.
func (s *Server) DocumentsAssignReviewer(ctx context.Context, req *oapi.AssignReviewerRequest, params oapi.DocumentsAssignReviewerParams) (oapi.DocumentsAssignReviewerRes, error) {
	uid, role, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.DocumentsAssignReviewerUnauthorized{Error: "unauthorized"}, nil
	}
	if !role.CanManageWorkspaces() {
		return &oapi.DocumentsAssignReviewerForbidden{Error: "only managers may assign reviewers"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsAssignReviewerNotFound{Error: "document not found"}, nil
	}
	reviewerID, ok := parseUintID(req.ReviewerId)
	if !ok {
		return &oapi.DocumentsAssignReviewerBadRequest{Error: "invalid reviewerId"}, nil
	}
	doc, err := s.DocumentRepo.FindByID(docID)
	if err != nil {
		return &oapi.DocumentsAssignReviewerNotFound{Error: "document not found"}, nil
	}
	doc.ReviewerID = &reviewerID
	if err := s.DocumentRepo.Update(doc); err != nil {
		logErr("documents.assign_reviewer.update", err)
		return &oapi.DocumentsAssignReviewerInternalServerError{Error: "failed to update document"}, nil
	}
	logAudit(s.Audit, models.AuditActionDocumentReviewerAssigned, uid, uintPtr(doc.ID), "document", map[string]any{
		"reviewer_id": reviewerID,
	})
	notifyReviewerAssigned(s.Email, s.UserRepo, s.Hub, doc, reviewerID)
	return &oapi.UpdateDocumentResponse{Document: documentToOAPI(doc)}, nil
}

// DocumentsDelete implements DELETE /documents/{id}.
func (s *Server) DocumentsDelete(ctx context.Context, params oapi.DocumentsDeleteParams) (oapi.DocumentsDeleteRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.DocumentsDeleteUnauthorized{Error: "unauthorized"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsDeleteNotFound{Error: "document not found"}, nil
	}
	doc, err := s.DocumentRepo.FindByID(docID)
	if err != nil {
		return &oapi.DocumentsDeleteNotFound{Error: "document not found"}, nil
	}
	if err := s.DocumentRepo.Delete(docID); err != nil {
		logErr("documents.delete", err)
		return &oapi.DocumentsDeleteInternalServerError{Error: "failed to delete document"}, nil
	}
	logAudit(s.Audit, models.AuditActionDocumentDeleted, uid, uintPtr(docID), "document", map[string]any{
		"name": doc.Name,
	})
	return &oapi.MessageResponse{Message: "document deleted"}, nil
}

