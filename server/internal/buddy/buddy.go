package buddy

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis 实现：会话/配对/加油全部落 Redis，多实例安全、重启存活。
//
// 键设计（计时靠 started_at 时间戳，前端按墙钟计算）：
//   xzj:buddy:sess:{uid}    → Session JSON，TTL=beatTimeout（心跳续期；过期即离线）
//   xzj:buddy:matched:{uid} → 搭子 uid，TTL=beatTimeout（双方心跳都会续）
//   xzj:buddy:queue         → LIST 等待匹配队列
//   xzj:buddy:cheer:{uid}   → 收到的加油计数，TTL=5min
//   xzj:buddy:online        → ZSET，score=最后心跳时间戳（统计在线数）

// Session 一个正在写作的用户会话
type Session struct {
	UserID     string    `json:"user_id"`
	Nickname   string    `json:"nickname"`
	ProjectID  uint      `json:"project_id"`
	StartedAt  time.Time `json:"started_at"`
	LastBeat   time.Time `json:"last_beat"`
	WordsDelta int       `json:"words_delta"`
	Matched    string    `json:"matched,omitempty"` // 配对的搭子 userID；空=未配对
}

const beatTimeout = 40 * time.Second // 心跳超时：会话过期即离线

var rdb *redis.Client
var rctx = context.Background()

const (
	keySess    = "xzj:buddy:sess:"
	keyMatched = "xzj:buddy:matched:"
	keyQueue   = "xzj:buddy:queue"
	keyCheer   = "xzj:buddy:cheer:"
	keyOnline  = "xzj:buddy:online"
)

func init() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       envInt("REDIS_DB", 0),
	})
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func sessKey(uid string) string    { return keySess + uid }
func matchedKey(uid string) string { return keyMatched + uid }
func cheerKey(uid string) string   { return keyCheer + uid }

// Heartbeat 上报/刷新会话
func Heartbeat(userID, nickname string, projectID uint, wordsDelta int) *Session {
	now := time.Now()
	s := readSession(userID)
	if s == nil {
		s = &Session{UserID: userID, StartedAt: now}
	}
	s.Nickname = nickname
	s.ProjectID = projectID
	s.WordsDelta = wordsDelta
	s.LastBeat = now
	writeSession(s)
	// 在线表登记 + 配对键双向续期
	rdb.ZAdd(rctx, keyOnline, redis.Z{Score: float64(now.Unix()), Member: userID})
	if peer, err := rdb.Get(rctx, matchedKey(userID)).Result(); err == nil {
		rdb.Expire(rctx, matchedKey(userID), beatTimeout)
		rdb.Expire(rctx, matchedKey(peer), beatTimeout)
		s.Matched = peer
	} else {
		s.Matched = ""
	}
	return s
}

// StopWriting 主动结束
func StopWriting(userID string) {
	if peer, err := rdb.Get(rctx, matchedKey(userID)).Result(); err == nil {
		rdb.Del(rctx, matchedKey(peer))
		publishEvent(peer, Event{Type: "peer_left"})
	}
	rdb.Del(rctx, sessKey(userID), matchedKey(userID), cheerKey(userID))
	rdb.ZRem(rctx, keyOnline, userID)
	rdb.LRem(rctx, keyQueue, 1, userID)
}

func readSession(uid string) *Session {
	raw, err := rdb.Get(rctx, sessKey(uid)).Bytes()
	if err != nil {
		return nil
	}
	var s Session
	if json.Unmarshal(raw, &s) != nil {
		return nil
	}
	return &s
}

func writeSession(s *Session) {
	b, _ := json.Marshal(s)
	rdb.Set(rctx, sessKey(s.UserID), b, beatTimeout)
}

// MatchRequested 请求配对
func MatchRequested(userID string) (matched bool, peer *Session) {
	// 已配对：直接返回
	if peerID, err := rdb.Get(rctx, matchedKey(userID)).Result(); err == nil {
		if s := readSession(peerID); s != nil {
			return true, s
		}
		rdb.Del(rctx, matchedKey(userID))
	}
	// 从队列头弹候选人（不能是自己、必须活着）
	for {
		cand, err := rdb.LPop(rctx, keyQueue).Result()
		if err != nil {
			break // 队列空
		}
		if cand == userID {
			rdb.RPush(rctx, keyQueue, userID) // 自己放回队尾
			break
		}
		if s := readSession(cand); s != nil {
			// 配对成功：双向写 matched
			rdb.Set(rctx, matchedKey(cand), userID, beatTimeout)
			rdb.Set(rctx, matchedKey(userID), cand, beatTimeout)
			if cur := readSession(userID); cur != nil {
				cur.Matched = cand
				writeSession(cur)
				// 实时通知等待方（对方可能正在 3s 轮询之外）
				publishEvent(cand, Event{Type: "matched", Peer: cur})
			}
			s.Matched = userID
			writeSession(s)
			return true, s
		}
		// 死候选人：丢弃，继续弹
	}
	// 没配上：入队等待
	rdb.RPush(rctx, keyQueue, userID)
	return false, nil
}

// LeaveMatch 退出匹配
func LeaveMatch(userID string) {
	rdb.LRem(rctx, keyQueue, 1, userID)
	if peer, err := rdb.Get(rctx, matchedKey(userID)).Result(); err == nil {
		rdb.Del(rctx, matchedKey(peer), matchedKey(userID))
		publishEvent(peer, Event{Type: "peer_left"})
	}
}

// Cheer 送加油
func Cheer(fromUserID string) {
	peer, err := rdb.Get(rctx, matchedKey(fromUserID)).Result()
	if err != nil {
		return
	}
	rdb.Incr(rctx, cheerKey(peer))
	rdb.Expire(rctx, cheerKey(peer), 5*time.Minute)
	publishEvent(peer, Event{Type: "cheer"})
}

// BuddyState 搭子状态视图（轮询返回）
type BuddyState struct {
	You       *Session `json:"you"`
	Peer      *Session `json:"peer,omitempty"`
	PeerCheer int      `json:"peer_cheer,omitempty"` // 对方给你送的加油数（读后清零）
	Waiting   bool     `json:"waiting"`              // 是否在匹配队列中
	QueueLen  int      `json:"queue_len"`            // 当前排队人数（含自己）
	Online    int      `json:"online"`               // 正在写作的总人数
}

// GetState 状态视图
func GetState(userID string) BuddyState {
	// 清扫在线表过期成员
	cutoff := float64(time.Now().Add(-beatTimeout).Unix())
	rdb.ZRemRangeByScore(rctx, keyOnline, "-inf", strconv.FormatFloat(cutoff, 'f', 0, 64))
	online, _ := rdb.ZCard(rctx, keyOnline).Result()

	st := BuddyState{Online: int(online)}
	if s := readSession(userID); s != nil {
		if peer, err := rdb.Get(rctx, matchedKey(userID)).Result(); err == nil {
			s.Matched = peer
		}
		st.You = s
		if peerID, err := rdb.Get(rctx, matchedKey(userID)).Result(); err == nil {
			if p := readSession(peerID); p != nil {
				p.Matched = userID
				st.Peer = p
			}
		}
	}
	if q, err := rdb.LRange(rctx, keyQueue, 0, -1).Result(); err == nil {
		st.QueueLen = len(q)
		for _, uid := range q {
			if uid == userID {
				st.Waiting = true
				break
			}
		}
	}
	return st
}

// ConsumeCheer 读走加油数（GetDel 原子清零）
func ConsumeCheer(userID string) int {
	n, err := rdb.GetDel(rctx, cheerKey(userID)).Int()
	if err != nil {
		return 0
	}
	return n
}
