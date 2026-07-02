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
        <button class="btn primary" @click="showCreateModal = true">
          <span>+</span> 新建书籍
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="projects.length === 0" class="empty-state">
      <div class="empty-icon">📚</div>
      <h3>还没有书籍</h3>
      <p>开始你的第一本书吧</p>
      <button class="btn primary" @click="showCreateModal = true">创建第一本书</button>
    </div>

    <div v-else class="books-grid">
      <div
        v-for="project in projects"
        :key="project.id || project.project_id"
        class="book-card"
        @click="openBook(project)"
      >
        <div class="book-cover">
          <div class="book-icon">📖</div>
        </div>
        <div class="book-info">
          <h3>{{ project.name }}</h3>
          <p class="book-meta">{{ project.description || '暂无简介' }}</p>
        </div>
        <div class="card-actions">
          <button class="card-btn" @click.stop="startRename(project)" title="重命名">✏️</button>
          <button class="card-btn" @click.stop="exportBook(project)" title="导出">📥</button>
          <button class="card-btn delete" @click.stop="deleteBook(project)" title="删除">×</button>
        </div>
      </div>

      <div class="book-card add-card" @click="showCreateModal = true">
        <div class="book-cover add">
          <span>+</span>
        </div>
        <div class="book-info">
          <h3>新建书籍</h3>
          <p class="book-meta">开始创作</p>
        </div>
      </div>
    </div>

    <!-- 新建书籍弹窗 -->
    <div v-if="showCreateModal" class="modal-overlay" @click="showCreateModal = false">
      <div class="modal" @click.stop>
        <h3>新建书籍</h3>
        <div class="form-group">
          <label>书籍名称</label>
          <input v-model="newBookName" placeholder="输入书籍名称" class="input" @keyup.enter="createBook" />
        </div>
        <div class="form-group">
          <label>简介（可选）</label>
          <input v-model="newBookDesc" placeholder="一句话描述这本书" class="input" />
        </div>
        <div class="modal-actions">
          <button class="btn" @click="showCreateModal = false">取消</button>
          <button class="btn primary" @click="createBook" :disabled="creating">
            {{ creating ? '创建中...' : '创建' }}
          </button>
        </div>
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
              <span class="trash-meta">{{ item.type }} · 删除于 {{ formatDate(item.deleted_at) }}</span>
            </div>
            <button class="btn small" @click="restoreDoc(item.id)">恢复</button>
          </div>
        </div>
        <div v-if="trashItems.length > 0" class="modal-actions">
          <button class="btn danger" @click="emptyTrash">清空回收站</button>
          <button class="btn" @click="showTrash = false">关闭</button>
        </div>
        <div v-else class="modal-actions">
          <button class="btn" @click="showTrash = false">关闭</button>
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
            <input v-model="editNickname" class="input" placeholder="设置昵称" />
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
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { projectApi, trashApi, userApi } from '../core/api'
import { toast } from '../composables/useToast'

const router = useRouter()
const projects = ref<any[]>([])
const loading = ref(false)
const showCreateModal = ref(false)
const newBookName = ref('')
const newBookDesc = ref('')
const creating = ref(false)

// 回收站
const showTrash = ref(false)
const trashItems = ref<any[]>([])
const trashLoading = ref(false)

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
})

async function loadProjects() {
  loading.value = true
  try {
    const res = await projectApi.list()
    projects.value = res.data?.data || res.data || []
  } catch (e: any) {
    if (e.response?.status === 401) {
      router.push('/login')
    }
  } finally {
    loading.value = false
  }
}

async function createBook() {
  if (!newBookName.value.trim()) return
  creating.value = true
  try {
    const res = await projectApi.create({
      name: newBookName.value.trim(),
      description: newBookDesc.value.trim()
    })
    const project = res.data?.data || res.data
    showCreateModal.value = false
    newBookName.value = ''
    newBookDesc.value = ''
    router.push(`/editor/${project.id || project.project_id}`)
  } catch (e: any) {
    toast.error(e.response?.data?.error || '创建失败')
  } finally {
    creating.value = false
  }
}

function openBook(project: any) {
  const id = project.id || project.project_id
  router.push(`/editor/${id}`)
}

async function deleteBook(project: any) {
  if (!confirm(`确定删除《${project.name}》？删除后可在回收站恢复`)) return
  try {
    await projectApi.delete(project.id || project.project_id)
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
async function exportBook(project: any) {
  try {
    const res = await projectApi.exportProject(project.id || project.project_id)
    const data = res.data?.data || res.data
    const docs = data.document_contents || []
    let md = `# ${data.project.name}\n\n`
    if (data.project.description) md += `> ${data.project.description}\n\n`
    for (const doc of docs) {
      if (doc.type === 'chapter' || doc.type === 'volume') {
        md += `## ${doc.title}\n\n${doc.content || ''}\n\n`
      }
    }
    const blob = new Blob([md], { type: 'text/markdown;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${data.project.name}.md`
    a.click()
    URL.revokeObjectURL(url)
    toast.success('导出成功')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '导出失败')
  }
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

async function restoreDoc(docId: number) {
  try {
    await trashApi.restore(docId)
    await loadTrash()
    toast.success('已恢复')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '恢复失败')
  }
}

async function emptyTrash() {
  if (!confirm('确定清空回收站？此操作不可恢复')) return
  try {
    await trashApi.empty()
    trashItems.value = []
    toast.success('已清空')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '清空失败')
  }
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
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 24px;
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

.book-card:hover .card-actions {
  opacity: 1;
}

.card-actions {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity var(--transition-fast);
  z-index: 10;
}

.card-btn {
  width: 28px;
  height: 28px;
  background: rgba(0,0,0,0.6);
  color: #fff;
  border: none;
  border-radius: 50%;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-btn.delete:hover { background: #ff3b30; }

.book-cover {
  width: 100%;
  aspect-ratio: 3/4;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
  font-size: 48px;
  box-shadow: var(--shadow-md);
  transition: box-shadow var(--transition-normal);
}

.book-card:hover .book-cover {
  box-shadow: var(--shadow-lg);
}

.book-cover.add {
  background: transparent;
  border: 2px dashed var(--border);
  box-shadow: none;
  font-size: 40px;
  color: var(--text-tertiary);
}

.book-card.add-card:hover .book-cover {
  border-color: var(--accent);
  color: var(--accent);
  box-shadow: none;
}

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

/* 弹窗 */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.4);
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
</style>
