package buddy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"xiezuojun-server/internal/model"
)

// Redis 实现：会话/配对/加油全部落 Redis，多实例安全、重启存活。
//
// 键设计（计时靠 started_at 时间戳，前端按墙钟计算）：
//   xzj:buddy:sess:{uid}    → Session JSON，TTL=beatTimeout（心跳续期；过期即离线）
//   xzj:buddy:matched:{uid} → 搭子 uid，TTL=beatTimeout（双方心跳都会续）
//   xzj:buddy:queue         → LIST 等待匹配队列
//   xzj:buddy:cheer:{uid}   → 收到的加油计数，TTL=5min
//   xzj:buddy:online        → ZSET，score=最后心跳时间戳（统计在线数）
//   xzj:buddy:daily:2006-01-02 → HASH{w:今日净增字数, ms:今日净码字毫秒}，TTL=48h（里程碑按今日口径）
//
// 计时口径（2026-10-26 重构）：时长以「前端累计、暂停不计时」为准——
// 心跳上报 elapsed_sec（本端累计秒数）+ timer_state，对方卡片直接展示该值并本地续滚；
// started_at 仅保留作会话起点参考，不再用于对方时长计算（旧口径含暂停/挂机时间）。

// Session 一个正在写作的用户会话
type Session struct {
	UserID     string    `json:"user_id"`
	Nickname   string    `json:"nickname"`
	ProjectID  uint      `json:"project_id"`
	StartedAt  time.Time `json:"started_at"`
	LastBeat   time.Time `json:"last_beat"`
	WordsDelta int       `json:"words_delta"`
	ElapsedSec int       `json:"elapsed_sec,omitempty"`  // 本端累计码字秒数（暂停不计时，新一场归零）
	TimerState string    `json:"timer_state,omitempty"` // running | paused
	Matched    string    `json:"matched,omitempty"`     // 配对的搭子 userID；空=未配对
}

// TodayStat 今日写作统计（净增口径：删字/新一场回退都如实减）
type TodayStat struct {
	Words int   `json:"words"`
	Ms    int64 `json:"ms"`
}

const beatTimeout = 40 * time.Second // 心跳超时：会话过期即离线

var rdb *redis.Client
var rctx = context.Background()
var db *gorm.DB // SetDB 注入（main 启动时）；桌面版同款注入

// SetDB 注入 GORM 句柄（writing_sessions 落库用）
func SetDB(d *gorm.DB) { db = d }

const (
	keySess    = "xzj:buddy:sess:"
	keyMatched = "xzj:buddy:matched:"
	keyQueue   = "xzj:buddy:queue"
	keyCheer   = "xzj:buddy:cheer:"
	keyOnline  = "xzj:buddy:online"
	keyDaily   = "xzj:buddy:daily:" // + uid + ":" + 2006-01-02（按日分键，跨日自然清零）
	keyGoal    = "xzj:buddy:goal:"  // + uid（每日码字目标，跟账号走，无 TTL）
)

