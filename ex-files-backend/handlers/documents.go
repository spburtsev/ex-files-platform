package handlers

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	docsv1 "github.com/spburtsev/ex-files-backend/gen/documents/v1"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

type DocumentHandler struct {
	Repo     services.DocumentRepository
	Storage  services.StorageService
	Audit    services.AuditRepository
	UserRepo services.UserRepository
	Email    services.EmailService
	Hub      *services.SSEHub
}

func documentToProto(d *models.Document) *docsv1.Document {
	pb := &docsv1.Document{
		Id:           uint64(d.ID),
		Name:         d.Name,
		MimeType:     d.MimeType,
		Size:         d.Size,
		Hash:         d.Hash,
		Status:       string(d.Status),
		UploaderId:   uint64(d.UploaderID),
		UploaderName: d.Uploader.Name,
		IssueId:      uint64(d.IssueID),
		CreatedAt:    timestamppb.New(d.CreatedAt),
		UpdatedAt:    timestamppb.New(d.UpdatedAt),
	}
	if d.ReviewerID != nil {
		pb.ReviewerId = uint64(*d.ReviewerID)
		pb.ReviewerName = d.Reviewer.Name
	}
	pb.ReviewerNote = d.ReviewerNote
	return pb
}

func versionToProto(v *models.DocumentVersion) *docsv1.DocumentVersion {
	return &docsv1.DocumentVersion{
		Id:           uint64(v.ID),
		DocumentId:   uint64(v.DocumentID),
		Version:      int32(v.Version),
		Hash:         v.Hash,
		Size:         v.Size,
		StorageKey:   v.StorageKey,
		UploaderId:   uint64(v.UploaderID),
		UploaderName: v.Uploader.Name,
		CreatedAt:    timestamppb.New(v.CreatedAt),
	}
}

// Upload uploads a new document to an issue.
// @Summary      Upload document
// @Tags         documents
// @Accept       multipart/form-data
// @Produce      application/x-protobuf
// @Param        id    path      int   true  "Issue ID"
// @Param        file  formData  file  true  "Document file"
// @Success      201   {object}  swagUploadDocumentResponse  "Protobuf: documents.v1.UploadDocumentResponse"
// @Failure      400   {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /issues/{id}/documents [post]
func (h *DocumentHandler) Upload(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	issueID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid issue id"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	// Hash the file content
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash file"})
		return
	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Reject if this issue already has a document with the same content.
	if existing, err := h.Repo.FindByIssueAndHash(uint(issueID), hash); err == nil && existing != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("issue already has a document with the same content: %q", existing.Name),
		})
		return
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check existing documents"})
		return
	}

	// Reset file reader for upload
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	// Create document record
	doc := models.Document{
		Name:        header.Filename,
		MimeType:    header.Header.Get("Content-Type"),
		Size:        header.Size,
		Hash:        hash,
		Status:     models.DocumentStatusPending,
		UploaderID: userID,
		IssueID:    uint(issueID),
	}
	if err := h.Repo.Create(&doc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document"})
		return
	}

	// Storage key: issues/<issueID>/documents/<docID>/v1/<filename>
	storageKey := fmt.Sprintf("issues/%d/documents/%d/v1/%s", issueID, doc.ID, header.Filename)

	if err := h.Storage.Upload(c.Request.Context(), storageKey, file, header.Size, doc.MimeType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file"})
		return
	}

	// Create version 1
	version := models.DocumentVersion{
		DocumentID: doc.ID,
		Version:    1,
		Hash:       hash,
		Size:       header.Size,
		StorageKey: storageKey,
		UploaderID: userID,
	}
	if err := h.Repo.CreateVersion(&version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create version"})
		return
	}

	logAudit(h.Audit, models.AuditActionDocumentUploaded, userID, uintPtr(doc.ID), "document", map[string]any{
		"name":     doc.Name,
		"hash":     hash,
		"issue_id": issueID,
	})

	// Reload doc to get uploader preloaded
	loadedDoc, err := h.Repo.FindByID(doc.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reload document"})
		return
	}
	doc = *loadedDoc

	protobufResponse(c, http.StatusCreated, &docsv1.UploadDocumentResponse{
		Document: documentToProto(&doc),
		Version: &docsv1.DocumentVersion{
			Id:         uint64(version.ID),
			DocumentId: uint64(version.DocumentID),
			Version:    int32(version.Version),
			Hash:       version.Hash,
			Size:       version.Size,
			StorageKey: version.StorageKey,
			UploaderId: uint64(version.UploaderID),
			CreatedAt:  timestamppb.New(version.CreatedAt),
		},
	})
}

