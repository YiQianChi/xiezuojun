// UsageTracker — 用量追踪与配额管理
// 双写：内存（快速配额检查）+ MySQL（持久化）

package gateway

import (
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
)

// UsageTracker 用量追踪器
type UsageTracker struct {
	mu sync.RWMutex
	db *gorm.DB // MySQL，可为 nil（降级为纯内存）

	// 内存存储（快速配额检查）
	dailyUsage   map[string]int // userID -> 今日已用 tokens
	monthlyUsage map[string]int // userID -> 本月已用 tokens
	records      []UsageRecord  // 历史记录（最近 1000 条，用于 GetUsageStats）

	// 每日重置
	lastResetDay int
}

// NewUsageTracker 创建追踪器
func NewUsageTracker() *UsageTracker {
	return &UsageTracker{
		dailyUsage:   make(map[string]int),
		monthlyUsage: make(map[string]int),
		records:      make([]UsageRecord, 0),
		lastResetDay: time.Now().Day(),
	}
}

// SetDB 设置数据库连接（可选）
func (t *UsageTracker) SetDB(db *gorm.DB) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.db = db
}

// Track 记录用量
func (t *UsageTracker) Track(record UsageRecord) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 检查是否需要重置日用量
	now := time.Now()
	if now.Day() != t.lastResetDay {
		t.dailyUsage = make(map[string]int)
		t.lastResetDay = now.Day()
	}

	// 累加用量
	t.dailyUsage[record.UserID] += record.TotalTokens
	t.monthlyUsage[record.UserID] += record.TotalTokens

	// 保存到内存记录
	t.records = append(t.records, record)
	if len(t.records) > 1000 {
		t.records = t.records[len(t.records)-500:]
	}

	// 持久化到 MySQL
	if t.db != nil {
		dbRecord := struct {
			UserID           string    `gorm:"size:50;index"`
			ProjectID        string    `gorm:"size:50;index"`
			ModelID          string    `gorm:"size:100"`
			PromptTokens     int
			CompletionTokens int
			TotalTokens      int
			Cost             float64
			AIMode           string    `gorm:"size:20"`
			DurationMs       int64
			CreatedAt        time.Time `gorm:"index"`
		}{
			UserID:           record.UserID,
			ProjectID:        record.ProjectID,
			ModelID:          record.ModelID,
			PromptTokens:     record.PromptTokens,
			CompletionTokens: record.CompletionTokens,
			TotalTokens:      record.TotalTokens,
			Cost:             record.Cost,
			AIMode:           record.AIMode,
			DurationMs:       record.DurationMs,
			CreatedAt:        record.Timestamp,
		}
		if err := t.db.Table("usage_records").Create(&dbRecord).Error; err != nil {
			log.Printf("[usage_tracker] failed to persist record: %v", err)
		}
	}

	return nil
}

// CheckQuota 检查配额
func (t *UsageTracker) CheckQuota(userID string, tier string) (*QuotaStatus, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	quota := getQuotaByTier(tier)

	usedToday := t.dailyUsage[userID]
	usedMonth := t.monthlyUsage[userID]

	status := &QuotaStatus{
		Allowed:    true,
		Remaining:  quota.DailyTokens - usedToday,
		UsedToday:  usedToday,
		LimitToday: quota.DailyTokens,
		UsedMonth:  usedMonth,
		LimitMonth: quota.MonthlyTokens,
	}

	if usedToday >= quota.DailyTokens {
		status.Allowed = false
		status.Reason = fmt.Sprintf("daily quota exceeded: %d/%d", usedToday, quota.DailyTokens)
		return status, nil
	}

	if usedMonth >= quota.MonthlyTokens {
		status.Allowed = false
		status.Reason = fmt.Sprintf("monthly quota exceeded: %d/%d", usedMonth, quota.MonthlyTokens)
		return status, nil
	}

	return status, nil
}

// GetUsageStats 获取用户用量统计
func (t *UsageTracker) GetUsageStats(userID string) map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)

	var weeklyTokens int
	var weeklyCost float64
	modeCounts := make(map[string]int)

	for _, r := range t.records {
		if r.UserID == userID && r.Timestamp.After(weekAgo) {
			weeklyTokens += r.TotalTokens
			weeklyCost += r.Cost
			modeCounts[r.AIMode]++
		}
	}

	return map[string]interface{}{
		"daily_used":    t.dailyUsage[userID],
		"monthly_used":  t.monthlyUsage[userID],
		"weekly_tokens": weeklyTokens,
		"weekly_cost":   weeklyCost,
		"mode_counts":   modeCounts,
	}
}

// getQuotaByTier 根据等级获取配额
func getQuotaByTier(tier string) Quota {
	switch tier {
	case "pro", "vip":
		return DefaultQuota.ProTier
	case "enterprise", "svip":
		return DefaultQuota.EnterpriseTier
	default:
		return DefaultQuota.FreeTier
	}
}
