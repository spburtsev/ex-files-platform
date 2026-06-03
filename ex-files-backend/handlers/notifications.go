package handlers

import (
	"fmt"
	"log/slog"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

// notifyDocumentEvent sends an email to the uploader about a document status change
// and broadcasts an SSE event targeted at the uploader. Failures are logged but
// never block the response. The issue lookup powers a deep-link payload so the
// frontend toast can navigate straight to the workspace/issue.
func notifyDocumentEvent(
	email services.EmailService,
	userRepo services.UserRepository,
	issueRepo services.IssueRepository,
	hub *services.SSEHub,
	doc *models.Document,
	eventType string,
	subject string,
	bodyHTML string,
) {
	if hub != nil {
		payload := map[string]any{
			"status":   string(doc.Status),
			"name":     doc.Name,
			"issue_id": doc.IssueID,
		}
		if issueRepo != nil {
			if issue, err := issueRepo.FindByID(doc.IssueID); err == nil {
				payload["workspace_id"] = issue.WorkspaceID
			}
		}
		hub.Broadcast(services.SSEEvent{
			Type:       eventType,
			DocumentID: doc.ID,
			UserID:     doc.UploaderID,
			Payload:    payload,
		})
	}

	if email == nil || userRepo == nil {
		return
	}
	uploader, err := userRepo.FindByID(doc.UploaderID)
	if err != nil {
		slog.Error("failed to find uploader", "component", "notify", "uploader_id", doc.UploaderID, "error", err)
		return
	}
	if err := email.Send(uploader.Email, subject, bodyHTML); err != nil {
		slog.Error("failed to send email", "component", "notify", "email", uploader.Email, "error", err)
	}
}

func notifyApprovalProgress(hub *services.SSEHub, doc *models.Document, approvalCount, requiredApprovals int) {
	if hub == nil {
		return
	}
	hub.Broadcast(services.SSEEvent{
		Type:       "document.approval_added",
		DocumentID: doc.ID,
		Payload: map[string]any{
			"status":             string(doc.Status),
			"name":               doc.Name,
			"issue_id":           doc.IssueID,
			"approval_count":     approvalCount,
			"required_approvals": requiredApprovals,
		},
	})
}

// notifyReviewerAssigned sends an email to the assigned reviewer and a
// targeted SSE event so their UI can toast the assignment immediately.
func notifyReviewerAssigned(
	email services.EmailService,
	userRepo services.UserRepository,
	issueRepo services.IssueRepository,
	hub *services.SSEHub,
	doc *models.Document,
	reviewerID uint,
) {
	if hub != nil {
		payload := map[string]any{
			"reviewer_id": reviewerID,
			"name":        doc.Name,
			"issue_id":    doc.IssueID,
		}
		if issueRepo != nil {
			if issue, err := issueRepo.FindByID(doc.IssueID); err == nil {
				payload["workspace_id"] = issue.WorkspaceID
			}
		}
		hub.Broadcast(services.SSEEvent{
			Type:       "document.reviewer_assigned",
			DocumentID: doc.ID,
			UserID:     reviewerID,
			Payload:    payload,
		})
	}

	if email == nil || userRepo == nil {
		return
	}
	reviewer, err := userRepo.FindByID(reviewerID)
	if err != nil {
		slog.Error("failed to find reviewer", "component", "notify", "reviewer_id", reviewerID, "error", err)
		return
	}
	subject := fmt.Sprintf("You have been assigned to review: %s", doc.Name)
	body := fmt.Sprintf(
		"<p>You have been assigned to review the document <strong>%s</strong>.</p>"+
			"<p>Please log in to the platform to review it.</p>",
		doc.Name,
	)
	if err := email.Send(reviewer.Email, subject, body); err != nil {
		slog.Error("failed to send email", "component", "notify", "email", reviewer.Email, "error", err)
	}
}