func init() {
	// ponytail-desktop: DESKTOP=1 时用内存 miniredis，桌面版无需装 Redis
	if os.Getenv("DESKTOP") == "1" {
		mr := miniredis.NewMiniRedis()
		if err := mr.Start(); err != nil {
			panic("miniredis start failed: " + err.Error())
		}
		rdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
		return
	}
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
func dailyKey(uid string) string {
	return keyDaily + uid + ":" + time.Now().In(time.Local).Format("2006-01-02")
}

func goalKey(uid string) string { return keyGoal + uid }

// GetGoal 每日码字目标（0=未设置，前端用默认值）
func GetGoal(uid string) int {
	v, err := rdb.Get(rctx, goalKey(uid)).Result()
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(v)
	return n
}

// SetGoal 每日码字目标存账号（Redis 持久，无 TTL）
func SetGoal(uid string, goal int) {
	rdb.Set(rctx, goalKey(uid), goal, 0)
}

// ReportWords 裸写上报：只累计今日净增字数（不涉搭子会话/计时状态）
func ReportWords(uid string, delta int) TodayStat {
	return updateDaily(uid, int64(delta), 0)
}

// updateDaily 今日净增统计：单调增量（新-旧>=0）直接累计，回退（新<旧，删字/换章/新一场回退）如实减
func updateDaily(userID string, words, elapsedMs int64) TodayStat {
	key := dailyKey(userID)
	pipe := rdb.Pipeline()
	wCmd := pipe.HIncrBy(rctx, key, "w", words)
	msCmd := pipe.HIncrBy(rctx, key, "ms", elapsedMs)
	pipe.Expire(rctx, key, 48*time.Hour)
	_, _ = pipe.Exec(rctx)
	return TodayStat{Words: int(wCmd.Val()), Ms: msCmd.Val()}
}

// readDaily 读今日统计（没有=零值）
func readDaily(userID string) TodayStat {
	m, err := rdb.HGetAll(rctx, dailyKey(userID)).Result()
	if err != nil || len(m) == 0 {
		return TodayStat{}
	}
	w, _ := strconv.ParseInt(m["w"], 10, 64)
	ms, _ := strconv.ParseInt(m["ms"], 10, 64)
	return TodayStat{Words: int(w), Ms: ms}
}

// Heartbeat 上报/刷新会话。timerState: running|paused；elapsedSec 为前端累计码字秒数（暂停不计时）
func Heartbeat(userID, nickname string, projectID uint, wordsDelta, elapsedSec int, timerState string) *Session {
	now := time.Now()
	s := readSession(userID)
	prevDelta := 0
	prevElapsed := 0
	if s != nil {
		prevDelta = s.WordsDelta
		prevElapsed = s.ElapsedSec
	} else {
		s = &Session{UserID: userID, StartedAt: now}
	}
	s.Nickname = nickname
	s.ProjectID = projectID
	s.WordsDelta = wordsDelta
	s.ElapsedSec = elapsedSec
	s.TimerState = timerState
	s.LastBeat = now
	writeSession(s)
	// 在线表登记
	rdb.ZAdd(rctx, keyOnline, redis.Z{Score: float64(now.Unix()), Member: userID})
	// 今日累计（净增口径，仅 running 时累计，防暂停期间心跳刷出重复时间）
	var today TodayStat
	prevToday := readDaily(userID)
	if timerState == "running" {
		today = updateDaily(userID, int64(wordsDelta-prevDelta), int64(elapsedSec-prevElapsed)*1000)
	} else {
		today = prevToday
	}
	if peer, err := rdb.Get(rctx, matchedKey(userID)).Result(); err == nil {
		rdb.Expire(rctx, matchedKey(userID), beatTimeout)
		rdb.Expire(rctx, matchedKey(peer), beatTimeout)
		s.Matched = peer
		// 里程碑：今日字数每跨 500 的倍数 → 实时推给搭子（负数按 0 算，删字/回退不触发）
		oldLv := max0(prevToday.Words) / 500
		newLv := max0(today.Words) / 500
		if newLv > oldLv {
			publishEvent(peer, Event{Type: "peer_milestone", Peer: s, Milestone: newLv * 500})
		}
	}
	return s
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// StopWriting 主动结束（会话清掉；今日统计已在心跳里累计，15s 心跳粒度内的尾差可接受）
func StopWriting(userID string) {
	if peer, err := rdb.Get(rctx, matchedKey(userID)).Result(); err == nil {
		rdb.Del(rctx, matchedKey(peer))
		publishEvent(peer, Event{Type: "peer_left"})
	}
	rdb.Del(rctx, sessKey(userID), matchedKey(userID), cheerKey(userID))
	rdb.ZRem(rctx, keyOnline, userID)
	rdb.LRem(rctx, keyQueue, 1, userID)
}

// FinishSession 一场结束：写 writing_sessions 表（时长=前端累计口径，调用方上报）
func FinishSession(userID string, projectID uint, startedAt, endedAt time.Time, durationSec, wordsDelta int) {
	if db == nil {
		return // db 未注入（桌面版异常路径）——不落库但不炸
	}
	if endedAt.Before(startedAt) {
		endedAt = startedAt
	}
	db.Create(&model.WritingSession{
		UserID: userID, ProjectID: projectID,
		StartedAt: startedAt, EndedAt: endedAt,
		DurationSec: durationSec, WordsDelta: wordsDelta,
	})
}

// ReportData 分享战报数据（今日口径，跨项目全量——发圈晒的是今天所有写作）
type ReportData struct {
	Date        string `json:"date"`         // 10月26日 · 星期一
	TodayWords  int64  `json:"today_words"`  // 今日字数（word_count_logs 净增）
	TodayMin    int    `json:"today_min"`    // 今日码字分钟
	Rate        string `json:"rate"`         // 字/分钟（今日，1 位小数）
	StreakDays  int    `json:"streak_days"`  // 连续写作天数（跨项目）
	Trend       []TrendPoint `json:"trend"`  // 近 14 天字数
}

// TrendPoint 每日字数
type TrendPoint struct {
	Date  string `json:"date"`
	Words int64  `json:"words"`
}

var weekdays = [...]string{"日", "一", "二", "三", "四", "五", "六"}

// GetReport 分享战报聚合
func GetReport(userID string) ReportData {
	rp := ReportData{}
	if db == nil {
		return rp
	}
	now := time.Now()
	rp.Date = fmt.Sprintf("%d月%d日 · 星期%s", now.Month(), now.Day(), weekdays[now.Weekday()])
	loc := now.Location()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	start14 := startOfToday.AddDate(0, 0, -13)

	// 今日字数 + 14 天趋势（word_count_logs 全项目净增）
	var words int64
	db.Model(&model.WordCountLog{}).
		Where("user_id = ? AND created_at >= ?", userID, startOfToday).
		Select("COALESCE(SUM(delta),0)").Scan(&words)
	rp.TodayWords = words

	type row struct {
		Date  string
		Words int64
	}
	var rows []row
	db.Model(&model.WordCountLog{}).
		Select("DATE(created_at) as date, COALESCE(SUM(delta),0) as words").
		Where("user_id = ? AND created_at >= ?", userID, start14).
		Group("DATE(created_at)").Scan(&rows)
	m := make(map[string]int64, len(rows))
	for _, r := range rows {
		if r.Words < 0 {
			r.Words = 0
		}
		m[r.Date] = r.Words
	}
	for i := 0; i < 14; i++ {
		d := start14.AddDate(0, 0, i).Format("2006-01-02")
		rp.Trend = append(rp.Trend, TrendPoint{Date: d, Words: m[d]})
	}

	// 今日时长（writing_sessions）
	var sec int64
	db.Model(&model.WritingSession{}).
		Where("user_id = ? AND started_at >= ?", userID, startOfToday).
		Select("COALESCE(SUM(duration_sec),0)").Scan(&sec)
	rp.TodayMin = int(sec / 60)

	// 速率：今日字数 ÷ 今日纯码字分钟（有 10 分钟以上才算，否则空）
	if rp.TodayMin >= 10 {
		rp.Rate = fmt.Sprintf("%.1f", float64(rp.TodayWords)/float64(rp.TodayMin))
	}

	// 连续天数（跨项目）
	var drows []row
	db.Model(&model.WordCountLog{}).
		Select("DISTINCT DATE(created_at) as date").
		Where("user_id = ?", userID).Scan(&drows)
	set := make(map[string]bool, len(drows))
	for _, r := range drows {
		set[r.Date] = true
	}
	cur := startOfToday
	for {
		if set[cur.Format("2006-01-02")] {
			rp.StreakDays++
			cur = cur.AddDate(0, 0, -1)
			continue
		}
		if rp.StreakDays == 0 { // 今天没写从昨天算
			cur = cur.AddDate(0, 0, -1)
			if set[cur.Format("2006-01-02")] {
				rp.StreakDays++
				cur = cur.AddDate(0, 0, -1)
				continue
			}
		}
		break
	}
	return rp
}

// SessionStats 时长统计视图（统计面板用）
type SessionStats struct {
	TodayMin     int   `json:"today_min"`
	WeekMin      int   `json:"week_min"`
	TotalMin     int   `json:"total_min"`
	SessionCount int64 `json:"session_count"`
}

// GetSessionStats 时长统计（今天/本周/累计分钟数 + 总场数）
func GetSessionStats(userID string, projectID uint) SessionStats {
	if db == nil {
		return SessionStats{}
	}
	now := time.Now()
	loc := now.Location()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	startOfWeek := startOfToday.AddDate(0, 0, -6)
	st := SessionStats{}
	scope := func() *gorm.DB {
		q := db.Model(&model.WritingSession{}).Where("user_id = ?", userID)
		if projectID > 0 {
			q = q.Where("project_id = ?", projectID)
		}
		return q.Session(&gorm.Session{NewDB: true})
	}
	var sec int64
	scope().Select("COALESCE(SUM(duration_sec),0)").Where("started_at >= ?", startOfToday).Scan(&sec)
	st.TodayMin = int(sec / 60)
	sec = 0
	scope().Select("COALESCE(SUM(duration_sec),0)").Where("started_at >= ?", startOfWeek).Scan(&sec)
	st.WeekMin = int(sec / 60)
	sec = 0
	scope().Select("COALESCE(SUM(duration_sec),0)").Scan(&sec)
	st.TotalMin = int(sec / 60)
	scope().Count(&st.SessionCount)
	return st
}

// GetToday 今日写作统计（总结/刻度条用）
func GetToday(userID string) TodayStat {
	return readDaily(userID)
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

// Nudge 快捷互动（喝杯水/歇一会等）：SSE 实时推给搭子。
// 冷却由前端控制（60s/种），服务端只做同种 10s 限频防连点。
var validNudges = map[string]bool{"water": true, "rest": true, "fire": true, "together": true}

func Nudge(fromUserID, nudge string) {
	if !validNudges[nudge] {
		return
	}
	peer, err := rdb.Get(rctx, matchedKey(fromUserID)).Result()
	if err != nil {
		return
	}
	cdKey := "xzj:buddy:nudgecd:" + fromUserID + ":" + nudge
	if ok, _ := rdb.SetNX(rctx, cdKey, 1, 10*time.Second).Result(); !ok {
		return // 同种 10s 内已发过
	}
	from := ""
	if s := readSession(fromUserID); s != nil {
		from = s.Nickname
	}
	publishEvent(peer, Event{Type: "peer_nudge", Nudge: nudge, From: from})
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
