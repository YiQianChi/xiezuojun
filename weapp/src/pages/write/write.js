// 原生码字模式：章节树 + 大输入框 + 自动保存 + 字数；深度编辑（排版/AI）进网页编辑器
const api = require('../../utils/api')

// HTML ↔ 纯文本（章正文基本是 <p> 段落；粗体等富文本标记在码字模式下舍弃）
function htmlToText(html) {
  if (!html) return ''
  return html
    .replace(/<h1[^>]*>/gi, '').replace(/<\/h1>/gi, '\n')
    .replace(/<\/(p|div|h[1-6]|li)>/gi, '\n')
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<[^>]+>/g, '')
    .replace(/&nbsp;/g, ' ').replace(/&amp;/g, '&').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}
function textToHtml(text) {
  const esc = text
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  return esc.split(/\n/).map(l => '<p>' + (l.trim() === '' ? '<br>' : l) + '</p>').join('')
}
function countWords(t) {
  return (t || '').replace(/\s/g, '').length
}

Page({
  data: {
    projectId: 0,
    publicId: '',
    bookName: '',
    docs: [],          // 卷/章树（只含 volume+chapter）
    flatChapters: [],  // 平铺章节（供切章）
    cur: null,         // 当前章节 {id,title,updated_at}
    text: '',
    wc: 0,
    saving: '就绪',
    drawer: false
  },

  onLoad(options) {
    const pid = Number(options.pid || 0)
    const pub = options.pub || ''
    this.setData({ projectId: pid, publicId: pub })
    this.loadAll().then(() => {
      // 默认打开最后一章（接着写）
      const chapters = this.data.flatChapters
      if (chapters.length) this.openChapter(chapters[chapters.length - 1])
    })
  },

  async loadAll() {
    try {
      const res = await api.get('/projects/' + this.data.projectId)
      const p = res.data || res
      this.setData({ bookName: p.name || '' })
      wx.setNavigationBarTitle({ title: p.name || '写作' })
    } catch (e) { /* 书名拿不到不阻塞 */ }
    try {
      const res = await api.get('/documents?project_id=' + this.data.projectId)
      const list = (res.data || res || []).filter(d => d.type === 'volume' || d.type === 'chapter')
      const vols = list.filter(d => d.type === 'volume' && (!d.parent_id || d.parent_id === 0))
      const loose = list.filter(d => d.type === 'chapter' && (!d.parent_id || d.parent_id === 0))
      const tree = []
      for (const v of vols) {
        tree.push({ ...v, isVol: true, children: list.filter(d => d.type === 'chapter' && d.parent_id === v.id) })
      }
      for (const c of loose) tree.push({ ...c, isVol: false, children: [] })
      const flat = []
      for (const n of tree) { if (!n.isVol) flat.push(n); for (const c of n.children || []) flat.push(c) }
      this.setData({ docs: tree, flatChapters: flat })
    } catch (e) {
      wx.showToast({ title: '目录加载失败', icon: 'none' })
    }
  },

  async openChapter(ch) {
    await this.flushSave()
    try {
      const res = await api.get('/documents/' + ch.id)
      const d = res.data || res
      const doc = d.document || d
      this._updatedAt = doc.updated_at || ''
      const text = htmlToText(d.content || '')
      this._origText = text
      this.setData({ cur: { id: ch.id, title: ch.title || doc.title }, text, wc: countWords(text), saving: '就绪', drawer: false })
    } catch (e) {
      wx.showToast({ title: '打开章节失败', icon: 'none' })
    }
  },

  onInput(e) {
    const text = e.detail.value
    this.setData({ text, wc: countWords(text), saving: '未保存' })
    if (this._saveTimer) clearTimeout(this._saveTimer)
    this._saveTimer = setTimeout(() => this.flushSave(), 1500)
  },

  async flushSave() {
    if (this._saveTimer) { clearTimeout(this._saveTimer); this._saveTimer = null }
    const { cur, text } = this.data
    if (!cur || text === this._origText) return
    this.setData({ saving: '保存中…' })
    try {
      const res = await api.patch('/documents/' + cur.id, {
        content: textToHtml(text),
        last_known_updated_at: this._updatedAt || ''
      })
      const d = res.data || res
      this._updatedAt = (d.updated_at) || this._updatedAt
      this._origText = text
      this.setData({ saving: '已保存' })
    } catch (e) {
      // 409=云端有新版本：拉云端覆盖本地（单设备码字场景风险极低）
      if (e.status === 409 || (e.data && e.data.code === 'document_conflict')) {
        this.openChapter(cur)
        wx.showToast({ title: '云端有更新，已加载云端版', icon: 'none' })
      } else {
        this.setData({ saving: '保存失败' })
        wx.showToast({ title: (e.data && e.data.error) || '保存失败', icon: 'none' })
      }
    }
  },

  onHide() { this.flushSave() },
  onUnload() { this.flushSave() },

  toggleDrawer() {
    this.setData({ drawer: !this.data.drawer })
    if (this.data.drawer) this.loadAll()
  },

  pickChapter(e) {
    const id = Number(e.currentTarget.dataset.id)
    const ch = this.data.flatChapters.find(c => c.id === id)
    if (ch) this.openChapter(ch)
  },

  newChapter() {
    // 挂到当前章所在卷（或顶层）
    const cur = this.data.cur
    const parentVol = cur ? (this.data.docs.find(v => v.isVol && (v.children || []).some(c => c.id === cur.id)) || null) : null
    const parentId = parentVol ? parentVol.id : null
    api.post('/documents', {
      project_id: this.data.projectId,
      type: 'chapter',
      title: '第 ' + (this.data.flatChapters.length + 1) + ' 章',
      parent_id: parentId
    }).then(() => {
      this.loadAll().then(() => {
        const flat = this.data.flatChapters
        if (flat.length) this.openChapter(flat[flat.length - 1])
      })
    }).catch(e => wx.showToast({ title: (e.data && e.data.error) || '新建失败', icon: 'none' }))
  },

  // 深度编辑（排版/AI/设定）→ 网页完整编辑器
  fullEditor() {
    this.flushSave().then(() => {
      if (this.data.publicId) api.openEditor(this.data.publicId)
    })
  }
})
