<template>
  <div class="context-panel" :class="{ collapsed: !open }">
    <div
      class="context-panel-toggle"
      @click="$emit('update:open', !open)"
      :title="open ? '收起' : '展开'"
    >
      <svg v-if="open" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="9 18 15 12 9 6"/></svg>
      <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="15 18 9 12 15 6"/></svg>
    </div>

    <div class="context-panel-inner" v-show="open">
      <div class="context-header">
        <h4>AI 参考设定</h4>
      </div>

      <div class="context-body">
        <div v-if="loading" class="context-empty">
          <div class="ctx-icon">⏳</div>
          <div>加载中...</div>
        </div>

        <div v-else-if="settingDocs.length === 0" class="context-empty">
          <div class="ctx-icon">💡</div>
          <div>暂无设定文档<br><span class="hint">在「设定」tab 中创建大纲/人物/世界观后，这里可以选择哪些参与 AI</span></div>
        </div>

        <template v-else>
          <!-- 按类型分组 -->
          <div v-for="group in groupedDocs" :key="group.type" class="setting-group">
            <div class="group-header">
              <span class="group-label">{{ group.label }}</span>
              <span class="group-count">{{ group.enabled }}/{{ group.docs.length }}</span>
            </div>
            <div
              v-for="doc in group.docs"
              :key="doc.id"
              class="setting-row"
              :class="{ disabled: !doc.ai_include }"
              @click="toggleDoc(doc)"
            >
              <div class="row-toggle" :class="{ on: doc.ai_include }">
                <span class="toggle-dot"></span>
              </div>
              <span class="row-title">{{ doc.title }}</span>
              <span v-if="doc.used !== undefined" class="row-tokens" :class="{ unused: !doc.used }">
                {{ doc.used ? doc.tokens + 't' : '未用' }}
              </span>
            </div>
          </div>

          <div v-if="usedSettings.length > 0" class="context-stats">
            本次参考 {{ usedCount }}/{{ usedSettings.length }} 张 · {{ totalTokens }} tokens
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { documentApi } from '../../core/api'
import { toast } from '../../composables/useToast'

export interface UsedSetting {
  id: number
  title: string
  type: 'outline' | 'characters' | 'worldbuilding'
  used: boolean
  tokens: number
}

interface SettingDocItem {
  id: number
  title: string
  type: string
  ai_include: boolean | null
  used?: boolean
  tokens?: number
}

const props = defineProps<{
  open: boolean
  projectId: number
  usedSettings: UsedSetting[]
}>()

defineEmits<{
  'update:open': [v: boolean]
  jump: [docId: number]
}>()

const loading = ref(false)
const settingDocs = ref<SettingDocItem[]>([])

const GROUP_LABELS: Record<string, string> = {
  outline: '大纲',
  characters: '人物',
  worldbuilding: '世界观',
  storyline: '故事主线',
  chapter_outline: '章节大纲',
}

const groupedDocs = computed(() => {
  const groups: { type: string; label: string; docs: SettingDocItem[]; enabled: number }[] = []
  for (const type of ['outline', 'storyline', 'chapter_outline', 'characters', 'worldbuilding']) {
    const docs = settingDocs.value.filter(d => d.type === type)
    if (docs.length === 0) continue
    groups.push({
      type,
      label: GROUP_LABELS[type] || type,
      docs,
      enabled: docs.filter(d => d.ai_include !== false).length,
    })
  }
  // 其他类型
  const others = settingDocs.value.filter(d => !['outline', 'storyline', 'chapter_outline', 'characters', 'worldbuilding'].includes(d.type))
  if (others.length > 0) {
    groups.push({ type: 'other', label: '其他', docs: others, enabled: others.filter(d => d.ai_include !== false).length })
  }
  return groups
})

const usedCount = computed(() => props.usedSettings.filter(s => s.used).length)
const totalTokens = computed(() => props.usedSettings.reduce((sum, s) => sum + (s.tokens || 0), 0))

// 合并 usedSettings 到 settingDocs 显示
watch(() => props.usedSettings, (used) => {
  if (used.length === 0) return
  const map = new Map(used.map(s => [s.id, s]))
  settingDocs.value = settingDocs.value.map(d => {
    const u = map.get(d.id)
    return u ? { ...d, used: u.used, tokens: u.tokens } : d
  })
}, { deep: true })

