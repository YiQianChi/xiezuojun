package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"xiezuojun-server/internal/model"
)

// Handler 后台管理：用户管理 / 数据看板 / 内容监管 / 公告反馈
// 鉴权两层：JWT 登录 + users.is_admin 查库校验（管理员标记不进 JWT，改库即时生效）
type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// RequireAdmin 管理员校验中间件（挂在 JWTAuth 之后）
func (h *Handler) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("userID")
		uid, _ := userID.(string)
		if uid == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}
		var u model.User
		if err := h.db.Where("user_id = ?", uid).First(&u).Error; err != nil || !u.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "无管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func RegisterRoutes(g *gin.RouterGroup, h *Handler) {
	adminG := g.Group("/admin")
	adminG.Use(h.RequireAdmin())

	// 数据看板
	adminG.GET("/stats/overview", h.StatsOverview)
	adminG.GET("/stats/daily", h.StatsDaily)
	adminG.GET("/stats/ai-usage", h.StatsAIUsage)

	// 用户管理
	adminG.GET("/users", h.ListUsers)
	adminG.GET("/users/:user_id", h.GetUser)
	adminG.PATCH("/users/:user_id", h.UpdateUser)
	adminG.PATCH("/users/:user_id/status", h.SetUserStatus)
	adminG.PATCH("/users/:user_id/plan", h.SetUserPlan)
	adminG.POST("/users/:user_id/reset-password", h.ResetUserPassword)

	// 内容监管
	adminG.GET("/projects", h.ListProjects)
	adminG.GET("/projects/:id/documents", h.ListProjectDocuments)
	adminG.GET("/documents/:id/content", h.GetDocumentContent)
	adminG.PATCH("/projects/:id/status", h.SetProjectStatus)

	// 公告
	adminG.GET("/announcements", h.ListAnnouncements)
	adminG.POST("/announcements", h.CreateAnnouncement)
	adminG.DELETE("/announcements/:id", h.DeleteAnnouncement)

	// 反馈
	adminG.GET("/feedbacks", h.ListFeedbacks)
	adminG.PATCH("/feedbacks/:id", h.ResolveFeedback)

	// 公开的（登录即可，非管理员也能看公告/提反馈）
	g.GET("/announcements", h.PublicListAnnouncements)
	g.POST("/feedbacks", h.SubmitFeedback)
}

// ---------- 数据看板 ----------

func (h *Handler) StatsOverview(c *gin.Context) {
	var userTotal, projectTotal, docTotal int64
	var activeToday, newUsers7d int64
	var aiCalls7d int64
	var aiTokens7d int64

	h.db.Model(&model.User{}).Where("status >= 0").Count(&userTotal)
	h.db.Model(&model.Project{}).Where("status > 0").Count(&projectTotal)
	h.db.Model(&model.Document{}).Count(&docTotal)

	h.db.Model(&model.WordCountLog{}).Where("created_at >= ?", todayStart()).Select("DISTINCT user_id").Count(&activeToday)
	h.db.Model(&model.User{}).Where("created_at >= ?", time.Now().AddDate(0, 0, -7)).Count(&newUsers7d)
	h.db.Model(&model.UsageRecord{}).Where("created_at >= ?", time.Now().AddDate(0, 0, -7)).Count(&aiCalls7d)
	h.db.Model(&model.UsageRecord{}).Where("created_at >= ?", time.Now().AddDate(0, 0, -7)).
		Select("COALESCE(SUM(total_tokens),0)").Scan(&aiTokens7d)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"user_total":    userTotal,
		"project_total": projectTotal,
		"doc_total":     docTotal,
		"active_today":  activeToday,
		"new_users_7d":  newUsers7d,
		"ai_calls_7d":   aiCalls7d,
		"ai_tokens_7d":  aiTokens7d,
	}})
}

