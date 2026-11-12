# emoji → lucide 矢量图标 替换规范（方案 A 细线条）

依赖已装：`lucide-vue-next`。全局 CSS 已加 `svg.lucide { vertical-align:-0.18em }`（src/style.css），混排对齐不用另调。

## 规则

1. **模板里的 UI emoji** → lucide 组件：`import { Lock } from 'lucide-vue-next'`（script setup 直接 import；Options API 注册到 components）。模板 `<Lock :size="16"/> 码字锁`。
   - 尺寸：带文字的按钮/页签/标签 15；纯图标小按钮 16；工具栏图标钮 18；空态/弹窗大图标 28~32。
   - 不改类名和 DOM 结构，只换 emoji 本身（emoji 与前后文字间的空格：图标组件后留一个空格，前面不留）。
2. **属性字符串**（title / placeholder / aria-label）：直接删掉 emoji 和后面的空格。
3. **JS 字符串**（toast 消息、拼接文案）：直接删掉 emoji+空格（toast 本身有颜色语义）。
4. **配置数组里的 icon 字段**（如 `{ id:'water', icon:'💧' }`）：改成存组件——`import { markRaw } from 'vue'`，`icon: markRaw(GlassWater)`，模板用 `<component :is="a.icon" :size="15"/>`。
5. **KEEP（不动）**：会存进数据库或由用户挑选的 emoji——头像/封面/贴纸选择器数组（AVA_EMOJIS、EMOJIS、charAvatar 默认 🧑、✒️✏️🖊️📚🎨🎬🎵💎 / 🔥❄️🌙☀️⚡🌟🎭🧩 / 🗂️🏷️⭐💡🔒⚔️🏰🗺️ 等 picker 集）、可自选 emoji 的文件夹默认图标、奖牌 🥇🥈🥉（排名字符串）。判断标准：它是"界面控件"还是"用户内容"？拿不准就 KEEP 并在汇报里列出。
6. 完成后 `cd /root/workspace/projects/xiezuojun/web && npx vue-tsc --noEmit 2>&1 | head -30`，确认自己改的文件没有新错误。

## 映射表（UI 图标）

| emoji | lucide | | emoji | lucide |
|---|---|---|---|---|
| ✕ | X | | 🔍 | Search |
| ✓ | Check | | 🎉 | PartyPopper |
| ⚠️ | TriangleAlert | | 📐 | LayoutDashboard |
| ★/⭐ | Star | | 📌/📍 | Pin |
| ✎/✏ | Pencil | | 🧹 | WandSparkles |
| ✍️ | PenLine | | 🎓 | GraduationCap |
| ✚ | Plus | | ⬇ | Download |
| ➜ | ArrowRightToLine | | 🎯 | Target |
| ❝ | Quote | | 🏅 | Medal |
| 📖 | BookOpen | | 🎨 | Palette |
| 📄 | FileText | | 🗂 | SquareKanban |
| 🔥 | Flame | | 💧 | GlassWater |
| 📋 | ListTree | | ☕ | Coffee |
| 🌍 | Globe2 | | ⛔ | Ban |
| 🔒 | Lock | | 👈 | MousePointerClick |
| 👥 | Users | | ✨空白新书 | FilePlus2 |
| 🗑 | Trash2 | | ✨通用闪光 | Sparkles |
| 📡 | WifiOff | | 📥 | FileUp |
| 🧵 | GitBranch | | 📁 | Folder |
| 📱 | Smartphone | | 📂 | FolderOpen |
| 📑 | ListOrdered | | 👤 | User |
| 🏁 | Flag | | 👆 | Pointer |
| 🌙 | Moon | | 🖱️ | MousePointer2 |
| 💡 | Lightbulb | | 🎲 | Dices |
| 🛡 | ShieldCheck | | ✉️ | Mail |
| 📊 | ChartNoAxesColumn | | 🏆 | Trophy |
| 📚 | BookCopy | | 📅 | Calendar |
| 🎙 | Mic | | 🪝 | Anchor |
| 📝 | FilePen | | 🎁 | Gift |
| ✅ | CircleCheckBig | | 📶 | Signal |
| 💬 | MessageCircle | | 🔋 | BatteryFull |
| ✦ | Sparkles | | 💾 | Save |
| ⚡ | Zap | | 🕒 | History |
| 🤝 | Handshake | | 📜 | ScrollText |
| 📢 | Megaphone | | 🏷 | Tag |
| 🧠 | Brain | | 🏰 | Castle |
| ⚙️ | Settings2 | | 🗺 | Map |
| 🔗 | Link2 | | 👏 | HeartHandshake |
| 💛 | Heart | | 🚪 | DoorOpen |
