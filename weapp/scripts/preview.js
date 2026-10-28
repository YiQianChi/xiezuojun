// 生成预览二维码（真机扫=打开开发版小程序）
const ci = require('miniprogram-ci')
const path = require('path')

;(async () => {
  const project = new ci.Project({
    appid: 'wxf49c97a20a588960',
    type: 'miniProgram',
    projectPath: path.resolve(__dirname, '..'),
    privateKeyPath: path.resolve(__dirname, '../keys/private.wxf49c97a20a588960.key'),
    ignores: ['node_modules/**/*', 'scripts/**/*']
  })
  const result = await ci.preview({
    project,
    desc: '预览版：web-view 壳',
    setting: { es6: true, minify: true },
    qrcodeFormat: 'image',
    qrcodeOutputDest: path.resolve(__dirname, '../preview-qr.jpg'),
    onProgressUpdate: () => {}
  })
  console.log('PREVIEW OK', JSON.stringify(result).slice(0, 200))
})().catch(e => { console.error('PREVIEW FAIL:', e.message); process.exit(1) })