// UploadVersion uploads a new version of an existing document.
// @Summary      Upload new version
// @Tags         documents
// @Accept       multipart/form-data
// @Produce      application/x-protobuf
// @Param        id    path      int   true  "Document ID"
// @Param        file  formData  file  true  "Document file"
// @Success      201   {object}  swagUploadDocumentResponse  "Protobuf: documents.v1.UploadDocumentResponse"
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id}/versions [post]
func (h *DocumentHandler) UploadVersion(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	doc, err := h.Repo.FindByID(uint(docID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash file"})
		return
	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	latestVersion, err := h.Repo.LatestVersionNumber(uint(docID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get version info"})
		return
	}
	newVersion := latestVersion + 1

	storageKey := fmt.Sprintf("issues/%d/documents/%d/v%d/%s", doc.IssueID, doc.ID, newVersion, header.Filename)

	if err := h.Storage.Upload(c.Request.Context(), storageKey, file, header.Size, header.Header.Get("Content-Type")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file"})
		return
	}

	version := models.DocumentVersion{
		DocumentID: uint(docID),
		Version:    newVersion,
		Hash:       hash,
		Size:       header.Size,
		StorageKey: storageKey,
		UploaderID: userID,
	}
	if err := h.Repo.CreateVersion(&version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create version"})
		return
	}

	logAudit(h.Audit, models.AuditActionVersionCreated, userID, uintPtr(doc.ID), "document", map[string]any{
		"version": newVersion,
		"hash":    hash,
	})

	protobufResponse(c, http.StatusCreated, &docsv1.UploadDocumentResponse{
		Document: documentToProto(doc),
		Version: &docsv1.DocumentVersion{
			Id:         uint64(version.ID),
			DocumentId: uint64(version.DocumentID),
			Version:    int32(version.Version),
			Hash:       version.Hash,
			Size:       version.Size,
			StorageKey: version.StorageKey,
			UploaderId: uint64(version.UploaderID),
			CreatedAt:  timestamppb.New(version.CreatedAt),
		},
	})
}

// List returns documents for an issue with optional search and status filter.
// @Summary      List documents
// @Tags         documents
// @Produce      application/x-protobuf
// @Param        id        path   int     true   "Issue ID"
// @Param        search    query  string  false  "Search by name"
// @Param        status    query  string  false  "Filter by status"
// @Param        page      query  int     false  "Page number"     default(1)
// @Param        per_page  query  int     false  "Items per page"  default(20)
// @Success      200  {object}  swagListDocumentsResponse  "Protobuf: documents.v1.ListDocumentsResponse"
// @Header       200  {int}     X-Total-Count
// @Header       200  {int}     X-Total-Pages
// @Header       200  {int}     X-Page
// @Header       200  {int}     X-Per-Page
// @Security     BearerAuth || CookieAuth
// @Router       /issues/{id}/documents [get]
func (h *DocumentHandler) List(c *gin.Context) {
	issueID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid issue id"})
		return
	}

	page, perPage := parsePagination(c)
	offset := (page - 1) * perPage
	search := c.Query("search")
	status := c.Query("status")

	docs, total, err := h.Repo.ListByIssue(uint(issueID), search, status, perPage, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch documents"})
		return
	}

	setPaginationHeaders(c, page, perPage, total)

	pbDocs := make([]*docsv1.Document, len(docs))
	for i := range docs {
		pbDocs[i] = documentToProto(&docs[i])
	}

	protobufResponse(c, http.StatusOK, &docsv1.ListDocumentsResponse{
		Documents: pbDocs,
	})
}

// Get returns a document with all its versions.
// @Summary      Get document
// @Tags         documents
// @Produce      application/x-protobuf
// @Param        id   path      int  true  "Document ID"
// @Success      200  {object}  swagGetDocumentResponse  "Protobuf: documents.v1.GetDocumentResponse"
// @Failure      404  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id} [get]
func (h *DocumentHandler) Get(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	doc, err := h.Repo.FindByID(uint(docID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	versions, err := h.Repo.GetVersions(uint(docID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch versions"})
		return
	}

	pbVersions := make([]*docsv1.DocumentVersion, len(versions))
	for i := range versions {
		pbVersions[i] = versionToProto(&versions[i])
	}

	protobufResponse(c, http.StatusOK, &docsv1.GetDocumentResponse{
		Document: &docsv1.DocumentDetail{
			Document: documentToProto(doc),
			Versions: pbVersions,
		},
	})
}

// File streams the binary contents of a document version.
// @Summary      Stream version file
// @Tags         documents
// @Produce      application/octet-stream
// @Param        id         path      int  true  "Document ID"
// @Param        versionId  path      int  true  "Version ID"
// @Success      200  {file}    binary
// @Failure      404  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id}/versions/{versionId}/file [get]
func (h *DocumentHandler) File(c *gin.Context) {
	versionID, err := strconv.ParseUint(c.Param("versionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version id"})
		return
	}

	version, err := h.Repo.GetVersion(uint(versionID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}

	reader, err := h.Storage.Get(c.Request.Context(), version.StorageKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	defer reader.Close()

	contentType := "application/octet-stream"
	if doc, err := h.Repo.FindByID(version.DocumentID); err == nil && doc.MimeType != "" {
		contentType = doc.MimeType
	}
	c.Header("Content-Type", contentType)
	if version.Size > 0 {
		c.Header("Content-Length", strconv.FormatInt(version.Size, 10))
	}
	if _, err := io.Copy(c.Writer, reader); err != nil {
		// Headers already sent — log and bail.
		return
	}
}

// Download returns a presigned URL for downloading a document version.
// @Summary      Download version
// @Tags         documents
// @Produce      application/x-protobuf
// @Param        id         path      int  true  "Document ID"
// @Param        versionId  path      int  true  "Version ID"
// @Success      200  {object}  swagDownloadURLResponse  "Protobuf: documents.v1.GetDownloadURLResponse"
// @Failure      404  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id}/versions/{versionId}/download [get]
func (h *DocumentHandler) Download(c *gin.Context) {
	versionID, err := strconv.ParseUint(c.Param("versionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version id"})
		return
	}

	version, err := h.Repo.GetVersion(uint(versionID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}

	url, err := h.Storage.PresignedURL(c.Request.Context(), version.StorageKey, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate download URL"})
		return
	}

	protobufResponse(c, http.StatusOK, &docsv1.GetDownloadURLResponse{
		Url: url,
	})
}

// Submit transitions a document from pending to in_review. Only the uploader.
// @Summary      Submit document for review
// @Tags         documents
// @Produce      application/x-protobuf
// @Param        id   path      int  true  "Document ID"
// @Success      200  {object}  swagUpdateDocumentResponse  "Protobuf: documents.v1.UpdateDocumentResponse"
// @Failure      403  {object}  swagErrorResponse
// @Failure      422  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id}/submit [post]
func (h *DocumentHandler) Submit(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	doc, err := h.Repo.FindByID(uint(docID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	if doc.UploaderID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the uploader may submit this document"})
		return
	}

	if !doc.CanTransitionTo(models.DocumentStatusInReview) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "document cannot be submitted from its current status"})
		return
	}

	doc.Status = models.DocumentStatusInReview
	doc.ReviewerNote = ""
	if err := h.Repo.Update(doc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update document"})
		return
	}

	logAudit(h.Audit, models.AuditActionDocumentSubmitted, userID, uintPtr(doc.ID), "document", map[string]any{
		"document_id": doc.ID,
	})

	protobufResponse(c, http.StatusOK, &docsv1.UpdateDocumentResponse{Document: documentToProto(doc)})
}

// AssignReviewer sets the reviewer for a document. Only managers and root users.
// @Summary      Assign reviewer
// @Tags         documents
// @Accept       json
// @Produce      application/x-protobuf
// @Param        id    path      int                        true  "Document ID"
// @Param        body  body      swagAssignReviewerRequest  true  "Reviewer payload"
// @Success      200   {object}  swagUpdateDocumentResponse "Protobuf: documents.v1.UpdateDocumentResponse"
// @Failure      403   {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id}/reviewer [put]
func (h *DocumentHandler) AssignReviewer(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	role, ok := mustGetRole(c)
	if !ok {
		return
	}
	if !role.CanManageWorkspaces() {
		c.JSON(http.StatusForbidden, gin.H{"error": "only managers may assign reviewers"})
		return
	}

	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	var body struct {
		ReviewerID uint `json:"reviewer_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reviewer_id is required"})
		return
	}

	doc, err := h.Repo.FindByID(uint(docID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	doc.ReviewerID = &body.ReviewerID
	if err := h.Repo.Update(doc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update document"})
		return
	}

	logAudit(h.Audit, models.AuditActionDocumentReviewerAssigned, userID, uintPtr(doc.ID), "document", map[string]any{
		"reviewer_id": body.ReviewerID,
	})

	notifyReviewerAssigned(h.Email, h.UserRepo, h.Hub, doc, body.ReviewerID)

	protobufResponse(c, http.StatusOK, &docsv1.UpdateDocumentResponse{Document: documentToProto(doc)})
}

// canReview returns true if the caller is the assigned reviewer, a manager, or root.
func canReview(doc *models.Document, callerID uint, role models.Role) bool {
	if role.CanManageWorkspaces() {
		return true
	}
	return doc.ReviewerID != nil && *doc.ReviewerID == callerID
}

// Approve transitions a document from in_review to approved.
// @Summary      Approve document
// @Tags         documents
// @Produce      application/x-protobuf
// @Param        id   path      int  true  "Document ID"
// @Success      200  {object}  swagUpdateDocumentResponse  "Protobuf: documents.v1.UpdateDocumentResponse"
// @Failure      403  {object}  swagErrorResponse
// @Failure      422  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id}/approve [post]
func (h *DocumentHandler) Approve(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	role, ok := mustGetRole(c)
	if !ok {
		return
	}
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	doc, err := h.Repo.FindByID(uint(docID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	if !canReview(doc, userID, role) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to review this document"})
		return
	}

	if !doc.CanTransitionTo(models.DocumentStatusApproved) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "document cannot be approved from its current status"})
		return
	}

	doc.Status = models.DocumentStatusApproved
	if err := h.Repo.Update(doc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update document"})
		return
	}

	logAudit(h.Audit, models.AuditActionDocumentApproved, userID, uintPtr(doc.ID), "document", map[string]any{
		"document_id": doc.ID,
	})

	notifyDocumentEvent(h.Email, h.UserRepo, h.Hub, doc, "document.approved",
		fmt.Sprintf("Document approved: %s", doc.Name),
		fmt.Sprintf("<p>Your document <strong>%s</strong> has been approved.</p>", doc.Name),
	)

	protobufResponse(c, http.StatusOK, &docsv1.UpdateDocumentResponse{Document: documentToProto(doc)})
}

// Reject transitions a document from in_review to rejected.
// @Summary      Reject document
// @Tags         documents
// @Accept       json
// @Produce      application/x-protobuf
// @Param        id    path      int                    true  "Document ID"
// @Param        body  body      swagReviewNoteRequest  false "Rejection note"
// @Success      200   {object}  swagUpdateDocumentResponse  "Protobuf: documents.v1.UpdateDocumentResponse"
// @Failure      403   {object}  swagErrorResponse
// @Failure      422   {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id}/reject [post]
func (h *DocumentHandler) Reject(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	role, ok := mustGetRole(c)
	if !ok {
		return
	}
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	var body struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)

	doc, err := h.Repo.FindByID(uint(docID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	if !canReview(doc, userID, role) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to review this document"})
		return
	}

	if !doc.CanTransitionTo(models.DocumentStatusRejected) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "document cannot be rejected from its current status"})
		return
	}

	doc.Status = models.DocumentStatusRejected
	doc.ReviewerNote = body.Note
	if err := h.Repo.Update(doc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update document"})
		return
	}

	logAudit(h.Audit, models.AuditActionDocumentRejected, userID, uintPtr(doc.ID), "document", map[string]any{
		"document_id": doc.ID,
		"note":        body.Note,
	})

	notifyDocumentEvent(h.Email, h.UserRepo, h.Hub, doc, "document.rejected",
		fmt.Sprintf("Document rejected: %s", doc.Name),
		fmt.Sprintf("<p>Your document <strong>%s</strong> has been rejected.</p><p>Reason: %s</p>", doc.Name, body.Note),
	)

	protobufResponse(c, http.StatusOK, &docsv1.UpdateDocumentResponse{Document: documentToProto(doc)})
}

// RequestChanges transitions a document from in_review to changes_requested.
// @Summary      Request changes
// @Tags         documents
// @Accept       json
// @Produce      application/x-protobuf
// @Param        id    path      int                    true  "Document ID"
// @Param        body  body      swagReviewNoteRequest  false "Change request note"
// @Success      200   {object}  swagUpdateDocumentResponse  "Protobuf: documents.v1.UpdateDocumentResponse"
// @Failure      403   {object}  swagErrorResponse
// @Failure      422   {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id}/request-changes [post]
func (h *DocumentHandler) RequestChanges(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	role, ok := mustGetRole(c)
	if !ok {
		return
	}
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	var body struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)

	doc, err := h.Repo.FindByID(uint(docID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	if !canReview(doc, userID, role) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to review this document"})
		return
	}

	if !doc.CanTransitionTo(models.DocumentStatusChangesRequested) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "document cannot have changes requested from its current status"})
		return
	}

	doc.Status = models.DocumentStatusChangesRequested
	doc.ReviewerNote = body.Note
	if err := h.Repo.Update(doc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update document"})
		return
	}

	logAudit(h.Audit, models.AuditActionDocumentChangesRequested, userID, uintPtr(doc.ID), "document", map[string]any{
		"document_id": doc.ID,
		"note":        body.Note,
	})

	notifyDocumentEvent(h.Email, h.UserRepo, h.Hub, doc, "document.changes_requested",
		fmt.Sprintf("Changes requested: %s", doc.Name),
		fmt.Sprintf("<p>Changes have been requested for your document <strong>%s</strong>.</p><p>Note: %s</p>", doc.Name, body.Note),
	)

	protobufResponse(c, http.StatusOK, &docsv1.UpdateDocumentResponse{Document: documentToProto(doc)})
}

// Resubmit transitions a document from changes_requested to in_review. Only the uploader.
// @Summary      Resubmit document
// @Tags         documents
// @Produce      application/x-protobuf
// @Param        id   path      int  true  "Document ID"
// @Success      200  {object}  swagUpdateDocumentResponse  "Protobuf: documents.v1.UpdateDocumentResponse"
// @Failure      403  {object}  swagErrorResponse
// @Failure      422  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id}/resubmit [post]
func (h *DocumentHandler) Resubmit(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	doc, err := h.Repo.FindByID(uint(docID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	if doc.UploaderID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the uploader may resubmit this document"})
		return
	}

	if !doc.CanTransitionTo(models.DocumentStatusInReview) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "document cannot be resubmitted from its current status"})
		return
	}

	doc.Status = models.DocumentStatusInReview
	doc.ReviewerNote = ""
	if err := h.Repo.Update(doc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update document"})
		return
	}

	logAudit(h.Audit, models.AuditActionDocumentSubmitted, userID, uintPtr(doc.ID), "document", map[string]any{
		"document_id": doc.ID,
		"resubmit":    true,
	})

	protobufResponse(c, http.StatusOK, &docsv1.UpdateDocumentResponse{Document: documentToProto(doc)})
}

// Delete removes a document.
// @Summary      Delete document
// @Tags         documents
// @Produce      application/x-protobuf
// @Param        id   path      int  true  "Document ID"
// @Success      200  {object}  swagMessageResponse  "Protobuf: documents.v1.DeleteDocumentResponse"
// @Failure      404  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id} [delete]
func (h *DocumentHandler) Delete(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	doc, err := h.Repo.FindByID(uint(docID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	if err := h.Repo.Delete(uint(docID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete document"})
		return
	}

	logAudit(h.Audit, models.AuditActionDocumentDeleted, userID, uintPtr(uint(docID)), "document", map[string]any{
		"name": doc.Name,
	})

	protobufResponse(c, http.StatusOK, &docsv1.DeleteDocumentResponse{
		Message: "document deleted",
	})
}
