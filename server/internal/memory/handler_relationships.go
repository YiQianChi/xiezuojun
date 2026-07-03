package memory

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetRelationships GET /memory/relationships?project_id=xxx
func (h *Handler) GetRelationships(c *gin.Context) {
	uid, err := getUserID(c)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, err.Error())
		return
	}
	pid, err := getProjectID(c, "")
	if err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.storage.ReadRelationships(uid, pid)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// PutRelationships PUT /memory/relationships?project_id=xxx
func (h *Handler) PutRelationships(c *gin.Context) {
	uid, err := getUserID(c)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, err.Error())
		return
	}
	pid, err := getProjectID(c, "")
	if err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	var data RelationshipData
	if err := c.ShouldBindJSON(&data); err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.storage.WriteRelationships(uid, pid, &data); err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}
