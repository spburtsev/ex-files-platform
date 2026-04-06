package handlers

import (
	"fmt"
	"log/slog"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

// notifyDocumentEvent sends an email to the uploader about a document status change
// and broadcasts an SSE event. Failures are logged but never block the response.
func notifyDocumentEvent(
	email services.EmailService,
	userRepo services.UserRepository,
	hub *services.SSEHub,
	doc *models.Document,
	eventType string,
	subject string,
	bodyHTML string,
) {
	// SSE broadcast
	if hub != nil {
		hub.Broadcast(services.SSEEvent{
			Type:       eventType,
			DocumentID: doc.ID,
			Payload:    map[string]any{"status": string(doc.Status), "name": doc.Name},
		})
	}

	// Email to uploader
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

// notifyReviewerAssigned sends an email to the assigned reviewer.
func notifyReviewerAssigned(
	email services.EmailService,
	userRepo services.UserRepository,
	hub *services.SSEHub,
	doc *models.Document,
	reviewerID uint,
) {
	if hub != nil {
		hub.Broadcast(services.SSEEvent{
			Type:       "document.reviewer_assigned",
			DocumentID: doc.ID,
			Payload:    map[string]any{"reviewer_id": reviewerID, "name": doc.Name},
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
