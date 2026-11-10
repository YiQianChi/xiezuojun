<template>
  <div class="bookshelf">
    <div class="bookshelf-header">
      <div>
        <h2>我的书架</h2>
        <p class="subtitle">{{ projects.length }} 本书</p>
      </div>
      <div class="header-actions">
        <button class="btn ghost" @click="showTrash = true">回收站</button>
        <button class="btn ghost" @click="showUserCenter = true">用户中心</button>
        <button class="btn primary" @click="openCreateModal">
          <span>+</span> 新建书籍
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="projects.length === 0" class="empty-state">
      <div class="empty-icon">📚</div>
      <h3>还没有书籍</h3>
      <p>开始你的第一本书吧</p>
      <button class="btn primary" @click="openCreateModal">创建第一本书</button>
    </div>

    <div v-else class="books-grid">
      <div
        v-for="project in projects"
        :key="project.id || project.project_id"
        class="book-card"
        @click="openBook(project)"
      >
        <div class="bc2-cover">
          <img v-if="project.cover_url" :src="project.cover_url" alt="" class="bc2-img" />
          <div v-else class="bc2-zi" :style="gradFor(project.name)"><span>{{ (project.name || '书').trim().charAt(0) }}</span></div>
        </div>
        <div class="book-info">
          <h3>{{ project.name }}</h3>
          <p v-if="project.description" class="book-meta">{{ project.description }}</p>
          <p class="bc2-meta">
            <span>{{ relTime(project.updated_at) }}</span><i></i>
            <span>{{ fmtWords(project.word_total) }}</span>
            <template v-if="project.chapter_count"><i></i><span>第 {{ project.chapter_count }} 章</span></template>
          </p>
        </div>
        <!-- 卡片操作收进 hover ⋯（触屏常显）；放卡片根节点避免被封面 overflow 裁剪 -->
        <div class="bc2-ops" @click.stop>
          <button class="bc2-more" title="封面 / 重命名 / 导出 / 删除" @click="toggleExportMenu(project.id || project.project_id)">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="5" cy="12" r="1.4" fill="currentColor"/><circle cx="12" cy="12" r="1.4" fill="currentColor"/><circle cx="19" cy="12" r="1.4" fill="currentColor"/></svg>
          </button>
          <div v-if="exportMenuId === (project.id || project.project_id)" class="bc2-menu">
            <button @click="exportMenuId = null; pickCover(project)">上传封面</button>
            <button @click="exportMenuId = null; startRename(project)">重命名</button>
            <button @click="exportBook(project, 'md')">导出 Markdown</button>
            <button @click="exportBook(project, 'txt')">导出 TXT</button>
            <button @click="exportBook(project, 'epub')">导出 EPUB</button>
            <button @click="exportBook(project, 'docx')">导出 DOCX</button>
            <div class="menu-sep"></div>
            <button class="danger" @click="exportMenuId = null; deleteBook(project)">删除</button>
          </div>
        </div>
      </div>

      <div class="book-card add-card" @click="openCreateModal">
        <div class="bc2-cover add">
          <span>+</span>
        </div>
        <div class="book-info">
          <h3>新建书籍</h3>
          <p class="book-meta">开始创作</p>
        </div>
      </div>
    </div>

    <!-- 新建书籍弹窗（空白 / 导入 TXT 二合一） -->
    <div v-if="showCreateModal" class="modal-overlay" @click="closeCreateModal">
      <div class="modal" @click.stop>
        <h3>新建书籍</h3>
        <div class="create-mode-tabs">
          <button
            class="mode-tab"
            :class="{ active: createMode === 'blank' }"
            @click="createMode = 'blank'"
          >✨ 空白新书</button>
          <button
            class="mode-tab"
            :class="{ active: createMode === 'import' }"
            @click="createMode = 'import'"
          >📥 导入 TXT</button>
        </div>

        <template v-if="createMode === 'blank'">
          <div class="form-group">
            <label>书籍名称</label>
            <input v-model="newBookName" placeholder="书名" class="input" @keyup.enter="createBook" />
          </div>
          <div class="form-group">
            <label>简介（可选）</label>
            <input v-model="newBookDesc" placeholder="简介（可选）" class="input" />
          </div>
          <div class="modal-actions">
            <button class="btn" @click="closeCreateModal">取消</button>
            <button class="btn primary" @click="createBook" :disabled="creating">
              {{ creating ? '创建中...' : '创建' }}
            </button>
          </div>
        </template>

        <ImportBookPanel v-else ref="importPanelRef" @imported="onImported" />
      </div>
    </div>

    <!-- 重命名弹窗 -->
    <div v-if="renameModal" class="modal-overlay" @click="renameModal = false">
      <div class="modal" @click.stop>
        <h3>重命名书籍</h3>
        <div class="form-group">
          <label>书籍名称</label>
          <input v-model="renameName" class="input" @keyup.enter="confirmRename" />
        </div>
        <div class="form-group">
          <label>简介</label>
          <input v-model="renameDesc" class="input" />
        </div>
        <div class="modal-actions">
          <button class="btn" @click="renameModal = false">取消</button>
          <button class="btn primary" @click="confirmRename" :disabled="renaming">保存</button>
        </div>
      </div>
    </div>

    <!-- 回收站弹窗 -->
    <div v-if="showTrash" class="modal-overlay" @click="showTrash = false">
      <div class="modal trash-modal" @click.stop>
        <h3>回收站</h3>
        <div v-if="trashLoading" class="loading">加载中...</div>
        <div v-else-if="trashItems.length === 0" class="empty-state small">
          <p>回收站为空</p>
        </div>
        <div v-else class="trash-list">
          <div v-for="item in trashItems" :key="item.id" class="trash-item">
            <div class="trash-info">
              <strong>{{ item.title }}</strong>
              <span class="trash-meta">{{ item.type === 'project' ? '书籍' : item.type }} · 删除于 {{ formatDate(item.trashed_at || item.deleted_at) }}</span>
            </div>
            <button class="btn small" @click="restoreDoc(item)">恢复</button>
            <button class="btn small danger" @click="openDestroyDoc(item)">彻底删除</button>
          </div>
        </div>
        <div v-if="trashItems.length > 0" class="modal-actions">
          <button class="btn danger" @click="openEmptyTrash">清空回收站</button>
          <button class="btn" @click="showTrash = false">关闭</button>
        </div>
        <div v-else class="modal-actions">
          <button class="btn" @click="showTrash = false">关闭</button>
        </div>
      </div>
    </div>

    <!-- 确认彻底删除/清空 弹窗 -->
    <div v-if="confirmModal" class="modal-overlay" @click="closeConfirm">
      <div class="modal confirm-modal-box" @click.stop>
        <div class="confirm-icon">⚠️</div>
        <h3>{{ confirmModal.title }}</h3>
        <p class="confirm-desc">{{ confirmModal.desc }}</p>
        <p class="confirm-warn">⚠️ 删除后不可恢复</p>
        <p class="confirm-hint">请输入 <strong>确认删除</strong> 以继续</p>
        <input v-model="confirmText" class="input" placeholder="确认删除" />
        <div class="modal-actions">
          <button class="btn" @click="closeConfirm">取消</button>
          <button class="btn danger" :disabled="confirmText !== '确认删除'" @click="confirmModal.action">确认删除</button>
        </div>
      </div>
    </div>

    <!-- 用户中心弹窗 -->
    <div v-if="showUserCenter" class="modal-overlay" @click="showUserCenter = false">
      <div class="modal" @click.stop>
        <h3>用户中心</h3>
        <div v-if="userInfo" class="user-info">
          <div class="form-group">
            <label>邮箱</label>
            <input :value="userInfo.email" class="input" disabled />
          </div>
          <div class="form-group">
            <label>昵称</label>
            <input v-model="editNickname" class="input" placeholder="昵称" />
          </div>
          <div class="usage-stats" v-if="usage">
            <div class="usage-row">
              <span>今日字数</span>
              <strong>{{ usage.today_words }}</strong>
            </div>
            <div class="usage-row">
              <span>本月字数</span>
              <strong>{{ usage.month_words }}</strong>
            </div>
            <div class="usage-row">
              <span>累计字数</span>
              <strong>{{ usage.total_words }}</strong>
            </div>
          </div>
          <div class="plan-badge">{{ userInfo.plan || 'free' }}</div>
        </div>
        <div class="modal-actions">
          <button class="btn" @click="showUserCenter = false">取消</button>
          <button class="btn primary" @click="saveProfile" :disabled="savingProfile">保存</button>
        </div>
      </div>
    </div>

    <input ref="coverInput" type="file" accept="image/jpeg,image/png,image/webp" style="display:none" @change="onCoverPicked" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { projectApi, trashApi, userApi } from '../core/api'
