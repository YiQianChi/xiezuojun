// 原生导入：txt → 卷/章解析（逻辑移植自 web ImportBookPanel，保持同规则）→ 分批入库
const api = require('./api')

const VOLUME_RE = /^[ \t　]*(第\s*[0-9零一二三四五六七八九十百千万两]+\s*卷[^\n]{0,40}|卷\s*[0-9零一二三四五六七八九十百千万两]+[^\n]{0,30})[ \t　]*$/
const CHAPTER_RE = /^[ \t　]*(第\s*[0-9零一二三四五六七八九十百千万两]+\s*[章节回话](?:[^，。！？\n]{0,40}[。！？]?)?|第\s*[0-9]{1,4}\s*[、.．]\s*[^\s第][^\n]{0,78}|序章|楔子|引子|尾声|终章|番外[^\n]{0,30}|Chapter\s+\d+[^\n]{0,40}|[0-9]{1,4}[、.．]\s*[^\s第][^\n]{0,38})[ \t　]*$/

function parseVolumes(text) {
  const lines = text.split(/\r\n|\r|\n/)
  const vols = []
  let curVol = null, curCh = null, foundChapter = false
  const ensureVol = () => { if (!curVol) { curVol = { title: '', chapters: [] }; vols.push(curVol) } return curVol }
  for (const line of lines) {
    if (/^[　\t]{1,}/.test(line)) { if (curCh) curCh.body.push(line); continue }
    const t = line.trim()
    if (t && t.length <= 60 && VOLUME_RE.test(t)) { curVol = { title: t, chapters: [] }; vols.push(curVol); curCh = null; continue }
    if (t && t.length <= 95 && CHAPTER_RE.test(t)) {
      foundChapter = true
      curCh = { title: t, body: [] }
      ensureVol().chapters.push(curCh)
      continue
    }
    if (curCh) curCh.body.push(line)
  }
  if (!foundChapter) return []
  return vols
    .filter(v => v.chapters.length)
    .map(v => ({
      title: v.title,
      chapters: v.chapters.map(c => ({ title: c.title, content: c.body.join('\n').replace(/^\n+|\n+$/g, '') }))
    }))
}

function splitBySize(text, size) {
  const paras = text.split(/\n+/)
  const chapters = []
  let buf = [], bufLen = 0
  const flush = () => {
    if (!buf.length) return
    chapters.push({ title: '第 ' + (chapters.length + 1) + ' 部分', content: buf.join('\n') })
    buf = []; bufLen = 0
  }
  for (const p of paras) { buf.push(p); bufLen += p.length; if (bufLen >= size) flush() }
  flush()
  return [{ title: '', chapters }]
}

// 选文件 → 解析 → onParsed(书名, 卷/章统计, 待导入函数)
function pickAndParse() {
  return new Promise((resolve, reject) => {
    wx.chooseMessageFile({
      count: 1, type: 'file',
      extensions: ['txt'],
      success: res => {
        const f = res.tempFiles && res.tempFiles[0]
        if (!f) return reject(new Error('未选择文件'))
        const name = (f.name || '导入的书').replace(/\.txt$/i, '')
        wx.getFileSystemManager().readFile({
          filePath: f.path,
          encoding: 'utf8',
          success: r => {
            const text = r.data || ''
            if (!text.trim()) return reject(new Error('文件是空的'))
            let vols = parseVolumes(text.trim())
            let mode = 'chapters'
            if (!vols.length) { vols = splitBySize(text.trim(), 3000); mode = 'split' }
            const total = vols.reduce((n, v) => n + v.chapters.length, 0)
            resolve({ name, vols, total, mode, raw: text.trim() })
          },
          fail: () => reject(new Error('读文件失败（仅支持 UTF-8 txt）'))
        })
      },
      fail: () => reject(new Error('cancel'))
    })
  })
}

// 分批导入；onProgress(done, total)
async function runImport(bookName, vols, onProgress) {
  const stream = []
  for (const v of vols) for (const ch of v.chapters) stream.push({ volTitle: v.title || '', chapter: ch })
  const BATCH = 500
  let projectId = null, publicId = '', first = true
  for (let i = 0; i < stream.length; i += BATCH) {
    const batchVols = []
    for (const item of stream.slice(i, i + BATCH)) {
      const last = batchVols[batchVols.length - 1]
      if (last && last.title === item.volTitle) last.chapters.push(item.chapter)
      else batchVols.push({ title: item.volTitle, chapters: [item.chapter] })
    }
    const res = first
      ? await api.post('/projects/import', { name: bookName, volumes: batchVols })
      : await api.post('/projects/import/batch', { project_id: projectId, name: bookName, volumes: batchVols })
    const d = res.data || res
    projectId = d.project_id || projectId
    publicId = d.public_id || (d.project && d.project.public_id) || publicId
    first = false
    if (onProgress) onProgress(Math.min(i + BATCH, stream.length), stream.length)
  }
  return { projectId, publicId, total: stream.length }
}

module.exports = { pickAndParse, runImport }
