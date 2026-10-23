<template>
      <div class="editor-page" :class="{ 'focus-mode': focusMode }" :style="displayVars">
    <!-- ponytail-focus: 专注模式底部字数激励（C 方案） -->
    <div v-if="focusMode" class="focus-today-words">今天 +{{ todayWords }} 字</div>
    <!-- 顶部栏 -->
    <EditorHeader
      :project-name="projectName"
      :current-doc-name="currentDocDisplayName"
      :mode="mode"
      :active-doc-id="activeDocId"
      :saving-version="savingVersion"
      :focus-mode="focusMode"
      :reference-open="referencePanelOpen"
      @back="goBack"
      @switch-mode="switchMode"
      @save-version="openSaveVersionModal"
      @open-history="openVersionModal"
      @toggle-focus="focusMode = !focusMode"
      @toggle-reference="referencePanelOpen = !referencePanelOpen"
      @toggle-sidebar="mobileSidebarOpen = !mobileSidebarOpen"
    />

    <div class="editor-body">
      <!-- 移动端抽屉遮罩 -->
      <div v-if="mobileSidebarOpen" class="sidebar-scrim" @click="mobileSidebarOpen = false"></div>
      <!-- 左侧文档树（移动端=抽屉） -->
      <EditorSidebar
        :class="{ 'drawer-open': mobileSidebarOpen }"
        class="doc-tree"
        :branches="branches"
        :active-branch-id="activeBranchId"
        :documents="documents"
        :setting-root-nodes="settingRootNodes"
        :article-root-nodes="articleRootNodes"
        :active-doc-id="activeDocId"
        :loading-docs="loadingDocs"
        @switch-branch="switchBranch"
        @init-default="initDefaultStructure"
        @select-doc="openDoc"
        @context-menu="openDocContextMenu"
        @drop-node="handleDropNode"
        @add-doc-root="createDocAtRoot"
        @add-setting="createSettingAtRoot"
        @add-article="createVolumeAtRoot"
        @add-doc-in="createDocInFolder"
        @cycle-status="handleCycleStatus"
      />

      <!-- 编辑器 -->
      <div class="editor-main" :class="{ 'ai-panel-open': contextPanelOpen }" v-show="mode === 'write'">
        <div v-if="draftBanner && draftBanner.docId === activeDocId" class="draft-banner" style="display:flex;align-items:center;justify-content:space-between;padding:8px 16px;background:#fef3cd;border-bottom:1px solid #f0d57a;font-size:14px;color:#856404;">
          <span>📝 检测到未保存的本地草稿，是否恢复？</span>
          <div class="draft-actions" style="display:flex;gap:8px;">
            <button class="btn primary" @click="restoreDraft">恢复</button>
            <button class="btn" @click="discardDraft">丢弃</button>
          </div>
        </div>
        <div v-if="compareActive" class="compare-view">
          <div class="compare-topbar">
            <div>
              <strong>对比当前版本</strong>
              <span>{{ selectedVersion?.label || '历史版本' }}</span>
            </div>
            <div class="compare-actions">
              <label class="sync-toggle">
                <span>同步滚动</span>
                <input v-model="compareSyncScroll" type="checkbox" />
                <i></i>
              </label>
              <button class="btn" @click="closeCompareView">退出对比</button>
              <button class="btn primary" @click="applyCompareCurrent">保存当前修改</button>
            </div>
          </div>
          <div class="compare-panes">
            <div class="compare-pane old">
              <div class="pane-title">历史版本（只读）</div>
              <div class="diff-lines" ref="compareOldRef" @scroll="syncCompareScroll('old')">
                <div
                  v-for="(row, index) in compareRows"
                  :key="'old-' + index"
                  class="diff-line"
                  :class="row.type"
                >
                  {{ row.oldText }}
                </div>
              </div>
            </div>
            <div class="compare-pane current">
              <div class="pane-title">当前版本（可修改）</div>
              <textarea ref="compareCurrentRef" v-model="compareCurrentContent" @input="onCompareCurrentInput" @scroll="syncCompareScroll('current')"></textarea>
            </div>
          </div>
        </div>
        <div v-else-if="!activeDocId" class="editor-empty">
          <p>👈 请选择或新建一个章节</p>
        </div>
        <div v-else class="editor-wrapper" @contextmenu.prevent="openContextMenu">
          <div class="tiptap-container">
            <editor-content v-if="editor" :editor="editor" />
          </div>
        </div>
        <SearchPanel
          :visible="searchVisible"
          :scope="searchScope"
          :query="searchQuery"
          :total="searchTotal"
          :active-index="searchActiveIndex"
          :loading="searchLoading"
          :project-matches="searchProjectMatches"
          :replace-value="replaceText"
          @close="closeSearch"
          @next="searchNext"
          @prev="searchPrev"
          @update:query="onSearchQueryUpdate"
          @update:scope="onSearchScopeChange"
          @update:replace-value="replaceText = $event"
          @jump-project="onJumpProject"
          @replace-one="() => { replaceOne(); autoSave() }"
          @replace-all="() => { const n = replaceAll(); autoSave(); toast.success(`已替换 ${n} 处`) }"
        />

        <!-- AI 参考面板（写作模式） -->
        <AIContextPanel
          :open="contextPanelOpen"
          :project-id="projectId"
          :used-settings="usedSettings"
          @update:open="contextPanelOpen = $event"
          @jump="jumpToSetting"
        />

        <!-- B5: 分屏参考面板 -->
        <ReferencePanel
          v-show="referencePanelOpen"
          :documents="documents"
          :project-id="projectId"
          @close="referencePanelOpen = false"
        />
      </div>

      <!-- 图谱浮层（编辑器内右下角） -->
      <div class="graph-fab">
        <button class="graph-fab-btn" @click="graphOpen = !graphOpen" :title="graphOpen ? '收起图谱' : '关系图谱'">
          <span v-if="!graphOpen">🔗</span>
          <span v-else>✕</span>
        </button>
        <div class="graph-fab-panel" :class="{ maximized: graphMaximized }" v-show="graphOpen" @click.stop>
          <button class="graph-resize-btn" @click="graphMaximized = !graphMaximized" :title="graphMaximized ? '缩小' : '放大'">
            <svg v-if="!graphMaximized" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M8 3H5a2 2 0 0 0-2 2v3"/><path d="M16 3h3a2 2 0 0 1 2 2v3"/><path d="M8 21H5a2 2 0 0 1-2-2v-3"/><path d="M16 21h3a2 2 0 0 0 2-2v-3"/></svg>
            <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M8 3v3a2 2 0 0 1-2 2H3"/><path d="M21 8h-3a2 2 0 0 1-2-2V3"/><path d="M3 16h3a2 2 0 0 1 2 2v3"/><path d="M16 21v-3a2 2 0 0 1 2-2h3"/></svg>
          </button>
          <RelationshipGraph
            :project-id="String(projectId)"
            :chapters="chapterList"
            :current-chapter="currentChapterIndex"
          />
        </div>
      </div>

      <!-- 计划模式 -->
      <PlanView v-if="mode === 'plan'" :project-name="projectName" :project-id="projectId" @jump-to-doc="jumpToSetting" />

      <!-- 审核模式 -->
      <ReviewView v-if="mode === 'review'" :project-id="projectId" @jump-to-source="handleReviewJump" @ai-fix="handleReviewAiFix" />

      <!-- AI 选区工具栏（选择时） -->
      <div
        v-if="aiToolbar.visible"
        class="ai-toolbar"
        :style="{ left: aiToolbar.x + 'px', top: aiToolbar.y + 'px' }"
      >
        <button @mousedown.prevent @click="prepareAIAction('continue')">续写</button>
        <button @mousedown.prevent @click="prepareAIAction('rewrite')">改写</button>
        <button @mousedown.prevent @click="prepareAIAction('polish')">润色</button>
        <button @mousedown.prevent @click="prepareAIAction('expand')">扩写</button>
      </div>

      <!-- AI 底部固定栏（方向输入 / 生成中 / 结果预览） -->
      <div v-if="aiPhase !== 'idle'" class="ai-bottom-bar" :class="'phase-' + aiPhase">
        <!-- 方向输入 -->
        <template v-if="aiPhase === 'inputting'">
          <button class="ai-icon-btn" @click="cancelInputting" aria-label="取消">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 6l12 12M18 6L6 18"/></svg>
          </button>
          <div class="ai-input-wrap">
            <input
              ref="aiDirectionInputRef"
              v-model="aiDirection"
              class="ai-direction-input"
              type="text"
              :placeholder="pendingAction === 'continue' ? '续写方向（可选）' : pendingAction === 'rewrite' ? '改写方向（可选）' : pendingAction === 'polish' ? '润色方向（可选）' : '扩写方向（可选）'"
              @keydown.enter.prevent="executeAIAction"
              @keydown.escape.prevent="cancelInputting"
            />
          </div>
          <button class="ai-icon-btn primary" @click="executeAIAction" aria-label="生成">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linejoin="round"><polygon points="6 4 20 12 6 20 6 4" fill="currentColor"/></svg>
          </button>
        </template>
        <!-- 生成中 -->
        <template v-if="aiPhase === 'generating'">
          <span class="ai-bottom-hint">生成中…</span>
          <button class="ai-icon-btn danger" @click="stopAIGeneration" aria-label="停止">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><rect x="5" y="5" width="14" height="14" rx="2"/></svg>
          </button>
        </template>
        <!-- 结果预览 -->
        <template v-if="aiPhase === 'preview'">
          <span class="ai-bottom-hint">已生成</span>
          <button class="ai-icon-btn" @click="regenerateAIResult" :disabled="aiLoading" aria-label="重新生成">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 11-3-6.7L21 8M21 3v5h-5"/></svg>
          </button>
          <button class="ai-icon-btn danger" @click="discardAIResult" aria-label="丢弃">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6"/></svg>
          </button>
          <button class="ai-icon-btn success" @click="acceptAIResult" aria-label="采用">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
          </button>
        </template>
      </div>

      <!-- 编辑器右键菜单 -->
      <div
        v-if="contextMenu.visible"
        class="context-menu"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
      >
        <div class="ctx-highlight-row">
          <span
            v-for="c in highlightColors"
            :key="c"
            class="hl-color-dot"
            :style="{ background: c }"
            @click="contextAction('highlight', c)"
          ></span>
          <span class="hl-color-dot hl-clear" @click="contextAction('clearHighlight')"></span>
        </div>
        <div class="ctx-sep"></div>
        <div @click="contextAction('copy')">复制</div>
        <div @click="contextAction('cut')">剪切</div>
        <div @click="contextAction('paste')">粘贴</div>
        <div class="ctx-sep"></div>
        <div class="ctx-strike-row">
          <span class="ctx-icon-btn" @click="contextAction('strike')" title="删除线">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="4" y1="12" x2="20" y2="12"/></svg>
          </span>
          <span class="ctx-icon-btn" @click="contextAction('unstrike')" title="取消删除线">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="4" y1="12" x2="20" y2="12"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </span>
        </div>
        <div class="ctx-sep"></div>
        <div @click="contextAction('oneClickFormat')">🧹 一键排版</div>
      </div>

      <!-- 文档树右键菜单 -->
      <div
        v-if="docContextMenu.visible"
        class="context-menu"
        :style="{ left: docContextMenu.x + 'px', top: docContextMenu.y + 'px' }"
      >
        <template v-if="docContextMenu.doc">
          <div @click="renameDoc">重命名</div>
          <div v-if="!isFolderNode(docContextMenu.doc)" class="submenu-trigger">
            标记为 ▸
            <div class="context-submenu">
              <div @click="handleDocStatusChange(docContextMenu.doc, 'draft')"><span class="status-dot-mini" style="background:#8e8e93"></span> 草稿</div>
              <div @click="handleDocStatusChange(docContextMenu.doc, 'revising')"><span class="status-dot-mini" style="background:#f5a623"></span> 修改中</div>
              <div @click="handleDocStatusChange(docContextMenu.doc, 'final')"><span class="status-dot-mini" style="background:#34c759"></span> 定稿</div>
            </div>
          </div>
          <div @click="openFolderProps">属性</div>
          <div @click="deleteDoc" class="danger">删除</div>
        </template>
        <template v-else>
          <div @click="newDocUnder('chapter')">新建章节</div>
          <div @click="newDocUnder('volume')">新建卷</div>
        </template>
      </div>
    </div>

    <!-- 底部状态栏 -->
    <div class="status-bar">
      <span class="status-item">{{ wordCount }} 字</span>
      <span v-if="isOffline" class="status-item offline-indicator" title="当前离线，内容已自动存在本机，恢复网络后会自动上传">📡 离线</span>
      <span class="status-item">{{ saveStatus }}</span>
      <button class="status-item speech-btn" :class="{ speaking: isSpeaking }" @click="toggleSpeech" :disabled="ttsLoading" :title="isSpeaking ? '停止朗读' : '朗读（选区优先）'">
        {{ ttsLoading ? '⏳' : isSpeaking ? '⏹' : '🔊' }}
      </button>
      <select v-model="selectedVoice" class="status-item voice-select" title="朗读音色">
        <option value="zh-CN">默认女声</option>
        <option v-for="v in voiceList" :key="v.voice" :value="v.voice">{{ v.name }}</option>
      </select>
      <button class="status-item scene-nav-btn" @click="jumpScene(-1)" title="上一个场景 (Alt+↑)">⬆️</button>
      <button class="status-item scene-nav-btn" @click="jumpScene(1)" title="下一个场景 (Alt+↓)">⬇️</button>

      <!-- 显示设置：字号/行距/字距/底部留白 -->
      <DisplaySettings :overrides="displayOverrides" @update="onDisplaySetting" />

      <!-- 工具箱：低频功能收起来 -->
      <div class="toolbox-trigger" :class="{ open: toolboxOpen }">
        <button class="memory-btn" @click.stop="toolboxOpen = !toolboxOpen" title="工具箱">
          <span v-if="!toolboxOpen">⋯</span>
          <span v-else>✕</span>
        </button>
        <div class="toolbox-popup" v-show="toolboxOpen" @click.stop>
          <div class="toolbox-grid">
            <button class="tool-btn" :class="{ active: memoryOpen }" @click="memoryOpen = !memoryOpen; toolboxOpen = false">
              <span class="tool-icon">🧠</span>
              <span class="tool-label">记忆</span>
            </button>
            <button class="tool-btn" :class="{ active: threadOpen }" @click="threadOpen = !threadOpen; toolboxOpen = false">
              <span class="tool-icon">🧵</span>
              <span class="tool-label">伏笔</span>
            </button>
            <button class="tool-btn" :class="{ active: trashOpen }" @click="trashOpen = !trashOpen; toolboxOpen = false">
              <span class="tool-icon">🗑</span>
              <span class="tool-label">回收站</span>
            </button>
            <button class="tool-btn" :class="{ active: statsOpen }" @click="statsOpen = !statsOpen; toolboxOpen = false">
              <span class="tool-icon">📊</span>
              <span class="tool-label">统计</span>
            </button>
            <button class="tool-btn" :class="{ active: storyboardOpen }" @click="storyboardOpen = !storyboardOpen; toolboxOpen = false">
              <span class="tool-icon">🎴</span>
              <span class="tool-label">大纲卡</span>
            </button>
            <button class="tool-btn" :class="{ active: chatOpen }" @click="chatOpen = !chatOpen; toolboxOpen = false">
              <span class="tool-icon">💬</span>
              <span class="tool-label">AI对话</span>
            </button>
            <button class="tool-btn" :class="{ active: graphOpen }" @click="graphOpen = !graphOpen; toolboxOpen = false">
              <span class="tool-icon">🔗</span>
              <span class="tool-label">关系图</span>
            </button>
            <button class="tool-btn" @click="oneClickFormat(); toolboxOpen = false">
              <span class="tool-icon">🧹</span>
              <span class="tool-label">一键排版</span>
            </button>
            <button class="tool-btn" @click="helpOpen = true; toolboxOpen = false">
              <span class="tool-icon">⌨️</span>
              <span class="tool-label">快捷键</span>
            </button>
            <button class="tool-btn" @click="replayOnboarding">
              <span class="tool-icon">🎓</span>
              <span class="tool-label">新手引导</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <BuddyWindow :current-word-count="wordCount" />

    <!-- 工具箱面板：记忆/伏笔/回收站/统计/AI对话（从状态栏触发，弹窗位置统一） -->
    <div v-if="memoryOpen" class="panel-popup memory-popup" @click.stop>
      <div class="panel-head"><span>🧠 记忆</span><button @click="memoryOpen = false">✕</button></div>
      <MemoryPanel :project-id="String(projectId)" />
    </div>
    <div v-if="threadOpen" class="panel-popup thread-popup" @click.stop>
      <div class="panel-head"><span>🧵 伏笔追踪</span><button @click="threadOpen = false">✕</button></div>
      <ThreadPanel :project-id="String(projectId)" />
    </div>
    <div v-if="trashOpen" class="panel-popup trash-popup" @click.stop>
      <div class="panel-head"><span>🗑 回收站</span><button @click="trashOpen = false">✕</button></div>
      <TrashPanel ref="trashPanelRef" :project-id="String(projectId)" @restored="onTrashRestored" />
    </div>
    <div v-if="statsOpen" class="panel-popup stats-popup" @click.stop>
      <div class="panel-head"><span>📊 写作统计</span><button @click="statsOpen = false">✕</button></div>
      <WritingStatsPanel :project-id="projectId" />
    </div>
    <div v-if="chatOpen" class="panel-popup chat-popup" @click.stop>
      <div class="panel-head"><span>💬 AI 对话</span><button @click="chatOpen = false">✕</button></div>
      <AIChatPanel :project-id="projectId" />
    </div>

    <!-- E5: 快捷键帮助弹窗 -->
    <Teleport to="body">
      <div v-if="helpOpen" class="help-overlay" @click="helpOpen = false">
        <div class="help-modal" @click.stop>
          <div class="help-head">
            <span>⌨️ 快捷键</span>
            <button @click="helpOpen = false" class="help-close">✕</button>
          </div>
          <div class="help-body">
            <div class="help-row"><kbd>Ctrl+S</kbd><span>保存</span></div>
            <div class="help-row"><kbd>Ctrl+F</kbd><span>搜索（本章）</span></div>
            <div class="help-row"><kbd>Ctrl+Shift+F</kbd><span>搜索（全项目）</span></div>
            <div class="help-row"><kbd>Ctrl+P</kbd><span>切换模式（构思→写作）</span></div>
            <div class="help-row"><kbd>Ctrl+Shift+Z</kbd><span>专注模式</span></div>
            <div class="help-row"><kbd>Alt+↑</kbd><span>上一个场景</span></div>
            <div class="help-row"><kbd>Alt+↓</kbd><span>下一个场景</span></div>
            <div class="help-row"><kbd>Ctrl+/</kbd><span>本帮助</span></div>
            <div class="help-row"><kbd>Esc</kbd><span>关闭弹窗/退出专注</span></div>
          </div>
          <div class="help-tip">场景分隔符：在段落开头输入 ✧ 或 --- 即可创建场景分隔</div>
        </div>
      </div>
    </Teleport>

    <!-- 章节大纲卡片视图（写作模式底部浮层） -->
    <div v-if="storyboardOpen && mode === 'write'" class="storyboard-drawer">
      <StoryboardView
        :documents="documents"
        :active-doc-id="activeDocId"
        @select-doc="openDoc"
        @reorder="handleStoryboardReorder"
      />
    </div>

    <!-- 新建文档弹窗 -->
    <CreateDocModal
      v-model="showCreateDocModal"
      :doc-type="createDocType"
      :folder-type-options="createDocFolderTypeOptions"
      @create="onCreateDocConfirm"
    />

    <!-- 首次进入：初始化向导（第一步首章名称 → 第二步作品相关文件） -->
    <div v-if="showSettingsInit" class="modal-overlay" @click="skipSettingsInit">
      <div class="settings-init-modal" @click.stop>
        <div class="si-steps">第 {{ settingsInitStep }} 步 / 共 2 步</div>

        <template v-if="settingsInitStep === 1">
          <h3>第一章叫什么？</h3>
          <p class="si-desc">只写章节名就好（如「雨夜」），第几章系统按位置自动排。</p>
          <input v-model="firstChapterNameInput" class="si-input" placeholder="雨夜" @keyup.enter="settingsInitStep = 2" />
          <div class="si-actions">
            <button class="si-btn" @click="skipSettingsInit">跳过</button>
            <button class="si-btn primary" @click="settingsInitStep = 2">下一步</button>
          </div>
        </template>

        <template v-else>
          <h3>要为这本书建哪些作品相关文件？</h3>
          <p class="si-desc">不建也没关系，以后随时可以在左侧「＋ 新建设定」补。</p>
          <div class="si-options">
            <label v-for="s in settingsInitOptions" :key="s.value" class="si-option">
              <input type="checkbox" v-model="s.checked" /> {{ s.label }}
            </label>
          </div>
          <div class="si-actions">
            <button class="si-btn" @click="settingsInitStep = 1">上一步</button>
            <button class="si-btn primary" @click="confirmSettingsInit" :disabled="settingsInitCreating">
              {{ settingsInitCreating ? '创建中…' : '完成' }}
            </button>
          </div>
        </template>
      </div>
    </div>

    <!-- 文件夹属性弹窗 -->
    <FolderPropsModal
      v-model="showFolderPropsModal"
      :doc="folderPropsDoc"
      @save="onFolderPropsSave"
    />

    <!-- 保存版本名称弹窗 -->
    <SaveVersionModal
      v-model="showSaveVersionModal"
      :saving="savingVersion"
      @save="onSaveVersionConfirm"
    />

    <!-- 版本历史弹窗 -->
    <VersionHistoryModal
      v-model="showVersionModal"
      :loading="loadingVersions"
      :restoring="restoringVersion"
      :versions="versions"
      :selected-version="selectedVersion"
      @select="selectVersion"
      @compare="startCompareWithCurrent"
      @restore="restoreSelectedVersion"
    />

    <!-- 专注模式退出提示 -->
    <div class="focus-exit-hint" @click="focusMode = false">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M8 3v3a2 2 0 0 1-2 2H3"/><path d="M21 8h-3a2 2 0 0 1-2-2V3"/><path d="M3 16h3a2 2 0 0 1 2 2v3"/><path d="M16 21v-3a2 2 0 0 1 2-2h3"/></svg>
      <span>退出专注 (Esc)</span>
    </div>

    <!-- 重命名弹窗 -->
    <div v-if="renameModal" class="confirm-overlay" @click="renameModal = null">
      <div class="confirm-modal-box rename-box" @click.stop>
        <h3>重命名</h3>
        <input
          ref="renameInputRef"
          v-model="renameModal.title"
          class="rename-input"
          @keydown.enter.prevent="confirmRename"
          @keydown.escape.prevent="renameModal = null"
        />
        <div class="confirm-buttons">
          <button class="btn-cancel" @click="renameModal = null">取消</button>
          <button class="btn-confirm" @click="confirmRename">确认</button>
        </div>
      </div>
    </div>

    <!-- 通用确认弹窗 -->
    <div v-if="confirmModal" class="confirm-overlay" @click="confirmModal = null">
      <div class="confirm-modal-box" @click.stop>
        <div class="confirm-icon">⚠️</div>
        <h3>{{ confirmModal.title }}</h3>
        <p class="confirm-desc">{{ confirmModal.desc }}</p>
        <div class="confirm-buttons">
          <button class="btn-cancel" @click="confirmModal = null">取消</button>
          <button class="btn-confirm" @click="confirmModal.action()">确认</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { EditorContent } from '@tiptap/vue-3'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import Highlight from '@tiptap/extension-highlight'
