<template>
  <div class="rel-graph">
    <div class="graph-toolbar">
      <button class="g-btn" @click="addNode">+ 角色</button>
      <button class="g-btn" @click="addLink">+ 关系</button>
      <button class="g-btn" @click="addType">+ 类型</button>
      <button class="g-btn" @click="exportJson">导出</button>
    </div>

    <div class="graph-stats" v-if="nodes.length">
      {{ nodes.length }} 角色 · {{ links.length }} 关系
    </div>

    <div ref="svgContainer" class="graph-canvas"></div>

    <!-- 图例 -->
    <div class="graph-legend" v-if="relTypes.length">
      <div
        v-for="t in relTypes"
        :key="t.key"
        class="legend-item"
        :class="{ active: typeFilter === t.key }"
        @click="toggleTypeFilter(t.key)"
      >
        <span class="legend-line" :style="legendStyle(t)"></span>
        {{ t.label }}
        <span v-if="!PRESET_TYPES.has(t.key)" class="legend-del" @click.stop="deleteType(t.key)">✕</span>
      </div>
    </div>

    <!-- 添加角色弹窗 -->
    <Teleport to="body">
    <div v-if="showAddNode" class="modal-overlay" @click="showAddNode = false">
      <div class="modal" @click.stop>
        <h3>添加角色</h3>
        <label>角色名</label>
        <input v-model="newNodeName" placeholder="如：张三" @keyup.enter="confirmAddNode">
        <label>身份</label>
        <input v-model="newNodeRole" placeholder="如：主角">
        <label>颜色</label>
        <div class="color-grid">
          <button
            v-for="c in COLORS"
            :key="c"
            class="color-swatch"
            :class="{ selected: newNodeColor === c }"
            :style="{ background: c }"
            @click="newNodeColor = c"
          ></button>
        </div>
        <div class="modal-actions">
          <button class="btn-cancel" @click="showAddNode = false">取消</button>
          <button class="btn-ok" @click="confirmAddNode">确定</button>
        </div>
      </div>
    </div>
    </Teleport>

    <!-- 添加关系弹窗 -->
    <Teleport to="body">
    <div v-if="showAddLink" class="modal-overlay" @click="showAddLink = false">
      <div class="modal modal-link" @click.stop>
        <h3>添加关系</h3>
        <div class="link-roles">
          <div class="role-col">
            <div class="role-col-head">源角色<span v-if="newLinkSource" class="picked">{{ newLinkSource }}</span></div>
            <input v-model="linkSearch1" placeholder="搜索角色..." class="search-input">
            <div class="role-list">
              <div v-for="n in filteredNodes1" :key="n.name" class="role-item" :class="{ active: newLinkSource === n.name }" @click="newLinkSource = n.name">
                <span class="role-dot" :style="{ background: n.color }"></span>
                <span class="role-name">{{ n.name }}</span>
                <span class="role-tag">{{ n.role }}</span>
              </div>
              <div v-if="!filteredNodes1.length" class="role-empty">无匹配</div>
            </div>
          </div>
          <div class="role-arrow">→</div>
          <div class="role-col">
            <div class="role-col-head">目标角色<span v-if="newLinkTarget" class="picked">{{ newLinkTarget }}</span></div>
            <input v-model="linkSearch2" placeholder="搜索角色..." class="search-input">
            <div class="role-list">
              <div v-for="n in filteredNodes2" :key="n.name" class="role-item" :class="{ active: newLinkTarget === n.name }" @click="newLinkTarget = n.name">
                <span class="role-dot" :style="{ background: n.color }"></span>
                <span class="role-name">{{ n.name }}</span>
                <span class="role-tag">{{ n.role }}</span>
              </div>
              <div v-if="!filteredNodes2.length" class="role-empty">无匹配</div>
            </div>
          </div>
        </div>
        <label>关系类型</label>
        <div class="type-buttons">
          <button
            v-for="t in relTypes"
            :key="t.key"
            class="type-btn"
            :class="{ selected: newLinkType === t.key }"
            :style="{ borderColor: t.color, background: newLinkType === t.key ? t.color : 'transparent', color: newLinkType === t.key ? '#fff' : t.color }"
            @click="newLinkType = t.key"
          >{{ t.label }}</button>
        </div>
        <label>关系描述</label>
        <input v-model="newLinkLabel" placeholder="如：两情相悦" class="link-desc-input">
        <div class="modal-actions">
          <button class="btn-cancel" @click="showAddLink = false">取消</button>
          <button class="btn-ok" @click="confirmAddLink">确定</button>
        </div>
      </div>
    </div>
    </Teleport>

    <!-- 添加类型弹窗 -->
    <Teleport to="body">
    <div v-if="showAddType" class="modal-overlay" @click="showAddType = false">
      <div class="modal" @click.stop>
        <h3>添加关系类型</h3>
        <label>类型名称</label>
        <input v-model="newTypeLabel" placeholder="如：宿敌">
        <label>颜色</label>
        <div class="color-grid">
          <button
            v-for="c in COLORS"
            :key="c"
            class="color-swatch"
            :class="{ selected: newTypeColor === c }"
            :style="{ background: c }"
            @click="newTypeColor = c"
          ></button>
        </div>
        <label>线型</label>
        <div class="dash-buttons">
          <button class="type-btn" :class="{ selected: newTypeDash.length === 0 }" @click="newTypeDash = []">实线</button>
          <button class="type-btn" :class="{ selected: newTypeDash.length === 2 && newTypeDash[0] === 6 }" @click="newTypeDash = [6, 4]">虚线</button>
          <button class="type-btn" :class="{ selected: newTypeDash.length === 2 && newTypeDash[0] === 2 }" @click="newTypeDash = [2, 2]">点线</button>
        </div>
        <div class="modal-actions">
          <button class="btn-cancel" @click="showAddType = false">取消</button>
          <button class="btn-ok" @click="confirmAddType">确定</button>
        </div>
      </div>
    </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed, shallowRef } from 'vue'
