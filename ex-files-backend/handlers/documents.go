package handlers

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"time"

	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/logging"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
)

func documentToOAPI(d *models.Document, approvals []models.DocumentApproval, requiredApprovals int) oapi.Document {
	if requiredApprovals < 1 {
		requiredApprovals = 1
	}
	out := oapi.Document{
		ID:                formatID(d.ID),
		Name:              d.Name,
		MimeType:          d.MimeType,
		Size:              d.Size,
		Hash:              d.Hash,
		Status:            oapi.DocumentStatus(d.Status),
		UploaderId:        formatID(d.UploaderID),
		UploaderName:      d.Uploader.Name,
		IssueId:           formatID(d.IssueID),
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
		RequiredApprovals: oapi.NewOptInt32(int32(requiredApprovals)),
		ApprovalCount:     oapi.NewOptInt32(int32(len(approvals))),
	}
	if d.ReviewerID != nil {
		out.ReviewerId = oapi.NewOptNilString(formatID(*d.ReviewerID))
		out.ReviewerName = oapi.NewOptNilString(d.Reviewer.Name)
	}
	if d.ReviewerNote != "" {
		out.ReviewerNote = oapi.NewOptString(d.ReviewerNote)
	}
	if len(approvals) > 0 {
		list := make([]oapi.DocumentApproval, 0, len(approvals))
		for i := range approvals {
			list = append(list, oapi.DocumentApproval{
				ReviewerId:   formatID(approvals[i].ReviewerID),
				ReviewerName: approvals[i].Reviewer.Name,
				CreatedAt:    approvals[i].CreatedAt,
			})
		}
		out.Approvals = list
	}
	return out
}

func requiredApprovalsOf(issue *models.Issue) int {
	if issue == nil || issue.RequiredApprovals < 1 {
		return 1
	}
	return issue.RequiredApprovals
}