import { toast } from '../composables/useToast'
import ImportBookPanel from '../components/ImportBookPanel.vue'

const router = useRouter()
const projects = ref<any[]>([])
const loading = ref(false)


const showCreateModal = ref(false)
const createMode = ref<'blank' | 'import'>('blank')
const importPanelRef = ref<{ reset: () => void } | null>(null)
const newBookName = ref('')
const newBookDesc = ref('')
const creating = ref(false)

// 回收站
const showTrash = ref(false)
const trashItems = ref<any[]>([])
const trashLoading = ref(false)
const confirmModal = ref<{ title: string; desc: string; action: () => void } | null>(null)
const confirmText = ref('')

// 用户中心
const showUserCenter = ref(false)
const userInfo = ref<any>(null)
const editNickname = ref('')
const usage = ref<any>(null)
const savingProfile = ref(false)

// 重命名
const renameModal = ref(false)
const renameName = ref('')
const renameDesc = ref('')
const renaming = ref(false)
const renameTarget = ref<any>(null)

onMounted(() => {
  loadProjects()
  // 首页工作台「导入书稿」深链：直接弹导入 tab
  if (router.currentRoute.value.query.import) {
    showCreateModal.value = true
    createMode.value = 'import'
  }
  document.addEventListener('click', closeExportMenu)
})