import { memoryApi } from '../../core/api'
import { toast } from '../../composables/useToast'

const props = defineProps<{ projectId: string }>()

const COLORS = ['#4cc9f0', '#ff4d6d', '#ffd166', '#06d6a0', '#b388ff', '#ff6b35', '#f0e68c', '#118ab2', '#ef476f', '#8338ec']
const PRESET_TYPES = new Set(['friend', 'love', 'enemy', 'family', 'master'])

interface RNode { name: string; role: string; color: string }
interface RLink { source: string; target: string; type: string; label: string }
interface RType { key: string; label: string; color: string; dash: number[] }

const nodes = ref<RNode[]>([])
const links = ref<RLink[]>([])
const relTypes = ref<RType[]>([
  { key: 'friend', label: '朋友', color: '#4cc9f0', dash: [] },
  { key: 'love', label: '恋人', color: '#ff4d6d', dash: [] },
  { key: 'enemy', label: '敌人', color: '#ff6b35', dash: [6, 4] },
  { key: 'family', label: '亲属', color: '#f0e68c', dash: [] },
  { key: 'master', label: '师徒', color: '#b388ff', dash: [2, 2] },
])

const svgContainer = ref<HTMLElement | null>(null)
const d3 = shallowRef<any>(null)
let sim: any = null
let svgSel: any = null
let gEl: any = null
let linkGroup: any = null
let nodeGroup: any = null
let zoomBehavior: any = null
let saveTimer: any = null

// UI state
const showAddNode = ref(false)
const showAddLink = ref(false)
const showAddType = ref(false)
const newNodeName = ref('')
const newNodeRole = ref('')
const newNodeColor = ref(COLORS[0])
const newLinkSource = ref('')
const newLinkTarget = ref('')
const newLinkType = ref('friend')
const newLinkLabel = ref('')
const newTypeLabel = ref('')
const newTypeColor = ref(COLORS[0])
const newTypeDash = ref<number[]>([])
const linkSearch1 = ref('')
const linkSearch2 = ref('')
const typeFilter = ref('')
const selected = ref<{ type: string; data: any } | null>(null)
const connectedNames = ref<Set<string>>(new Set())

const filteredNodes1 = computed(() => {
  const q = linkSearch1.value
  return q ? nodes.value.filter(n => n.name.includes(q)) : nodes.value
})
const filteredNodes2 = computed(() => {
  const q = linkSearch2.value
  return q ? nodes.value.filter(n => n.name.includes(q)) : nodes.value
})

function legendStyle(t: RType) {
  const base: any = { background: t.color }
  if (t.dash.length) base.borderTop = `2px dashed ${t.color}`
  return base
}

