// Package sprint 拼字房：多人约同一时段比字数（竞品橙瓜码字/作家助手「拼字星球」同型功能）。
//
// 设计（MVP，2026-10-30 拍板口径）：
//   - 纯内存房间表 + 进程内 SSE（单实例部署；桌面版无 Redis 依赖，重启丢房可接受）
//   - 房间性质二选一：🔒 仅邀请（默认，房间号/邀请链接/密码进门）/ 🌍 公开（进快速匹配池）
//     —— 一期不做房间大厅列表（冷启动空列表比没有更伤），公开房只被「快速匹配」发现
//   - 模式：time=拼时间（倒计时比谁写得多）/ words=拼字数（先到 N 字者胜）
//   - 字数口径：客户端上报「本局累计字数」（进房时基线 + 编辑器字数增量），服务端只存展示
//   - 榜单实时性：每个成员心跳（约 5s）广播一次房间快照；倒计时由 EndsAt+server_time 客户端自算
package sprint

import (
	"math/rand"
	"sort"
	"sync"
	"time"
)

// Player 房间内一个成员
type Player struct {
	UserID   string    `json:"user_id"`
	Nickname string    `json:"nickname"`
	Words    int       `json:"words"` // 本局累计字数（客户端上报）
	Online   bool      `json:"online"`
	LastBeat time.Time `json:"-"`
	JoinedAt time.Time `json:"joined_at"`
}

// Room 一间拼字房
type Room struct {
	Code       string // 4 位房间号，活跃房中唯一
	Name       string
	Mode       string // time | words
	Target     int    // time=分钟；words=字数
	IsPublic   bool
	Password   string // 可选，仅邀请房常用
	MaxPlayers int
	HostUID    string
	State      string // waiting | running | finished
	CreatedAt  time.Time
	StartedAt  time.Time
	EndsAt     time.Time // time 模式有值

	Players map[string]*Player
	Order   []string // 加入顺序（waiting 态排序用）

	FinishedAt time.Time
}

const (
	beatTimeout    = 40 * time.Second // 心跳超时判离线
	roomTTL        = 10 * time.Minute // 结算后房间保留时长（供围观/重连看结果）
	waitingTTL     = 2 * time.Hour    // 无人开的等待房回收
	defaultMaxRoom = 8
)

var (
	mu       sync.Mutex
	rooms    = map[string]*Room{}
	userRoom = map[string]string{} // uid → code
	subsMu   sync.Mutex
	subs     = map[string]map[chan []byte]struct{}{} // uid → 订阅者集合
)

// ============ SSE（进程内广播） ============

func subscribe(uid string) chan []byte {
	ch := make(chan []byte, 16)
	subsMu.Lock()
	if subs[uid] == nil {
		subs[uid] = map[chan []byte]struct{}{}
	}
	subs[uid][ch] = struct{}{}
	subsMu.Unlock()
	return ch
}

func unsubscribe(uid string, ch chan []byte) {
	subsMu.Lock()
	if m, ok := subs[uid]; ok {
		delete(m, ch)
		if len(m) == 0 {
			delete(subs, uid)
		}
	}
	subsMu.Unlock()
}

// broadcastRoom 把房间快照推给房内所有成员（按人定制 me 字段，否则前端 find 不到自己）
func broadcastRoom(r *Room) {
	for uid := range r.Players {
		publish(uid, marshalEvent("room", viewOf(r, uid)))
	}
}

func publish(uid string, b []byte) {
	subsMu.Lock()
	defer subsMu.Unlock()
	for ch := range subs[uid] {
		select {
		case ch <- b:
		default: // 慢消费者丢帧，等下一拍快照
		}
	}
}

// ============ 房间操作 ============

func genCode() string {
	for i := 0; i < 50; i++ {
		c := ""
		for j := 0; j < 4; j++ {
			c += string(rune('0' + rand.Intn(10)))
		}
		if rooms[c] == nil {
			return c
		}
	}
	return string(rune('A' + rand.Intn(26))) + time.Now().Format("150405") // 兜底
}

