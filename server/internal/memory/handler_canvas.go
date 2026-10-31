package memory

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetCanvas GET /memory/canvas?project_id=xxx
func (h *Handler) GetCanvas(c *gin.Context) {
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
	data, err := h.storage.ReadCanvas(uid, pid)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", data)
}

// PutCanvas PUT /memory/canvas?project_id=xxx（body=整份画布 JSON）
func (h *Handler) PutCanvas(c *gin.Context) {
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
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, canvasMaxBytes+1))
	if err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	if len(body) > canvasMaxBytes {
		errorJSON(c, http.StatusRequestEntityTooLarge, "画布数据超过 2MiB 限制")
		return
	}
	if err := h.storage.WriteCanvas(uid, pid, body); err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