async function loadProjects() {
  loading.value = true
  try {
    const res = await projectApi.list()
    projects.value = res.data?.data || res.data || []
    // 拼字邀请落地闭环：已登录收卡/点链接落在书架时，自动进最近一本书——编辑器 init 会弹「加入房间」
    if (localStorage.getItem('xzj-sprint-invite') && projects.value.length) {
      const p = projects.value[0]
      toast.info('🏁 收到拼字邀请，正在带你去加入')
      router.push(`/editor/${p.public_id || p.id}`)
    }
  } catch (e: any) {
    if (e.response?.status === 401) {
      router.push('/login')
    }
  } finally {
    loading.value = false
  }
}

function openCreateModal() {
  createMode.value = 'blank'
  showCreateModal.value = true
}

function closeCreateModal() {
  showCreateModal.value = false
  importPanelRef.value?.reset()
}

async function createBook() {
  if (!newBookName.value.trim()) return
  creating.value = true
  try {
    const res = await projectApi.create({
      name: newBookName.value.trim(),
      description: newBookDesc.value.trim(),
      setting_types: [] // 作品相关文件不在建书时创建——进编辑器后弹窗让作者选
    })
    const project = res.data?.data || res.data
    showCreateModal.value = false
    newBookName.value = ''
    newBookDesc.value = ''
    router.push(`/editor/${project.public_id || project.id}`)
  } catch (e: any) {
    toast.error(e.response?.data?.error || '创建失败')
  } finally {
    creating.value = false
  }
}

function openBook(project: any) {
  // 编辑器 URL 用随机 public_id，不暴露自增序号
  const id = project.public_id || project.id || project.project_id
  router.push(`/editor/${id}`)
}

