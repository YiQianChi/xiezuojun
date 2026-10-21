package buddy

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// SSE 实时推送：配对成功/对方离开/收到加油即时下发，不等心跳轮询。
// 通道：xzj:buddy:evt:{uid}（Redis Pub/Sub，多实例安全）。
// 客户端断线自行重连；心跳轮询仍保留作为兜底和会话续期。

const keyEvents = "xzj:buddy:evt:"

// Event 推送给单个用户的实时事件
type Event struct {
	Type      string   `json:"type"`                // matched | peer_left | cheer | peer_milestone | peer_nudge
	Peer      *Session `json:"peer,omitempty"`      // matched 时带上对方会话
	Milestone int      `json:"milestone,omitempty"` // peer_milestone：对方跨过的字数档（500 的倍数）
	Nudge     string   `json:"nudge,omitempty"`     // peer_nudge：互动类型（water/rest/fire/together）
	From      string   `json:"from,omitempty"`      // peer_nudge：发送方昵称
}

func publishEvent(uid string, ev Event) {
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	rdb.Publish(rctx, keyEvents+uid, b)
}

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

	sub := rdb.Subscribe(rctx, keyEvents+uid)
	defer sub.Close()
	ch := sub.Channel()

	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()

	fmt.Fprint(w, ": connected\n\n")
	w.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg.Payload)
			w.Flush()
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			w.Flush()
		}
	}
}
