package project

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"xiezuojun-server/internal/model"
	"xiezuojun-server/internal/store"
)

const maxDocumentVersions = 3

func (h *Handler) loadDocumentContent(doc model.Document) string {
	content := ""
	proj, err := h.getProject(doc.ProjectID)
	if err == nil && proj != nil {
		if doc.ParentID != nil && *doc.ParentID > 0 {
			content, _ = h.ds.loadDocContent(proj.UserID, proj.ID, proj.Name, doc.ID, *doc.ParentID)
		} else {
			content, _ = h.ds.loadDocContent(proj.UserID, proj.ID, proj.Name, doc.ID)
		}
	}
	if content == "" {
		content, _ = h.ds.loadDocContentFallback(doc.ID)
	}
	return content
}

func (h *Handler) snapshotDocumentVersion(doc model.Document, content string) error {
	if doc.Type == "volume" {
		return nil
	}
	return h.db.Transaction(func(tx *gorm.DB) error {
		var latest model.DocumentVersion
		if err := tx.Where("document_id = ?", doc.ID).Order("created_at DESC, id DESC").First(&latest).Error; err == nil {
			if latest.ContentSnapshot == content {
				return nil
			}
		} else if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		version := model.DocumentVersion{
			DocumentID:      doc.ID,
			ContentSnapshot: content,
			WordCount:       countWords(content),
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}

		var keepIDs []uint
		if err := tx.Model(&model.DocumentVersion{}).
			Where("document_id = ?", doc.ID).
			Order("created_at DESC, id DESC").
			Limit(maxDocumentVersions).
			Pluck("id", &keepIDs).Error; err != nil {
			return err
		}
		if len(keepIDs) >= maxDocumentVersions {
			return tx.Where("document_id = ? AND id NOT IN ?", doc.ID, keepIDs).Delete(&model.DocumentVersion{}).Error
		}
		return nil
	})
}

func (h *Handler) listDocumentVersionsForDoc(c *gin.Context, doc model.Document) {
	var versions []model.DocumentVersion
	if err := h.db.Where("document_id = ?", doc.ID).
		Order("created_at DESC, id DESC").
		Limit(maxDocumentVersions).
		Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": versions})
}

func (h *Handler) ListDocumentVersions(c *gin.Context) {
	doc, err := h.findDoc(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	h.listDocumentVersionsForDoc(c, *doc)
}

func (h *Handler) GetDocumentVersion(c *gin.Context) {
	versionID, err := strconv.Atoi(c.Param("version_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version_id"})
		return
	}
	var version model.DocumentVersion
	if err := h.db.First(&version, uint(versionID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": version})
}

func (h *Handler) RestoreDocumentVersion(c *gin.Context) {
	userID := getUserID(c)
	versionID, err := strconv.Atoi(c.Param("version_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version_id"})
		return
	}

	var version model.DocumentVersion
	if err := h.db.First(&version, uint(versionID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var doc model.Document
	if err := h.db.First(&doc, version.DocumentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}
	if doc.Status != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document is deleted"})
		return
	}

	currentContent := h.loadDocumentContent(doc)
	if err := h.snapshotDocumentVersion(doc, currentContent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "snapshot current: " + err.Error()})
		return
	}

	now := time.Now()
	updates := map[string]interface{}{
		"word_count":  countWords(version.ContentSnapshot),
		"search_text": version.ContentSnapshot,
		"updated_at":  now,
	}
	if err := h.db.Model(&model.Document{}).Where("id = ?", doc.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	proj, err := h.getProject(doc.ProjectID)
	if err == nil && proj != nil {
		h.locks.Lock(proj.UserID)
		h.updateDocWithMeta(proj.UserID, proj.ID, proj.Name, doc, version.ContentSnapshot, false)
		h.locks.Unlock(proj.UserID)
	}
	_ = h.db.First(&doc, doc.ID).Error
	_ = h.pruneDocumentVersions(doc.ID)
	projName := h.getProjectName(doc.ProjectID)
	_ = store.AppendChangelog(h.paths, userID, store.ChangeEntry{
		Action: "restore_doc_version", Project: projName,
		Doc: strconv.FormatUint(uint64(doc.ID), 10), Summary: "恢复版本: " + doc.Title,
	})
	c.JSON(http.StatusOK, gin.H{"message": "restored", "data": gin.H{"document": doc, "content": version.ContentSnapshot}})
}

func (h *Handler) pruneDocumentVersions(docID uint) error {
	var keepIDs []uint
	if err := h.db.Model(&model.DocumentVersion{}).
		Where("document_id = ?", docID).
		Order("created_at DESC, id DESC").
		Limit(maxDocumentVersions).
		Pluck("id", &keepIDs).Error; err != nil {
		return err
	}
	if len(keepIDs) == 0 {
		return nil
	}
	return h.db.Where("document_id = ? AND id NOT IN ?", docID, keepIDs).Delete(&model.DocumentVersion{}).Error
}