// --- D3 dynamic load ---
async function loadD3() {
  if ((window as any).d3) return (window as any).d3
  await new Promise<void>((resolve, reject) => {
    const s = document.createElement('script')
    s.src = 'https://d3js.org/d3.v7.min.js'
    s.onload = () => resolve()
    s.onerror = () => reject(new Error('Failed to load D3'))
    document.head.appendChild(s)
  })
  return (window as any).d3
}

function initGraph() {
  if (!d3.value || !svgContainer.value) return
  const d3lib = d3.value
  const W = svgContainer.value.clientWidth || 280
  const H = svgContainer.value.clientHeight || 320

  svgSel = d3lib.select(svgContainer.value).append('svg')
    .attr('width', '100%').attr('height', '100%')
    .attr('viewBox', `0 0 ${W} ${H}`)
  svgSel.append('defs')

  // zoom
  zoomBehavior = d3lib.zoom().scaleExtent([0.3, 4]).on('zoom', (e: any) => gEl.attr('transform', e.transform))
  svgSel.call(zoomBehavior)

  gEl = svgSel.append('g')
  linkGroup = gEl.append('g').attr('class', 'links')
  nodeGroup = gEl.append('g').attr('class', 'nodes')

  sim = d3lib.forceSimulation([])
    .force('link', d3lib.forceLink([]).id((d: any) => d.name).distance(70).strength(0.15))
    .force('charge', d3lib.forceManyBody().strength(-350))
    .force('center', d3lib.forceCenter(W / 2, H / 2))
    .force('collide', d3lib.forceCollide(28))
    .force('x', d3lib.forceX(W / 2).strength(0.04))
    .force('y', d3lib.forceY(H / 2).strength(0.04))
    .on('tick', ticked)
}

function updateMarkers() {
  const defs = svgSel.select('defs')
  defs.selectAll('marker').remove()
  relTypes.value.forEach(t => {
    const m = defs.append('marker').attr('id', `arrow-${t.key}`).attr('viewBox', '0 0 10 10')
      .attr('refX', 24).attr('refY', 5).attr('markerWidth', 6).attr('markerHeight', 6).attr('orient', 'auto')
    m.append('path').attr('d', 'M0,0 L10,5 L0,10 Z').attr('fill', t.color)
  })
}

function getTypeMeta(key: string): RType | undefined {
  return relTypes.value.find(t => t.key === key)
}

function linkName(v: any): string { return typeof v === 'string' ? v : v?.name || '' }
function isRelated(a: any, b: any): boolean {
  return links.value.some(l =>
    (linkName(l.source) === a.name && linkName(l.target) === b.name) ||
    (linkName(l.source) === b.name && linkName(l.target) === a.name)
  )
}
function render() {
  if (!d3.value || !sim) return

  updateMarkers()

  // links
  const linkSel = linkGroup.selectAll('g').data(links.value, (d: any) =>
    `${d.source?.name || d.source}-${d.target?.name || d.target}`)

  const linkEnter = linkSel.enter().append('g')
  linkEnter.append('line')
  linkEnter.append('text').attr('class', 'link-label')

  linkSel.exit().remove()

  const allLinks = linkEnter.merge(linkSel)
  allLinks.select('line')
    .attr('stroke', (d: any) => getTypeMeta(d.type)?.color || '#555')
    .attr('stroke-width', 2)
    .attr('stroke-dasharray', (d: any) => {
      const t = getTypeMeta(d.type)
      return t && t.dash.length ? t.dash.join(' ') : null
    })
    .attr('marker-end', (d: any) => `url(#arrow-${d.type})`)
    .style('opacity', (d: any) => {
      if (typeFilter.value) return d.type === typeFilter.value ? 1 : 0.04
      if (selected.value?.type === 'node') {
        const sn = selected.value.data.name
        return (d.source?.name === sn || d.target?.name === sn) ? 1 : 0.05
      }
      return 0.7
    })

  allLinks.select('.link-label')
    .text((d: any) => d.label || '')
    .attr('font-size', 8)
    .attr('fill', '#888')
    .attr('text-anchor', 'middle')
    .attr('pointer-events', 'none')
    .style('opacity', (d: any) => {
      if (typeFilter.value) return d.type === typeFilter.value ? 1 : 0.04
      if (selected.value?.type === 'node') {
        const sn = selected.value.data.name
        return (d.source?.name === sn || d.target?.name === sn) ? 1 : 0.05
      }
      return 0.7
    })

  // nodes
  const nodeSel = nodeGroup.selectAll('g').data(nodes.value, (d: any) => d.name)

  const nodeEnter = nodeSel.enter().append('g')
    .call(d3.value.drag()
      .on('start', (e: any, d: any) => { if (!e.active) sim.alphaTarget(0.3).restart(); d.fx = d.x; d.fy = d.y })
      .on('drag', (e: any, d: any) => { d.fx = e.x; d.fy = e.y })
      .on('end', (e: any, d: any) => { if (!e.active) sim.alphaTarget(0); d.fx = null; d.fy = null }))
    .on('click', (e: any, d: any) => { e.stopPropagation(); selectNode(d) })

  nodeEnter.append('circle').attr('r', 10)
  nodeEnter.append('text').attr('class', 'node-name')
  nodeEnter.append('text').attr('class', 'node-role')

  nodeSel.exit().remove()

  const allNodes = nodeEnter.merge(nodeSel)
  allNodes.select('circle')
    .attr('fill', (d: any) => d.color)
    .attr('stroke', 'rgba(255,255,255,0.3)')
    .attr('stroke-width', 2)
    .attr('opacity', (d: any) => {
      if (typeFilter.value) return 0.2
      if (!selected.value || selected.value.type !== 'node') return 1
      return (selected.value.data.name === d.name || isRelated(selected.value.data, d)) ? 1 : 0.15
    })

  allNodes.select('.node-name')
    .text((d: any) => d.name)
    .attr('font-size', 9)
    .attr('fill', '#fff')
    .attr('text-anchor', 'middle')
    .attr('dy', -16)
    .attr('pointer-events', 'none')

  allNodes.select('.node-role')
    .text((d: any) => d.role)
    .attr('font-size', 7)
    .attr('fill', '#999')
    .attr('text-anchor', 'middle')
    .attr('dy', 22)
    .attr('pointer-events', 'none')

  // update sim
  sim.nodes(nodes.value)
  sim.force('link').links(links.value)
  sim.alpha(1).restart()

  // click blank to deselect
  svgSel.on('click', () => { selected.value = null; connectedNames.value = new Set(); render() })
}

