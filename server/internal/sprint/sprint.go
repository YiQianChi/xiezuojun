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
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
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
	EndsAt     time.Time // 两种模式都有值：time=目标时长；words=60min 上限

	AutoStart     bool      // 快速匹配房：2 人即自动倒计时开局（自建房不自动）
	AutoStartAt   time.Time // 自动开局时刻（零=未在倒计时）
	AutoCancelled bool      // 房主取消过自动开局，本轮不再自动武装

	Players map[string]*Player
	Order   []string // 加入顺序（waiting 态排序用）

	FinishedAt time.Time
}

const (
	beatTimeout    = 40 * time.Second  // 心跳超时判离线
	roomTTL        = 10 * time.Minute  // 结算后房间保留时长（供围观/重连看结果）
	waitingTTL     = 2 * time.Hour     // 无人开的等待房回收
	defaultMaxRoom = 8
	autoStartDelay = 10 * time.Second  // 快速匹配房凑齐 2 人后的开局倒计时
	wordsCap       = 60 * time.Minute  // 拼字数模式时间上限：到时按字数结算，防永远 running
)

var (
	mu       sync.Mutex
	rooms    = map[string]*Room{}
	userRoom = map[string]string{} // uid → code
	subsMu   sync.Mutex
	subs     = map[string]map[chan []byte]struct{}{} // uid → 订阅者集合

	rdbPersist *redis.Client // 房间快照持久化（nil=纯内存：桌面版/未配 Redis）
	rctx       = context.Background()
)

// ---- Redis 持久化：部署/重启不断局 ----
//   xzj:sprint:room:{code} → Room JSON（TTL 2h 兜底）
//   xzj:sprint:idx         → SET 房间号索引（启动恢复用）
// 桌面版（DESKTOP=1）与未配 REDIS_ADDR 时自动退化为纯内存。
// 注意：不在这里用包级 init()——DESKTOP 开关在 main 里 Setenv，包级 init 拿不到
//（buddy/vcode 踩过的坑）；由 RegisterRoutes 显式启动。

const (
	keyRoomPrefix = "xzj:sprint:room:"
	keyRoomIdx    = "xzj:sprint:idx"
)

func initPersistence() {
	if os.Getenv("DESKTOP") == "1" {
		return // 桌面版：纯内存
	}
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return // 未配置 Redis：退化为纯内存
	}
	rdbPersist = redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("REDIS_PASSWORD")})
	restoreFromRedis()
}

func persistRoom(r *Room) {
	if rdbPersist == nil {
		return
	}
	if b, err := json.Marshal(r); err == nil {
		rdbPersist.Set(rctx, keyRoomPrefix+r.Code, b, 2*time.Hour)
		rdbPersist.SAdd(rctx, keyRoomIdx, r.Code)
	}
}

func dropPersist(code string) {
	if rdbPersist == nil {
		return
	}
	rdbPersist.Del(rctx, keyRoomPrefix+code)
	rdbPersist.SRem(rctx, keyRoomIdx, code)
}

// restoreFromRedis 启动恢复：没结算的房间捞回内存；过期房顺手清掉。
// 恢复时全员 LastBeat 重置为 now——给 40s 心跳宽限，避免重启瞬间被判离线清场。
func restoreFromRedis() {
	if rdbPersist == nil {
		return
	}
	codes, err := rdbPersist.SMembers(rctx, keyRoomIdx).Result()
	if err != nil {
		return
	}
	now := time.Now()
	for _, code := range codes {
		b, err := rdbPersist.Get(rctx, keyRoomPrefix+code).Bytes()
		if err != nil {
			rdbPersist.SRem(rctx, keyRoomIdx, code)
			continue
		}
		var r Room
		if json.Unmarshal(b, &r) != nil || len(r.Players) == 0 {
			dropPersist(code)
			continue
		}
		mu.Lock()
		if r.State == "finished" && now.Sub(r.FinishedAt) > roomTTL {
			mu.Unlock(); dropPersist(code); continue
		}
		if r.State == "waiting" && now.Sub(r.CreatedAt) > waitingTTL {
			mu.Unlock(); dropPersist(code); continue
		}
		if rooms[code] == nil {
			for _, p := range r.Players {
				p.LastBeat = now
				p.Online = true
			}
			rooms[code] = &r
			for uid := range r.Players {
				userRoom[uid] = code
			}
		}
		mu.Unlock()
	}
}

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
	if r.State == "finished" {
		return nil, "本局已结束，等房主开下一局"
	}
	// running 也放行：中途加入，字数从 0 起跑、按剩余时间拼（竞品同款）
	if len(r.Players) >= r.MaxPlayers {
		return nil, "房间已满"
	}
	if r.Password != "" && r.Password != password {
		return nil, "密码不对"
	}
	joinRoom(r, uid, nickname)
	return r, ""
}