func (s *Server) documentResponse(doc *models.Document, issue *models.Issue) oapi.Document {
	var approvals []models.DocumentApproval
	if s.ApprovalRepo != nil {
		if a, err := s.ApprovalRepo.ListByDocument(doc.ID); err == nil {
			approvals = a
		} else {
			logErr("documents.response.approvals", err)
		}
	}
	return documentToOAPI(doc, approvals, requiredApprovalsOf(issue))
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

	required := 1
	if issue, err := s.IssueRepo.FindByID(issueID); err == nil {
		required = requiredApprovalsOf(issue)
	}
	approvalsByDoc := make(map[uint][]models.DocumentApproval)
	if s.ApprovalRepo != nil && len(docs) > 0 {
		ids := make([]uint, len(docs))
		for i := range docs {
			ids[i] = docs[i].ID
		}
		if all, err := s.ApprovalRepo.ListByDocumentIDs(ids); err == nil {
			for _, a := range all {
				approvalsByDoc[a.DocumentID] = append(approvalsByDoc[a.DocumentID], a)
			}
		} else {
			logErr("documents.list.approvals", err)
		}
	}

	out := make([]oapi.Document, len(docs))
	for i := range docs {
		out[i] = documentToOAPI(&docs[i], approvalsByDoc[docs[i].ID], required)
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

	storageKey := fmt.Sprintf("issues/%d/documents/%d/%s", issueID, doc.ID, mp.Name)
	if err := s.Storage.Upload(ctx, storageKey, mp.File, mp.Size, mimeType); err != nil {
		logErr("documents.upload.storage", err)
		return &oapi.DocumentsUploadInternalServerError{Error: "failed to upload file"}, nil
	}

	doc.StorageKey = storageKey
	if err := s.DocumentRepo.Update(&doc); err != nil {
		logErr("documents.upload.update_key", err)
		return &oapi.DocumentsUploadInternalServerError{Error: "failed to persist storage key"}, nil
	}

	logging.Audit(ctx, "document.uploaded", uid,
		slog.String("target_type", "document"),
		slog.Uint64("target_id", uint64(doc.ID)),
		slog.String("name", doc.Name),
		slog.String("hash", hash),
		slog.Uint64("issue_id", uint64(issueID)),
	)

	loaded, err := s.DocumentRepo.FindByID(doc.ID)
	if err != nil {
		logErr("documents.upload.reload", err)
		return &oapi.DocumentsUploadInternalServerError{Error: "failed to reload document"}, nil
	}

	return &oapi.UploadDocumentResponse{
		Document: s.documentResponse(loaded, issue),
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
	return &oapi.GetDocumentResponse{Document: s.documentResponse(doc, nil)}, nil
}

// DocumentsGetFile implements GET /documents/{id}/file.
func (s *Server) DocumentsGetFile(ctx context.Context, params oapi.DocumentsGetFileParams) (oapi.DocumentsGetFileRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.DocumentsGetFileUnauthorized{Error: "unauthorized"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsGetFileNotFound{Error: "invalid document id"}, nil
	}
	doc, err := s.DocumentRepo.FindByID(docID)
	if err != nil {
		return &oapi.DocumentsGetFileNotFound{Error: "document not found"}, nil
	}
	reader, err := s.Storage.Get(ctx, doc.StorageKey)
	if err != nil {
		logErr("documents.file.read", err)
		return &oapi.DocumentsGetFileInternalServerError{Error: "failed to read file"}, nil
	}
	out := oapi.DocumentsGetFileOKHeaders{
		Response: oapi.DocumentsGetFileOK{Data: reader},
	}
	if doc.Size > 0 {
		out.ContentLength = optInt64(doc.Size)
	}
	return &out, nil
}

// DocumentsGetDownloadUrl implements GET /documents/{id}/download.
func (s *Server) DocumentsGetDownloadUrl(ctx context.Context, params oapi.DocumentsGetDownloadUrlParams) (oapi.DocumentsGetDownloadUrlRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.DocumentsGetDownloadUrlUnauthorized{Error: "unauthorized"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.DocumentsGetDownloadUrlNotFound{Error: "invalid document id"}, nil
	}
	doc, err := s.DocumentRepo.FindByID(docID)
	if err != nil {
		return &oapi.DocumentsGetDownloadUrlNotFound{Error: "document not found"}, nil
	}
	signedURL, err := s.Storage.PresignedURL(ctx, doc.StorageKey, 15*time.Minute)
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
	logging.Audit(ctx, "document.submitted", uid,
		slog.String("target_type", "document"),
		slog.Uint64("target_id", uint64(doc.ID)),
	)
	return &oapi.UpdateDocumentResponse{Document: s.documentResponse(doc, nil)}, nil
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
	if s.ApprovalRepo != nil {
		if err := s.ApprovalRepo.DeleteByDocument(doc.ID); err != nil {
			logErr("documents.resubmit.clear_approvals", err)
		}
	}
	logging.Audit(ctx, "document.submitted", uid,
		slog.String("target_type", "document"),
		slog.Uint64("target_id", uint64(doc.ID)),
		slog.Bool("resubmit", true),
	)
	return &oapi.UpdateDocumentResponse{Document: s.documentResponse(doc, nil)}, nil
}

func (s *Server) issuePanel(issueID uint) (map[uint]bool, []uint, error) {
	reviewers, err := s.IssueRepo.GetReviewers(issueID)
	if err != nil {
		return nil, nil, err
	}
	set := make(map[uint]bool, len(reviewers))
	ids := make([]uint, 0, len(reviewers))
	for i := range reviewers {
		if set[reviewers[i].ID] {
			continue
		}
		set[reviewers[i].ID] = true
		ids = append(ids, reviewers[i].ID)
	}
	return set, ids, nil
}

func canReviewPanel(panel map[uint]bool, issueCreatorID, callerID uint, role models.Role) bool {
	if len(panel) == 0 {
		return role.CanManageWorkspaces() || issueCreatorID == callerID
	}
	return panel[callerID]
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
	issue, err := s.IssueRepo.FindByID(doc.IssueID)
	if err != nil {
		logErr("documents.approve.issue_lookup", err)
		return &oapi.DocumentsApproveInternalServerError{Error: "failed to load issue"}, nil
	}
	panel, panelIDs, err := s.issuePanel(doc.IssueID)
	if err != nil {
		logErr("documents.approve.panel", err)
		return &oapi.DocumentsApproveInternalServerError{Error: "failed to load reviewers"}, nil
	}
	if !canReviewPanel(panel, issue.CreatorID, uid, role) {
		return &oapi.DocumentsApproveForbidden{Error: "not authorized to review this document"}, nil
	}
	if !doc.CanTransitionTo(models.DocumentStatusApproved) {
		return &oapi.DocumentsApproveUnprocessableEntity{Error: "document cannot be approved from its current status"}, nil
	}

	// Record this reviewer's approval (idempotent on the unique index).
	if s.ApprovalRepo != nil {
		if err := s.ApprovalRepo.Create(&models.DocumentApproval{DocumentID: doc.ID, ReviewerID: uid}); err != nil {
			logErr("documents.approve.record", err)
			return &oapi.DocumentsApproveInternalServerError{Error: "failed to record approval"}, nil
		}
	}

	threshold := requiredApprovalsOf(issue)
	count := 1
	if len(panel) == 0 {
		threshold = 1 // empty panel: a single manager/creator approval resolves
	} else if s.ApprovalRepo != nil {
		c, err := s.ApprovalRepo.CountByReviewers(doc.ID, panelIDs)
		if err != nil {
			logErr("documents.approve.count", err)
			return &oapi.DocumentsApproveInternalServerError{Error: "failed to count approvals"}, nil
		}
		count = int(c)
	}

	if count >= threshold {
		doc.Status = models.DocumentStatusApproved
		if err := s.DocumentRepo.Update(doc); err != nil {
			logErr("documents.approve.update", err)
			return &oapi.DocumentsApproveInternalServerError{Error: "failed to update document"}, nil
		}
		if !issue.Resolved {
			issue.Resolved = true
			if err := s.IssueRepo.Update(issue); err != nil {
				logErr("documents.approve.issue_resolve", err)
			}
		}
		logging.Audit(ctx, "document.approved", uid,
			slog.String("target_type", "document"),
			slog.Uint64("target_id", uint64(doc.ID)),
			slog.Int("approval_count", count),
			slog.Int("required_approvals", threshold),
		)
		notifyDocumentEvent(s.Email, s.UserRepo, s.IssueRepo, s.Hub, doc, "document.approved",
			fmt.Sprintf("Document approved: %s", doc.Name),
			fmt.Sprintf("<p>Your document <strong>%s</strong> has been approved.</p>", doc.Name),
		)
		notifyApprovalProgress(s.Hub, doc, count, threshold)
	} else {
		if doc.Status != models.DocumentStatusInReview {
			doc.Status = models.DocumentStatusInReview
			if err := s.DocumentRepo.Update(doc); err != nil {
				logErr("documents.approve.promote", err)
				return &oapi.DocumentsApproveInternalServerError{Error: "failed to update document"}, nil
			}
		}
		logging.Audit(ctx, "document.approval_added", uid,
			slog.String("target_type", "document"),
			slog.Uint64("target_id", uint64(doc.ID)),
			slog.Int("approval_count", count),
			slog.Int("required_approvals", threshold),
		)
		notifyApprovalProgress(s.Hub, doc, count, threshold)
	}
	return &oapi.UpdateDocumentResponse{Document: s.documentResponse(doc, issue)}, nil
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
	issue, err := s.IssueRepo.FindByID(doc.IssueID)
	if err != nil {
		logErr("documents.reject.issue_lookup", err)
		return &oapi.DocumentsRejectInternalServerError{Error: "failed to load issue"}, nil
	}
	panel, _, err := s.issuePanel(doc.IssueID)
	if err != nil {
		logErr("documents.reject.panel", err)
		return &oapi.DocumentsRejectInternalServerError{Error: "failed to load reviewers"}, nil
	}
	if !canReviewPanel(panel, issue.CreatorID, uid, role) {
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
	logging.Audit(ctx, "document.rejected", uid,
		slog.String("target_type", "document"),
		slog.Uint64("target_id", uint64(doc.ID)),
		slog.String("note", note),
	)
	notifyDocumentEvent(s.Email, s.UserRepo, s.IssueRepo, s.Hub, doc, "document.rejected",
		fmt.Sprintf("Document rejected: %s", doc.Name),
		fmt.Sprintf("<p>Your document <strong>%s</strong> has been rejected.</p><p>Reason: %s</p>", doc.Name, note),
	)
	return &oapi.UpdateDocumentResponse{Document: s.documentResponse(doc, issue)}, nil
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
	issue, err := s.IssueRepo.FindByID(doc.IssueID)
	if err != nil {
		logErr("documents.request_changes.issue_lookup", err)
		return &oapi.DocumentsRequestChangesInternalServerError{Error: "failed to load issue"}, nil
	}
	panel, _, err := s.issuePanel(doc.IssueID)
	if err != nil {
		logErr("documents.request_changes.panel", err)
		return &oapi.DocumentsRequestChangesInternalServerError{Error: "failed to load reviewers"}, nil
	}
	if !canReviewPanel(panel, issue.CreatorID, uid, role) {
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
	logging.Audit(ctx, "document.changes_requested", uid,
		slog.String("target_type", "document"),
		slog.Uint64("target_id", uint64(doc.ID)),
		slog.String("note", note),
	)
	notifyDocumentEvent(s.Email, s.UserRepo, s.IssueRepo, s.Hub, doc, "document.changes_requested",
		fmt.Sprintf("Changes requested: %s", doc.Name),
		fmt.Sprintf("<p>Changes have been requested for your document <strong>%s</strong>.</p><p>Note: %s</p>", doc.Name, note),
	)
	return &oapi.UpdateDocumentResponse{Document: s.documentResponse(doc, issue)}, nil
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
	logging.Audit(ctx, "document.reviewer_assigned", uid,
		slog.String("target_type", "document"),
		slog.Uint64("target_id", uint64(doc.ID)),
		slog.Uint64("reviewer_id", uint64(reviewerID)),
	)
	notifyReviewerAssigned(s.Email, s.UserRepo, s.IssueRepo, s.Hub, doc, reviewerID)
	return &oapi.UpdateDocumentResponse{Document: s.documentResponse(doc, nil)}, nil
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
	logging.Audit(ctx, "document.deleted", uid,
		slog.String("target_type", "document"),
		slog.Uint64("target_id", uint64(docID)),
		slog.String("name", doc.Name),
	)
	return &oapi.MessageResponse{Message: "document deleted"}, nil
}
