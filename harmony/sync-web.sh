#!/bin/bash
# 构建鸿蒙壳用的前端产物：注入 API 绝对地址 + 拷进 rawfile
# 用法：在仓库根目录 ./harmony/sync-web.sh
set -e
cd "$(dirname "$0")/../web"
npx vite build --mode harmony --outDir dist-harmony
cd ..
rm -rf harmony/entry/src/main/resources/rawfile/web
mkdir -p harmony/entry/src/main/resources/rawfile/web
cp -r web/dist-harmony/* harmony/entry/src/main/resources/rawfile/web/
echo "✓ 前端产物已同步到 harmony/entry/src/main/resources/rawfile/web/"