// 导入完成 → 关弹窗直接进新书编辑器（ImportBookPanel 回传 public_id 或数字 id）
function onImported(projectId: string | number) {
  showCreateModal.value = false
  router.push(`/editor/${projectId}`)
}

async function deleteBook(project: any) {
  if (!confirm(`确定删除《${project.name}》？删除后可在回收站恢复`)) return
  try {
    await projectApi.delete(project.public_id || project.id)
    await loadProjects()
  } catch (e: any) {
    toast.error(e.response?.data?.error || '删除失败')
  }
}

// 重命名
function startRename(project: any) {
  renameTarget.value = project
  renameName.value = project.name
  renameDesc.value = project.description || ''
  renameModal.value = true
}

async function confirmRename() {
  if (!renameName.value.trim() || !renameTarget.value) return
  renaming.value = true
  try {
    await projectApi.update(renameTarget.value.id || renameTarget.value.project_id, {
      name: renameName.value.trim(),
      description: renameDesc.value.trim()
    })
    renameModal.value = false
    await loadProjects()
    toast.success('已更新')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '更新失败')
  } finally {
    renaming.value = false
  }
}

// 导出
const exportMenuId = ref<number | null>(null)

// ===== 封面上传 =====
const coverInput = ref<HTMLInputElement | null>(null)
let coverProject: any = null
function pickCover(project: any) {
  coverProject = project
  if (coverInput.value) {
    coverInput.value.value = ''
    coverInput.value.click()
  }
}
async function onCoverPicked(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || !coverProject) return
  if (file.size > 5 * 1024 * 1024) { toast.error('图片不能超过 5MB'); return }
  const fd = new FormData()
  fd.append('cover', file)
  try {
    const res = await projectApi.uploadCover(coverProject.public_id || coverProject.id, fd)
    const url = res.data?.data?.cover_url
    if (url) coverProject.cover_url = url
    toast.success('封面已更新')
  } catch (err: any) {
    toast.error(err?.response?.data?.error || '上传失败')
  }
}

function toggleExportMenu(id: number) {
  exportMenuId.value = exportMenuId.value === id ? null : id
}

// ===== 卡片信息行 helpers（UX 批2 方案A）=====
function relTime(t?: string) {
  if (!t) return ''
  const d = new Date(t), now = new Date()
  const ms = now.getTime() - d.getTime()
  if (ms < 60000) return '刚刚'
  const pad = (n: number) => String(n).padStart(2, '0')
  const sameDay = (a: Date, b: Date) => a.toDateString() === b.toDateString()
  if (sameDay(d, now)) return `今天 ${pad(d.getHours())}:${pad(d.getMinutes())}`
  if (sameDay(d, new Date(now.getTime() - 86400000))) return `昨天 ${pad(d.getHours())}:${pad(d.getMinutes())}`
  const day = Math.floor(ms / 86400000)
  if (day < 30) return `${day} 天前`
  if (day < 365) return `${Math.floor(day / 30)} 个月前`
  return `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`
}

function fmtWords(n?: number) {
  if (!n) return '0 字'
  if (n < 10000) return `${n} 字`
  return `${(n / 10000).toFixed(1).replace(/\.0$/, '')} 万字`
}

// 首字封面渐变：按书名 hash 从低饱和暖冷六组里挑，同书恒定（与编辑器侧栏书卡同语言）
function gradFor(name?: string) {
  const s = name || '书'
  let h = 0
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0
  const palettes = [
    ['#8fc59b', '#5f9c6e'], ['#a8c5d8', '#7396ad'], ['#d8c39f', '#b08560'],
    ['#c5abd0', '#9a7ba8'], ['#e0b1a4', '#c07b66'], ['#a4c9c0', '#6f9f93'],
  ]
  const [a, b] = palettes[h % palettes.length]
  return `background: linear-gradient(135deg, ${a}, ${b})`
}

// 点击外部关闭导出菜单
function closeExportMenu() { exportMenuId.value = null }

