package user

// bindWechatPhone 合并/绑定逻辑自测（sqlite 内存库，不碰 Redis/微信 API）

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"xiezuojun-server/internal/model"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.AuthIdentity{}, &model.Project{}); err != nil {
		t.Fatal(err)
	}
	return &Handler{db: db}
}

func seedUser(t *testing.T, db *gorm.DB, userID string, identities ...string) {
	t.Helper()
	if err := db.Create(&model.User{UserID: userID, Nickname: "n", Plan: "free", Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	for _, p := range identities {
		identifier := p + "-" + userID
		if err := db.Create(&model.AuthIdentity{UserID: userID, Provider: p, Identifier: identifier}).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestBindWechatPhone_MergeOrphan(t *testing.T) {
	h := newTestHandler(t)
	// 孤儿微信号：只有 wechat identity + 一本书
	seedUser(t, h.db, "orphan", "wechat")
	h.db.Create(&model.Project{UserID: "orphan", Name: "书"})
	// 老账号：手机 + 邮箱
	seedUser(t, h.db, "olduser", "phone", "email")
	h.db.Model(&model.AuthIdentity{}).Where("user_id = ? AND provider = 'phone'", "olduser").Update("identifier", "13800000000")

	user, isNew, merged, err := h.bindWechatPhone("wechat-orphan", "13800000000")
	if err != nil || user == nil {
		t.Fatalf("err=%v", err)
	}
	if !merged || isNew || user.UserID != "olduser" {
		t.Fatalf("merged=%v isNew=%v uid=%s", merged, isNew, user.UserID)
	}
	// 作品迁走
	var p model.Project
	h.db.First(&p)
	if p.UserID != "olduser" {
		t.Fatalf("project not migrated: %s", p.UserID)
	}
	// 孤儿号禁用，微信身份已转移
	var orphan model.User
	h.db.Where("user_id = ?", "orphan").First(&orphan)
	if orphan.Status != 0 {
		t.Fatal("orphan not disabled")
	}
	var wi model.AuthIdentity
	h.db.Where("provider = 'wechat'").First(&wi)
	if wi.UserID != "olduser" {
		t.Fatal("wechat identity not transferred")
	}
}

func TestBindWechatPhone_BrandNew(t *testing.T) {
	h := newTestHandler(t)
	user, isNew, merged, err := h.bindWechatPhone("wx-new", "13900000000")
	if err != nil {
		t.Fatal(err)
	}
	if !isNew || merged || user.Phone != "13900000000" {
		t.Fatalf("isNew=%v merged=%v phone=%s", isNew, merged, user.Phone)
	}
	var cnt int64
	h.db.Model(&model.AuthIdentity{}).Where("user_id = ?", user.UserID).Count(&cnt)
	if cnt != 2 {
		t.Fatalf("expected 2 identities, got %d", cnt)
	}
}

func TestBindWechatPhone_AttachPhoneToWechatUser(t *testing.T) {
	h := newTestHandler(t)
	seedUser(t, h.db, "wxuser", "wechat")
	user, isNew, merged, err := h.bindWechatPhone("wechat-wxuser", "13700000000")
	if err != nil {
		t.Fatal(err)
	}
	if isNew || merged || user.UserID != "wxuser" {
		t.Fatalf("isNew=%v merged=%v uid=%s", isNew, merged, user.UserID)
	}
	var cnt int64
	h.db.Model(&model.AuthIdentity{}).Where("user_id = ? AND provider = 'phone' AND identifier = '13700000000'", "wxuser").Count(&cnt)
	if cnt != 1 {
		t.Fatal("phone identity not created")
	}
	var u model.User
	h.db.Where("user_id = ?", "wxuser").First(&u)
	if u.Phone != "13700000000" {
		t.Fatal("users.phone not updated")
	}
}

func TestBindWechatPhone_AttachWechatToPhoneUser(t *testing.T) {
	h := newTestHandler(t)
	seedUser(t, h.db, "phoneuser", "phone")
	h.db.Model(&model.AuthIdentity{}).Where("user_id = ?", "phoneuser").Update("identifier", "13600000000")
	user, _, merged, err := h.bindWechatPhone("wx-first-time", "13600000000")
	if err != nil {
		t.Fatal(err)
	}
	if merged || user.UserID != "phoneuser" {
		t.Fatalf("merged=%v uid=%s", merged, user.UserID)
	}
	var cnt int64
	h.db.Model(&model.AuthIdentity{}).Where("user_id = ? AND provider = 'wechat' AND identifier = 'wx-first-time'", "phoneuser").Count(&cnt)
	if cnt != 1 {
		t.Fatal("wechat identity not attached")
	}
}

func TestBindWechatPhone_Conflict(t *testing.T) {
	h := newTestHandler(t)
	// 微信号所属账号还有邮箱身份（完整账号），不能静默并入别人的手机号账号
	seedUser(t, h.db, "fullwx", "wechat", "email")
	seedUser(t, h.db, "phoneuser", "phone")
	h.db.Model(&model.AuthIdentity{}).Where("user_id = ?", "phoneuser").Update("identifier", "13500000000")
	_, _, _, err := h.bindWechatPhone("wechat-fullwx", "13500000000")
	if !errors.Is(err, errAccountConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	// 两个账号都应完好
	var u model.User
	h.db.Where("user_id = ?", "fullwx").First(&u)
	if u.Status != 1 {
		t.Fatal("full account should stay active")
	}
}