import { Table } from '@tiptap/extension-table'
import { TableRow } from '@tiptap/extension-table-row'
import { TableCell } from '@tiptap/extension-table-cell'
import { TableHeader } from '@tiptap/extension-table-header'
// ponytail: 4 fixed highlight colors, no config needed
const highlightColors = ['#ffeb3b', '#4caf50', '#2196f3', '#ff5252']
import { documentApi, branchApi, projectApi, ttsApi } from '../core/api'
import { chapterDisplayTitle, stripChapterPrefix, chapterOrdinal, numToChinese } from '../core/chapterTitle'
import SearchPanel from '../components/SearchPanel.vue'
import PlanView from '../components/PlanView.vue'
import ReviewView from '../components/ReviewView.vue'
import EditorHeader from '../components/editor/EditorHeader.vue'
import EditorSidebar from '../components/editor/EditorSidebar.vue'
import AIContextPanel from '../components/editor/AIContextPanel.vue'
import AIChatPanel from '../components/editor/AIChatPanel.vue'
import ReferencePanel from '../components/editor/ReferencePanel.vue'
import MemoryPanel from '../components/editor/MemoryPanel.vue'
import RelationshipGraph from '../components/editor/RelationshipGraph.vue'
import ThreadPanel from '../components/editor/ThreadPanel.vue'
import TrashPanel from '../components/editor/TrashPanel.vue'
import StoryboardView from '../components/editor/StoryboardView.vue'
import WritingStatsPanel from '../components/editor/WritingStatsPanel.vue'
import BuddyWindow from '../components/editor/BuddyWindow.vue'
import DisplaySettings from '../components/editor/DisplaySettings.vue'
import CreateDocModal from '../components/editor/CreateDocModal.vue'
import FolderPropsModal from '../components/editor/FolderPropsModal.vue'
import SaveVersionModal from '../components/editor/SaveVersionModal.vue'
import VersionHistoryModal from '../components/editor/VersionHistoryModal.vue'
import { useEditorSearch, createSearchHighlightPlugin } from '../composables/useEditorSearch'
import { useAIToolbar, AIMark, AIStrikethrough } from '../composables/useAIToolbar'
import { useVersionManager } from '../composables/useVersionManager'
import { toast } from '../composables/useToast'

