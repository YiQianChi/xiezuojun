<template>
  <div class="rel-graph">
    <div class="graph-toolbar">
      <button class="g-btn" @click="addNode">+ 角色</button>
      <button class="g-btn" @click="addLink">+ 关系</button>
      <button class="g-btn" @click="addType">+ 类型</button>
      <button class="g-btn" @click="importJson">导入</button>
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

    <!-- 添加/编辑角色弹窗 -->
    <div v-if="showAddNode" class="modal-overlay" @click="showAddNode = false">
      <div class="modal" @click.stop>
        <h3>{{ editMode ? '编辑角色' : '添加角色' }}</h3>
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
          <button v-if="editMode" class="btn-del" @click="deleteNode">删除</button>
          <button class="btn-cancel" @click="showAddNode = false">取消</button>
          <button class="btn-ok" @click="confirmAddNode">{{ editMode ? '保存' : '确定' }}</button>
        </div>
      </div>
    </div>

    <!-- 添加关系弹窗 -->
    <div v-if="showAddLink" class="modal-overlay" @click="showAddLink = false">
      <div class="modal" @click.stop>
        <h3>添加关系</h3>
        <label>源角色</label>
        <input v-model="linkSearch1" placeholder="搜索..." class="search-input">
        <select v-model="newLinkSource" size="5" class="role-select">
          <option v-for="n in filteredNodes1" :key="n.name" :value="n.name">{{ n.name }} ({{ n.role }})</option>
        </select>
        <label>目标角色</label>
        <input v-model="linkSearch2" placeholder="搜索..." class="search-input">
        <select v-model="newLinkTarget" size="5" class="role-select">
          <option v-for="n in filteredNodes2" :key="n.name" :value="n.name">{{ n.name }} ({{ n.role }})</option>
        </select>
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
        <input v-model="newLinkLabel" placeholder="如：两情相悦">
        <div class="modal-actions">
          <button class="btn-cancel" @click="showAddLink = false">取消</button>
          <button class="btn-ok" @click="confirmAddLink">确定</button>
        </div>
      </div>
    </div>

    <!-- 关系右键菜单 -->
    <div v-if="showLinkMenu" class="ctx-menu" :style="{ left: ctxPos.x + 'px', top: ctxPos.y + 'px' }" @click.stop>
      <button class="ctx-item" @click="editLinkFromMenu">编辑</button>
      <button class="ctx-item ctx-danger" @click="deleteLinkFromMenu">删除</button>
    </div>

    <!-- 添加类型弹窗 -->
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

    <input ref="fileInput" type="file" accept=".json" style="display:none" @change="onFileSelected">
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
const fileInput = ref<HTMLElement | null>(null)
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
const editMode = ref(false)
const editingNode = ref<RNode | null>(null)
const showLinkMenu = ref(false)
const ctxPos = ref({ x: 0, y: 0 })
let ctxLink: RLink | null = null

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
  linkEnter.on('contextmenu', (e: any, d: any) => openLinkMenu(e, d))

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
    .on('dblclick', (e: any, d: any) => { e.stopPropagation(); editNode(d) })
    .on('contextmenu', (e: any, d: any) => {
      e.preventDefault(); e.stopPropagation()
      ctxPos.value = { x: e.clientX, y: e.clientY }
      editingNode.value = d; editMode.value = true
      showAddNode.value = true
      newNodeName.value = d.name; newNodeRole.value = d.role; newNodeColor.value = d.color
    })

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
  svgSel.on('click', () => { selected.value = null; connectedNames.value = new Set(); showLinkMenu.value = false; render() })
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

// --- Add/edit node ---
function addNode() { showAddNode.value = true; editMode.value = false; newNodeName.value = ''; newNodeRole.value = ''; newNodeColor.value = COLORS[nodes.value.length % COLORS.length] }
function editNode(d: any) {
  showAddNode.value = true; editMode.value = true; editingNode.value = d
  newNodeName.value = d.name; newNodeRole.value = d.role; newNodeColor.value = d.color
}
function confirmAddNode() {
  const name = newNodeName.value.trim()
  if (!name) { toast.error('请输入角色名'); return }
  if (editMode.value && editingNode.value) {
    const oldName = editingNode.value.name
    const n = nodes.value.find(n => n.name === oldName)
    if (n) { n.name = name; n.role = newNodeRole.value || '角色'; n.color = newNodeColor.value }
    if (oldName !== name) {
      // update links referencing old name
      links.value.forEach(l => {
        if (l.source === oldName) l.source = name
        if (l.target === oldName) l.target = name
        const src = linkName(l.source); if (src === oldName) l.source = name
        const tgt = linkName(l.target); if (tgt === oldName) l.target = name
      })
    }
    editingNode.value = null
  } else {
    if (nodes.value.some(n => n.name === name)) { toast.error('角色已存在'); return }
    nodes.value.push({ name, role: newNodeRole.value || '角色', color: newNodeColor.value })
  }
  showAddNode.value = false
  render()
  scheduleSave()
}
function deleteNode() {
  if (!editingNode.value) return
  const name = editingNode.value.name
  nodes.value = nodes.value.filter(n => n.name !== name)
  links.value = links.value.filter(l => linkName(l.source) !== name && linkName(l.target) !== name)
  editingNode.value = null
  showAddNode.value = false
  selected.value = null
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
  const data = { nodes: nodes.value, links: links.value.map(l => ({ source: linkName(l.source), target: linkName(l.target), type: l.type, label: l.label })), types: relTypes.value }
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'relationships.json'
  a.click()
  URL.revokeObjectURL(url)
}

