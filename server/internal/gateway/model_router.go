// ModelRouter — 模型路由与选型

package gateway

import "fmt"

// ModelRouter 模型路由器
type ModelRouter struct {
	models []ModelConfig
}

// NewModelRouter 创建路由器
func NewModelRouter() *ModelRouter {
	return &ModelRouter{
		models: DefaultModels,
	}
}

// Route 根据请求选择最合适的模型
func (r *ModelRouter) Route(req RouteRequest) RouteResult {
	// 1. 筛选可用模型
	candidates := make([]ModelConfig, 0)
	for _, m := range r.models {
		// 检查上下文窗口是否足够
		if m.Capability.ContextWindow < req.ContextTokens {
			continue
		}
		// 检查是否支持 JSON（如果需要）
		if req.RequireJSON && !m.Capability.SupportsJSON {
			continue
		}
		candidates = append(candidates, m)
	}

	if len(candidates) == 0 {
		// 没有可用模型，返回默认模型（带警告）
		return RouteResult{
			Model:  r.models[0],
			Reason: "no model fits constraints, falling back to default",
		}
	}

	// 2. 按优先级排序（已在 DefaultModels 中排序）
	// 3. 选择第一个可用模型
	selected := candidates[0]

	// 4. 计算预估成本
	estimatedTokens := req.ContextTokens + 1000 // 粗略估算输出 1000 tokens
	cost := float64(estimatedTokens) / 1000 * selected.CostPer1K

	return RouteResult{
		Model:         selected,
		Reason:        fmt.Sprintf("context=%dtokens, mode=%s, tier=%s", req.ContextTokens, req.AIMode, req.UserTier),
		EstimatedCost: cost,
	}
}

// RegisterModel 注册新模型
func (r *ModelRouter) RegisterModel(model ModelConfig) {
	r.models = append(r.models, model)
}

// GetModels 获取所有模型配置
func (r *ModelRouter) GetModels() []ModelConfig {
	return r.models
}