async function loadSettings() {
  if (!props.projectId) return
  loading.value = true
  try {
    const res = await documentApi.list(String(props.projectId))
    const data = res.data?.data
    const allDocs = Array.isArray(data) ? data : []
    // ponytail: note docs grouped by parent's type instead of generic "其他"
    const typeById = new Map<number, string>()
    allDocs.forEach((d: any) => { if (d.id) typeById.set(d.id, d.type) })
    // ponytail: root folder hidden when it has child notes
    const SETTING_TYPES = ['outline', 'characters', 'worldbuilding', 'storyline', 'chapter_outline']
    const hasChildNotes = new Set<number>()
    allDocs.forEach((d: any) => { if (d.type === 'note' && d.parent_id) hasChildNotes.add(d.parent_id) })
    settingDocs.value = allDocs
      .filter((d: any) => [...SETTING_TYPES, 'note'].includes(d.type))
      .filter((d: any) => !(SETTING_TYPES.includes(d.type) && hasChildNotes.has(d.id)))
      .map((d: any) => {
        let groupType = d.type
        if (d.type === 'note' && d.parent_id) {
          const pt = typeById.get(d.parent_id)
          if (pt && SETTING_TYPES.includes(pt)) groupType = pt
        }
        return { id: d.id, title: d.title, type: groupType, ai_include: d.ai_include }
      })
  } catch {
    toast.error('加载设定列表失败')
  } finally {
    loading.value = false
  }
}

async function toggleDoc(doc: SettingDocItem) {
  const newVal = doc.ai_include === false ? true : false
  doc.ai_include = newVal
  try {
    await documentApi.update(doc.id, { ai_include: newVal })
  } catch {
    doc.ai_include = !newVal
    toast.error('更新失败')
  }
}

// ponytail: load on mount, not on first open — panel is collapsed by default
watch(() => props.projectId, () => loadSettings(), { immediate: true })
</script>

<style scoped>
.context-panel {
  position: absolute;
  top: 12px;
  right: 12px;
  bottom: 12px;
  width: 280px;
  background: var(--bg-secondary, #ffffff);
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.06), 0 0 0 1px rgba(0, 0, 0, 0.04);
  z-index: 300;
  display: flex;
  flex-direction: row;
  overflow: hidden;
  transition: width 0.2s ease;
  font-family: 'SF Pro Display', 'PingFang SC', -apple-system, sans-serif;
}
.context-panel.collapsed {
  width: 24px;
  background: transparent;
  box-shadow: none;
}
.context-panel-toggle {
  width: 24px;
  min-width: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--text-secondary, #86868b);
  background: var(--bg-tertiary, #f0f0f0);
  border-radius: 6px 0 0 6px;
  transition: background 0.15s;
}
.context-panel.collapsed .context-panel-toggle {
  background: var(--bg-secondary, #ffffff);
  border-radius: 6px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1), 0 0 0 1px rgba(0, 0, 0, 0.05);
}
.context-panel-toggle:hover { background: rgba(0, 0, 0, 0.08); color: var(--text-primary, #1d1d1f); }

.context-panel-inner {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.context-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px 10px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
}
.context-header h4 {
  margin: 0;
  font-size: 12px;
  color: var(--text-tertiary, #aeaeb2);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 600;
}

.context-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px 12px;
}
.context-empty {
  padding: 30px 12px;
  text-align: center;
  color: var(--text-tertiary, #aeaeb2);
  font-size: 12px;
  line-height: 1.6;
}
.context-empty .ctx-icon { font-size: 24px; margin-bottom: 8px; opacity: 0.6; }
.context-empty .hint { color: var(--text-tertiary, #aeaeb2); opacity: 0.8; font-size: 11px; }

.setting-group { margin-bottom: 12px; }
.group-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 4px 6px;
}
.group-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary, #86868b);
}
.group-count {
  font-size: 10px;
  color: var(--text-tertiary, #aeaeb2);
  font-variant-numeric: tabular-nums;
}

.setting-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 8px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.12s;
  margin-bottom: 2px;
}
.setting-row:hover { background: rgba(0, 0, 0, 0.04); }
.setting-row.disabled { opacity: 0.5; }

.row-toggle {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 1.5px solid var(--text-tertiary, #c7c7cc);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.15s;
}
.row-toggle.on {
  border-color: var(--accent, #007aff);
  background: var(--accent, #007aff);
}
.toggle-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: white;
  opacity: 0;
  transition: opacity 0.15s;
}
.row-toggle.on .toggle-dot { opacity: 1; }

.row-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--text-primary, #1d1d1f);
}

.row-tokens {
  font-size: 10px;
  color: var(--text-tertiary, #aeaeb2);
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}
.row-tokens.unused { color: var(--text-tertiary, #c7c7cc); }

.context-stats {
  font-size: 11px;
  color: var(--text-tertiary, #aeaeb2);
  padding: 8px 4px 4px;
  border-top: 1px dashed rgba(0, 0, 0, 0.05);
  margin-top: 4px;
}
</style>