function importJson() { fileInput.value?.click() }
function onFileSelected(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  const reader = new FileReader()
  reader.onload = () => {
    try {
      const d = JSON.parse(reader.result as string)
      if (Array.isArray(d.nodes)) nodes.value = d.nodes
      if (Array.isArray(d.links)) links.value = d.links
      if (Array.isArray(d.types) && d.types.length) relTypes.value = d.types
      render()
      scheduleSave()
      toast.success('导入成功')
    } catch { toast.error('JSON 格式错误') }
  }
  reader.readAsText(f)
  ;(e.target as HTMLInputElement).value = '' // ponytail: reset so same file can re-trigger
}

// --- Link context menu ---
function openLinkMenu(e: MouseEvent, linkData: any) {
  e.preventDefault()
  e.stopPropagation()
  ctxLink = linkData
  ctxPos.value = { x: e.clientX, y: e.clientY }
  showLinkMenu.value = true
}
function deleteLinkFromMenu() {
  if (!ctxLink) return
  const s = linkName(ctxLink.source), t = linkName(ctxLink.target)
  links.value = links.value.filter(l => !(linkName(l.source) === s && linkName(l.target) === t))
  showLinkMenu.value = false
  ctxLink = null
  render()
  scheduleSave()
}
function editLinkFromMenu() {
  if (!ctxLink) return
  // fill add-link form with existing values for editing
  showAddLink.value = true
  newLinkSource.value = linkName(ctxLink.source)
  newLinkTarget.value = linkName(ctxLink.target)
  newLinkType.value = ctxLink.type
  newLinkLabel.value = ctxLink.label
  // remove old link; confirmAddLink will add the new one
  const s = linkName(ctxLink.source), t = linkName(ctxLink.target)
  links.value = links.value.filter(l => !(linkName(l.source) === s && linkName(l.target) === t))
  showLinkMenu.value = false
  ctxLink = null
  render()
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

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.4);
  backdrop-filter: blur(6px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 500;
}
.modal {
  background: #252545;
  border: 1px solid rgba(255,255,255,0.12);
  border-radius: 14px;
  padding: 20px;
  width: 320px;
  max-height: 80vh;
  overflow-y: auto;
  color: #eee;
}
.modal h3 { margin: 0 0 12px; font-size: 15px; }
.modal label { display: block; margin: 8px 0 4px; font-size: 11px; color: #aaa; }
.modal input, .modal select {
  width: 100%;
  padding: 6px 10px;
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.12);
  border-radius: 6px;
  color: #fff;
  font-size: 13px;
  outline: none;
}
.modal input:focus, .modal select:focus { border-color: #4cc9f0; }
.modal select option { color: #333; background: #fff; }
.search-input { margin-bottom: 4px !important; }
.role-select {
  width: 100%;
  padding: 4px;
  font-size: 12px;
}
.color-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 6px;
}
.color-swatch {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  transition: transform 0.1s;
}
.color-swatch.selected { border-color: #fff; transform: scale(1.1); }
.type-buttons, .dash-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.type-btn {
  padding: 5px 12px;
  font-size: 12px;
  border: 1px solid #555;
  border-radius: 6px;
  background: transparent;
  color: #aaa;
  cursor: pointer;
  transition: all 0.12s;
}
.type-btn.selected { color: #fff; }
.modal-actions {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}
.modal-actions button {
  flex: 1;
  padding: 8px;
  border-radius: 8px;
  border: none;
  font-size: 13px;
  cursor: pointer;
}
.btn-cancel { background: rgba(255,255,255,0.1); color: #ccc; }
.btn-ok { background: #4cc9f0; color: #111; font-weight: 600; }
.btn-del { background: #ff3b30; color: #fff; font-weight: 600; }
.ctx-menu {
  position: fixed;
  z-index: 600;
  background: #252545;
  border: 1px solid rgba(255,255,255,0.12);
  border-radius: 8px;
  padding: 4px;
  box-shadow: 0 4px 16px rgba(0,0,0,0.4);
}
.ctx-item {
  display: block;
  width: 80px;
  padding: 6px 12px;
  font-size: 12px;
  border: none;
  background: transparent;
  color: #eee;
  cursor: pointer;
  border-radius: 4px;
  text-align: left;
}
.ctx-item:hover { background: rgba(255,255,255,0.1); }
.ctx-danger { color: #ff3b30; }
</style>