function ticked() {
  if (!linkGroup || !nodeGroup) return
  linkGroup.selectAll('line')
    .attr('x1', (d: any) => d.source.x).attr('y1', (d: any) => d.source.y)
    .attr('x2', (d: any) => d.target.x).attr('y2', (d: any) => d.target.y)
  linkGroup.selectAll('.link-label')
    .attr('x', (d: any) => (d.source.x + d.target.x) / 2)
    .attr('y', (d: any) => (d.source.y + d.target.y) / 2 - 4)
  nodeGroup.selectAll('g').attr('transform', (d: any) => `translate(${d.x},${d.y})`)
}

function selectNode(d: any) {
  if (selected.value?.data?.name === d.name) {
    selected.value = null
  } else {
    selected.value = { type: 'node', data: d }
  }
  typeFilter.value = ''
  render()
}

function toggleTypeFilter(key: string) {
  typeFilter.value = typeFilter.value === key ? '' : key
  if (typeFilter.value) selected.value = null
  render()
}

// --- Add node ---
function addNode() { showAddNode.value = true; newNodeName.value = ''; newNodeRole.value = ''; newNodeColor.value = COLORS[nodes.value.length % COLORS.length] }
function confirmAddNode() {
  const name = newNodeName.value.trim()
  if (!name) { toast.error('请输入角色名'); return }
  if (nodes.value.some(n => n.name === name)) { toast.error('角色已存在'); return }
  nodes.value.push({ name, role: newNodeRole.value || '角色', color: newNodeColor.value })
  showAddNode.value = false
  render()
  scheduleSave()
}

// --- Add link ---
function addLink() {
  if (nodes.value.length < 2) { toast.error('至少需要2个角色'); return }
  showAddLink.value = true
  newLinkSource.value = nodes.value[0].name
  newLinkTarget.value = nodes.value[1].name
  newLinkType.value = relTypes.value[0].key
  newLinkLabel.value = ''
  linkSearch1.value = ''
  linkSearch2.value = ''
}
function confirmAddLink() {
  if (newLinkSource.value === newLinkTarget.value) { toast.error('不能选同一个角色'); return }
  links.value.push({ source: newLinkSource.value, target: newLinkTarget.value, type: newLinkType.value, label: newLinkLabel.value })
  showAddLink.value = false
  render()
  scheduleSave()
}

