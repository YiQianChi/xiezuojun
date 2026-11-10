package sprint

import (
	"testing"
	"time"
)

// 测试辅助：直接造房进内存表（绕过 HTTP 层）
func mkRoom(code, mode string, public bool) *Room {
	r := &Room{
		Code: code, Name: "t", Mode: mode, Target: 15,
		IsPublic: public, MaxPlayers: 4, HostUID: "h0",
		State: "waiting", CreatedAt: time.Now(),
		Players: map[string]*Player{},
	}
	mu.Lock()
	rooms[code] = r
	mu.Unlock()
	return r
}

func cleanupRooms(codes ...string) {
	mu.Lock()
	for _, c := range codes {
		if r := rooms[c]; r != nil {
			for uid := range r.Players {
				delete(userRoom, uid)
			}
			delete(rooms, c)
		}
	}
	mu.Unlock()
}

func TestAutoStartLifecycle(t *testing.T) {
	r := mkRoom("T001", "time", true)
	r.AutoStart = true
	r.HostUID = "u1"
	defer cleanupRooms("T001")

	joinRoom(r, "u1", "甲")
	if !r.AutoStartAt.IsZero() {
		t.Fatal("1 人不应武装自动开局")
	}
	joinRoom(r, "u2", "乙")
	if r.AutoStartAt.IsZero() {
		t.Fatal("2 人应武装自动开局")
	}
	v := viewOf(r, "u1")
	if v.AutoStartMs <= 0 || !v.IsQuick {
		t.Fatalf("视图应带 auto_start_at_ms/is_quick: %+v", v)
	}
	// 非房主不能取消
	if _, msg := CancelAutoStart("u2"); msg == "" {
		t.Fatal("非房主取消应被拒")
	}
	if _, msg := CancelAutoStart("u1"); msg != "" {
		t.Fatalf("房主取消失败: %s", msg)
	}
	if !r.AutoStartAt.IsZero() || !r.AutoCancelled {
		t.Fatal("取消后应清零并置 AutoCancelled")
	}
	// 取消后新成员进来不再自动武装
	joinRoom(r, "u3", "丙")
	if !r.AutoStartAt.IsZero() {
		t.Fatal("AutoCancelled 后不应重新武装")
	}
	// 人掉回 1 撤倒计时
	r.AutoCancelled = false
	armAutoStart(r)
	if r.AutoStartAt.IsZero() {
		t.Fatal("3 人应重新武装")
	}
	Leave("u3")
	Leave("u2")
	if !r.AutoStartAt.IsZero() {
		t.Fatal("人数 <2 应撤倒计时")
	}
	// 到点自动开局
	joinRoom(r, "u2", "乙")
	if r.AutoStartAt.IsZero() {
		t.Fatal("2 人应武装")
	}
	r.AutoStartAt = time.Now().Add(-time.Second)
	checkAutoStart(r)
	if r.State != "running" {
		t.Fatalf("到点应自动开局，state=%s", r.State)
	}
}

func TestHostMigrationOnLeave(t *testing.T) {
	r := mkRoom("T002", "time", false)
	defer cleanupRooms("T002")
	joinRoom(r, "u1", "甲")
	joinRoom(r, "u2", "乙")
	r.HostUID = "u1"
	Leave("u1")
	if rooms["T002"] == nil {
		t.Fatal("房主离开等待房不应解散")
	}
	if r.HostUID != "u2" {
		t.Fatalf("房主应传位给 u2，得 %s", r.HostUID)
	}
	// 房主掉线（超时判离线）也传位：模拟清扫逻辑
	r.Players["u2"].Online = false
	r.Players["u2"].LastBeat = time.Now().Add(-beatTimeout - time.Second)
	joinRoom(r, "u3", "丙") // u3 在线
	migrateHost(r)
	if r.HostUID != "u3" {
		t.Fatalf("房主掉线应传位给在线的 u3，得 %s", r.HostUID)
	}
}

func TestRestartKeepsCode(t *testing.T) {
	r := mkRoom("T003", "time", false)
	joinRoom(r, "u1", "甲")
	joinRoom(r, "u2", "乙")
	r.HostUID = "u1"
	if _, msg := Start("u1"); msg != "" {
		t.Fatalf("开局失败: %s", msg)
	}
	r.Players["u1"].Words = 100
	mu.Lock()
	finishRoom(r)
	mu.Unlock()
	if r.State != "finished" {
		t.Fatal("应已结算")
	}
	Leave("u2") // running/finished 离开不清成绩
	if _, msg := Restart("u2"); msg == "" {
		t.Fatal("非房主 restart 应被拒")
	}
	if _, msg := Restart("u1"); msg != "" {
		t.Fatalf("restart 失败: %s", msg)
	}
	if r.State != "waiting" || r.Players["u1"].Words != 0 || !r.EndsAt.IsZero() {
		t.Fatalf("restart 应重置回 waiting 且清字数: state=%s words=%d", r.State, r.Players["u1"].Words)
	}
	if r.Code != "T003" {
		t.Fatal("restart 不应换房间号")
	}
}

func TestQuickMatchPools(t *testing.T) {
	defer cleanupRooms("T100", "T200")
	w := mkRoom("T100", "time", true)
	w.HostUID = "q0"
	joinRoom(w, "q0", "甲")
	w.MaxPlayers = 2 // 再进 1 人就满
	run := mkRoom("T200", "time", true)
	run.State = "running"
	run.MaxPlayers = 2 // 进 1 人就满
	run.EndsAt = time.Now().Add(10 * time.Minute)
	joinRoom(run, "r0", "乙")

	a := QuickMatch("qa", "测甲")
	if a == nil || a.Code != "T100" {
		t.Fatalf("应优先进等待房 T100，得 %v", a)
	}
	b := QuickMatch("qb", "测乙")
	if b == nil || b.Code != "T200" {
		t.Fatalf("等待房满员应次选 running 房 T200，得 %v", b)
	}
	// 剩余 <3 分钟的 running 房不再进池，另开新房
	run2 := mkRoom("T201", "time", true)
	run2.State = "running"
	run2.EndsAt = time.Now().Add(time.Minute)
	defer cleanupRooms("T201")
	c := QuickMatch("qc", "测丙")
	if c == nil || c.Code == "T201" || !c.AutoStart {
		t.Fatalf("无房可进应新开自动开局房，得 %v", c)
	}
}

func TestWordsModeHasCap(t *testing.T) {
	r := mkRoom("T300", "words", false)
	defer cleanupRooms("T300")
	r.Target = 2000
	joinRoom(r, "u1", "甲")
	r.HostUID = "u1"
	if _, msg := Start("u1"); msg != "" {
		t.Fatalf("开局失败: %s", msg)
	}
	if r.EndsAt.IsZero() {
		t.Fatal("words 模式也应有 EndsAt 上限")
	}
	// 到时按字数结算
	mu.Lock()
	r.EndsAt = time.Now().Add(-time.Second)
	finishIfDue(r)
	mu.Unlock()
	if r.State != "finished" {
		t.Fatal("words 到上限应结算")
	}
}