// validateCreate 建房参数校验与归一
func validateCreate(name, mode string, target int) (string, int, bool) {
	if mode != "time" && mode != "words" {
		return "", 0, false
	}
	if mode == "time" {
		if target < 5 || target > 60 {
			target = 15
		}
	} else {
		if target < 200 || target > 20000 {
			target = 2000
		}
	}
	if name == "" {
		name = "拼字房"
	}
	if len([]rune(name)) > 20 {
		name = string([]rune(name)[:20])
	}
	return name, target, true
}

// Create 建房（自动离开旧房）
func Create(uid, nickname, name, mode string, target int, isPublic bool, password string, maxPlayers int) *Room {
	name, target, ok := validateCreate(name, mode, target)
	if !ok {
		return nil
	}
	if maxPlayers < 2 || maxPlayers > defaultMaxRoom {
		maxPlayers = 4
	}
	mu.Lock()
	defer mu.Unlock()
	LeaveLocked(uid)
	r := &Room{
		Code:       genCode(),
		Name:       name,
		Mode:       mode,
		Target:     target,
		IsPublic:   isPublic,
		MaxPlayers: maxPlayers,
		HostUID:    uid,
		State:      "waiting",
		CreatedAt:  time.Now(),
		Players:    map[string]*Player{},
	}
	if !isPublic && password != "" {
		r.Password = password
	}
	rooms[r.Code] = r
	joinRoom(r, uid, nickname)
	return r
}

// Join 凭房间号（+密码）加入
func Join(uid, nickname, code, password string) (*Room, string) {
	mu.Lock()
	defer mu.Unlock()
	LeaveLocked(uid)
	r := rooms[code]
	if r == nil {
		return nil, "房间不存在或已解散"
	}
	if r.State != "waiting" {
		return nil, "这一局已经在拼了，等下一局吧"
	}
	if len(r.Players) >= r.MaxPlayers {
		return nil, "房间已满"
	}
	if r.Password != "" && r.Password != password {
		return nil, "密码不对"
	}
	joinRoom(r, uid, nickname)
	return r, ""
}

// QuickMatch 快速匹配：优先并入等待中的公开房，没有就开一间公开默认房
func QuickMatch(uid, nickname string) *Room {
	mu.Lock()
	defer mu.Unlock()
	LeaveLocked(uid)
	// 找还有空位、开不超过 10 分钟的公开等待房
	now := time.Now()
	var codes []string
	for c, r := range rooms {
		if r.IsPublic && r.State == "waiting" && len(r.Players) < r.MaxPlayers && now.Sub(r.CreatedAt) < 10*time.Minute {
			codes = append(codes, c)
		}
	}
	if len(codes) > 0 {
		sort.Strings(codes)
		r := rooms[codes[0]]
		joinRoom(r, uid, nickname)
		return r
	}
	r := &Room{
		Code:       genCode(),
		Name:       "快速拼字房",
		Mode:       "time",
		Target:     15,
		IsPublic:   true,
		MaxPlayers: 4,
		HostUID:    uid,
		State:      "waiting",
		CreatedAt:  now,
		Players:    map[string]*Player{},
	}
	rooms[r.Code] = r
	joinRoom(r, uid, nickname)
	return r
}

func joinRoom(r *Room, uid, nickname string) {
	if nickname == "" {
		nickname = "菌" + uid[len(uid)-3:]
	}
	r.Players[uid] = &Player{
		UserID:   uid,
		Nickname: nickname,
		Online:   true,
		LastBeat: time.Now(),
		JoinedAt: time.Now(),
	}
	r.Order = append(r.Order, uid)
	userRoom[uid] = r.Code
	broadcastRoom(r)
}