func (h *Handler) StatsDaily(c *gin.Context) {
	days := 30
	since := todayStart().AddDate(0, 0, -(days - 1))

	// 注册
	type row struct {
		Date string `json:"date"`
		N    int64  `json:"n"`
	}
	var regs, words []row
	h.db.Model(&model.User{}).
		Select("DATE_FORMAT(created_at,'%m-%d') as date, COUNT(*) as n").
		Where("created_at >= ?", since).Group("date").Order("date").Scan(&regs)

	// 活跃（按天有写作记录的独立用户数）+ 字数
	var act []row
	h.db.Model(&model.WordCountLog{}).
		Select("DATE_FORMAT(created_at,'%m-%d') as date, COUNT(DISTINCT user_id) as n").
		Where("created_at >= ? AND delta > 0", since).Group("date").Order("date").Scan(&act)

	h.db.Model(&model.WordCountLog{}).
		Select("DATE_FORMAT(created_at,'%m-%d') as date, COALESCE(SUM(delta),0) as n").
		Where("created_at >= ? AND delta > 0", since).Group("date").Order("date").Scan(&words)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"registrations": regs, "actives": act, "words": words,
	}})
}

func (h *Handler) StatsAIUsage(c *gin.Context) {
	type row struct {
		ModelID string  `json:"model_id"`
		Calls   int64   `json:"calls"`
		Tokens  int64   `json:"tokens"`
		Cost    float64 `json:"cost"`
	}
	var rows []row
	h.db.Model(&model.UsageRecord{}).
		Select("model_id, COUNT(*) as calls, COALESCE(SUM(total_tokens),0) as tokens, COALESCE(SUM(cost),0) as cost").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).
		Group("model_id").Order("calls DESC").Scan(&rows)

	// 按模式（续写/润色/对话等）
	type modeRow struct {
		AIMode string `json:"ai_mode"`
		Calls  int64  `json:"calls"`
	}
	var modes []modeRow
	h.db.Model(&model.UsageRecord{}).
		Select("ai_mode, COUNT(*) as calls").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).
		Group("ai_mode").Order("calls DESC").Scan(&modes)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"by_model": rows, "by_mode": modes}})
}

// ---------- 用户管理 ----------

func (h *Handler) ListUsers(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	const size = 20

	tx := h.db.Model(&model.User{})
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("email LIKE ? OR nickname LIKE ? OR user_id LIKE ?", like, like, like)
	}
	var total int64
	tx.Count(&total)

	var users []model.User
	tx.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&users)

	// 附带每用户的书数
	type res struct {
		model.User
		ProjectCount int64 `json:"project_count"`
	}
	out := make([]res, 0, len(users))
	for _, u := range users {
		var pc int64
		h.db.Model(&model.Project{}).Where("user_id = ? AND status > 0", u.UserID).Count(&pc)
		out = append(out, res{User: u, ProjectCount: pc})
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"total": total, "page": page, "list": out}})
}

func (h *Handler) GetUser(c *gin.Context) {
	uid := c.Param("user_id")
	var u model.User
	if err := h.db.Where("user_id = ?", uid).First(&u).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	var projects []model.Project
	h.db.Where("user_id = ? AND status > 0", uid).Order("id DESC").Find(&projects)

	// 写作统计：总字数增量、活跃天数、最近 30 天字数
	var totalWords, activeDays int64
	h.db.Model(&model.WordCountLog{}).Where("user_id = ? AND delta > 0", uid).
		Select("COALESCE(SUM(delta),0)").Scan(&totalWords)
	h.db.Model(&model.WordCountLog{}).Where("user_id = ? AND delta > 0", uid).
		Select("COUNT(DISTINCT DATE(created_at))").Scan(&activeDays)

	// AI 用量：30 天调用数 + tokens
	var aiCalls, aiTokens int64
	h.db.Model(&model.UsageRecord{}).Where("user_id = ? AND created_at >= ?", uid, time.Now().AddDate(0, 0, -30)).Count(&aiCalls)
	h.db.Model(&model.UsageRecord{}).Where("user_id = ? AND created_at >= ?", uid, time.Now().AddDate(0, 0, -30)).
		Select("COALESCE(SUM(total_tokens),0)").Scan(&aiTokens)

	// 最近登录设备（refresh_tokens 最新 5 条）
	type deviceRow struct {
		DeviceID   string    `json:"device_id"`
		IP         string    `json:"ip"`
		ExpiresAt  time.Time `json:"expires_at"`
		CreatedAt  time.Time `json:"created_at"`
	}
	var devices []deviceRow
	h.db.Model(&model.RefreshToken{}).Where("user_id = ?", uid).
		Order("id DESC").Limit(5).Scan(&devices)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"user": u, "projects": projects,
		"stats": gin.H{
			"total_words": totalWords, "active_days": activeDays,
			"ai_calls_30d": aiCalls, "ai_tokens_30d": aiTokens,
		},
		"devices": devices,
	}})
}

