package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/services"
)

type VerifyHandler struct {
	Repo services.DocumentRepository
}

// Verify checks whether a document with the given hash exists.
// @Summary      Verify document hash
// @Tags         verify
// @Produce      json
// @Param        hash  query     string  true  "SHA-256 document hash"
// @Success      200   {object}  swagVerifyResponse
// @Failure      400   {object}  swagErrorResponse
// @Router       /verify [get]
func (h *VerifyHandler) Verify(c *gin.Context) {
	hash := c.Query("hash")
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hash query parameter is required"})
		return
	}

	doc, err := h.Repo.FindByHash(hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"verified": false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to look up document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"verified":      true,
		"document_name": doc.Name,
		"status":        doc.Status,
		"notarized_at":  doc.CreatedAt,
		"hash":          doc.Hash,
	})
}
