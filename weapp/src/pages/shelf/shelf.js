// 原生书架：书籍卡片 + 新建/重命名/删除/回收站，全部走原生
const api = require('../../utils/api')

Page({
  data: {
    projects: [],
    trash: [],
    loading: true,
    showCreate: false,
    newName: '',
    creating: false,
    renameTarget: null,
    renameName: '',
    showTrash: false
  },

  onShow() { this.load() },

  async load() {
    this.setData({ loading: true })
    try {
      const res = await api.get('/projects')
      const list = (res.data || res || []).map(p => ({ ...p, cover_url: api.coverUrl(p.cover_url) }))
      this.setData({ projects: list, loading: false })
    } catch (e) {
      this.setData({ loading: false })
      if (e.message !== 'unauthorized') wx.showToast({ title: '加载失败，下拉重试', icon: 'none' })
    }
  },

  onPullDownRefresh() {
    this.load().then(() => wx.stopPullDownRefresh())
  },

  openBook(e) {
    const pid = e.currentTarget.dataset.pid
    if (pid) api.openEditor(pid)
  },

  // ---- 新建 ----
  openCreate() { this.setData({ showCreate: true, newName: '' }) },
  onName(e) { this.setData({ newName: e.detail.value }) },
  cancelCreate() { this.setData({ showCreate: false }) },
  async createProject() {
    const name = this.data.newName.trim()
    if (!name || this.data.creating) return
    this.setData({ creating: true })
    try {
      const res = await api.post('/projects', { name, description: '', setting_types: [] })
      const p = res.data || res
      this.setData({ showCreate: false, creating: false })
      api.openEditor(p.public_id || String(p.id))
    } catch (e) {
      this.setData({ creating: false })
      wx.showToast({ title: (e.data && e.data.error) || '创建失败', icon: 'none' })
    }
  },

  // ---- 重命名 ----
  startRename(e) {
    const p = this.data.projects.find(x => String(x.id) === String(e.currentTarget.dataset.id))
    if (!p) return
    this.setData({ renameTarget: p, renameName: p.name })
  },
  onRename(e) { this.setData({ renameName: e.detail.value }) },
  cancelRename() { this.setData({ renameTarget: null }) },
  async doRename() {
    const name = this.data.renameName.trim()
    const t = this.data.renameTarget
    if (!name || !t) return
    try {
      await api.patch('/projects/' + (t.public_id || t.id), { name })
      this.setData({ renameTarget: null })
      this.load()
    } catch (e) {
      wx.showToast({ title: (e.data && e.data.error) || '重命名失败', icon: 'none' })
    }
  },

  // ---- 删除（软删进回收站） ----
  delBook(e) {
    const p = this.data.projects.find(x => String(x.id) === String(e.currentTarget.dataset.id))
    if (!p) return
    wx.showModal({
      title: '删除《' + p.name + '》？',
      content: '删除后可在回收站恢复',
      confirmText: '删除',
      confirmColor: '#d05252',
      success: r => {
        if (!r.confirm) return
        api.del('/projects/' + (p.public_id || p.id)).then(() => {
          wx.showToast({ title: '已移入回收站', icon: 'none' })
          this.load()
        }).catch(err => wx.showToast({ title: (err.data && err.data.error) || '删除失败', icon: 'none' }))
      }
    })
  },

  // ---- 回收站 ----
  async openTrash() {
    this.setData({ showTrash: true, trash: [] })
    try {
      const res = await api.get('/trash')
      const items = (res.data || res || []).filter(i => i.type === 'project')
      this.setData({ trash: items })
    } catch (e) {
      wx.showToast({ title: '回收站加载失败', icon: 'none' })
    }
  },
  closeTrash() { this.setData({ showTrash: false }) },
  restoreBook(e) {
    const id = e.currentTarget.dataset.id
    api.post('/trash/restore', { project_id: Number(id), doc_id: 0 }).then(() => {
      wx.showToast({ title: '已恢复', icon: 'none' })
      this.openTrash()
      this.load()
    }).catch(err => wx.showToast({ title: (err.data && err.data.error) || '恢复失败', icon: 'none' }))
  },
  destroyBook(e) {
    const id = e.currentTarget.dataset.id
    wx.showModal({
      title: '彻底删除',
      content: '删除后不可恢复，确定吗？',
      confirmText: '彻底删除',
      confirmColor: '#d05252',
      success: r => {
        if (!r.confirm) return
        api.post('/trash/destroy', { project_id: Number(id), doc_id: 0 }).then(() => {
          this.openTrash()
        }).catch(err => wx.showToast({ title: (err.data && err.data.error) || '删除失败', icon: 'none' }))
      }
    })
  },
  emptyTrash() {
    if (!this.data.trash.length) return
    wx.showModal({
      title: '清空回收站',
      content: '将彻底删除全部 ' + this.data.trash.length + ' 本书，不可恢复。',
      confirmText: '清空',
      confirmColor: '#d05252',
      success: r => {
        if (!r.confirm) return
        api.del('/trash').then(() => {
          wx.showToast({ title: '已清空', icon: 'none' })
          this.setData({ showTrash: false })
          this.load()
        }).catch(err => wx.showToast({ title: (err.data && err.data.error) || '清空失败', icon: 'none' }))
      }
    })
  }
})