// UpdateUser 编辑用户资料（昵称/手机/邮箱/套餐/状态，按提交的字段部分更新）
func (h *Handler) UpdateUser(c *gin.Context) {
	var req struct {
		Nickname *string `json:"nickname"`
		Phone    *string `json:"phone"`
		Email    *string `json:"email"`
		Plan     *string `json:"plan"`
		Status   *int    `json:"status"`
		IsAdmin  *bool   `json:"is_admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	updates := map[string]interface{}{}
	if req.Nickname != nil {
		if len([]rune(*req.Nickname)) > 50 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "昵称过长"})
			return
		}
		updates["nickname"] = strings.TrimSpace(*req.Nickname)
	}
	if req.Phone != nil {
		updates["phone"] = strings.TrimSpace(*req.Phone)
	}
	if req.Email != nil {
		e := strings.TrimSpace(*req.Email)
		if !strings.Contains(e, "@") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式不正确"})
			return
		}
		// 邮箱唯一性
		var cnt int64
		h.db.Model(&model.User{}).Where("email = ? AND user_id != ?", e, c.Param("user_id")).Count(&cnt)
		if cnt > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被其它账号使用"})
			return
		}
		updates["email"] = e
	}
	if req.Plan != nil {
		if *req.Plan != "free" && *req.Plan != "vip" && *req.Plan != "svip" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "plan 取值非法"})
			return
		}
		updates["plan"] = *req.Plan
	}
	if req.Status != nil {
		if *req.Status != 0 && *req.Status != 1 && *req.Status != -1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status 取值非法"})
			return
		}
		updates["status"] = *req.Status
	}
	if req.IsAdmin != nil {
		updates["is_admin"] = *req.IsAdmin
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有要更新的字段"})
		return
	}
	res := h.db.Model(&model.User{}).Where("user_id = ?", c.Param("user_id")).Updates(updates)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

func (h *Handler) SetUserStatus(c *gin.Context) {
	var req struct {
		Status int `json:"status"` // 1=正常 0=禁用 -1=注销（软删）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	res := h.db.Model(&model.User{}).Where("user_id = ?", c.Param("user_id")).
		Update("status", req.Status)
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

func (h *Handler) SetUserPlan(c *gin.Context) {
	var req struct {
		Plan string `json:"plan"` // free / vip / svip
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if req.Plan != "free" && req.Plan != "vip" && req.Plan != "svip" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan 取值非法"})
		return
	}
	res := h.db.Model(&model.User{}).Where("user_id = ?", c.Param("user_id")).
		Update("plan", req.Plan)
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

// ResetUserPassword 管理员直接设置新密码（bcrypt 后写 users + user_auth_identities 两表）
func (h *Handler) ResetUserPassword(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码至少 6 位"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加密失败"})
		return
	}
	uid := c.Param("user_id")
	res := h.db.Model(&model.User{}).Where("user_id = ?", uid).Update("password_hash", string(hash))
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	h.db.Model(&model.AuthIdentity{}).Where("user_id = ? AND provider = 'email'", uid).
		Update("credential_hash", string(hash))
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

// ---------- 内容监管 ----------

func (h *Handler) ListProjects(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	const size = 20

	tx := h.db.Model(&model.Project{})
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("name LIKE ? OR user_id IN (SELECT user_id FROM users WHERE email LIKE ?)", like, like)
	}
	var total int64
	tx.Count(&total)

	var projects []model.Project
	tx.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&projects)

	// 附作者邮箱
	uidSet := map[string]bool{}
	for _, p := range projects {
		uidSet[p.UserID] = true
	}
	emails := map[string]string{}
	if len(uidSet) > 0 {
		var us []model.User
		h.db.Where("user_id IN ?", keysOf(uidSet)).Find(&us)
		for _, u := range us {
			emails[u.UserID] = u.Email
		}
	}
	type res struct {
		model.Project
		OwnerEmail string `json:"owner_email"`
	}
	out := make([]res, 0, len(projects))
	for _, p := range projects {
		out = append(out, res{Project: p, OwnerEmail: emails[p.UserID]})
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"total": total, "page": page, "list": out}})
}

func (h *Handler) ListProjectDocuments(c *gin.Context) {
	id := c.Param("id")
	var docs []model.Document
	h.db.Where("project_id = ?", id).Order("sort_order").Find(&docs)
	type res struct {
		ID    uint   `json:"id"`
		Title string `json:"title"`
		Type  string `json:"type"`
		Words int    `json:"words"`
	}
	out := make([]res, 0, len(docs))
	for _, d := range docs {
		out = append(out, res{ID: d.ID, Title: d.Title, Type: d.Type, Words: len([]rune(d.SearchText))})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (h *Handler) GetDocumentContent(c *gin.Context) {
	var d model.Document
	if err := h.db.First(&d, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文档不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id": d.ID, "title": d.Title, "type": d.Type,
		"content": d.SearchText, // 正文权威源在文件；监管读 search_text 缓存即可
	}})
}

// SetProjectStatus 下架/恢复书籍（0=下架 1=正常）
func (h *Handler) SetProjectStatus(c *gin.Context) {
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	res := h.db.Model(&model.Project{}).Where("id = ?", c.Param("id")).Update("status", req.Status)
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "书籍不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

// ---------- 公告 ----------

func (h *Handler) ListAnnouncements(c *gin.Context) {
	var list []Announcement
	h.db.Where("status = 1").Order("id DESC").Limit(50).Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) PublicListAnnouncements(c *gin.Context) {
	var list []Announcement
	h.db.Where("status = 1").Order("id DESC").Limit(10).Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) CreateAnnouncement(c *gin.Context) {
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标题不能为空"})
		return
	}
	a := Announcement{Title: req.Title, Content: req.Content, Status: 1}
	if err := h.db.Create(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": a})
}

func (h *Handler) DeleteAnnouncement(c *gin.Context) {
	h.db.Model(&Announcement{}).Where("id = ?", c.Param("id")).Update("status", 0)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

// ---------- 反馈 ----------

func (h *Handler) SubmitFeedback(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
		Contact string `json:"contact"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "反馈内容不能为空"})
		return
	}
	userID, _ := c.Get("userID")
	uid, _ := userID.(string)
	f := Feedback{UserID: uid, Content: req.Content, Contact: req.Contact}
	if err := h.db.Create(&f).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

func (h *Handler) ListFeedbacks(c *gin.Context) {
	onlyOpen := c.Query("open") == "1"
	tx := h.db.Model(&Feedback{})
	if onlyOpen {
		tx = tx.Where("resolved = 0")
	}
	var total int64
	tx.Count(&total)
	var list []Feedback
	tx.Order("id DESC").Limit(100).Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"total": total, "list": list}})
}

func (h *Handler) ResolveFeedback(c *gin.Context) {
	h.db.Model(&Feedback{}).Where("id = ?", c.Param("id")).Update("resolved", 1)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

// ---------- helpers ----------

func todayStart() time.Time {
	n := time.Now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
