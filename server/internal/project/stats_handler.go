package project

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"xiezuojun-server/internal/model"
)

// WritingStats GET /api/projects/:id/stats — 项目维度的写作统计
func (h *Handler) WritingStats(c *gin.Context) {
	pid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if !h.requireProjectOwner(c, uint(pid)) {
		return
	}
	userID := getUserID(c)

	now := time.Now()
	loc := now.Location()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	startOfWeek := startOfToday.AddDate(0, 0, -6) // 含今天共7天
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)

	var today, week, month int64
	h.db.Model(&model.WordCountLog{}).
		Where("user_id = ? AND project_id = ? AND created_at >= ?", userID, pid, startOfToday).
		Select("COALESCE(SUM(delta), 0)").Scan(&today)
	h.db.Model(&model.WordCountLog{}).
		Where("user_id = ? AND project_id = ? AND created_at >= ?", userID, pid, startOfWeek).
		Select("COALESCE(SUM(delta), 0)").Scan(&week)
	h.db.Model(&model.WordCountLog{}).
		Where("user_id = ? AND project_id = ? AND created_at >= ?", userID, pid, startOfMonth).
		Select("COALESCE(SUM(delta), 0)").Scan(&month)

	var totalWords int64
	h.db.Model(&model.Document{}).
		Where("project_id = ? AND status = 1", pid).
		Select("COALESCE(SUM(word_count), 0)").Scan(&totalWords)

	streak := h.calcStreakDays(userID, uint(pid))
	daily := h.dailyWords(userID, uint(pid), 14)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"today_words": today,
		"week_words":  week,
		"month_words": month,
		"total_words": totalWords,
		"streak_days": streak,
		"daily":       daily,
	}})
}

type dailyEntry struct {
	Date  string `json:"date"`
	Words int    `json:"words"`
}

// dailyWords 最近 n 天每天的 SUM(delta)，无记录的天补零。
func (h *Handler) dailyWords(userID string, pid uint, days int) []dailyEntry {
	type row struct {
		Date  string
		Words int
	}
	var rows []row
	now := time.Now()
	loc := now.Location()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(days - 1))
	h.db.Model(&model.WordCountLog{}).
		Select("DATE(created_at) as date, COALESCE(SUM(delta), 0) as words").
		Where("user_id = ? AND project_id = ? AND created_at >= ?", userID, pid, start).
		Group("DATE(created_at)").
		Order("date DESC").
		Scan(&rows)

	m := make(map[string]int, len(rows))
	for _, r := range rows {
		m[r.Date] = r.Words
	}
	result := make([]dailyEntry, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		result = append(result, dailyEntry{Date: d, Words: m[d]})
	}
	return result
}

// calcStreakDays 从今天往前，连续有写作记录的天数。今天没写则从昨天起算。
func (h *Handler) calcStreakDays(userID string, pid uint) int {
	type row struct {
		Date string
	}
	var rows []row
	h.db.Model(&model.WordCountLog{}).
		Select("DISTINCT DATE(created_at) as date").
		Where("user_id = ? AND project_id = ?", userID, pid).
		Scan(&rows)
	if len(rows) == 0 {
		return 0
	}
	set := make(map[string]bool, len(rows))
	for _, r := range rows {
		set[r.Date] = true
	}

	now := time.Now()
	loc := now.Location()
	cur := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	streak := 0
	for {
		if set[cur.Format("2006-01-02")] {
			streak++
			cur = cur.AddDate(0, 0, -1)
			continue
		}
		if streak == 0 {
			// 今天没写，从昨天开始算
			cur = cur.AddDate(0, 0, -1)
			if set[cur.Format("2006-01-02")] {
				streak++
				cur = cur.AddDate(0, 0, -1)
				continue
			}
		}
		break
	}
	return streak
}
