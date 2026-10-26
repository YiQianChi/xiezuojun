package buddy

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册搭子功能路由（挂到 authd 组下）
func RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/buddy/heartbeat", handleHeartbeat)
	rg.POST("/buddy/stop", handleStop)
	rg.POST("/buddy/match", handleMatch)
	rg.POST("/buddy/leave-match", handleLeaveMatch)
	rg.POST("/buddy/cheer", handleCheer)
	rg.POST("/buddy/nudge", handleNudge)
	rg.GET("/buddy/state", handleState)
	rg.GET("/buddy/today", handleToday)  // 今日写作统计（净增字数/净码字时长）
	rg.GET("/buddy/events", handleEvents) // SSE 实时推送
}

func handleHeartbeat(c *gin.Context) {
	uid := c.GetString("userID")
	var req struct {
		Nickname    string `json:"nickname"`
		ProjectID   uint   `json:"project_id"`
		WordsDelta  int    `json:"words_delta"`
		ElapsedSec  int    `json:"elapsed_sec"`  // 前端累计码字秒数（暂停不计时）
		TimerState  string `json:"timer_state"`  // running | paused
	}
	_ = c.ShouldBindJSON(&req) // body 可为空
	if req.TimerState != "running" && req.TimerState != "paused" {
		req.TimerState = "running" // 兼容旧前端：无状态默认 running
	}
	s := Heartbeat(uid, req.Nickname, req.ProjectID, req.WordsDelta, req.ElapsedSec, req.TimerState)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"session":     s,
		"state":       GetState(uid),
		"today":       GetToday(uid),        // 今日净增字数/净码字时长（里程碑今日口径）
		"server_time": time.Now().UnixMilli(), // 本机钟校准
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

func handleNudge(c *gin.Context) {
	uid := c.GetString("userID")
	var req struct {
		Nudge string `json:"nudge"`
	}
	_ = c.ShouldBindJSON(&req)
	Nudge(uid, req.Nudge)
	c.JSON(http.StatusOK, gin.H{"data": "nudged"})
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

func handleToday(c *gin.Context) {
	uid := c.GetString("userID")
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"today":       GetToday(uid),
		"server_time": time.Now().UnixMilli(),
	}})
}
