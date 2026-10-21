# 写作菌：Z盘(服务器) → D盘(本地) 工程同步脚本
# 用法：右键"使用 PowerShell 运行"，或在 PowerShell 里 ./sync.ps1
# 排除构建缓存（oh_modules/build/.hvigor/.idea），只同步源码和产物

$src = "Z:\harmony"
$dst = "D:\xiezuojun\harmony"

if (-not (Test-Path $src)) {
    Write-Host "✗ 找不到 $src —— 先确认 Z 盘已映射（\\10.10.30.183\xzj）" -ForegroundColor Red
    exit 1
}

Write-Host "→ 同步 $src → $dst ..." -ForegroundColor Cyan
# /MIR 镜像（删掉本地多余文件）；/XD 排除目录；/XF 排除签名信息文件（本地已有各自的）
robocopy $src $dst /MIR /XD oh_modules build .hvigor .idea /XF "*.hvigor" | Out-Null

if ($LASTEXITCODE -ge 8) {
    Write-Host "✗ 同步失败（robocopy 退出码 $LASTEXITCODE）" -ForegroundColor Red
    exit 1
}

# ohpm 缓存目录被 /MIR 删了也没事，DevEco sync 时会自动重建
Write-Host "✓ 同步完成 → DevEco 打开 D:\xiezuojun\harmony 构建即可" -ForegroundColor Green
Write-Host ""
Write-Host "本次同步的关键文件版本：" -ForegroundColor Gray
Get-ChildItem "$dst\entry\src\main\resources\rawfile\web\index.html" -ErrorAction SilentlyContinue | ForEach-Object { Write-Host ("  rawfile/web  " + $_.LastWriteTime) }
Get-ChildItem "$dst\AppScope\resources\base\media\app_icon.png" -ErrorAction SilentlyContinue | ForEach-Object { Write-Host ("  app_icon     " + $_.LastWriteTime) }
pause