async function exportBook(project: any, format: 'md' | 'txt' | 'epub' | 'docx') {
  exportMenuId.value = null
  const pid = project.id || project.project_id
  const name = project.name || '未命名'
  try {
    if (format === 'md') {
      const res = await projectApi.exportProject(pid)
      const data = res.data?.data || res.data
      const docs = data.document_contents || []
      let md = `# ${data.project.name}\n\n`
      if (data.project.description) md += `> ${data.project.description}\n\n`
      for (const doc of docs) {
        if (doc.type === 'chapter' || doc.type === 'volume') {
          md += `## ${doc.title}\n\n${doc.content || ''}\n\n`
        }
      }
      downloadBlob(new Blob([md], { type: 'text/markdown;charset=utf-8' }), `${name}.md`)
    } else if (format === 'epub') {
      const res = await projectApi.exportEPUB(pid)
      downloadBlob(new Blob([res.data], { type: 'application/epub+zip' }), `${name}.epub`)
    } else if (format === 'docx') {
      const res = await projectApi.exportDOCX(pid)
      downloadBlob(new Blob([res.data], { type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' }), `${name}.docx`)
    } else {
      const res = await projectApi.exportTXT(pid)
      downloadBlob(new Blob([res.data], { type: 'text/plain;charset=utf-8' }), `${name}.txt`)
    }
    toast.success('导出成功')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '导出失败')
  }
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

// 回收站
async function loadTrash() {
  trashLoading.value = true
  try {
    const res = await trashApi.list()
    const data = res.data?.data
    trashItems.value = Array.isArray(data) ? data : []
  } catch {
    trashItems.value = []
  } finally {
    trashLoading.value = false
  }
}

async function restoreDoc(item: any) {
  try {
    // ponytail: project 类型走 project_id 恢复，doc 类型走 doc_id
    const payload = item.type === 'project'
      ? { project_id: item.id || item.project_id, doc_id: 0 }
      : { project_id: item.project_id, doc_id: item.id }
    await trashApi.restore(payload)
    await loadTrash()
    await loadProjects()
    toast.success('已恢复')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '恢复失败')
  }
}

async function emptyTrash() {
  try {
    await trashApi.empty()
    trashItems.value = []
    toast.success('已清空')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '清空失败')
  } finally {
    closeConfirm()
  }
}

function openEmptyTrash() {
  confirmText.value = ''
  confirmModal.value = {
    title: '清空回收站',
    desc: `确定要清空回收站吗？将删除全部 ${trashItems.value.length} 个项目。`,
    action: emptyTrash,
  }
}

function closeConfirm() {
  confirmModal.value = null
  confirmText.value = ''
}

async function destroyDoc() {
  const item = (confirmModal.value as any)?._item
  if (!item) return
  try {
    if (item.type === 'project') {
      await trashApi.destroy({ project_id: item.id || item.project_id, doc_id: 0 })
    } else {
      await trashApi.destroy({ project_id: item.project_id, doc_id: item.id })
    }
    await loadTrash()
    await loadProjects()
    toast.success('已彻底删除')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '删除失败')
  } finally {
    closeConfirm()
  }
}

function openDestroyDoc(item: any) {
  confirmText.value = ''
  confirmModal.value = {
    title: '彻底删除',
    desc: `确定要彻底删除「${item.title}」吗？`,
    action: destroyDoc,
    _item: item,
  } as any
}

// 用户中心
async function loadUserInfo() {
  try {
    const [meRes, usageRes] = await Promise.all([userApi.me(), userApi.usage()])
    userInfo.value = meRes.data?.data || meRes.data
    editNickname.value = userInfo.value?.nickname || ''
    usage.value = usageRes.data?.data || usageRes.data
  } catch {
    // ignore
  }
}

async function saveProfile() {
  savingProfile.value = true
  try {
    await userApi.updateProfile({ nickname: editNickname.value.trim() })
    toast.success('已保存')
    showUserCenter.value = false
  } catch (e: any) {
    toast.error(e.response?.data?.error || '保存失败')
  } finally {
    savingProfile.value = false
  }
}

