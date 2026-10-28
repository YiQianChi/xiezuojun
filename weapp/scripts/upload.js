// 上传小程序代码到微信后台（miniprogram-ci）
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
  const result = await ci.upload({
    project,
    version: '0.1.0',
    desc: '首版：web-view 承载 xiezuojun.com + 原生登录页',
    setting: { es6: true, minify: true },
    onProgressUpdate: () => {}
  })
  console.log('UPLOAD OK', JSON.stringify(result))
})().catch(e => { console.error('UPLOAD FAIL:', e.message); process.exit(1) })