// Leave 主动离开（房主离开=解散等待房；拼字中离开=标记退赛，保留成绩到结算）
func Leave(uid string) {
	mu.Lock()
	defer mu.Unlock()
	LeaveLocked(uid)
}

func LeaveLocked(uid string) {
	code := userRoom[uid]
	if code == "" {
		return
	}
	r := rooms[code]
	delete(userRoom, uid)
	if r == nil {
		return
	}
	delete(r.Players, uid)
	for i, u := range r.Order {
		if u == uid {
			r.Order = append(r.Order[:i], r.Order[i+1:]...)
			break
		}
	}
	if r.State == "waiting" {
		if len(r.Players) == 0 || r.HostUID == uid {
			// 房主走人 or 空房：等待房解散，通知残余成员
			ev := marshalEvent("dissolved", &RoomView{Code: code, State: "waiting"})
			for u := range r.Players {
				publish(u, ev)
				delete(userRoom, u)
			}
			delete(rooms, code)
		} else {
			broadcastRoom(r)
		}
	} else {
		// running/finished：标记离线，成绩保留到房间回收
		if len(onlinePlayers(r)) == 0 {
			finishIfDue(r)
		} else {
			broadcastRoom(r)
		}
	}
}

// Start 房主开局
func Start(uid string) (*Room, string) {
	mu.Lock()
	defer mu.Unlock()
	r := roomOf(uid)
	if r == nil {
		return nil, "不在房间里"
	}
	if r.HostUID != uid {
		return nil, "只有房主能开始"
	}
	if r.State != "waiting" {
		return r, ""
	}
	if len(r.Players) < 1 {
		return nil, "房间里没人"
	}
	r.State = "running"
	r.StartedAt = time.Now()
	if r.Mode == "time" {
		r.EndsAt = r.StartedAt.Add(time.Duration(r.Target) * time.Minute)
	}
	for _, p := range r.Players {
		p.LastBeat = time.Now()
	}
	broadcastRoom(r)
	return r, ""
}

// Heartbeat 上报本局字数；返回房间快照
func Heartbeat(uid string, words int) *RoomView {
	mu.Lock()
	defer mu.Unlock()
	r := roomOf(uid)
	if r == nil {
		return nil
	}
	if p := r.Players[uid]; p != nil {
		if words < 0 {
			words = 0
		}
		p.Words = words
		p.Online = true
		p.LastBeat = time.Now()
	}
	// 拼字数模式：有人达标即结算
	if r.State == "running" && r.Mode == "words" {
		for _, p := range r.Players {
			if p.Words >= r.Target {
				finishRoom(r)
				break
			}
		}
	}
	if r.State == "running" {
		finishIfDue(r)
	}
	if r.State == "running" || r.State == "waiting" {
		broadcastRoom(r) // 心跳驱动榜单广播（约 5s 一拍）
	}
	return viewOf(r, uid)
}

// roomOf 调用方需持锁
func roomOf(uid string) *Room {
	if code := userRoom[uid]; code != "" {
		return rooms[code]
	}
	return nil
}

func onlinePlayers(r *Room) []*Player {
	var out []*Player
	for _, p := range r.Players {
		if p.Online {
			out = append(out, p)
		}
	}
	return out
}

func finishIfDue(r *Room) {
	if r.State == "running" && r.Mode == "time" && !time.Now().Before(r.EndsAt) {
		finishRoom(r)
	}
}

func finishRoom(r *Room) {
	if r.State == "finished" {
		return
	}
	r.State = "finished"
	r.FinishedAt = time.Now()
	for uid := range r.Players {
		publish(uid, marshalEvent("finished", viewOf(r, uid)))
	}
}

// ============ 快照 ============

// PlayerView 面向前端的成员视图
type PlayerView struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Words    int    `json:"words"`
	Online   bool   `json:"online"`
	Rank     int    `json:"rank"` // running/finished 态有效
	IsHost   bool   `json:"is_host"`
}

