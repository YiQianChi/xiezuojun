package buddy

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册搭子功能路由（挂到 authd 组下）
func RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/buddy/heartbeat", handleHeartbeat)
	rg.POST("/buddy/stop", handleStop)
	rg.POST("/buddy/match", handleMatch)
	rg.POST("/buddy/leave-match", handleLeaveMatch)
	rg.POST("/buddy/cheer", handleCheer)
	rg.GET("/buddy/state", handleState)
	rg.GET("/buddy/events", handleEvents) // SSE 实时推送
}

func handleHeartbeat(c *gin.Context) {
	uid := c.GetString("userID")
	var req struct {
		Nickname   string `json:"nickname"`
		ProjectID  uint   `json:"project_id"`
		WordsDelta int    `json:"words_delta"`
	}
	_ = c.ShouldBindJSON(&req) // body 可为空
	s := Heartbeat(uid, req.Nickname, req.ProjectID, req.WordsDelta)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"session": s,
		"state":   GetState(uid),
	}})
}

func handleStop(c *gin.Context) {
	uid := c.GetString("userID")
	StopWriting(uid)
	c.JSON(http.StatusOK, gin.H{"data": "stopped"})
}

func handleMatch(c *gin.Context) {
	uid := c.GetString("userID")
	matched, peer := MatchRequested(uid)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"matched": matched,
		"peer":    peer,
	}})
}

func handleLeaveMatch(c *gin.Context) {
	uid := c.GetString("userID")
	LeaveMatch(uid)
	c.JSON(http.StatusOK, gin.H{"data": "left"})
}

func handleCheer(c *gin.Context) {
	uid := c.GetString("userID")
	Cheer(uid)
	c.JSON(http.StatusOK, gin.H{"data": "cheered"})
}

func handleState(c *gin.Context) {
	uid := c.GetString("userID")
	q := c.Query("consume_cheer")
	state := GetState(uid)
	if q == "1" {
		state.PeerCheer = ConsumeCheer(uid)
	}
	_ = strconv.Itoa // keep import parity
	c.JSON(http.StatusOK, gin.H{"data": state})
}
