# 写作菌：服务器 → 本地 D 盘 工程同步（不依赖 Z 盘映射，直接走 UNC 路径）
# 用法：右键"使用 PowerShell 运行"，或：powershell -ExecutionPolicy Bypass -File sync.ps1
# robocopy 退出码 0-7 都是成功；脚本结束会显示关键文件版本

$src = "\\10.10.30.183\xzj\harmony"
$dst = "D:\xiezuojun\harmony"

if (-not (Test-Path $src)) {
    Write-Host "✗ 连不上 $src" -ForegroundColor Red
    Write-Host "  → 先在资源管理器地址栏输入 \\10.10.30.183\xzj 认证一次" -ForegroundColor Yellow
    Write-Host "    （账号 xzjshare，勾「记住我的凭据」），然后重跑本脚本" -ForegroundColor Yellow
    pause; exit 1
}

Write-Host "→ 同步 $src → $dst ..." -ForegroundColor Cyan
robocopy $src $dst /MIR /XD oh_modules build .hvigor .idea /NFL /NDL /NJH | Out-Null

if ($LASTEXITCODE -ge 8) {
    Write-Host "✗ 同步失败（robocopy 退出码 $LASTEXITCODE）" -ForegroundColor Red
    pause; exit 1
}

Write-Host "✓ 同步完成" -ForegroundColor Green
$bp = Get-Content "$dst\entry\build-profile.json5" -TotalCount 2 -ErrorAction SilentlyContinue
if ($bp -match '"apiType"') {
    Write-Host "✓ entry 配置正常（apiType）" -ForegroundColor Green
} else {
    Write-Host "⚠ entry/build-profile.json5 内容异常（没有 apiType），告诉 Hermes" -ForegroundColor Yellow
}
$raw = Get-Item "$dst\entry\src\main\resources\rawfile\web\index.html" -ErrorAction SilentlyContinue
if ($raw) { Write-Host ("  rawfile/web  版本: " + $raw.LastWriteTime) -ForegroundColor Gray }
pause
