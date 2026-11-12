// 目录树/文件夹自选图标：矢量（lucide）。key 存库；老数据的 emoji 渲染时经 EMOJI_TO_KEY 自动映射
import { type Component } from 'vue'
import {
  Folder, FolderOpen, ClipboardList, Users, Globe2, BookOpen, NotebookTabs, FilePen, FileText,
  SquareKanban, Tag, Star, Lightbulb, Lock, Swords, Castle, Map, PenTool, Pencil, PenLine,
  BookCopy, Palette, Clapperboard, Music, Gem, Flame, Snowflake, Moon, Sun, Zap, Sparkles,
  Drama, Puzzle,
} from 'lucide-vue-next'

export const NODE_ICONS: { key: string; icon: Component }[] = [
  { key: 'folder', icon: Folder },
  { key: 'clipboard', icon: ClipboardList },
  { key: 'users', icon: Users },
  { key: 'globe', icon: Globe2 },
  { key: 'book', icon: BookOpen },
  { key: 'tabs', icon: NotebookTabs },
  { key: 'memo', icon: FilePen },
  { key: 'page', icon: FileText },
  { key: 'divider', icon: SquareKanban },
  { key: 'tag', icon: Tag },
  { key: 'star', icon: Star },
  { key: 'bulb', icon: Lightbulb },
  { key: 'lock', icon: Lock },
  { key: 'swords', icon: Swords },
  { key: 'castle', icon: Castle },
  { key: 'map', icon: Map },
  { key: 'nib', icon: PenTool },
  { key: 'pencil', icon: Pencil },
  { key: 'pen', icon: PenLine },
  { key: 'books', icon: BookCopy },
  { key: 'palette', icon: Palette },
  { key: 'movie', icon: Clapperboard },
  { key: 'music', icon: Music },
  { key: 'gem', icon: Gem },
  { key: 'flame', icon: Flame },
  { key: 'snow', icon: Snowflake },
  { key: 'moon', icon: Moon },
  { key: 'sun', icon: Sun },
  { key: 'zap', icon: Zap },
  { key: 'sparkle', icon: Sparkles },
  { key: 'drama', icon: Drama },
  { key: 'puzzle', icon: Puzzle },
]

const KEY_TO_COMP: Record<string, Component> = Object.fromEntries(NODE_ICONS.map(n => [n.key, n.icon]))

// 老数据迁移：库里存过的 emoji → key（含 type 默认表里的 📂）
export const EMOJI_TO_KEY: Record<string, string> = {
  '📁': 'folder', '📂': 'folder-open', '📋': 'clipboard', '👥': 'users', '🌍': 'globe',
  '📖': 'book', '📑': 'tabs', '📝': 'memo', '📄': 'page', '🗂️': 'divider', '🏷️': 'tag',
  '⭐': 'star', '💡': 'bulb', '🔒': 'lock', '⚔️': 'swords', '🏰': 'castle', '🗺️': 'map',
  '✒️': 'nib', '✏️': 'pencil', '🖊️': 'pen', '📚': 'books', '🎨': 'palette', '🎬': 'movie',
  '🎵': 'music', '💎': 'gem', '🔥': 'flame', '❄️': 'snow', '🌙': 'moon', '☀️': 'sun',
  '⚡': 'zap', '🌟': 'sparkle', '🎭': 'drama', '🧩': 'puzzle',
}
KEY_TO_COMP['folder-open'] = FolderOpen

// node.type 没自选图标时的默认
const TYPE_DEFAULT: Record<string, string> = {
  volume: 'folder', folder: 'folder-open', outline: 'clipboard', characters: 'users',
  worldbuilding: 'globe', storyline: 'book', chapter_outline: 'tabs',
  chapter: 'page', note: 'page',
}

/** 存库值（key 或老 emoji）→ 矢量组件；返回 null 表示按文本渲染（角色卡头像等用户内容） */
export function resolveNodeIcon(icon: string | undefined, type: string): Component | null {
  if (icon) return KEY_TO_COMP[icon] || KEY_TO_COMP[EMOJI_TO_KEY[icon] || ''] || null
  return KEY_TO_COMP[TYPE_DEFAULT[type] || 'page'] || null
}
