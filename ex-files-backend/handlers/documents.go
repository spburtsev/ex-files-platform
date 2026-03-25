package handlers

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"

	docsv1 "github.com/spburtsev/ex-files-backend/gen/documents/v1"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

type DocumentHandler struct {
	Repo    services.DocumentRepository
	Storage services.StorageService
	Audit   services.AuditRepository
}

func documentToProto(d *models.Document) *docsv1.Document {
	return &docsv1.Document{
		Id:           uint64(d.ID),
		Name:         d.Name,
		MimeType:     d.MimeType,
		Size:         d.Size,
		Hash:         d.Hash,
		Status:       string(d.Status),
		UploaderId:   uint64(d.UploaderID),
		UploaderName: d.Uploader.Name,
		WorkspaceId:  uint64(d.WorkspaceID),
		CreatedAt:    timestamppb.New(d.CreatedAt),
		UpdatedAt:    timestamppb.New(d.UpdatedAt),
	}
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

func (h *DocumentHandler) Upload(c *gin.Context) {
	userID, _ := c.Get("user_id")
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
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
		Status:      models.DocumentStatusPending,
		UploaderID:  userID.(uint),
		WorkspaceID: uint(wsID),
	}
	if err := h.Repo.Create(&doc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document"})
		return
	}

	// Storage key: workspaces/<wsID>/documents/<docID>/v1/<filename>
	storageKey := fmt.Sprintf("workspaces/%d/documents/%d/v1/%s", wsID, doc.ID, header.Filename)

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
		UploaderID: userID.(uint),
	}
	if err := h.Repo.CreateVersion(&version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create version"})
		return
	}

	logAudit(h.Audit, models.AuditActionDocumentUploaded, userID.(uint), uintPtr(doc.ID), "document", map[string]any{
		"name":         doc.Name,
		"hash":         hash,
		"workspace_id": wsID,
	})

	// Reload doc to get uploader preloaded
	doc.Uploader = models.User{}
	loadedDoc, _ := h.Repo.FindByID(doc.ID)
	if loadedDoc != nil {
		doc = *loadedDoc
	}

	c.JSON(http.StatusCreated, &docsv1.UploadDocumentResponse{
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

func (h *DocumentHandler) UploadVersion(c *gin.Context) {
	userID, _ := c.Get("user_id")
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

	storageKey := fmt.Sprintf("workspaces/%d/documents/%d/v%d/%s", doc.WorkspaceID, doc.ID, newVersion, header.Filename)

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
		UploaderID: userID.(uint),
	}
	if err := h.Repo.CreateVersion(&version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create version"})
		return
	}

	logAudit(h.Audit, models.AuditActionVersionCreated, userID.(uint), uintPtr(doc.ID), "document", map[string]any{
		"version": newVersion,
		"hash":    hash,
	})

	c.JSON(http.StatusCreated, &docsv1.UploadDocumentResponse{
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

func (h *DocumentHandler) List(c *gin.Context) {
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	page, perPage := parsePagination(c)
	offset := (page - 1) * perPage
	search := c.Query("search")
	status := c.Query("status")

	docs, total, err := h.Repo.ListByWorkspace(uint(wsID), search, status, perPage, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch documents"})
		return
	}

	setPaginationHeaders(c, page, perPage, total)

	pbDocs := make([]*docsv1.Document, len(docs))
	for i := range docs {
		pbDocs[i] = documentToProto(&docs[i])
	}

	c.JSON(http.StatusOK, &docsv1.ListDocumentsResponse{
		Documents: pbDocs,
	})
}

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

	c.JSON(http.StatusOK, &docsv1.GetDocumentResponse{
		Document: &docsv1.DocumentDetail{
			Document: documentToProto(doc),
			Versions: pbVersions,
		},
	})
}

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

	c.JSON(http.StatusOK, &docsv1.GetDownloadURLResponse{
		Url: url,
	})
}

func (h *DocumentHandler) Delete(c *gin.Context) {
	userID, _ := c.Get("user_id")
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

	logAudit(h.Audit, models.AuditActionDocumentUploaded, userID.(uint), uintPtr(uint(docID)), "document", map[string]any{
		"name":   doc.Name,
		"action": "deleted",
	})

	c.JSON(http.StatusOK, &docsv1.DeleteDocumentResponse{
		Message: "document deleted",
	})
}
