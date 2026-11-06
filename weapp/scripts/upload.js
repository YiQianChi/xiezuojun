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
    version: '0.6.7',
    desc: '退出登录同步清网页登录态(经 /logout)；配合网页工作台/编辑器更新',
    setting: { es6: true, minify: true },
    onProgressUpdate: () => {}
  })
  console.log('UPLOAD OK', JSON.stringify(result))
})().catch(e => { console.error('UPLOAD FAIL:', e.message); process.exit(1) })
