// handler_characters.go — 角色管理 API

package memory

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListCharacters 列出角色
// GET /memory/characters?project_id=xxx
func (h *Handler) ListCharacters(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, err.Error())
		return
	}
	pid, err := getProjectID(c, "")
	if err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	names, err := h.storage.ListCharacters(userID, pid)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": names})
}

// GetCharacter 读取角色
// GET /memory/characters/:name?project_id=xxx
func (h *Handler) GetCharacter(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, err.Error())
		return
	}
	pid, err := getProjectID(c, "")
	if err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	name := c.Param("name")
	if name == "" {
		errorJSON(c, http.StatusBadRequest, "name required")
		return
	}
	content, err := h.storage.ReadCharacter(userID, pid, name)
	if err != nil {
		errorJSON(c, http.StatusNotFound, "character not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"name": name, "content": content}})
}

// WriteCharacter 写角色卡
// PUT /memory/characters/:name?project_id=xxx
func (h *Handler) WriteCharacter(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, err.Error())
		return
	}
	pid, err := getProjectID(c, "")
	if err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	name := c.Param("name")
	if name == "" {
		errorJSON(c, http.StatusBadRequest, "name required")
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.storage.WriteCharacter(userID, pid, name, req.Content); err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

// DeleteCharacter 删角色
// DELETE /memory/characters/:name?project_id=xxx
func (h *Handler) DeleteCharacter(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, err.Error())
		return
	}
	pid, err := getProjectID(c, "")
	if err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	name := c.Param("name")
	if name == "" {
		errorJSON(c, http.StatusBadRequest, "name required")
		return
	}
	if err := h.storage.DeleteCharacter(userID, pid, name); err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