const route = useRoute()
const router = useRouter()

const projectId = Number(route.params.projectId)
try { localStorage.setItem('xzj-active-project', String(projectId)) } catch {}

const editor = shallowRef<any>(null)
const documents = ref<any[]>([])
const branches = ref<any[]>([])
const activeBranchId = ref<number | null>(null)
const activeDocId = ref<number | null>(null)
const currentDocName = ref('')
// 顶栏章节名和侧栏一致：第N章 + 作者写的名字（序号自动）
const currentDocDisplayName = computed(() => {
  const d = documents.value.find((x: any) => x.id === activeDocId.value)
  if (d && d.type === 'chapter') return chapterDisplayTitle(d, documents.value)
  return currentDocName.value
})
const projectName = ref('')
const loadingDocs = ref(false)
const wordCount = ref(0)
const saveStatus = ref('')
const draftBanner = ref<{ docId: number; text: string } | null>(null)
const isSpeaking = ref(false)
const ttsLoading = ref(false)
let currentAudio: HTMLAudioElement | null = null

async function toggleSpeech() {
  if (isSpeaking.value) {
    currentAudio?.pause()
    currentAudio = null
    isSpeaking.value = false
    return
  }
  let text = ''
  if (editor.value) {
    const { from, to } = editor.value.state.selection
    text = from < to ? editor.value.state.doc.textBetween(from, to, '\n') : getEditorText()
  }
  if (!text.trim()) { toast.info('没有可朗读的内容'); return }
  // ponytail: Edge TTS 限 5000 字，超出截断
  if (text.length > 5000) {
    text = text.slice(0, 5000)
    toast.info('内容超过5000字，已截断朗读')
  }
  ttsLoading.value = true
  try {
    const res = await ttsApi.synthesize(text, selectedVoice.value)
    const blob = new Blob([res.data], { type: 'audio/mpeg' })
    const url = URL.createObjectURL(blob)
    currentAudio = new Audio(url)
    currentAudio.onended = () => { isSpeaking.value = false; URL.revokeObjectURL(url) }
    currentAudio.onerror = () => { isSpeaking.value = false; URL.revokeObjectURL(url) }
    await currentAudio.play()
    isSpeaking.value = true
  } catch (e) {
    toast.error('朗读失败，请重试')
    console.error('TTS error', e)
  } finally {
    ttsLoading.value = false
  }
}

// E2: 加载 TTS 音色列表
const voiceList = ref<any[]>([])
async function loadVoices() {
  try {
    const res = await ttsApi.voices()
    voiceList.value = res.data?.data || []
  } catch { /* ponytail: 语音列表加载失败不影响朗读 */ }
}

