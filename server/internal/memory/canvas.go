package memory

import (
	"encoding/json"
	"errors"
	"os"

	"xiezuojun-server/internal/store"
)

// 画布布局：前后端约定的 JSON 文档（卡片坐标/分区/视图/钉选），schema 由前端演进，
// 后端只做透明存储 + 大小防护，跟 RELATIONSHIPS.json 一样落在作品 memory 目录。
const canvasMaxBytes = 2 << 20 // 2 MiB

func (s *MemoryStorage) canvasPath(uid, pid string) string {
	return s.memoryPathFor(uid, pid, "CANVAS.json")
}

// memoryPathFor: 组装 memory 目录下的文件路径（与 relationshipsPath 同规则）
func (s *MemoryStorage) memoryPathFor(uid, pid, name string) string {
	return s.memoryPath(uid, pid) + string(os.PathSeparator) + name
}

// ReadCanvas 返回原始 JSON（不存在时返回空对象）；调用方（handler）直接透传给前端。
func (s *MemoryStorage) ReadCanvas(uid, pid string) (json.RawMessage, error) {
	b, err := os.ReadFile(s.canvasPath(uid, pid))
	if err != nil {
		if os.IsNotExist(err) {
			return json.RawMessage(`{"cards":[],"zones":[],"view":{"x":0,"y":0,"scale":1}}`), nil
		}
		return nil, err
	}
	return json.RawMessage(b), nil
}

// WriteCanvas 校验 JSON 合法性与大小后原子写。
func (s *MemoryStorage) WriteCanvas(uid, pid string, data json.RawMessage) error {
	if len(data) > canvasMaxBytes {
		return errors.New("画布数据超过 2MiB 限制")
	}
	if !json.Valid(data) {
		return errors.New("画布数据不是合法 JSON")
	}
	return store.AtomicWrite(s.canvasPath(uid, pid), data, 0644)
}