// --- Add type ---
function addType() { showAddType.value = true; newTypeLabel.value = ''; newTypeColor.value = COLORS[relTypes.value.length % COLORS.length]; newTypeDash.value = [] }
function confirmAddType() {
  const label = newTypeLabel.value.trim()
  if (!label) { toast.error('请输入类型名称'); return }
  const key = label // ponytail: use label as key, uniqueness by label
  if (relTypes.value.some(t => t.key === key)) { toast.error('类型已存在'); return }
  relTypes.value.push({ key, label, color: newTypeColor.value, dash: [...newTypeDash.value] })
  showAddType.value = false
  render()
  scheduleSave()
}

function deleteType(key: string) {
  if (PRESET_TYPES.has(key)) return
  if (links.value.some(l => l.type === key)) { toast.error('该类型下有关系，无法删除'); return }
  relTypes.value = relTypes.value.filter(t => t.key !== key)
  render()
  scheduleSave()
}

function exportJson() {
  const data = { nodes: nodes.value, links: links.value, types: relTypes.value }
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'relationships.json'
  a.click()
  URL.revokeObjectURL(url)
}

// --- Sync characters from API ---
async function syncCharacters() {
  try {
    const res = await memoryApi.listCharacters(props.projectId)
    const charNames: string[] = res.data?.data || []
    // Add new characters as nodes, keep existing
    for (const name of charNames) {
      if (!nodes.value.some(n => n.name === name)) {
        const idx = nodes.value.length
        nodes.value.push({ name, role: '角色', color: COLORS[idx % COLORS.length] })
      }
    }
    render()
    scheduleSave()
  } catch { /* ignore */ }
}

// --- Save / Load ---
function scheduleSave() {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(async () => {
    try {
      // D3 mutates link source/target to objects; serialize back to names
      const cleanLinks = links.value.map(l => ({
        source: linkName(l.source),
        target: linkName(l.target),
        type: l.type,
        label: l.label,
      }))
      await memoryApi.saveRelationships(props.projectId, {
        nodes: nodes.value,
        links: cleanLinks,
        types: relTypes.value,
      })
    } catch (e: any) {
      toast.error('保存关系数据失败')
    }
  }, 800)
}

async function loadData() {
  try {
    const res = await memoryApi.getRelationships(props.projectId)
    const d = res.data?.data
    if (d) {
      if (Array.isArray(d.nodes)) nodes.value = d.nodes
      if (Array.isArray(d.links)) links.value = d.links
      if (Array.isArray(d.types) && d.types.length) relTypes.value = d.types
    }
  } catch { /* 404 = empty, fine */ }
}

onMounted(async () => {
  await loadData()
  await syncCharacters()
  try {
    d3.value = await loadD3()
    initGraph()
    render()
  } catch (e: any) {
    toast.error('加载 D3 失败')
  }
})

onBeforeUnmount(() => {
  if (sim) sim.stop()
  if (svgSel) svgSel.remove()
  if (saveTimer) clearTimeout(saveTimer)
})
</script>

<style scoped>
.rel-graph {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  position: relative;
}