// RoomView 面向前端的房间快照
type RoomView struct {
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	Mode        string       `json:"mode"`
	Target      int          `json:"target"`
	TargetLabel string       `json:"target_label"`
	IsPublic    bool         `json:"is_public"`
	HasPass     bool         `json:"has_password"`
	MaxPlayers  int          `json:"max_players"`
	HostUID     string       `json:"host_uid"`
	State       string       `json:"state"`
	StartedAtMs int64        `json:"started_at_ms"`
	EndsAtMs    int64        `json:"ends_at_ms"`
	ServerNowMs int64        `json:"server_now_ms"`
	Players     []PlayerView `json:"players"`
	Me          string       `json:"me"`
}

func viewOf(r *Room, meUID string) *RoomView {
	v := &RoomView{
		Code:        r.Code,
		Name:        r.Name,
		Mode:        r.Mode,
		Target:      r.Target,
		IsPublic:    r.IsPublic,
		HasPass:     r.Password != "",
		MaxPlayers:  r.MaxPlayers,
		HostUID:     r.HostUID,
		State:       r.State,
		StartedAtMs: r.StartedAt.UnixMilli(),
		ServerNowMs: time.Now().UnixMilli(),
		Me:          meUID,
	}
	if r.Mode == "time" {
		v.TargetLabel = minutesLabel(r.Target)
		if !r.EndsAt.IsZero() {
			v.EndsAtMs = r.EndsAt.UnixMilli()
		}
	} else {
		v.TargetLabel = itoa(r.Target) + " 字"
	}
	// waiting=加入顺序；running/finished=字数降序
	type pe struct {
		p  *Player
		iq int
	}
	var list []pe
	for i, uid := range r.Order {
		if p, ok := r.Players[uid]; ok {
			list = append(list, pe{p, i})
		}
	}
	if r.State != "waiting" {
		sort.SliceStable(list, func(a, b int) bool {
			if list[a].p.Words != list[b].p.Words {
				return list[a].p.Words > list[b].p.Words
			}
			if list[a].p.Online != list[b].p.Online {
				return list[a].p.Online
			}
			return list[a].iq < list[b].iq
		})
	}
	for i, e := range list {
		v.Players = append(v.Players, PlayerView{
			UserID:   e.p.UserID,
			Nickname: e.p.Nickname,
			Words:    e.p.Words,
			Online:   e.p.Online,
			Rank:     i + 1,
			IsHost:   e.p.UserID == r.HostUID,
		})
	}
	return v
}

// GetRoomFor 当前所在房间快照（不在房=nil）
func GetRoomFor(uid string) *RoomView {
	mu.Lock()
	defer mu.Unlock()
	r := roomOf(uid)
	if r == nil {
		return nil
	}
	return viewOf(r, uid)
}

// ============ 后台清扫 ============

func init() {
	go func() {
		for range time.Tick(5 * time.Second) {
			mu.Lock()
			now := time.Now()
			for code, r := range rooms {
				// 心跳超时 → 离线标记（不清成绩）
				for _, p := range r.Players {
					if p.Online && now.Sub(p.LastBeat) > beatTimeout {
						p.Online = false
						if r.State == "running" {
							broadcastRoom(r)
						}
					}
				}
				finishIfDue(r)
				// 回收：结算房超 TTL / 等待房超时 / 全员离线的 running 房
				if r.State == "finished" && now.Sub(r.FinishedAt) > roomTTL {
					for uid := range r.Players {
						delete(userRoom, uid)
					}
					delete(rooms, code)
					continue
				}
				if r.State == "waiting" && (now.Sub(r.CreatedAt) > waitingTTL || len(onlinePlayers(r)) == 0) {
					for uid := range r.Players {
						delete(userRoom, uid)
					}
					delete(rooms, code)
					continue
				}
				if r.State == "running" && len(onlinePlayers(r)) == 0 {
					finishRoom(r)
				}
			}
			mu.Unlock()
		}
	}()
}