function formatDate(d: string) {
  if (!d) return ''
  return new Date(d).toLocaleDateString('zh-CN')
}

// ponytail: 弹窗打开时加载数据
watch(showTrash, (v) => { if (v) loadTrash() })
watch(showUserCenter, (v) => { if (v) loadUserInfo() })
</script>

<style scoped>
.bookshelf {
  max-width: 1200px;
  margin: 0 auto;
  padding: 40px 32px;
  width: 100%;
}

.bookshelf-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 32px;
}

.bookshelf-header h2 {
  font-family: var(--font-serif);
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.bookshelf-header .subtitle {
  color: var(--text-secondary);
  font-size: 13px;
}

.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.loading {
  text-align: center;
  padding: 80px;
  color: var(--text-secondary);
}

.empty-state {
  text-align: center;
  padding: 100px 0;
}

.empty-state.small { padding: 40px 0; }

.empty-icon {
  font-size: 64px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.empty-state h3 {
  font-size: 20px;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.empty-state p {
  color: var(--text-secondary);
  margin-bottom: 24px;
}

.books-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 220px));
  gap: 24px;
  justify-content: center;
}

.book-card {
  cursor: pointer;
  transition: all var(--transition-normal);
  position: relative;
  border-radius: var(--radius-md);
  padding: 8px;
}

.book-card:hover {
  transform: translateY(-6px);
}

/* 卡片 v2（UX 批2 方案A）：紧凑卡 + 首字渐变封面 + 信息行 + hover ⋯ */
.bc2-cover {
  width: 100%;
  aspect-ratio: 15 / 8;
  border-radius: var(--radius-md);
  overflow: hidden;
  margin-bottom: 10px;
  box-shadow: var(--shadow-md);
  transition: box-shadow var(--transition-normal);
  background: var(--bg-secondary);
}

.book-card:hover .bc2-cover { box-shadow: var(--shadow-lg); }

.bc2-img { width: 100%; height: 100%; object-fit: cover; display: block; }

.bc2-zi { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; }

.bc2-zi span {
  font-family: var(--font-serif);
  font-size: 44px;
  font-weight: 700;
  color: #fff;
  text-shadow: 0 1px 4px rgba(0, 0, 0, 0.25);
}

.bc2-cover.add {
  background: transparent;
  border: 2px dashed var(--border);
  box-shadow: none;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  color: var(--text-tertiary);
}

.book-card.add-card:hover .bc2-cover.add {
  border-color: var(--accent);
  color: var(--accent);
  box-shadow: none;
}

.bc2-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
  margin: 2px 0 0;
  white-space: nowrap;
  overflow: hidden;
}

.bc2-meta i { width: 3px; height: 3px; border-radius: 50%; background: var(--border); flex: none; }

/* hover ⋯（触屏常显）；挂卡片根节点，弹层不被封面 overflow 裁剪 */
.bc2-ops {
  position: absolute;
  top: 14px;
  right: 14px;
  opacity: 0;
  transition: opacity var(--transition-fast);
  z-index: 10;
}

.book-card:hover .bc2-ops { opacity: 1; }

@media (hover: none) { .bc2-ops { opacity: 1; } }

.bc2-more {
  width: 26px;
  height: 26px;
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.45);
  color: #fff;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  backdrop-filter: blur(4px);
}

.bc2-more:hover { background: rgba(0, 0, 0, 0.65); }

.bc2-menu {
  position: absolute;
  top: 30px;
  right: 0;
  background: var(--bg-secondary);
  border-radius: 10px;
  box-shadow: var(--shadow-md);
  overflow: hidden;
  z-index: 100;
  min-width: 150px;
  padding: 4px 0;
}

.bc2-menu button {
  display: block;
  width: 100%;
  text-align: left;
  border: none;
  background: transparent;
  padding: 8px 14px;
  font-size: 13px;
  color: var(--text-primary);
  cursor: pointer;
}