// QuickMatch 快速匹配：优先并入等待中的公开房；没有再进还有空位、剩余≥3 分钟的 running 房
// （running 本就支持中途加入——不进池会让两人先后匹配错开各开新房）
func QuickMatch(uid, nickname string) *Room {
	mu.Lock()
	defer mu.Unlock()
	LeaveLocked(uid)
	now := time.Now()
	var waiting, running []string
	for c, r := range rooms {
		if !r.IsPublic || len(r.Players) >= r.MaxPlayers {
			continue
		}
		if r.State == "waiting" && now.Sub(r.CreatedAt) < 10*time.Minute {
			waiting = append(waiting, c)
		} else if r.State == "running" && !r.EndsAt.IsZero() && r.EndsAt.Sub(now) >= 3*time.Minute {
			running = append(running, c)
		}
	}
	for _, pool := range [][]string{waiting, running} {
		if len(pool) == 0 {
			continue
		}
		sort.Strings(pool)
		r := rooms[pool[0]]
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
		AutoStart:  true, // 凑齐 2 人自动倒计时开局，不干等房主
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
	armAutoStart(r)
	persistRoom(r)
	broadcastRoom(r)
}

// armAutoStart 快速匹配房凑齐 2 人自动进入开局倒计时（房主取消过则不武装）
func armAutoStart(r *Room) {
	if r.AutoStart && r.State == "waiting" && !r.AutoCancelled && r.AutoStartAt.IsZero() && len(r.Players) >= 2 {
		r.AutoStartAt = time.Now().Add(autoStartDelay)
	}
}

// checkAutoStart 倒计时到点自动开局；人数掉回 1 则撤倒计时（心跳/清扫双入口调用）
func checkAutoStart(r *Room) {
	if r.State != "waiting" || r.AutoStartAt.IsZero() {
		return
	}
	if len(r.Players) < 2 {
		r.AutoStartAt = time.Time{}
		return
	}
	if !time.Now().Before(r.AutoStartAt) {
		startLocked(r)
	}
}

// migrateHost 房主离开/掉线：传位给加入最早的在线成员（调用方持锁）
func migrateHost(r *Room) {
	for _, uid := range r.Order {
		if p, ok := r.Players[uid]; ok && p.Online {
			r.HostUID = uid
			return
		}
	}
	if len(r.Order) > 0 {
		if _, ok := r.Players[r.Order[0]]; ok {
			r.HostUID = r.Order[0]
		}
	}
}

// Leave 主动离开（拼字中离开=标记退赛，保留成绩到结算；等待房房主离开=传位不散房）
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
		if len(r.Players) == 0 {
			delete(rooms, code)
			dropPersist(code)
			return
		}
		if r.HostUID == uid {
			migrateHost(r) // 房主走了传位给最早成员，房不散、邀请链接不变
		}
		checkAutoStart(r) // 人掉回 1 撤自动开局倒计时
		persistRoom(r)
		broadcastRoom(r)
	} else {
		// running/finished：标记离线，成绩保留到房间回收
		if len(onlinePlayers(r)) == 0 {
			finishIfDue(r)
		} else {
			persistRoom(r)
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
	startLocked(r)
	return r, ""
}

// startLocked 真正开局（房主手点与自动开局共用；调用方持锁）
func startLocked(r *Room) {
	r.State = "running"
	r.AutoStartAt = time.Time{}
	r.StartedAt = time.Now()
	if r.Mode == "time" {
		r.EndsAt = r.StartedAt.Add(time.Duration(r.Target) * time.Minute)
	} else {
		r.EndsAt = r.StartedAt.Add(wordsCap) // 拼字数没上限会永远 running，到时按字数结算
	}
	for _, p := range r.Players {
		p.LastBeat = time.Now()
	}
	persistRoom(r)
	broadcastRoom(r)
}

// Restart 同房再来一局：房主把结算房重置回等待。房间号不变=邀请链接仍有效、成员不散
func Restart(uid string) (*Room, string) {
	mu.Lock()
	defer mu.Unlock()
	r := roomOf(uid)
	if r == nil {
		return nil, "不在房间里"
	}
	if r.HostUID != uid {
		return nil, "只有房主能开下一局"
	}
	if r.State != "finished" {
		return nil, "本局还没结束"
	}
	r.State = "waiting"
	r.StartedAt, r.EndsAt, r.FinishedAt = time.Time{}, time.Time{}, time.Time{}
	r.AutoCancelled = false
	for _, p := range r.Players {
		p.Words = 0
		p.LastBeat = time.Now()
		p.Online = true // 宽限：按在线起算，没心跳的下一轮清扫会纠正
	}
	armAutoStart(r) // 快速匹配房人还在就接着自动开
	persistRoom(r)
	broadcastRoom(r)
	return r, ""
}

// CancelAutoStart 房主取消自动开局倒计时（本轮不再自动武装，改为手点开始）
func CancelAutoStart(uid string) (*Room, string) {
	mu.Lock()
	defer mu.Unlock()
	r := roomOf(uid)
	if r == nil {
		return nil, "不在房间里"
	}
	if r.HostUID != uid {
		return nil, "只有房主能取消"
	}
	r.AutoStartAt = time.Time{}
	r.AutoCancelled = true
	persistRoom(r)
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
	checkAutoStart(r) // 自动开局倒计时（心跳 5s 一拍驱动，比清扫更跟手）
	if r.State == "running" || r.State == "waiting" {
		persistRoom(r)
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
	// 两种模式 EndsAt 都有值（words=60min 上限），到时按字数结算
	if r.State == "running" && !r.EndsAt.IsZero() && !time.Now().Before(r.EndsAt) {
		finishRoom(r)
	}
}

func finishRoom(r *Room) {
	if r.State == "finished" {
		return
	}
	r.State = "finished"
	r.FinishedAt = time.Now()
	persistRoom(r)
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
	AutoStartMs int64        `json:"auto_start_at_ms"` // >0=自动开局倒计时截止时刻
	IsQuick     bool         `json:"is_quick"`         // 快速匹配房
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
	if !r.AutoStartAt.IsZero() {
		v.AutoStartMs = r.AutoStartAt.UnixMilli()
	}
	v.IsQuick = r.AutoStart
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

// RoomListItem 对外的公开房摘要（房间列表用）
type RoomListItem struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Mode        string `json:"mode"`
	TargetLabel string `json:"target_label"`
	State       string `json:"state"`
	Players     int    `json:"players"`
	MaxPlayers  int    `json:"max_players"`
	EndsAtMs    int64  `json:"ends_at_ms"`
	ServerNowMs int64  `json:"server_now_ms"`
}

// ListPublicRooms 公开房列表：waiting + running 都列（中途可进）；死房/满员标注由前端处理
func ListPublicRooms() []RoomListItem {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now()
	var out []RoomListItem
	for _, r := range rooms {
		if !r.IsPublic || r.State == "finished" {
			continue
		}
		// 等待超 10 分钟没人开的、全员离线的死房不列
		if r.State == "waiting" && (now.Sub(r.CreatedAt) > 10*time.Minute || len(onlinePlayers(r)) == 0) {
			continue
		}
		if r.State == "running" && len(onlinePlayers(r)) == 0 {
			continue
		}
		item := RoomListItem{
			Code:        r.Code,
			Name:        r.Name,
			Mode:        r.Mode,
			TargetLabel: viewOf(r, "").TargetLabel,
			State:       r.State,
			Players:     len(r.Players),
			MaxPlayers:  r.MaxPlayers,
			ServerNowMs: now.UnixMilli(),
		}
		if r.Mode == "time" && !r.EndsAt.IsZero() {
			item.EndsAtMs = r.EndsAt.UnixMilli()
		}
		out = append(out, item)
	}
	// 在拼的排前面，其次按人数降序
	sort.SliceStable(out, func(a, b int) bool {
		sa, sb := 0, 0
		if out[a].State == "running" {
			sa = 1
		}
		if out[b].State == "running" {
			sb = 1
		}
		if sa != sb {
			return sa > sb
		}
		if out[a].Players != out[b].Players {
			return out[a].Players > out[b].Players
		}
		return out[a].Code < out[b].Code
	})
	if len(out) > 20 {
		out = out[:20]
	}
	return out
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
						broadcastRoom(r) // waiting 也要广播：房主传位/成员离线徽标靠它
					}
				}
				// 房主掉线：等待房传位，别让房卡死没人能开局
				if r.State == "waiting" && r.HostUID != "" {
					if h := r.Players[r.HostUID]; h != nil && !h.Online {
						old := r.HostUID
						migrateHost(r)
						if r.HostUID != old {
							broadcastRoom(r)
						}
					}
				}
				checkAutoStart(r) // 自动开局倒计时到点（清扫兜底，心跳驱动为主）
				finishIfDue(r)
				// 回收：结算房超 TTL / 等待房超时 / 全员离线的 running 房
				if r.State == "finished" && now.Sub(r.FinishedAt) > roomTTL {
					for uid := range r.Players {
						delete(userRoom, uid)
					}
					delete(rooms, code)
					dropPersist(code)
					continue
				}
				if r.State == "waiting" && (now.Sub(r.CreatedAt) > waitingTTL || len(onlinePlayers(r)) == 0) {
					for uid := range r.Players {
						delete(userRoom, uid)
					}
					delete(rooms, code)
					dropPersist(code)
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