.graph-toolbar {
  display: flex;
  gap: 4px;
  padding: 6px 8px;
  flex-shrink: 0;
  flex-wrap: wrap;
}
.g-btn {
  padding: 4px 10px;
  font-size: 11px;
  border: 1px solid var(--border, rgba(0,0,0,0.1));
  border-radius: 6px;
  background: var(--bg-secondary, #f5f5f7);
  color: var(--text-primary, #333);
  cursor: pointer;
  transition: all 0.12s;
}
.g-btn:hover { background: var(--accent-light, rgba(0,122,255,0.08)); }

.graph-stats {
  font-size: 11px;
  color: var(--text-tertiary, #999);
  padding: 0 8px 4px;
  flex-shrink: 0;
}

.graph-canvas {
  flex: 1;
  min-height: 0;
  background: #1a1a2e;
  border-radius: 8px;
  margin: 0 8px;
  overflow: hidden;
  cursor: grab;
}
.graph-canvas:active { cursor: grabbing; }

.graph-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 6px 8px;
  flex-shrink: 0;
}
.legend-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  padding: 3px 8px;
  border-radius: 4px;
  cursor: pointer;
  background: rgba(0,0,0,0.04);
  color: var(--text-secondary, #666);
  transition: all 0.12s;
}
.legend-item.active { background: var(--accent-light, rgba(0,122,255,0.12)); font-weight: 500; }
.legend-line {
  display: inline-block;
  width: 16px;
  height: 2px;
  border-radius: 1px;
}
.legend-del {
  font-size: 9px;
  color: #ff3b30;
  padding: 0 2px;
}

/* Modal — unscoped because Teleport moves these to body */
</style>
<style>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.25);
  backdrop-filter: blur(6px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 500;
}
.modal {
  background: var(--bg-secondary, #fff);
  border: 1px solid var(--border, #00000014);
  border-radius: var(--radius-lg, 16px);
  padding: 24px;
  width: 320px;
  max-height: 80vh;
  overflow-y: auto;
  color: var(--text-primary, #1d1d1f);
  box-shadow: var(--shadow-xl, 0 20px 60px #0000001f);
}
.modal h3 { margin: 0 0 16px; font-size: 17px; font-weight: 700; color: var(--text-primary, #1d1d1f); }
.modal label { display: block; margin: 10px 0 5px; font-size: 12px; color: var(--text-secondary, #86868b); font-weight: 500; }
.modal label .picked { float: right; color: var(--accent, #007aff); font-size: 12px; font-weight: 600; }
.modal input, .modal select {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-tertiary, #e8e8ed);
  border: 1px solid var(--border, #00000014);
  border-radius: var(--radius-sm, 8px);
  color: var(--text-primary, #1d1d1f);
  font-size: 14px;
  outline: none;
  transition: border-color var(--transition-fast, .15s);
}
.modal input:focus, .modal select:focus { border-color: var(--accent, #007aff); }
.modal input::placeholder { color: var(--text-tertiary, #aeaeb2); }
.modal select option { color: var(--text-primary, #1d1d1f); background: var(--bg-secondary, #fff); }
.modal .search-input { margin-bottom: 4px !important; }
.modal .role-select {
  width: 100%;
  padding: 4px;
  font-size: 12px;
}

/* Link modal */
.modal-link { width: 520px; }
.link-roles {
  display: flex;
  gap: 0;
  align-items: stretch;
  margin-bottom: 12px;
}
.role-col {
  flex: 1;
  min-width: 0;
}
.role-arrow {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  font-size: 18px;
  color: var(--text-tertiary, #aeaeb2);
  flex-shrink: 0;
}
.role-col-head {
  font-size: 12px;
  color: var(--text-secondary, #86868b);
  margin-bottom: 4px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 500;
}
.role-col-head .picked {
  color: var(--accent, #007aff);
  font-weight: 600;
  font-size: 12px;
}
.role-list {
  max-height: 160px;
  overflow-y: auto;
  border: 1px solid var(--border, #00000014);
  border-radius: var(--radius-sm, 8px);
  background: var(--bg-tertiary, #e8e8ed);
}
.role-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  cursor: pointer;
  transition: background 0.1s;
}
.role-item:hover { background: rgba(0,0,0,0.04); }
.role-item.active { background: var(--accent-light, #007aff14); }
.role-item.active .role-name { color: var(--accent, #007aff); }
.role-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}
.role-name { font-size: 13px; color: var(--text-primary, #1d1d1f); }
.role-tag { font-size: 10px; color: var(--text-tertiary, #aeaeb2); margin-left: auto; }
.role-empty { padding: 16px; text-align: center; font-size: 12px; color: var(--text-tertiary, #aeaeb2); }
.link-desc-input { margin-bottom: 4px; }
.modal .color-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 6px;
}
.modal .color-swatch {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  transition: transform 0.1s;
}
.modal .color-swatch.selected { border-color: var(--accent, #007aff); transform: scale(1.1); }
.modal .type-buttons, .modal .dash-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.modal .type-btn {
  padding: 6px 14px;
  font-size: 13px;
  border: 1px solid var(--border, #00000014);
  border-radius: var(--radius-sm, 8px);
  background: var(--bg-tertiary, #e8e8ed);
  color: var(--text-secondary, #86868b);
  cursor: pointer;
  transition: all 0.12s;
}
.modal .type-btn.selected { color: #fff; border-color: var(--accent, #007aff); }
.modal .modal-actions {
  display: flex;
  gap: 8px;
  margin-top: 16px;
}
.modal .modal-actions button {
  flex: 1;
  padding: 10px;
  border-radius: var(--radius-sm, 8px);
  border: none;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}
.modal .btn-cancel { background: var(--bg-tertiary, #e8e8ed); color: var(--text-secondary, #86868b); }
.modal .btn-ok { background: var(--accent, #007aff); color: #fff; font-weight: 600; }
</style>
