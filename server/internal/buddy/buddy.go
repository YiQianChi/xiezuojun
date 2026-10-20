package buddy

import (
	"sync"
	"time"
)

// Session 一个正在写作的用户会话（纯内存，进程重启即清——MVP 够用）
type Session struct {
	UserID     string    `json:"user_id"`
	Nickname   string    `json:"nickname"`
	ProjectID  uint      `json:"project_id"`
	StartedAt  time.Time `json:"started_at"`
	LastBeat   time.Time `json:"last_beat"`
	WordsDelta int       `json:"words_delta"`
	Matched    string    `json:"matched,omitempty"` // 配对的搭子 userID；空=未配对
}

// Pair 一对匹配的搭子
type Pair struct {
	A, B     string // userID
	CheersA  int    // A 收到的加油次数
	CheersB  int
	CreatedAt time.Time
}

var (
	mu       sync.RWMutex
	sessions = map[string]*Session{} // userID → session
	pairs    = map[string]*Pair{}    // userID → pair（两边都有入口）
	// 匹配等待队列（FIFO）
	waitQueue []string
)

const (
	beatTimeout = 40 * time.Second // 心跳超时：会话视为结束
	matchWait   = 5 * time.Minute  // 排队超时：放弃
)

// Heartbeat 上报/刷新会话。返回当前会话（含配对信息）
func Heartbeat(userID, nickname string, projectID uint, wordsDelta int) *Session {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now()
	s, ok := sessions[userID]
	if !ok || now.Sub(s.LastBeat) > beatTimeout {
		// 新会话：旧的（若有）先清理
		cleanupUser(userID)
		s = &Session{UserID: userID, Nickname: nickname, ProjectID: projectID, StartedAt: now, WordsDelta: wordsDelta}
		sessions[userID] = s
	}
	s.LastBeat = now
	s.WordsDelta = wordsDelta
	s.Nickname = nickname
	return s
}

// StopWriting 主动结束：清理会话与配对
func StopWriting(userID string) {
	mu.Lock()
	defer mu.Unlock()
	cleanupUser(userID)
	delete(sessions, userID)
}

// cleanupUser 从等待队列和配对中摘除（需持锁）
func cleanupUser(userID string) {
	// 出队列
	for i, q := range waitQueue {
		if q == userID {
			waitQueue = append(waitQueue[:i], waitQueue[i+1:]...)
			break
		}
	}
	// 解除配对
	if p, ok := pairs[userID]; ok {
		other := p.A
		if p.A == userID {
			other = p.B
		}
		delete(pairs, other)
		delete(pairs, userID)
	}
}

// MatchRequested 加入匹配队列，返回配对结果：matched=true 时 peer 为搭子
func MatchRequested(userID string) (matched bool, peer *Session) {
	mu.Lock()
	defer mu.Unlock()
	// 已有配对直接返回
	if p, ok := pairs[userID]; ok {
		other := p.A
		if p.A == userID {
			other = p.B
		}
		if s, ok2 := sessions[other]; ok2 {
			return true, s
		}
	}
	// 已在队列：还在等
	for _, q := range waitQueue {
		if q == userID {
			return false, nil
		}
	}
	// 尝试从队列头取一个等待者（不能是自己）
	for i, cand := range waitQueue {
		if cand != userID {
			if s, ok := sessions[cand]; ok && time.Since(s.LastBeat) < beatTimeout {
				// 配对成功
				waitQueue = append(waitQueue[:i], waitQueue[i+1:]...)
				p := &Pair{A: cand, B: userID, CreatedAt: time.Now()}
				pairs[cand] = p
				pairs[userID] = p
				sessions[cand].Matched = userID
				if cur, ok2 := sessions[userID]; ok2 {
					cur.Matched = cand
				}
				return true, s
			}
			// 候选已死，踢出队列
			waitQueue = append(waitQueue[:i], waitQueue[i+1:]...)
			// 不 break，继续找下一个
		}
	}
	// 没人可配：入队等待
	waitQueue = append(waitQueue, userID)
	return false, nil
}

// LeaveMatch 退出匹配（排队中或已配对均可退）
func LeaveMatch(userID string) {
	mu.Lock()
	defer mu.Unlock()
	// 排队中：出队
	for i, q := range waitQueue {
		if q == userID {
			waitQueue = append(waitQueue[:i], waitQueue[i+1:]...)
			break
		}
	}
	// 已配对：解除（搭子会自动回到未配对态，前端轮询感知）
	if p, ok := pairs[userID]; ok {
		other := p.A
		if p.A == userID {
			other = p.B
		}
		delete(pairs, other)
		delete(pairs, userID)
		if s, ok2 := sessions[other]; ok2 {
			s.Matched = ""
		}
		if s, ok2 := sessions[userID]; ok2 {
			s.Matched = ""
		}
	}
}

// Cheer 给搭子送加油
func Cheer(fromUserID string) {
	mu.Lock()
	defer mu.Unlock()
	p, ok := pairs[fromUserID]
	if !ok {
		return
	}
	if p.A == fromUserID {
		p.CheersB++ // 加油送给对方
	} else {
		p.CheersA++
	}
}

// BuddyState 搭子状态视图（轮询返回）
type BuddyState struct {
	You       *Session `json:"you"`
	Peer      *Session `json:"peer,omitempty"`
	PeerCheer int      `json:"peer_cheer,omitempty"` // 对方给你送的加油数（读后清零语义由前端处理）
	Waiting   bool     `json:"waiting"`              // 是否在匹配队列中
	QueueLen  int      `json:"queue_len"`            // 当前排队人数（含自己）
	Online    int      `json:"online"`               // 正在写作的总人数
}

// GetState 查询某人视角的搭子状态
func GetState(userID string) BuddyState {
	mu.RLock()
	defer mu.RUnlock()
	st := BuddyState{Online: len(sessions)}
	if s, ok := sessions[userID]; ok && time.Since(s.LastBeat) < beatTimeout {
		st.You = cloneSession(s)
	}
	if p, ok := pairs[userID]; ok {
		other := p.A
		if p.A == userID {
			other = p.B
		}
		if s, ok2 := sessions[other]; ok2 && time.Since(s.LastBeat) < beatTimeout {
			st.Peer = cloneSession(s)
			if p.A == userID {
				st.PeerCheer = p.CheersA
			} else {
				st.PeerCheer = p.CheersB
			}
		}
	}
	for _, q := range waitQueue {
		if q == userID {
			st.Waiting = true
		}
	}
	st.QueueLen = len(waitQueue)
	return st
}

// ConsumeCheer 读取并清零收到的加油（前端拿到后展示一次）
func ConsumeCheer(userID string) int {
	mu.Lock()
	defer mu.Unlock()
	p, ok := pairs[userID]
	if !ok {
		return 0
	}
	var n int
	if p.A == userID {
		n = p.CheersA
		p.CheersA = 0
	} else {
		n = p.CheersB
		p.CheersB = 0
	}
	return n
}

func cloneSession(s *Session) *Session {
	c := *s
	return &c
}
