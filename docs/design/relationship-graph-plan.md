# 人物关系图谱 → 写作菌集成方案

## 位置
MemoryPanel.vue 第四个 tab「图谱」，跟「角色/摘要/背景」平级。
角色 tab 已管角色卡，图谱是同一批数据的可视化视图，认知零跳转。

## 数据结构
一个 JSON 文件：`memory/{uid}/{pid}/RELATIONSHIPS.json`
```json
{
  "nodes": [
    { "name": "陆云起", "role": "主角", "color": "#4cc9f0" }
  ],
  "links": [
    { "source": "陆云起", "target": "沈清霜", "type": "love", "label": "两情相悦" }
  ],
  "types": [
    { "key": "friend", "label": "朋友", "color": "#4cc9f0", "dash": [] }
  ]
}
```
- 节点 name 对齐 ListCharacters 返回的角色名，不重复存角色卡内容
- types 存自定义关系类型（5个预设+用户添加的）

## 后端（~80行 Go）

### 新文件 `internal/memory/relationships.go`
```go
// 存储层：读写 RELATIONSHIPS.json
func (s *MemoryStorage) ReadRelationships(uid, pid string) (*RelationshipData, error)
func (s *MemoryStorage) WriteRelationships(uid, pid string, data *RelationshipData) error
```
结构体：
```go
type RelationshipNode struct { Name string `json:"name"`; Role string `json:"role"`; Color string `json:"color"` }
type RelationshipLink struct { Source string `json:"source"`; Target string `json:"target"`; Type string `json:"type"`; Label string `json:"label"` }
type RelType struct { Key string `json:"key"`; Label string `json:"label"`; Color string `json:"color"`; Dash []int `json:"dash"` }
type RelationshipData struct { Nodes []RelationshipNode `json:"nodes"`; Links []RelationshipLink `json:"links"`; Types []RelType `json:"types"` }
```

### 新文件 `internal/memory/handler_relationships.go`
```go
// GET    /memory/relationships?project_id=xxx     → 读全量
// PUT    /memory/relationships?project_id=xxx     → 存全量（前端整体提交）
```
不用细粒度 CRUD，前端改完一次 PUT 全量。JSON 文件场景下全量写最简单可靠。

### `cmd/server/main.go` 加2行路由
```go
authd.GET("/memory/relationships", memHandler.GetRelationships)
authd.PUT("/memory/relationships", memHandler.PutRelationships)
```

## 前端

### 1. `src/core/api.ts` — memoryApi 加2个方法
```ts
getRelationships(projectId: string) {
  return api.get('/memory/relationships', { params: { project_id: projectId } })
},
saveRelationships(projectId: string, data: RelationshipData) {
  return api.put('/memory/relationships', data, { params: { project_id: projectId } })
},
```

### 2. `src/components/editor/RelationshipGraph.vue` — 新组件
从原型 `character-graph-55.html` 搬 D3 逻辑，改：
- 数据源：`props.projectId` → `onMounted` 调 `memoryApi.getRelationships`
- 角色名同步：`memoryApi.listCharacters` 拿到角色名列表，跟 nodes 对齐（新角色自动加入 nodes）
- 保存：`watch(nodes/links/types deep)` → debounce 500ms → `memoryApi.saveRelationships`
- 样式：适配 MemoryPanel 宽度（右侧面板 ~300px），图谱区域比原型小，节点 r=10 文字 10px
- d3.v7：`npm install d3` + `npm install -D @types/d3`（检查是否已装）

### 3. `MemoryPanel.vue` — 加 tab
```vue
<button class="mem-tab" :class="{ active: memTab === 'graph' }" @click="memTab = 'graph'">图谱</button>

<template v-if="memTab === 'graph'">
  <RelationshipGraph :project-id="projectId" />
</template>
```
memTab 类型加 `'graph'`。

## 执行步骤
1. 后端：建 relationships.go + handler_relationships.go + main.go 加路由
2. 前端：api.ts 加方法 → 建 RelationshipGraph.vue → MemoryPanel 加 tab
3. 部署 180 → 验证：打开编辑器 → 右侧面板点「图谱」→ 看到图 → 加角色/关系 → 刷新还在

## 不做
- 不加 DB 表（文件存储够用，跟角色卡/记忆文件一致）
- 不加独立页面/路由（MemoryPanel tab 足够）
- 不做关系数据的增量 API（全量 PUT 最简单，JSON 文件不存在并发写问题）
- 不做 AI 自动提取关系（后面再说，先手动画）