.bc2-menu button:hover { background: rgba(0, 0, 0, 0.05); }

.bc2-menu button.danger { color: #e5484d; }

.bc2-menu .menu-sep { height: 1px; background: var(--border); margin: 4px 0; }

.book-info h3 {
  font-family: var(--font-serif);
  font-size: 15px;
  font-weight: 600;
  margin: 0 0 4px;
  color: var(--text-primary);
}

.book-meta {
  font-size: 12px;
  color: var(--text-secondary);
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 建书弹窗的作品相关勾选组 */
.setting-checks {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 14px;
}
.setting-check {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 13px;
  color: var(--text-primary);
  cursor: pointer;
}
.setting-check input { accent-color: var(--accent, #007aff); }

/* 新建书籍弹窗的模式切换 tab */
.create-mode-tabs {
  display: flex;
  gap: 6px;
  margin-bottom: 18px;
  padding: 4px;
  background: var(--bg-tertiary, #f5f5f7);
  border-radius: 10px;
}
.mode-tab {
  flex: 1;
  padding: 8px 0;
  border: none;
  border-radius: 8px;
  background: transparent;
  font-size: 14px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all .15s;
}
.mode-tab.active {
  background: var(--bg-secondary, #fff);
  color: var(--text-primary);
  font-weight: 600;
  box-shadow: 0 1px 4px rgba(0,0,0,.08);
}

/* 弹窗 */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: var(--overlay-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  background: var(--bg-secondary, #fff);
  border-radius: var(--radius-lg, 16px);
  padding: 32px;
  width: 420px;
  max-width: 90vw;
  box-shadow: var(--shadow-xl, 0 20px 60px rgba(0,0,0,.15));
}

.modal h3 {
  font-size: 20px;
  margin-bottom: 20px;
  color: var(--text-primary);
}

.trash-modal { width: 560px; }

.confirm-modal-box {
  max-width: 380px;
  text-align: center;
}
.confirm-modal-box .confirm-icon { font-size: 32px; margin-bottom: 8px; }
.confirm-modal-box .confirm-desc { font-size: 13px; color: var(--text-secondary); margin: 8px 0 6px; }
.confirm-modal-box .confirm-warn { font-size: 13px; color: var(--danger, #ff4d6d); font-weight: 600; margin-bottom: 12px; }
.confirm-modal-box .confirm-hint { font-size: 12px; color: var(--text-secondary); margin-bottom: 8px; }
.confirm-modal-box .input { text-align: center; margin-bottom: 4px; }
/* 弹窗输入统一居中（Boss 2026-10-30） */
.modal .input { text-align: center; }
.confirm-modal-box .modal-actions { justify-content: center; }

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 20px;
}

.btn {
  border: 1px solid var(--border, #e0e0e0);
  background: transparent;
  padding: 8px 16px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  color: var(--text-primary);
  transition: all var(--transition-fast, .15s);
}

.btn:hover { background: var(--bg-tertiary, #f5f5f7); }
.btn:disabled { opacity: .5; cursor: not-allowed; }

.btn.primary {
  background: var(--accent, #007aff);
  color: #fff;
  border-color: var(--accent, #007aff);
}

.btn.primary:hover { opacity: .9; background: var(--accent, #007aff); }

.btn.ghost {
  border-color: transparent;
  color: var(--text-secondary);
}

.btn.ghost:hover { background: var(--bg-tertiary, #f5f5f7); }

.btn.danger {
  color: #ff3b30;
  border-color: rgba(255,59,48,.3);
}

.btn.danger:hover { background: rgba(255,59,48,.08); }

.btn.small { padding: 4px 12px; font-size: 12px; }

/* 回收站列表 */
.trash-list {
  max-height: 400px;
  overflow-y: auto;
}

.trash-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px solid var(--border, #eee);
}

.trash-info { display: flex; flex-direction: column; gap: 2px; }
.trash-info strong { font-size: 14px; color: var(--text-primary); }
.trash-meta { font-size: 12px; color: var(--text-tertiary); }

/* 用户中心 */
.user-info { margin-bottom: 8px; }

.usage-stats {
  margin: 16px 0;
  padding: 16px;
  background: var(--bg-tertiary, #f5f5f7);
  border-radius: 12px;
}

.usage-row {
  display: flex;
  justify-content: space-between;
  padding: 6px 0;
  font-size: 14px;
}

.usage-row span { color: var(--text-secondary); }
.usage-row strong { color: var(--text-primary); }

.plan-badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  background: rgba(0,122,255,.1);
  color: var(--accent, #007aff);
}

/* 大屏 1440+ */
@media (min-width: 1440px) {
  .bookshelf { max-width: 1400px; }
  .books-grid { grid-template-columns: repeat(auto-fill, minmax(200px, 240px)); gap: 28px; }
}

/* 2K 1920+ (含4K缩放150%) */
@media (min-width: 1920px) {
  .bookshelf { max-width: 1600px; padding: 48px 40px; }
  .bookshelf-header h2 { font-size: 32px; }
  .books-grid { grid-template-columns: repeat(auto-fill, minmax(220px, 280px)); gap: 32px; }
  .bc2-zi span { font-size: 56px; }
  .book-info h3 { font-size: 16px; }
  .book-meta { font-size: 13px; }
  .btn { font-size: 15px; padding: 9px 18px; }
  .modal { width: 480px; padding: 36px; }
}

/* 4K 2560+ (原生4K或2K缩放125%) */
@media (min-width: 2560px) {
  .bookshelf { max-width: 2000px; padding: 64px 48px; }
  .bookshelf-header h2 { font-size: 36px; }
  .bookshelf-header .subtitle { font-size: 16px; }
  .books-grid { grid-template-columns: repeat(auto-fill, minmax(260px, 320px)); gap: 36px; }
  .bc2-zi span { font-size: 64px; }
  .book-info h3 { font-size: 18px; }
  .book-meta { font-size: 14px; }
  .btn { font-size: 16px; padding: 10px 20px; }
  .modal { width: 560px; padding: 40px; }
  .modal h3 { font-size: 24px; }
  .empty-icon { font-size: 80px; }
  .empty-state h3 { font-size: 24px; }
  .loading { font-size: 18px; padding: 120px; }
}

/* 平板 */
@media (max-width: 1023px) {
  .bookshelf { padding: 32px 24px; }
  .books-grid { grid-template-columns: repeat(auto-fill, minmax(160px, 200px)); gap: 20px; }
}

/* 手机 */
@media (max-width: 720px) {
  .bookshelf { padding: 20px 16px; }
  .bookshelf-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  .header-actions { width: 100%; }
  .header-actions .btn { padding: 8px 14px; }
  .header-actions .btn.primary { flex: 1; }
  .bookshelf-header h2 { font-size: 22px; }
  .subtitle { font-size: 12px; }
  /* 触屏：固定两列，禁止横向溢出 */
  .books-grid { grid-template-columns: repeat(2, 1fr) !important; gap: 14px; }
  /* 触屏：卡片给实体卡底（透明卡片+文字行悬浮很怪） */
  .book-card { background: var(--bg-secondary, #fff); border: 1px solid var(--border, #ececf1); }
  .book-card:hover { transform: none; }
  .bc2-zi span { font-size: 34px; }
  .bc2-menu { min-width: 132px; }
  .book-info h3 { font-size: 14px; }
  .book-info { padding: 10px 12px 12px; }
  .book-meta { font-size: 11px; }
  .modal { width: 100%; padding: 24px 20px; }
  .trash-modal { width: 100%; }
  .empty-state { padding: 60px 0; }
  .empty-icon { font-size: 48px; }
}

/* 小手机 */
@media (max-width: 380px) {
  .books-grid { grid-template-columns: repeat(2, 1fr); gap: 10px; }
  .bc2-zi span { font-size: 28px; }
  .header-actions .btn { font-size: 12px; padding: 7px 10px; }
}
</style>
