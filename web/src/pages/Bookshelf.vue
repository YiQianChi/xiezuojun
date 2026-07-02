<template>
  <div class="bookshelf">
    <div class="bookshelf-header">
      <div>
        <h2>我的书架</h2>
        <p class="subtitle">{{ projects.length }} 本书</p>
      </div>
      <button class="btn primary" @click="showCreateModal = true">
        <span>+</span> 新建书籍
      </button>
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
        <button class="delete-btn" @click.stop="deleteBook(project)" title="删除">×</button>
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { projectApi } from '../core/api'
import { toast } from '../composables/useToast'

const router = useRouter()
const projects = ref<any[]>([])
const loading = ref(false)
const showCreateModal = ref(false)
const newBookName = ref('')
const newBookDesc = ref('')
const creating = ref(false)

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
  if (!confirm(`确定删除《${project.name}》？此操作不可恢复`)) return
  try {
    await projectApi.delete(project.id || project.project_id)
    await loadProjects()
  } catch (e: any) {
    toast.error(e.response?.data?.error || '删除失败')
  }
}
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

.loading {
  text-align: center;
  padding: 80px;
  color: var(--text-secondary);
}

.empty-state {
  text-align: center;
  padding: 100px 0;
}

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

.book-card:hover .delete-btn {
  opacity: 1;
}

.delete-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 28px;
  height: 28px;
  background: rgba(0,0,0,0.6);
  color: #fff;
  border: none;
  border-radius: 50%;
  cursor: pointer;
  opacity: 0;
  transition: all var(--transition-fast);
  font-size: 16px;
  line-height: 1;
  z-index: 10;
}

.delete-btn:hover {
  background: #ff3b30;
}

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
</style>