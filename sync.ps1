# 写作菌：服务器 → 本地 D 盘 工程同步（不依赖 Z 盘映射，直接走 UNC 路径）
# 用法：右键"使用 PowerShell 运行"，或：powershell -ExecutionPolicy Bypass -File sync.ps1
# 可带参数只同步一个模块：sync.ps1 weapp / sync.ps1 harmony / sync.ps1 all（默认 all）
# robocopy 退出码 0-7 都是成功；脚本结束会显示关键文件版本

$mod = if ($args.Count -gt 0) { $args[0] } else { "all" }
$root = "\\10.10.30.183\xzj"

if (-not (Test-Path $root)) {
    Write-Host "✗ 连不上 $root" -ForegroundColor Red
    Write-Host "  → 先在资源管理器地址栏输入 \\10.10.30.183\xzj 认证一次" -ForegroundColor Yellow
    Write-Host "    （账号 xzjshare，勾「记住我的凭据」），然后重跑本脚本" -ForegroundColor Yellow
    pause; exit 1
}

function Sync-Dir($name) {
    $src = Join-Path $root $name
    $dst = "D:\xiezuojun\$name"
    if (-not (Test-Path $src)) { Write-Host "✗ 服务器上没有 $src" -ForegroundColor Red; return $false }
    Write-Host "→ 同步 $src → $dst ..." -ForegroundColor Cyan
    robocopy $src $dst /MIR /XD oh_modules build .hvigor .idea node_modules keys /NFL /NDL /NJH | Out-Null
    if ($LASTEXITCODE -ge 8) { Write-Host "✗ $name 同步失败（robocopy 退出码 $LASTEXITCODE）" -ForegroundColor Red; return $false }
    Write-Host "✓ $name 同步完成" -ForegroundColor Green
    return $true
}

$ok = $true
if ($mod -eq "harmony" -or $mod -eq "all") { $ok = (Sync-Dir "harmony") -and $ok }
if ($mod -eq "weapp" -or $mod -eq "all")  { $ok = (Sync-Dir "weapp") -and $ok }

if (-not $ok) { pause; exit 1 }

# 模块自检
$wx = Get-Content "D:\xiezuojun\weapp\src\pages\login\login.wxml" -TotalCount 8 -ErrorAction SilentlyContinue
if ($wx -match 'btn-wx') {
    Write-Host "✓ weapp 登录页含微信一键登录按钮（新版）" -ForegroundColor Green
} elseif (Test-Path "D:\xiezuojun\weapp") {
    Write-Host "⚠ weapp 登录页没有微信登录按钮（旧版？），告诉 Hermes" -ForegroundColor Yellow
}
$bp = Get-Content "D:\xiezuojun\harmony\entry\build-profile.json5" -TotalCount 2 -ErrorAction SilentlyContinue
if ($bp -match '"apiType"') {
    Write-Host "✓ harmony entry 配置正常（apiType）" -ForegroundColor Green
}
pause