// E1: 场景分隔导航 — 查找 ✧/---/*** 开头的段落位置
const SCENE_SEP = /^(✧{1,}|={3,}|\*{3,}|-{3,}|※{1,}|◇{1,})/
function findScenePositions(): number[] {
  if (!editor.value) return []
  const positions: number[] = []
  editor.value.state.doc.descendants((node: any, pos: number) => {
    if (node.isText) {
      const text = node.text || ''
      if (SCENE_SEP.test(text.trim())) positions.push(pos)
    }
    return true
  })
  return positions
}
function jumpScene(dir: 1 | -1) {
  if (!editor.value) return
  const positions = findScenePositions()
  if (positions.length === 0) { toast.info('未找到场景分隔符（✧ 或 ---）'); return }
  const sel = editor.value.state.selection.from
  const target = dir === 1
    ? positions.find(p => p > sel + 1)
    : [...positions].reverse().find(p => p < sel - 1)
  if (target === undefined) { toast.info(dir === 1 ? '已是最后一个场景' : '已是第一个场景'); return }
  editor.value.chain().setTextSelection(target).run()
  editor.value.view.coordsAtPos(target)
  const coords = editor.value.view.coordsAtPos(target)
  const el = document.elementFromPoint(coords.left, coords.top)?.parentElement
  if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

const contextMenu = ref({ visible: false, x: 0, y: 0 })
// ponytail: 右键时编辑器会失焦丢选区，打开菜单前缓存
let cachedSelection: { text: string; start: number; end: number } | null = null
const docContextMenu = ref({ visible: false, x: 0, y: 0, doc: null as any })
const showCreateDocModal = ref(false)
const newDocName = ref('')
const createDocType = ref<string>('chapter')
const createDocParent = ref<number | null>(null)
const createDocFolderTypeOptions = computed(() => {
  // 只有「新建设定」弹窗需要选类型；卷/章节的弹窗都是单一用途
  if (createDocType.value !== 'setting') return undefined
  return [
    { value: 'outline', label: '📋 大纲' },
    { value: 'characters', label: '👥 人物' },
    { value: 'worldbuilding', label: '🌍 世界观' },
    { value: 'storyline', label: '📖 故事主线' },
    { value: 'chapter_outline', label: '📑 章节大纲' },
  ]
})

// 文件夹属性弹窗
const showFolderPropsModal = ref(false)
const folderPropsDoc = ref<any>(null)

let saveTimer: any = null
const isSettingContent = ref(false)

// 版本管理 composable
const {
  // 版本状态
  versions, selectedVersion, loadingVersions, savingVersion, restoringVersion,
  showVersionModal, showSaveVersionModal,
  // 版本操作
  openSaveVersionModal, onSaveVersionConfirm, openVersionModal,
  selectVersion, restoreSelectedVersion,
  // 对比状态
  compareActive, compareCurrentContent, compareSyncScroll,
  compareRows, compareOldRef, compareCurrentRef,
  // 对比操作
  startCompareWithCurrent, closeCompareView, onCompareCurrentInput, syncCompareScroll, applyCompareCurrent,
} = useVersionManager({
  editor, activeDocId, isSettingContent, wordCount, saveStatus,
  autoSave, loadDocs, getEditorText, editorContentFromText,
  hideAIToolbar: () => { aiToolbar.value.visible = false },
  cancelSaveTimer: () => { if (saveTimer) { clearTimeout(saveTimer); saveTimer = null } },
})
// template refs — used via ref="compareOldRef" / ref="compareCurrentRef" in <template>
void compareOldRef; void compareCurrentRef
let selectionHandler: (() => void) | null = null

// 搜索面板状态 & 逻辑（本章 / 全项目 两种模式）— composable
const {
  searchVisible,
  searchScope,
  searchQuery,
  searchTotal,
  searchActiveIndex,
  searchLoading,
  searchProjectMatches,
  replaceText,
  openSearch,
  closeSearch,
  onSearchQueryUpdate,
  onSearchScopeChange,
  searchNext,
  searchPrev,
  onJumpProject,
  applyEditorSearch,
  replaceOne,
  replaceAll,
} = useEditorSearch({
  editor,
  projectId,
  activeDocId,
  documents,
  openDoc: (...args) => openDoc(...args),
})

// 根节点（parent_id 为空的）
const rootNodes = computed(() => {
  return documents.value
    .filter(d => !d.parent_id)
    .sort((a, b) => (a.sort_order || 0) - (b.sort_order || 0))
})

// 关系图时间轴：正文目录下的章节文档（type=volume/chapter），按 sort_order 展平为线性列表
const CHAPTER_TYPES = ['volume', 'chapter']

function isChapterType(type: string): boolean {
  return CHAPTER_TYPES.includes(type)
}
const chapterList = computed(() => {
  return documents.value
    .filter(d => CHAPTER_TYPES.includes(d.type))
    .sort((a, b) => (a.sort_order || 0) - (b.sort_order || 0))
})
const currentChapterIndex = computed(() => {
  if (!activeDocId.value) return 0
  const idx = chapterList.value.findIndex(d => d.id === activeDocId.value)
  return idx >= 0 ? idx + 1 : 0
})

const memoryOpen = ref(false)
const graphOpen = ref(false)
const threadOpen = ref(false)
const trashOpen = ref(false)
const trashPanelRef = ref<any>(null)
const graphMaximized = ref(false)

// 通用确认弹窗
const confirmModal = ref<{ title: string; desc: string; action: () => void } | null>(null)
const renameModal = ref<{ docId: number; title: string } | null>(null)
const renameInputRef = ref<HTMLInputElement | null>(null)

// 顶部工作模式：计划 / 写作 / 审核
type EditorMode = 'plan' | 'write' | 'review'
const mode = ref<EditorMode>('write')

// AI 参考的设定文档（从后端返回）
interface UsedSetting {
  id: number
  title: string
  type: 'outline' | 'characters' | 'worldbuilding'
  used: boolean
  tokens: number
}
const usedSettings = ref<UsedSetting[]>([])
const contextPanelOpen = ref(false) // 右侧参考面板默认收起，AI 操作时自动弹出
const referencePanelOpen = ref(false) // B5: 分屏参考面板
const focusMode = ref(false) // 专注模式：隐藏侧栏+顶栏，编辑器全屏
// ponytail-mobile: 移动端抽屉侧栏（A 方案）——显示由 CSS @media 管，这里只管开合状态
const mobileSidebarOpen = ref(false)

// ponytail-focus: 专注模式「今天字数」——首次进入记基线，之后 = 当前字数 - 基线（最小 0）
const todayWords = ref(0)
let todayBaseline = -1
watch(focusMode, (on) => {
  if (on) {
    todayBaseline = wordCount.value
    todayWords.value = 0
  }
})
watch(wordCount, (n) => {
  if (focusMode.value && todayBaseline >= 0) {
    todayWords.value = Math.max(0, n - todayBaseline)
  }
})
const storyboardOpen = ref(false) // 章节大纲卡片视图
const statsOpen = ref(false) // 写作统计面板
const chatOpen = ref(false) // AI 对话面板
const helpOpen = ref(false) // E5: 快捷键帮助弹窗
const toolboxOpen = ref(false) // 工具箱弹出菜单

// ---- 首次进入：初始化向导（第一步首章名称 → 第二步作品相关文件） ----
const showSettingsInit = ref(false)
const settingsInitCreating = ref(false)
const settingsInitStep = ref(1)
const firstChapterNameInput = ref('')
const settingsInitOptions = ref([
  { value: 'outline', label: '📋 大纲', checked: true },
  { value: 'characters', label: '👥 人物设定', checked: true },
  { value: 'worldbuilding', label: '🌍 世界观', checked: true },
  { value: 'storyline', label: '📖 故事主线', checked: true },
  { value: 'chapter_outline', label: '📑 章节大纲', checked: true },
])
const SETTING_INIT_TYPES = ['outline', 'characters', 'worldbuilding', 'storyline', 'chapter_outline']

function maybeAskSettingsInit() {
  const skipKey = `xzj-settings-init-asked-${projectId}`
  try { if (localStorage.getItem(skipKey)) return } catch {}
  const hasSettings = documents.value.some((d: any) => SETTING_INIT_TYPES.includes(d.type))
  if (!hasSettings) {
    // 预填当前首章名（剥离「第X章」前缀，用户写了才发重命名）
    const first = documents.value
      .filter((d: any) => d.type === 'chapter')
      .sort((a: any, b: any) => (a.sort_order || 0) - (b.sort_order || 0))[0]
    firstChapterNameInput.value = first ? stripChapterPrefix(first.title) : ''
    settingsInitStep.value = 1
    showSettingsInit.value = true
    // 和新手引导错开：init 弹窗关闭后再让 tour 弹（事件驱动，见 OnboardingTour）
    window.dispatchEvent(new CustomEvent('settings-init-state', { detail: true }))
  }
}

function skipSettingsInit() {
  showSettingsInit.value = false
  window.dispatchEvent(new CustomEvent('settings-init-state', { detail: false }))
  try { localStorage.setItem(`xzj-settings-init-asked-${projectId}`, '1') } catch {}
}

async function confirmSettingsInit() {
  settingsInitCreating.value = true
  try {
    // 第一步：首章改名（用户没动就不发请求）
    const first = documents.value
      .filter((d: any) => d.type === 'chapter')
      .sort((a: any, b: any) => (a.sort_order || 0) - (b.sort_order || 0))[0]
    const newTitle = firstChapterNameInput.value.trim()
    if (first && newTitle && newTitle !== first.title) {
      await documentApi.update(first.id, { title: newTitle })
      // 同步内容里的 H1 标题行（种子首章的内容是建书时按旧名写的）
      try {
        const res = await documentApi.get(first.id)
        const docData = res.data?.data || res.data
        const content = typeof docData?.content === 'string' ? docData.content : ''
        const ord = chapterOrdinal(first, documents.value) || 1
        const display = `第${numToChinese(ord)}章 ${newTitle}`
        const updated = content.replace(/^#\s.*$/m, '# ' + display)
        if (updated !== content) await documentApi.update(first.id, { content: updated })
      } catch { /* 内容同步失败不阻断 */ }
    }
    // 第二步：作品相关文件
    const chosen = settingsInitOptions.value.filter(s => s.checked)
    for (let i = 0; i < chosen.length; i++) {
      const s = chosen[i]
      await documentApi.create({
        project_id: projectId,
        type: s.value,
        title: s.label.replace(/^\S+\s/, ''), // 去掉 emoji 前缀
        sort_order: i,
      })
    }
    skipSettingsInit()
    await loadDocs()
    // 首章改过名的话内容也变了，编辑器里正开着它就重新加载，防旧内容被自动保存盖回
    if (first && newTitle && newTitle !== first.title && activeDocId.value === first.id) {
      const refreshed = documents.value.find((d: any) => d.id === first.id)
      if (refreshed) await openDoc(refreshed)
    }
  } catch (e: any) {
    toast.error(e.response?.data?.error || '创建失败，请重试')
  } finally {
    settingsInitCreating.value = false
  }
}

// ---- 显示设置（字号/行距/字距）+ 打字机视线位置 ----
// 视线位置语义（Boss 定的）：码字行停在屏幕 N% 高度，眼睛聚焦屏幕中段。
// 实现 = 底部 padding 由视线位置动态推导（容器高 × (1-N%)，让最后一行能滚到视线线）
//        + 打字时光标低于视线线就把容器向上滚
const DS_KEY = 'xzj-display-settings'
const displayOverrides = ref<Record<string, number>>({})
try {
  displayOverrides.value = JSON.parse(localStorage.getItem(DS_KEY) || '{}')
  delete displayOverrides.value.bottomPad // 旧版「底部留白」已并入视线位置，清掉残留
} catch {}

const eyeLineRatio = computed(() => (displayOverrides.value.eyeLine ?? 45) / 100)
const typewriterPad = ref(60)
let twResizeObs: ResizeObserver | null = null

function updateTypewriterPad() {
  const c = document.querySelector('.tiptap-container')
  if (c) typewriterPad.value = Math.round(c.clientHeight * (1 - eyeLineRatio.value))
}

// 容器在 v-else 分支里（打开章节才渲染），首次 onMounted 时可能还不存在——
// 所以观察器要等容器出现再挂，不能只在 onMounted 挂一次
let twObserved: Element | null = null
function setupTypewriterObserver() {
  const c = document.querySelector('.tiptap-container')
  if (!c || c === twObserved) return
  twResizeObs?.disconnect()
  twResizeObs = new ResizeObserver(() => updateTypewriterPad())
  twResizeObs.observe(c)
  twObserved = c
  updateTypewriterPad()
}
watch(activeDocId, () => { nextTick(setupTypewriterObserver) })

// 打字时若光标低于视线线，把容器上滚让光标回到视线线（光标在线上不动，不打扰阅读回看）
function typewriterScroll() {
  if (!editor.value) return
  const container = document.querySelector('.tiptap-container') as HTMLElement | null
  if (!container) return
  try {
    const coords = editor.value.view.coordsAtPos(editor.value.state.selection.from)
    const cRect = container.getBoundingClientRect()
    const targetY = cRect.top + cRect.height * eyeLineRatio.value
    if (coords.bottom > targetY) {
      container.scrollTop += coords.bottom - targetY
    }
  } catch { /* 光标位置算不出时跳过 */ }
}

const displayVars = computed(() => {
  const o = displayOverrides.value
  const s: Record<string, string> = {}
  if (o.fontSize !== undefined) s['--ds-font-size'] = o.fontSize + 'px'
  if (o.lineHeight !== undefined) s['--ds-line-height'] = String(o.lineHeight)
  if (o.letterSpacing !== undefined) s['--ds-letter-spacing'] = o.letterSpacing + 'em'
  s['--ds-bottom-pad'] = typewriterPad.value + 'px' // 视线位置始终生效
  return s
})

function onDisplaySetting(key: string, value: number | undefined) {
  if (value === undefined) delete displayOverrides.value[key]
  else displayOverrides.value[key] = value
  try { localStorage.setItem(DS_KEY, JSON.stringify(displayOverrides.value)) } catch {}
}

watch(eyeLineRatio, () => { updateTypewriterPad() })
const selectedVoice = ref('zh-CN') // E2: TTS 音色选择

function replayOnboarding() {
  toolboxOpen.value = false
  window.dispatchEvent(new Event('replay-onboarding'))
}

// AI 工具栏 & 交互 —— composable
const {
  aiLoading,
  aiPhase,
  aiDirection,
  aiToolbar,
  aiInsert,
  showAIToolbar,
  pendingAction,
  prepareAIAction,
  executeAIAction,
  stopAIGeneration,
  acceptAIResult,
  discardAIResult,
  regenerateAIResult,
  cancelInputting,
  quickRewrite,
} = useAIToolbar({
  editor,
  projectId,
  activeDocId,
  usedSettings,
  saveStatus,
})

const aiDirectionInputRef = ref<HTMLInputElement | null>(null)
// ponytail: focus direction input when entering inputting phase
watch(aiPhase, (v) => {
  if (v === 'inputting') {
    contextPanelOpen.value = true // 使用 AI 时自动弹出右侧面板
    nextTick(() => aiDirectionInputRef.value?.focus())
  }
})

async function jumpToSetting(docId: number) {
  let doc = documents.value.find(d => d.id === docId)
  if (!doc) {
    await loadDocs()
    doc = documents.value.find(d => d.id === docId)
  }
  if (doc) {
    mode.value = 'write'
    await openDoc(doc, { keepSearch: true })
  }
}

// ponytail: chapter_title exact match first, then fuzzy title match
function findDocByLocus(locus: string, chapterTitle?: string) {
  if (chapterTitle) {
    const exact = documents.value.find(d => d.title === chapterTitle)
    if (exact) return exact
  }
  return documents.value.find(d =>
    d.title === locus || locus.includes(d.title) || d.title.includes(locus)
  )
}

async function handleReviewJump(issue: { locus: string; desc: string; chapter_title?: string }) {
  const doc = findDocByLocus(issue.locus, issue.chapter_title)
  if (!doc) return
  switchMode('write')
  await openDoc(doc)
  await nextTick()
  if (!editor.value) return
  const snippet = issue.desc?.length > 30 ? issue.desc.slice(0, 30) : (issue.desc || '')
  const fullText = editor.value.state.doc.textBetween(0, editor.value.state.doc.content.size, '\n')
  const pos = fullText.indexOf(snippet)
  if (pos < 0) return  // jumped to correct chapter, skip highlight if snippet not found
  editor.value.chain()
    .setTextSelection({ from: pos, to: pos + snippet.length })
    .run()
  const coords = editor.value.view.coordsAtPos(pos)
  const selEl = document.elementFromPoint(coords.left, coords.top)?.parentElement
  if (selEl) selEl.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

async function handleReviewAiFix(issue: any) {
  const doc = findDocByLocus(issue.locus, issue.chapter_title)
  if (!doc) return
  switchMode('write')
  await openDoc(doc)
  await nextTick()
  if (!editor.value) return
  const snippet = issue.desc?.length > 30 ? issue.desc.slice(0, 30) : (issue.desc || '')
  const fullText = editor.value.state.doc.textBetween(0, editor.value.state.doc.content.size, '\n')
  const pos = fullText.indexOf(snippet)
  if (pos >= 0) {
    editor.value.chain()
      .focus()
      .setTextSelection({ from: pos, to: pos + snippet.length })
      .run()
  }
  await quickRewrite(snippet, pos >= 0 ? pos : 0, pos >= 0 ? pos + snippet.length : 0)
}

// ponytail: click outside toolbox closes it; popup @click.stop prevents inner clicks from reaching here
function closeToolboxOnClickOutside() {
  if (toolboxOpen.value) toolboxOpen.value = false
}

function switchMode(next: EditorMode) {
  if (mode.value === next) return
  mode.value = next
  // 关闭浮层，避免残留
  if (aiPhase.value === 'generating') stopAIGeneration()
  aiToolbar.value.visible = false
  aiInsert.value.active = false
  contextMenu.value.visible = false
  toolboxOpen.value = false
  if (searchVisible.value) closeSearch()
}

// ponytail: 设定/文章分区——volume/chapter=文章侧，其余=设定侧
const settingRootNodes = computed(() =>
  rootNodes.value.filter(n => !isChapterType(n.type))
)
const articleRootNodes = computed(() =>
  rootNodes.value.filter(n => isChapterType(n.type))
)

onMounted(async () => {
  initEditor()
  window.addEventListener('keydown', handleGlobalKeydown)
  document.addEventListener('click', closeToolboxOnClickOutside)
  window.addEventListener('online', onOnlineStatusChange)
  window.addEventListener('offline', onOnlineStatusChange)
  loadProjectName()
  loadVoices() // E2: 非关键，不 await
  await loadBranches()
  await loadDocs()
  maybeAskSettingsInit()
  // 打字机视线：底距 + 容器尺寸变化（改窗口/手机键盘弹起）时重算
  setupTypewriterObserver()
  // 恢复上次的章节；新书（只有一章）直接打开第一章，不让新手面对空编辑器
  if (!activeDocId.value) {
    let target: any = null
    try {
      const lastId = Number(localStorage.getItem(`xzj-last-doc-${projectId}`) || 0)
      if (lastId) target = documents.value.find((d: any) => d.id === lastId && d.type === 'chapter')
    } catch {}
    if (!target) {
      const chapters = documents.value.filter((d: any) => d.type === 'chapter')
      if (chapters.length === 1) target = chapters[0]
    }
    if (target) openDoc(target)
  }
})

async function loadProjectName() {
  try {
    const res = await projectApi.get(projectId)
    const data = res.data?.data || res.data
    projectName.value = data?.name || ''
  } catch (e) {
    console.warn('加载项目名失败', e)
  }
}

onBeforeUnmount(() => {
  twResizeObs?.disconnect()
  twResizeObs = null
  window.removeEventListener('keydown', handleGlobalKeydown)
  document.removeEventListener('click', closeToolboxOnClickOutside)
  window.removeEventListener('online', onOnlineStatusChange)
  window.removeEventListener('offline', onOnlineStatusChange)
  if (editor.value) editor.value.destroy()
  if (saveTimer) clearTimeout(saveTimer)
  currentAudio?.pause()
  currentAudio = null
  if (selectionHandler) {
    document.removeEventListener('selectionchange', selectionHandler)
    selectionHandler = null
  }
})

function initEditor() {
  if (editor.value) editor.value.destroy()
  if (selectionHandler) {
    document.removeEventListener('selectionchange', selectionHandler)
  }

  editor.value = new Editor({
    content: '',
    extensions: [
      StarterKit,
      Highlight.configure({ multicolor: true }),
      Table.configure({ resizable: true }),
      TableRow,
      TableHeader,
      TableCell,
      AIMark,
      AIStrikethrough,
    ],
    // ponytail: 粘贴强制纯文本，去掉外部加粗/斜体等格式
    editorProps: {
      handlePaste: (view, event) => {
        const text = event.clipboardData?.getData('text/plain')
        if (text) {
          view.dispatch(view.state.tr.insertText(text))
          return true
        }
        return false
      },
    },
    onUpdate: ({ editor: ed }) => {
      if (isSettingContent.value) return
      const plainText = getEditorText(ed)
      wordCount.value = plainText.length
      saveStatus.value = '编辑中...'
      // ponytail: write localStorage draft on every keystroke window so
      // browser crash / tab close in the 1.5s debounce gap doesn't lose data.
      if (activeDocId.value) {
        try { localStorage.setItem('draft-' + activeDocId.value, getEditorHTML(ed)) } catch {}
      }
      if (saveTimer) clearTimeout(saveTimer)
      saveTimer = setTimeout(() => autoSave(), 1500)
      typewriterScroll() // 打字机视线：光标沉过视线线就把页面向上托
    }
  })

  // 注册搜索高亮 ProseMirror 插件
  editor.value.registerPlugin(createSearchHighlightPlugin())

  selectionHandler = () => {
    if (!editor.value || !editor.value.isFocused) return
    if (isSettingContent.value) return
    // AI 活动期间不响应选区变化，防止清空预览
    if (aiPhase.value !== 'idle') return
    const sel = getSelection()
    if (sel.text.trim()) {
      showAIToolbar(sel)
    } else {
      aiToolbar.value.visible = false
    }
  }
  document.addEventListener('selectionchange', selectionHandler)
}

function getSelection() {
  if (!editor.value) return { text: '', start: 0, end: 0 }
  // ponytail: editor.state.selection lags DOM on first selection (selectionchange fires
  // before ProseMirror syncs). Use view.posAtDOM() — always reads live DOM, never stale.
  const native = window.getSelection()
  if (!native || native.rangeCount === 0) return { text: '', start: 0, end: 0 }
  const text = native.toString()
  if (!text) return { text: '', start: 0, end: 0 }
  const view = editor.value.view
  const range = native.getRangeAt(0)
  const start = view.posAtDOM(range.startContainer, range.startOffset)
  const end = view.posAtDOM(range.endContainer, range.endOffset)
  return { text, start: Math.min(start, end), end: Math.max(start, end) }
}

// ponytail: parse Markdown table blocks (| col | col |) into ProseMirror table nodes
function parseMarkdownTable(lines: string[], startIdx: number): { tableNode: object; consumed: number } | null {
  const headerLine = lines[startIdx]
  // need at least header + separator + 1 row
  if (startIdx + 2 >= lines.length) return null
  const separatorLine = lines[startIdx + 1]
  // separator must be like |---|---| or | --- | :---: |
  if (!/^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)+\|?\s*$/.test(separatorLine)) return null

  // ponytail: simpler split — strip leading/trailing | then split
  const splitCells = (line: string) => {
    let s = line.trim()
    if (s.startsWith('|')) s = s.slice(1)
    if (s.endsWith('|')) s = s.slice(0, -1)
    return s.split('|').map(c => c.trim())
  }

  const headers = splitCells(headerLine)
  if (headers.length < 1) return null

  // ponytail: parse **bold** and *italic* inline marks into ProseMirror text nodes
  const makeTextContent = (text: string) => {
    if (!text) return []
    const nodes: object[] = []
    // split on **bold** or *italic* patterns
    const regex = /(\*\*[^*]+\*\*|\*[^*]+\*)/g
    let last = 0
    let m: RegExpExecArray | null
    while ((m = regex.exec(text)) !== null) {
      if (m.index > last) nodes.push({ type: 'text', text: text.slice(last, m.index) })
      const matched = m[0]
      if (matched.startsWith('**')) {
        nodes.push({ type: 'text', text: matched.slice(2, -2), marks: [{ type: 'bold' }] })
      } else {
        nodes.push({ type: 'text', text: matched.slice(1, -1), marks: [{ type: 'italic' }] })
      }
      last = m.index + matched.length
    }
    if (last < text.length) nodes.push({ type: 'text', text: text.slice(last) })
    return nodes
  }

  const rows: object[] = []
  // header row
  rows.push({
    type: 'tableRow',
    content: headers.map(h => ({ type: 'tableHeader', content: [{ type: 'paragraph', content: makeTextContent(h) }] }))
  })

  // body rows
  let consumed = 2 // header + separator
  for (let i = startIdx + 2; i < lines.length; i++) {
    const line = lines[i]
    if (!line.includes('|')) break
    const cells = splitCells(line)
    // pad/truncate to header length
    while (cells.length < headers.length) cells.push('')
    rows.push({
      type: 'tableRow',
      content: cells.slice(0, headers.length).map((c: string) => ({ type: 'tableCell', content: [{ type: 'paragraph', content: makeTextContent(c) }] }))
    })
    consumed++
  }

  if (rows.length < 2) return null // need at least header + 1 body row

  return { tableNode: { type: 'table', content: rows }, consumed }
}

function editorContentFromText(content: string) {
  if (!content) return { type: 'doc', content: [{ type: 'paragraph' }] }
  const lines = content.split('\n')
  const blocks: object[] = []
  let i = 0
  while (i < lines.length) {
    const line = lines[i]
    // detect markdown table: line has | and next line is separator
    if (line.includes('|') && i + 1 < lines.length && /^\s*\|?\s*:?-{2,}:?\s*(\|)/.test(lines[i + 1])) {
      const result = parseMarkdownTable(lines, i)
      if (result) {
        blocks.push(result.tableNode)
        i += result.consumed
        continue
      }
    }
    blocks.push({
      type: 'paragraph',
      content: line ? [{ type: 'text', text: line }] : []
    })
    i++
  }
  return { type: 'doc', content: blocks }
}

function getEditorText(ed = editor.value) {
  if (!ed) return ''
  const blocks: string[] = []
  ed.state.doc.forEach((node: any) => {
    blocks.push(node.textContent || '')
  })
  return blocks.join('\n')
}

// ponytail: save/load as HTML to preserve marks (highlight etc.)
function getEditorHTML(ed = editor.value) {
  return ed ? ed.getHTML() : ''
}

// —— 一键排版 ——
// 处理"内容写好了但排版乱"：①去掉每段首尾的空白字符（含手打的全角空格缩进）②删除空段落
// ③每段统一首行缩进两个全角空格 ④段落之间不留空行（网文排版习惯：段间靠 CSS margin 隔开）。
// 标题(h1-h3)、表格、代码块等非段落节点原样保留；文字的 marks（高亮等）通过保留 inline 内容不动而保住。
// 单个 ProseMirror 事务完成，Ctrl+Z 一步可撤销。
function oneClickFormat() {
  const ed = editor.value
  if (!ed) return
  const { state, view } = ed
  const { tr } = state
  const INDENT = '\u3000\u3000' // 两个全角空格

  // 从后往前处理，位置映射才不会互相干扰
  const paras: { pos: number; node: any }[] = []
  state.doc.forEach((node: any, offset: number) => {
    if (node.type.name === 'paragraph') paras.push({ pos: offset, node })
  })

  let changed = 0
  for (let i = paras.length - 1; i >= 0; i--) {
    const { pos, node } = paras[i]
    const text = node.textContent
    const cleaned = text.replace(/^[\s\u3000]+|[\s\u3000]+$/g, '')
    // 空段落（或只有空白）：删掉整个段落
    if (!cleaned) {
      tr.delete(pos, pos + node.nodeSize)
      changed++
      continue
    }
    // 已是标准缩进且无首尾空白：跳过
    if (text === INDENT + cleaned) continue
    const newText = INDENT + cleaned
    // 重建段落：保留 marks 的做法是复制首个 text node 的 marks 应用到新文本
    const firstText = node.content.firstChild
    const marks = firstText?.marks ?? []
    const paraType = state.schema.nodes.paragraph
    const newPara = paraType.create(null, state.schema.text(newText, marks))
    tr.replaceWith(pos, pos + node.nodeSize, newPara)
    changed++
  }

  if (changed > 0) {
    // 只把光标放回文档开头附近，避免位置失效
    tr.setSelection(state.selection.constructor.near(tr.doc.resolve(Math.min(state.selection.from, tr.doc.content.size))))
    view.dispatch(tr)
    autoSave()
    toast.success(`已整理 ${changed} 个段落`)
  } else {
    toast.info('排版已经很整齐，无需整理')
  }
}

async function loadBranches() {
  try {
    const res = await branchApi.list(projectId)
    branches.value = res.data?.data || res.data || []
    if (!activeBranchId.value && branches.value.length > 0) {
      activeBranchId.value = branches.value[0].id
    }
  } catch (e) {
    console.error('加载故事线失败', e)
    branches.value = []
  }
}

async function switchBranch() {
  compareActive.value = false
  activeDocId.value = null
  currentDocName.value = ''
  wordCount.value = 0
  saveStatus.value = ''
  if (editor.value) editor.value.commands.setContent('', { emitUpdate: false })
  await loadDocs()
}

async function loadDocs() {
  loadingDocs.value = true
  const docsCacheKey = `xzj-docs-${projectId}-${activeBranchId.value ?? 'main'}`
  try {
    const res = await documentApi.list(String(projectId), activeBranchId.value)
    const fresh = res.data?.data || res.data || []
    // ponytail: 展开状态持久化到 localStorage，按项目独立
    const expandKey = `tree-expand-${projectId}`
    const isFirstLoad = documents.value.length === 0
    let savedExpand: Set<number> | null = null
    if (isFirstLoad) {
      // 只有真的存过展开状态才用；否则走默认展开（原写法 '[]' 恒成功导致默认展开永不生效）
      const rawExpand = localStorage.getItem(expandKey)
      if (rawExpand !== null) {
        try { savedExpand = new Set(JSON.parse(rawExpand)) } catch {}
      }
    }
    const prev = new Map(documents.value.map(d => [d.id, d._expanded]))
    for (const d of fresh) {
      if (prev.has(d.id)) {
        d._expanded = prev.get(d.id)
      } else if (isFirstLoad && savedExpand) {
        d._expanded = savedExpand.has(d.id)
      } else if (isFirstLoad) {
        // 首次打开（无保存的展开状态）：卷和文件夹默认展开，新手进来就能看到第一章
        d._expanded = d.type === 'volume' || d.type === 'folder'
      }
    }
    documents.value = fresh
    // ponytail: 缓存文档树，离线刷新时兜底展示
    try { localStorage.setItem(docsCacheKey, JSON.stringify(fresh)) } catch {}
  } catch (e) {
    console.error('加载文档失败', e)
    // ponytail: 离线兜底——用上次缓存的文档树，作者断网刷新也能继续写
    try {
      const cached = JSON.parse(localStorage.getItem(docsCacheKey) || '[]')
      if (Array.isArray(cached) && cached.length) {
        documents.value = cached
      } else {
        documents.value = []
      }
    } catch { documents.value = [] }
  } finally {
    loadingDocs.value = false
  }
}

// ponytail: 展开状态持久化——监听 _expanded 变化，防抖写 localStorage
let expandSaveTimer: ReturnType<typeof setTimeout> | null = null
watch(documents, () => {
  if (expandSaveTimer) clearTimeout(expandSaveTimer)
  expandSaveTimer = setTimeout(() => {
    const ids = documents.value.filter(d => d._expanded).map(d => d.id)
    localStorage.setItem(`tree-expand-${projectId}`, JSON.stringify(ids))
  }, 500)
}, { deep: true })

async function initDefaultStructure() {
  const defaults = [
    { type: 'outline', title: '大纲', sort_order: 1 },
    { type: 'characters', title: '人物设定', sort_order: 2 },
    { type: 'worldbuilding', title: '世界观', sort_order: 3 },
    { type: 'storyline', title: '故事主线', sort_order: 4 },
    { type: 'chapter_outline', title: '章节大纲', sort_order: 5 },
    { type: 'volume', title: '文章', sort_order: 6 },
  ]
  for (const d of defaults) {
    try {
      await documentApi.create({
        project_id: projectId,
        title: d.title,
        type: d.type,
        sort_order: d.sort_order,
      })
    } catch (e) {
      console.error('创建默认结构失败', d, e)
    }
  }
  await loadDocs()
}

async function openDoc(doc: any, opts?: { keepSearch?: boolean }) {
  compareActive.value = false
  // ponytail-mobile: 移动端选中章节后收起抽屉
  mobileSidebarOpen.value = false
  // 切换文档时停止朗读
  if (currentAudio) {
    currentAudio.pause()
    currentAudio = null
    isSpeaking.value = false
  }
  // ponytail: flush pending auto-save BEFORE switching activeDocId.
  // Without this, a pending saveTimer fires after activeDocId changes but before
  // the new doc's content loads — saving the old doc's content over the new doc.
  // Just clearing the timer would lose up to 1.5s of edits; flushing saves them first.
  if (saveTimer) { clearTimeout(saveTimer); saveTimer = null; await autoSave() }
  activeDocId.value = doc.id
  currentDocName.value = doc.title
  // 记住每本书最后打开的章节，下次进编辑器直接续上
  try { localStorage.setItem(`xzj-last-doc-${projectId}`, String(doc.id)) } catch {}
  aiToolbar.value.visible = false
  // 切换章节时清空搜索状态与高亮（全项目跳转除外）
  if (!opts?.keepSearch && (searchQuery.value || searchTotal.value)) {
    searchQuery.value = ''
    searchTotal.value = 0
    searchActiveIndex.value = 0
    searchProjectMatches.value = []
    applyEditorSearch('', -1)
  }

  try {
    let content: string
    let fromCache = false
    try {
      const res = await documentApi.get(doc.id)
      const docData = res.data?.data || res.data
      content = (typeof docData?.content === 'string') ? docData.content : ''
      // ponytail: 缓存章节内容，离线时兜底可读
      try { localStorage.setItem('xzj-doc-content-' + doc.id, content) } catch {}
    } catch {
      // ponytail: 离线兜底——用缓存内容打开，作者断网也能翻章节看/改
      content = ''
      try { content = localStorage.getItem('xzj-doc-content-' + doc.id) || '' } catch {}
      fromCache = true
    }

    if (editor.value) {
      isSettingContent.value = true
      // ponytail: load HTML directly to preserve marks; fallback to plain text for old docs
      const html = content.trim().startsWith('<') ? content : null
      if (html) {
        editor.value.commands.setContent(html, { emitUpdate: false })
      } else {
        editor.value.commands.setContent(editorContentFromText(content), { emitUpdate: false })
      }
      wordCount.value = getEditorText().length
      saveStatus.value = fromCache ? '📡 离线读取' : '已保存'
      setTimeout(() => { isSettingContent.value = false }, 100)

      // ponytail: check for unsaved local draft (browser crashed during 1.5s debounce)
      const draftKey = 'draft-' + doc.id
      const draftHTML = (() => { try { return localStorage.getItem(draftKey) } catch { return null } })()
      if (draftHTML && draftHTML !== getEditorHTML()) {
        draftBanner.value = { docId: doc.id, text: draftHTML }
      } else {
        draftBanner.value = null
      }
    }
  } catch (e) {
    console.error('加载文档内容失败', e)
    isSettingContent.value = false
  }
}

let saveRetryCount = 0
// ponytail: offline state — no point burning retries when navigator.onLine is false;
// 'online' event listener triggers a catch-up save the moment network returns.
const isOffline = ref(!navigator.onLine)
function onOnlineStatusChange() {
  const offline = !navigator.onLine
  isOffline.value = offline
  if (!offline) {
    // 网络恢复：立即补传（草稿仍在 localStorage，丢不了）
    saveRetryCount = 0
    autoSave()
  }
}
async function autoSave() {
  if (!activeDocId.value || !editor.value) return
  if (isOffline.value) {
    saveStatus.value = '📡 离线中，已存本地'
    return
  }
  try {
    saveStatus.value = '保存中...'
    await documentApi.update(activeDocId.value, {
      content: getEditorHTML()
    })
    saveStatus.value = '已保存'
    saveRetryCount = 0
    // ponytail: server has it now, clear the local draft backup
    if (activeDocId.value) {
      localStorage.removeItem('draft-' + activeDocId.value)
    }
  } catch (e) {
    if (saveRetryCount < 3) {
      saveRetryCount++
      saveStatus.value = `保存失败，重试中(${saveRetryCount}/3)...`
      setTimeout(() => autoSave(), 2000 * saveRetryCount)
    } else {
      saveStatus.value = isOffline.value ? '📡 离线中，已存本地' : '⚠️ 保存失败，已存本地，恢复网络自动上传'
      saveRetryCount = 0
    }
  }
}

function restoreDraft() {
  if (!draftBanner.value || !editor.value) return
  isSettingContent.value = true
  editor.value.commands.setContent(draftBanner.value.text, { emitUpdate: false })
  wordCount.value = getEditorText().length
  saveStatus.value = '编辑中...'
  setTimeout(() => {
    isSettingContent.value = false
    // Trigger a save so the restored content hits the server
    if (saveTimer) clearTimeout(saveTimer)
    saveTimer = setTimeout(() => autoSave(), 1500)
  }, 100)
  draftBanner.value = null
}

function discardDraft() {
  if (!draftBanner.value) return
  localStorage.removeItem('draft-' + draftBanner.value.docId)
  draftBanner.value = null
}

function openCreateDocModal(parentId: number | null = null, type: string = 'chapter') {
  createDocParent.value = parentId
  createDocType.value = type
  newDocName.value = ''
  showCreateDocModal.value = true
}

// 拆分组件 CreateDocModal 触发（emit create）
async function onCreateDocConfirm(name: string, folderType?: string) {
  newDocName.value = name
  await createDoc(folderType)
}

async function createDoc(folderType?: string) {
  if (!newDocName.value.trim()) return
  try {
    // 计算 sort_order
    const siblings = documents.value.filter(d =>
      (createDocParent.value === null ? !d.parent_id : d.parent_id === createDocParent.value)
    )
    const maxSort = siblings.reduce((m, d) => Math.max(m, d.sort_order || 0), 0)

    // ponytail: type 由调用方决定，Tab 归属纯由 type 推导
    const actualType = folderType || createDocType.value

    const res = await documentApi.create({
      project_id: projectId,
      branch_id: activeBranchId.value || undefined,
      title: newDocName.value.trim(),
      type: actualType,
      parent_id: createDocParent.value || undefined,
      sort_order: maxSort + 1,
    })
    showCreateDocModal.value = false
    newDocName.value = ''
    await loadDocs()
    const newDoc = res.data?.data || res.data
    if (newDoc && actualType === 'chapter') {
      openDoc(newDoc)
    }
  } catch (e: any) {
    toast.error(e.response?.data?.error || '创建失败')
  }
}

function openDocContextMenu(e: MouseEvent, doc: any) {
  docContextMenu.value = {
    visible: true,
    x: e.clientX,
    y: e.clientY,
    doc
  }
  document.addEventListener('click', closeDocContextMenu, { once: true })
}

function closeDocContextMenu() {
  docContextMenu.value.visible = false
}

// ponytail: 底部按钮固定根目录新建
function createDocAtRoot() {
  openCreateDocModal(null, 'volume')
}

// ponytail: 文章区＋号=新建卷（层级：卷→章节；章节走每卷的「+ 在卷下新建章节」）
function createVolumeAtRoot() {
  openCreateDocModal(null, 'volume')
}

// ponytail: 设定区＋号——根目录建设定（弹窗里选类型：大纲/人物/世界观等）
function createSettingAtRoot() {
  openCreateDocModal(null, 'setting')
}

// ponytail: 文件夹内+按钮，根据文件夹类型决定子文档类型
function createDocInFolder(folder: any) {
  const type = isChapterType(folder.type) ? 'chapter' : 'note'
  openCreateDocModal(folder.id, type)
}

function newDocUnder(type: 'volume' | 'chapter') {
  const doc = docContextMenu.value.doc
  // 空白区域右键 → 根级创建；右键文件夹 → 在其下创建；右键文件 → 同级创建
  if (!doc) {
    openCreateDocModal(null, type)
  } else {
    const isFolder = isFolderNode(doc)
    const parentId = isFolder ? doc.id : (doc.parent_id || null)
    // 子文档类型跟随文件夹：章节文件夹→chapter，设定文件夹→note
    const actualType = isFolder && !isChapterType(doc.type) && type === 'chapter' ? 'note' : type
    openCreateDocModal(parentId, actualType)
  }
  closeDocContextMenu()
}

// ponytail: 判断是否文件夹容器——folder=设定文件夹, volume=文章文件夹, 其他设定类型也可展开
// ponytail: 设定区平铺后只有卷是容器；设定文档是纯文件（Boss 2026-10-22：设定下面不要有文件夹）
function isFolderNode(doc: any): boolean {
  return doc.type === 'volume'
}

// 右键文件夹 → 属性
function openFolderProps() {
  const doc = docContextMenu.value.doc
  if (!doc) return
  folderPropsDoc.value = doc
  showFolderPropsModal.value = true
  closeDocContextMenu()
}

async function onFolderPropsSave(data: { icon?: string; doc_status?: string; word_goal?: number }) {
  const doc = folderPropsDoc.value
  if (!doc) return
  try {
    const update: any = {}
    if (data.icon !== undefined) update.icon = data.icon
    if (data.doc_status !== undefined) update.doc_status = data.doc_status
    if (data.word_goal !== undefined) update.word_goal = data.word_goal
    await documentApi.update(doc.id, update)
    showFolderPropsModal.value = false
    await loadDocs()
  } catch (e: any) {
    toast.error(e.response?.data?.error || '更新属性失败')
  }
}

// B3: 状态切换（圆点点击循环 draft->revising->final->draft）
async function handleCycleStatus(doc: any) {
  const order = ['draft', 'revising', 'final']
  const cur = doc.doc_status || 'draft'
  const next = order[(order.indexOf(cur) + 1) % order.length]
  await handleDocStatusChange(doc, next)
}

// B3: 右键菜单设置状态
async function handleDocStatusChange(doc: any, newStatus: string) {
  closeDocContextMenu()
  try {
    await documentApi.update(doc.id, { doc_status: newStatus })
    const d = documents.value.find(x => x.id === doc.id)
    if (d) d.doc_status = newStatus
  } catch (e: any) {
    toast.error(e.response?.data?.error || '更新状态失败')
  }
}

async function renameDoc() {
  const doc = docContextMenu.value.doc
  if (!doc) return
  closeDocContextMenu()
  renameModal.value = { docId: doc.id, title: doc.title }
  nextTick(() => { renameInputRef.value?.focus(); renameInputRef.value?.select() })
}

async function confirmRename() {
  if (!renameModal.value) return
  const { docId, title } = renameModal.value
  if (!title.trim()) return
  const trimmed = title.trim()
  renameModal.value = null
  try {
    await documentApi.update(docId, { title: trimmed })
    await loadDocs()
    if (activeDocId.value === docId) currentDocName.value = trimmed
  } catch (e: any) {
    toast.error(e.response?.data?.error || '重命名失败')
  }
}

async function deleteDoc() {
  const doc = docContextMenu.value.doc
  if (!doc) return
  closeDocContextMenu()
  confirmModal.value = {
    title: '删除文档',
    desc: `确定删除「${doc.title}」？删除后可在回收站恢复。`,
    action: async () => {
      try {
        await documentApi.delete(doc.id)
        if (activeDocId.value === doc.id) {
          activeDocId.value = null
          currentDocName.value = ''
          if (editor.value) editor.value.commands.setContent('')
        }
        await loadDocs()
        trashPanelRef.value?.load()
        toast.success('已移入回收站')
      } catch (e: any) {
        toast.error(e.response?.data?.error || '删除失败')
      } finally {
        confirmModal.value = null
      }
    },
  }
}

async function onTrashRestored() {
  await loadDocs()
}

async function handleDropNode(srcId: number, dstId: number, position: 'before' | 'after' | 'inside') {
  const src = documents.value.find(d => d.id === srcId)
  const dst = documents.value.find(d => d.id === dstId)
  if (!src || !dst) return

  let newParentId: number | null
  let newSortOrder: number

  if (position === 'inside') {
    // 拖到文件夹中间 → 成为该文件夹的子节点
    newParentId = dst.id
    const children = documents.value.filter(d => d.parent_id === dst.id)
    newSortOrder = children.reduce((m, d) => Math.max(m, d.sort_order || 0), 0) + 1
  } else {
    // before / after → 成为该节点的同级
    newParentId = dst.parent_id || null
    const baseSort = dst.sort_order || 0
    newSortOrder = position === 'before' ? baseSort - 1 : baseSort + 1
  }

  try {
    await documentApi.move(srcId, {
      parent_id: newParentId !== null ? String(newParentId) : null,
      sort_order: newSortOrder,
    })
    await loadDocs()
  } catch (e: any) {
    toast.error(e.response?.data?.error || '移动失败')
  }
}

// ponytail: 复用 handleDropNode 的 move 逻辑，storyboard 拖拽只有 before/after
async function handleStoryboardReorder(payload: { fromId: number; toId: number; position: 'before' | 'after' }) {
  await handleDropNode(payload.fromId, payload.toId, payload.position)
}

function openContextMenu(e: MouseEvent) {
  // ponytail: 右键前缓存当前选区，右键动作后编辑器已失焦
  cachedSelection = getSelection()
  contextMenu.value = { visible: true, x: e.clientX, y: e.clientY }
  document.addEventListener('click', closeContextMenu, { once: true })
}

function closeContextMenu() {
  contextMenu.value.visible = false
}

function contextAction(action: string, color?: string) {
  if (!editor.value) return
  // 一键排版作用于整篇，不需要选区
  if (action === 'oneClickFormat') {
    contextMenu.value.visible = false
    oneClickFormat()
    return
  }
  const sel = cachedSelection || getSelection()
  cachedSelection = null
  if (!sel.text && action !== 'paste') return
  switch (action) {
    case 'highlight':
      editor.value.chain().setTextSelection({ from: sel.start, to: sel.end }).toggleHighlight({ color }).run()
      break
    case 'clearHighlight':
      editor.value.chain().setTextSelection({ from: sel.start, to: sel.end }).unsetHighlight().run()
      break
    case 'copy':
      navigator.clipboard.writeText(sel.text)
      break
    case 'cut':
      navigator.clipboard.writeText(sel.text)
      editor.value.commands.deleteRange({ from: sel.start, to: sel.end })
      break
    case 'paste':
      navigator.clipboard.readText().then(text => {
        editor.value?.commands.insertContentAt(sel.start, text)
      })
      break
    case 'strike':
      editor.value.chain().setTextSelection({ from: sel.start, to: sel.end }).setStrike().run()
      break
    case 'unstrike':
      editor.value.chain().setTextSelection({ from: sel.start, to: sel.end }).unsetStrike().run()
      break
  }
  contextMenu.value.visible = false
}

function handleGlobalKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
    e.preventDefault()
    // ponytail: cancel pending debounce so Ctrl+S saves immediately, not twice
    if (saveTimer) { clearTimeout(saveTimer); saveTimer = null }
    autoSave()
    return
  }
  // B4: Ctrl+P 循环切换工作模式
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'p') {
    e.preventDefault()
    const modes: EditorMode[] = ['plan', 'write'] // 审核暂时隐藏，别被快捷键切进去
    const idx = modes.indexOf(mode.value)
    switchMode(modes[(idx + 1) % modes.length])
    return
  }
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
    e.preventDefault()
    openSearch(e.shiftKey ? 'project' : 'chapter')
    return
  }
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'h') {
    e.preventDefault()
    openSearch('chapter')
    return
  }
  if (e.key === 'Escape' && searchVisible.value) {
    e.preventDefault()
    closeSearch()
    return
  }
  // ponytail: Esc 退出专注模式（搜索关闭后优先级最低）
  if (e.key === 'Escape' && focusMode.value) {
    e.preventDefault()
    focusMode.value = false
    return
  }
  // Ctrl/Cmd+Shift+Z 切换专注模式
  if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key.toLowerCase() === 'z') {
    e.preventDefault()
    focusMode.value = !focusMode.value
    return
  }
  // E5: Ctrl+/ 打开快捷键帮助
  if ((e.ctrlKey || e.metaKey) && e.key === '/') {
    e.preventDefault()
    helpOpen.value = !helpOpen.value
    return
  }
  // E1: Alt+↑/↓ 场景导航
  if (e.altKey && (e.key === 'ArrowUp' || e.key === 'ArrowDown')) {
    e.preventDefault()
    jumpScene(e.key === 'ArrowDown' ? 1 : -1)
    return
  }
}

function goBack() {
  router.push('/bookshelf')
}
</script>

<style scoped src="./editor-page.css"></style>
