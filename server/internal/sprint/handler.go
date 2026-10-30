package sprint

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册拼字房路由（authd 组下）；同时启动 Redis 持久化（桌面版/未配置时自动跳过）
func RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/sprint/create", handleCreate)
	rg.POST("/sprint/join", handleJoin)
	rg.POST("/sprint/quickmatch", handleQuickMatch)
	rg.POST("/sprint/leave", handleLeave)
	rg.POST("/sprint/start", handleStart)
	rg.POST("/sprint/heartbeat", handleHeartbeat)
	rg.GET("/sprint/room", handleRoom)
	rg.GET("/sprint/rooms", handleRooms) // 公开房列表
	rg.GET("/sprint/events", handleEvents) // SSE 实时推送

	// Redis 房间持久化（显式启动而非包级 init：DESKTOP 开关在 main 里 Setenv，包级 init 拿不到）
	go initPersistence()
}

// marshalEvent 组一个 SSE 事件包
func marshalEvent(typ string, view *RoomView) []byte {
	b, err := json.Marshal(map[string]interface{}{
		"type": typ,
		"room": view,
	})
	if err != nil {
		return []byte(`{"type":"ping"}`)
	}
	return b
}

func minutesLabel(n int) string { return itoa(n) + " 分钟" }
func itoa(n int) string         { return strconv.Itoa(n) }

func handleCreate(c *gin.Context) {
	uid := c.GetString("userID")
	var req struct {
		Nickname   string `json:"nickname"`
		Name       string `json:"name"`
		Mode       string `json:"mode"`       // time | words
		Target     int    `json:"target"`     // 分钟 或 字数
		IsPublic   bool   `json:"is_public"`  // false=仅邀请（默认）
		Password   string `json:"password"`   // 可选
		MaxPlayers int    `json:"max_players"`
	}
	_ = c.ShouldBindJSON(&req)
	r := Create(uid, req.Nickname, req.Name, req.Mode, req.Target, req.IsPublic, req.Password, req.MaxPlayers)
	if r == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room params"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": viewOf(r, uid)})
}

func handleJoin(c *gin.Context) {
	uid := c.GetString("userID")
	var req struct {
		Nickname string `json:"nickname"`
		Code     string `json:"code"`
		Password string `json:"password"`
	}
	_ = c.ShouldBindJSON(&req)
	r, msg := Join(uid, req.Nickname, req.Code, req.Password)
	if r == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": viewOf(r, uid)})
}

func handleQuickMatch(c *gin.Context) {
	uid := c.GetString("userID")
	var req struct {
		Nickname string `json:"nickname"`
	}
	_ = c.ShouldBindJSON(&req)
	r := QuickMatch(uid, req.Nickname)
	c.JSON(http.StatusOK, gin.H{"data": viewOf(r, uid)})
}

func handleLeave(c *gin.Context) {
	Leave(c.GetString("userID"))
	c.JSON(http.StatusOK, gin.H{"data": "left"})
}

func handleStart(c *gin.Context) {
	uid := c.GetString("userID")
	r, msg := Start(uid)
	if r == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": viewOf(r, uid)})
}

func handleHeartbeat(c *gin.Context) {
	uid := c.GetString("userID")
	var req struct {
		Words int `json:"words"`
	}
	_ = c.ShouldBindJSON(&req)
	view := Heartbeat(uid, req.Words)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"room":        view,
		"server_time": time.Now().UnixMilli(),
	}})
}

func handleRoom(c *gin.Context) {
	uid := c.GetString("userID")
	view := GetRoomFor(uid)
	c.JSON(http.StatusOK, gin.H{"data": view}) // 不在房=null，前端按 null 处理
}

func handleRooms(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": ListPublicRooms()})
}

// handleEvents SSE：房内成员收实时快照（join/leave/开始/结算/每次心跳的榜单）
func handleEvents(c *gin.Context) {
	uid := c.GetString("userID")
	if uid == "" {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	w := c.Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // 防 nginx 缓冲

	ch := subscribe(uid)
	defer unsubscribe(uid, ch)

	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()

	fmt.Fprint(w, ": connected\n\n")
	w.Flush()

	// 连上先补一拍当前房间快照（重连恢复用）
	if view := GetRoomFor(uid); view != nil {
		fmt.Fprintf(w, "data: %s\n\n", marshalEvent("room", view))
		w.Flush()
	}

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.Flush()
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			w.Flush()
		}
	}
}
