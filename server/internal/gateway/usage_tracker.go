// UsageTracker — 用量追踪与配额管理

package gateway

import (
	"fmt"
	"sync"
	"time"
)

// UsageTracker 用量追踪器
type UsageTracker struct {
	mu sync.RWMutex

	// 内存存储（后续可接入 Redis/数据库）
	dailyUsage  map[string]int // userID -> 今日已用 tokens
	monthlyUsage map[string]int // userID -> 本月已用 tokens
	records      []UsageRecord  // 历史记录

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

	// 保存记录
	t.records = append(t.records, record)

	// 限制内存中记录数，防止无限增长
	if len(t.records) > 10000 {
		t.records = t.records[len(t.records)-5000:]
	}

	return nil
}

// CheckQuota 检查配额
func (t *UsageTracker) CheckQuota(userID string, tier string) (*QuotaStatus, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// 获取配额配置
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

	// 统计最近 7 天的用量
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
	case "pro":
		return DefaultQuota.ProTier
	case "enterprise":
		return DefaultQuota.EnterpriseTier
	default:
		return DefaultQuota.FreeTier
	}
}
