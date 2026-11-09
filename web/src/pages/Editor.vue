<template>
      <div class="editor-page" :class="{ 'focus-mode': focusMode, 'mp-full': isMiniProgramEnv }" :style="displayVars">
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
      @toggle-reference="toggleReference"
      @toggle-sidebar="mobileSidebarOpen = !mobileSidebarOpen"
    />

    <!-- 手机档 H 方案（Boss 定稿）：字数/保存胶囊 + 贴边把手 + 玻璃工具轨；底栏退役（CSS 控制仅 ≤768px 显示） -->
    <div class="mp-chip">{{ wordCount }} 字{{ mpSaveChip }}<template v-if="isOffline"> · 离线</template></div>
    <div class="mp-handle" @click.stop="mpRailOpen = !mpRailOpen"></div>
    <div class="mp-rail" :class="{ open: mpRailOpen }" @click.stop>
      <button :class="{ on: !!wordLock }" @click="mpRailOpen = false; wordLockOpen = !wordLockOpen" title="码字锁">🔒</button>
      <button v-if="enableDictation" :class="{ on: dictating }" @click="toggleDictation" title="听写">{{ dictating ? '⏹' : '🎙' }}</button>
      <button :class="{ on: contextPanelOpen }" @click="mpRailOpen = false; contextPanelOpen = true" title="设定 / 灵感 / 自检">📖</button>
      <button :class="{ on: sensitiveOpen }" @click="mpRailOpen = false; sensitiveOpen = !sensitiveOpen" title="敏感词">◉<span v-if="sensitiveTotal > 0" class="mp-dot">{{ sensitiveTotal > 99 ? '99+' : sensitiveTotal }}</span></button>
      <button @click="mpRailOpen = false; toolboxOpen = !toolboxOpen" title="工具箱">⋯</button>
      <!-- mp-full（编辑器全屏）时顶栏隐藏，目录/模式切换挪进工具轨 -->
      <button v-if="isMiniProgramEnv" @click="mpRailOpen = false; mobileSidebarOpen = true" title="目录">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M4 6h16M4 12h16M4 18h10"/></svg>
      </button>
      <button v-if="isMiniProgramEnv" :class="{ on: mode !== 'write' }" @click="mpRailOpen = false; switchMode(mode === 'write' ? 'review' : 'write')" :title="mode === 'write' ? '切到审阅' : '切到写作'">
        <svg v-if="mode === 'write'" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7z"/><circle cx="12" cy="12" r="3"/></svg>
        <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 19l7-7 3 3-7 7-3-3z"/><path d="M18 13l-1.5-7.5L2 2l3.5 14.5L13 18l5-5z"/><path d="M2 2l7.586 7.586"/><circle cx="11" cy="11" r="2"/></svg>
      </button>
      <button class="mp-close" @click="mpRailOpen = false">✕</button>
    </div>

    <!-- 手机档翻章条：一体居中胶囊（2026-11-06 拍 demo A）——相邻章顺写；点中间章名开目录（长目录在抽屉里） -->
    <div class="mp-chapnav" v-if="!keyboardOpen && activeDocId && mode === 'write' && navChapters.length > 1">
      <button :disabled="!navPrev" title="上一章" @click="navPrev && openDoc(navPrev)">‹</button>
      <span class="mp-chapnav-name" @click="mobileSidebarOpen = true">{{ currentDocDisplayName }} · {{ navIndex }}/{{ navChapters.length }}</span>
      <!-- 最后一章：下一章变「+ 新章」——弹新建章节框（可留空=自动编号），Boss 2026-11-06 -->
      <button v-if="!navNext" class="chapnav-new" @click="createNextChapter">+ 新章</button>
      <button v-else title="下一章" @click="openDoc(navNext)">›</button>
    </div>

    <div class="editor-body">
      <!-- 移动端抽屉遮罩 -->
      <div v-if="mobileSidebarOpen" class="sidebar-scrim" @click="mobileSidebarOpen = false"></div>
      <!-- 左侧文档树（移动端=抽屉）；class 由 EditorSidebar 根元素自带，此处不重复传 -->
      <EditorSidebar
        :class="{ 'drawer-open': mobileSidebarOpen }"
        :documents="documents"
        :setting-root-nodes="settingRootNodes"
        :article-root-nodes="articleRootNodes"
        :active-doc-id="activeDocId"
        :loading-docs="loadingDocs"
        :project-name="projectName"
        :today-words="todayWords"
        :chapter-words="wordCount"
        :init-busy="inittingDefault"
        @init-default="initDefaultStructure"
        @select-doc="onSelectDoc"
        @context-menu="openDocContextMenu"
        @drop-node="handleDropNode"
        @drop-root="handleDropRoot"
        @add-doc-root="createDocAtRoot"
        @add-setting="createSettingAtRoot"
        @add-volume="createVolumeAtRoot"
        @add-prologue="createSpecialChapter('楔子')"
        @add-extra="createSpecialChapter('番外')"
        @add-article="createVolumeAtRoot"
        @goal-lock="onGoalLock"
        @open-trash="trashOpen = true"
        @add-doc-in="createDocInFolder"
        @cycle-status="handleCycleStatus"
      />

      <!-- 编辑器 -->
      <div
        class="editor-main"
        :class="{ 'ai-panel-open': contextPanelOpen, 'split-h': referencePanelOpen && splitDir === 'h', 'split-v': referencePanelOpen && splitDir === 'v' }"
        :style="referencePanelOpen ? { '--split-w': splitW + 'px', '--split-mh': splitH + 'px' } : undefined"
        v-show="mode === 'write'"
      >
        <!-- 方案二·左页页头：活动页指示 + 章名（分屏打开时才出现） -->
        <div v-if="referencePanelOpen" class="left-pane-head" :class="{ 'is-active': lastPane === 'left' }">
          <span class="lph-live" v-if="lastPane === 'left'">正在写</span>
          <span class="lph-title">{{ currentDocDisplayName || '未选择章节' }}</span>
        </div>
        <!-- 拼字房 HUD（C 形态）：顶部细横幅 + 右缘迷你榜；专注模式降级为胶囊+底部淡字 -->
        <div v-if="sprint.running && !focusMode" class="sprint-banner">
          <span class="sb-cd">{{ sprint.countdownText }}</span>
          <span class="sb-sep"></span>
          <span class="sb-me">我 第{{ sprint.myRank }}名 · {{ sprint.myWords }} 字</span>
          <span class="sb-sep"></span>
          <span class="sb-lead">{{ sprint.leadText }}</span>
          <button class="sb-btn" @click="sprint.openPanel()">房间 ▸</button>
        </div>
        <div v-if="sprint.running && !focusMode && sprint.room" class="sprint-rail">
          <div v-for="p in sprint.room.players" :key="p.user_id" class="sr-dot" :class="{ me: p.user_id === sprint.room.me, off: !p.online }" :title="p.nickname">
            <span class="sr-rk">{{ p.rank }}</span>
            <span class="sr-av">{{ sprint.avatar(p.nickname) }}</span>
            <span class="sr-wc">{{ p.words }}</span>
          </div>
        </div>

        <!-- 便签贴纸（C 方案）：贴在正文纸张右缘，实时量纸张对齐 -->
        <StickyNotes :project-id="projectId" />
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
          <!-- C 方案（2026-11-05 Boss 拍板）：空书不再让用户找目录，一键建第一章开写 -->
          <template v-if="chapterList.length === 0">
            <div class="ee-icon">✍️</div>
            <div class="ee-title">开始你的第一章</div>
            <div class="ee-sub">不用先研究目录——一键建好「第一章」直接开写</div>
            <button class="ee-go" :disabled="startingFirst" @click="startFirstChapter">{{ startingFirst ? '创建中…' : '✍️ 直接开写' }}</button>
            <button class="ee-ghost" @click="mobileSidebarOpen = true">先看看目录结构</button>
          </template>
          <template v-else>
            <p>👈 请选择或新建一个章节</p>
            <button class="ee-ghost" @click="mobileSidebarOpen = true">打开目录</button>
          </template>
        </div>
        <div v-else class="editor-wrapper" :class="{ 'show-line-numbers': lineNumbers, 'drop-target': leftDragOver }" @contextmenu.prevent="openContextMenu"
          @dragover.prevent="onLeftDragOver" @dragleave="onLeftDragLeave" @drop.prevent="onDropLeft">
          <div v-if="leftDragOver" class="drop-hint-left">📄 松开：在这页打开</div>
          <CharacterCardView
            v-if="activeDoc && activeDoc.type === 'character'"
            :doc="activeDoc"
            :editor="editor"
            @update-meta="onCharMetaUpdate"
            @update-title="onCharTitleUpdate"
          >
            <div class="tiptap-container">
              <editor-content v-if="editor" :editor="editor" />
            </div>
          </CharacterCardView>
          <template v-else>
            <div class="tiptap-container" :class="{ 'has-ch-heading': activeDoc && activeDoc.type === 'chapter' }">
              <!-- 分离方案：章节顶部虚拟章名头——动态「第N章 标题」，不进正文数据（C 破折号样式）。
                   放在滚动容器内、负 margin 压进纸张顶部：像正文第一行印在纸上、随内容一起滚动 -->
              <div v-if="activeDoc && activeDoc.type === 'chapter' && currentDocDisplayName" class="chapter-virtual-heading">
                <span class="vh-dash"></span>
                <span class="vh-title"><span v-if="chapterHeadingParts?.num" class="vh-num">{{ chapterHeadingParts.num }} </span>{{ chapterHeadingParts?.name }}</span>
                <span class="vh-dash vh-dash-r"></span>
              </div>
              <editor-content v-if="editor" :editor="editor" />
            </div>
          </template>
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
          :prev-chapter-id="prevChapterId"
          :current-chapter="currentChapterIndex"
          :get-text="getPlainTextForCheck"
          :sensitive-total="sensitiveTotal"
          @update:open="contextPanelOpen = $event"
          @jump="jumpToSetting"
          @insp-insert="insertInspToEditor"
          @open-canvas="canvasOpen = true"
        />

        <!-- 分屏：对等第二页（⇄可切左右/上下，侧栏可点/可拖进来） -->
        <SplitEditor
          v-show="referencePanelOpen"
          ref="splitRef"
          :documents="documents"
          :active-doc-id="activeDocId"
          :default-doc-id="splitDefaultId"
          :dir="splitDir"
          :active-pane="lastPane === 'right'"
          @close="referencePanelOpen = false"
          @swap="swapPanes"
          @toggle-dir="toggleSplitDir"
          @pane-focus="lastPane = 'right'"
          @resize-w="splitW = $event"
          @resize-h="splitH = $event"
        />
      </div>

      <!-- 图谱浮层（入口在工具箱「关系图」，右下角圆钮已撤） -->
      <div class="graph-fab" v-show="graphOpen" @click.stop>
        <div class="graph-fab-panel" :class="{ maximized: graphMaximized }">
          <button class="graph-resize-btn" @click="graphMaximized = !graphMaximized" :title="graphMaximized ? '缩小' : '放大'">
            <svg v-if="!graphMaximized" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M8 3H5a2 2 0 0 0-2 2v3"/><path d="M16 3h3a2 2 0 0 1 2 2v3"/><path d="M8 21H5a2 2 0 0 1-2-2v-3"/><path d="M16 21h3a2 2 0 0 0 2-2v-3"/></svg>
            <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M8 3v3a2 2 0 0 1-2 2H3"/><path d="M21 8h-3a2 2 0 0 1-2-2V3"/><path d="M3 16h3a2 2 0 0 1 2 2v3"/><path d="M16 21v-3a2 2 0 0 1 2-2h3"/></svg>
          </button>
          <button class="graph-close-btn" @click="graphOpen = false" title="收起图谱">✕</button>
          <RelationshipGraph
            :project-id="String(projectId)"
            :chapters="chapterList"
            :current-chapter="currentChapterIndex"
          />
        </div>
      </div>

      <!-- 审核模式 -->
      <ReviewView v-if="mode === 'review'" :project-id="projectId" @jump-to-source="handleReviewJump" @ai-fix="handleReviewAiFix" />

      <!-- AI 选区工具栏（选择时）。AI 全线暂下线（2026-11-09 Boss 指令），恢复=ENABLE_AI 改 true -->
      <div
        v-if="ENABLE_AI && aiToolbar.visible"
        class="ai-toolbar"
        :style="{ left: aiToolbar.x + 'px', top: aiToolbar.y + 'px' }"
      >
        <button @mousedown.prevent @click="prepareAIAction('continue')">续写</button>
        <button @mousedown.prevent @click="prepareAIAction('rewrite')">改写</button>
        <button @mousedown.prevent @click="prepareAIAction('polish')">润色</button>
        <button @mousedown.prevent @click="prepareAIAction('expand')">扩写</button>
      </div>

      <!-- AI 底部固定栏（方向输入 / 生成中 / 结果预览）。AI 暂下线，同上 -->
      <div v-if="ENABLE_AI && aiPhase !== 'idle'" class="ai-bottom-bar" :class="'phase-' + aiPhase">
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
        <div v-if="ctxHasSelection" @click="contextAction('thread')">🧵 记伏笔</div>
      </div>

      <!-- 文档树右键菜单 -->
      <div
        v-if="docContextMenu.visible"
        class="context-menu"
        :style="{ left: docContextMenu.x + 'px', top: docContextMenu.y + 'px' }"
      >
        <template v-if="docContextMenu.doc">
          <div v-if="ctxNewItem" @click="doCtxNew">{{ ctxNewItem.label }}</div>
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
          <div v-if="!isFolderNode(docContextMenu.doc) && docContextMenu.doc.type === 'chapter'" @click="openShareDoc">分享章节</div>
          <div v-if="!isFolderNode(docContextMenu.doc) && docContextMenu.doc.type === 'chapter' && docContextMenu.doc.share_code" @click="stopShareFromMenu" class="danger">停止分享</div>
          <div @click="deleteDoc" class="danger">删除</div>
        </template>
        <template v-else>
          <template v-if="docContextMenu.zone === 'related'">
            <div @click="newSettingFromMenu">新建设定</div>
          </template>
          <template v-else>
            <div @click="newDocUnder('chapter')">新建章节</div>
            <div @click="newDocUnder('volume')">新建卷</div>
            <div @click="createSpecialChapter('楔子'); closeDocContextMenu()">新建楔子</div>
            <div @click="createSpecialChapter('番外'); closeDocContextMenu()">新建番外</div>
          </template>
        </template>
      </div>
    </div>

    <!-- 底部状态栏 -->
    <div class="status-bar">
      <span class="status-item">{{ wordCount }} 字</span>
      <span v-if="isOffline" class="status-item offline-indicator" title="当前离线，内容已自动存在本机，恢复网络后会自动上传">📡 离线</span>
      <span class="status-item">{{ saveStatus }}</span>
      <!-- 敏感词检测：实时扫描，点开看命中清单，点词定位 -->
      <div class="toolbox-trigger" :class="{ open: sensitiveOpen }">
        <button class="status-item speech-btn sens-btn" :class="{ hit: sensitiveTotal > 0 }" @click.stop="sensitiveOpen = !sensitiveOpen" title="敏感词检测（发布前自查）">
          ◉ 敏感词 {{ sensitiveTotal }}
        </button>
        <div class="toolbox-popup sens-popup" v-show="sensitiveOpen" @click.stop>
          <div class="sens-head">风险词命中 · {{ sensitiveTotal }} 处</div>
          <div v-if="sensitiveHits.length === 0" class="sens-empty">✅ 未命中风险词</div>
          <div v-for="h in sensitiveHits" :key="h.word" class="sens-row" @click="jumpToSensitiveWord(h.word)">
            <span class="sens-word">{{ h.word }}</span>
            <span class="sens-count">×{{ h.count }}</span>
          </div>
          <div class="sens-foot">命中 ≠ 必删，发布前结合情节自查</div>
        </div>
      </div>

      <!-- 方案一·瘦身：左=看的信息，右=点的工具 -->
      <div class="status-spacer"></div>

      <!-- 码字锁：目标字数内拦切章/离开 -->
      <div class="toolbox-trigger" :class="{ open: wordLockOpen }">
        <button class="status-item speech-btn lock-btn" :class="{ on: !!wordLock }" @click.stop="wordLockOpen = !wordLockOpen" :title="wordLock ? `码字锁进行中 · 还差 ${lockRemaining} 字` : '码字锁：设个目标字数，写完才解锁'">
          {{ wordLock ? `🔒 差${lockRemaining}字` : '🔒 码字锁' }}
        </button>
        <div class="toolbox-popup lock-popup" v-show="wordLockOpen" @click.stop>
          <div class="sens-head">{{ wordLock ? '换目标 / 解锁' : '写满多少字解锁？' }}</div>
          <div class="lock-grid">
            <button v-for="t in [300, 500, 1000, 2000]" :key="t" class="lock-opt" @click="setWordLock(t)">{{ t }} 字</button>
          </div>
          <button v-if="wordLock" class="lock-cancel" @click="clearWordLock">放弃本次目标（解锁）</button>
        </div>
      </div>

      <!-- 听写：语音转文字（Chrome） -->
      <!-- 听写：2026-11-03 Boss 暂时下线（小程序 webview 不支持 Web Speech API，等火山 ASR 版再上） -->
      <button v-if="enableDictation" class="status-item speech-btn" :class="{ speaking: dictating }" @click="toggleDictation" :title="dictating ? '停止听写' : '🎙 听写（语音转文字）'">
        {{ dictating ? '⏹' : '🎙' }}
      </button>

      <!-- 手机排版预览（读者视角，正文实时同步） -->
      <button class="status-item speech-btn" @click="phonePreviewOpen = true" title="📱 手机排版预览（读者视角）">📱</button>

      <!-- 拼字房入口已深度融合进搭子悬浮窗「拼字」页签（状态栏不再单列） -->

      <!-- 显示设置：字号/行距/字距/底部留白 -->
      <DisplaySettings ref="dsRef" :overrides="displayOverrides" :line-numbers="lineNumbers" @update="onDisplaySetting" @toggle-line-numbers="onToggleLineNumbers" />

      <!-- 工具箱：低频功能收起来 -->
      <div class="toolbox-trigger" :class="{ open: toolboxOpen }">
        <button class="memory-btn" @click.stop="toolboxOpen = !toolboxOpen" title="工具箱">
          <span v-if="!toolboxOpen">⋯</span>
          <span v-else>✕</span>
        </button>
        <div class="toolbox-popup" v-show="toolboxOpen" @click.stop>
          <div class="toolbox-grid">
            <!-- AI 暂下线（2026-11-09）：记忆(AI 记忆)与 AI 对话入口收起，恢复=ENABLE_AI 改 true -->
            <button v-if="ENABLE_AI" class="tool-btn" :class="{ active: memoryOpen }" @click="memoryOpen = !memoryOpen; toolboxOpen = false">
              <span class="tool-icon">🧠</span>
              <span class="tool-label">记忆</span>
            </button>
            <button class="tool-btn" :class="{ active: statsOpen }" @click="statsOpen = !statsOpen; toolboxOpen = false">
              <span class="tool-icon">📊</span>
              <span class="tool-label">统计</span>
            </button>
            <!-- 显示设置：仅手机档入口（桌面用底栏 ⚙️，勿重复） -->
            <button class="tool-btn mp-only" @click="toolboxOpen = false; dsRef?.toggle()">
              <span class="tool-icon">Aa</span>
              <span class="tool-label">显示</span>
            </button>
            <button v-if="ENABLE_AI" class="tool-btn" :class="{ active: chatOpen }" @click="chatOpen = !chatOpen; toolboxOpen = false">
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

    <BuddyWindow :current-word-count="wordCount" :active-doc-id="activeDocId" />
    <!-- 键盘上方标点条：手机打字时的高频标点（选区点引号=包住） -->
    <PunctBar :show="keyboardOpen && editorFocused" :editor="editor" :baseline="openBaselineWords" />
    <SprintPanel />

    <!-- 拼字房专注模式降级：右上倒计时胶囊（贴退出胶囊）+ 底部一行淡字 -->
    <div v-if="sprint.running && focusMode" class="sprint-focus-pill" @click="sprint.openPanel()">🔥 {{ sprint.countdownText }} · 第{{ sprint.myRank }}名</div>
    <div v-if="sprint.running && focusMode" class="sprint-focus-line">{{ sprint.countdownText }} · 我第 {{ sprint.myRank }} 名 · 本局 +{{ sprint.myWords }} 字 · {{ sprint.leadText }}</div>

    <!-- 手机排版预览（读者视角） -->
    <PhonePreview
      :open="phonePreviewOpen"
      :title="phonePreviewTitle"
      :book-name="projectName || '未命名作品'"
      :text="phonePreviewText"
      :word-count="wordCount"
      :chapter-index="currentChapterIndex"
      :chapter-total="chapterList.length"
      @close="phonePreviewOpen = false"
    />

    <!-- 工具箱面板：记忆/伏笔/回收站/统计/AI对话（从状态栏触发，弹窗位置统一） -->
    <div v-if="ENABLE_AI && memoryOpen" class="panel-popup memory-popup" @click.stop>
      <div class="panel-head"><span>🧠 记忆</span><button @click="memoryOpen = false">✕</button></div>
      <MemoryPanel :project-id="String(projectId)" />
    </div>
    <div v-if="trashOpen" class="panel-popup trash-popup" @click.stop>
      <div class="panel-head"><span>🗑 回收站</span><button @click="trashOpen = false">✕</button></div>
      <TrashPanel ref="trashPanelRef" :project-id="String(projectId)" @restored="onTrashRestored" />
    </div>
    <div v-if="statsOpen" class="panel-popup stats-popup" @click.stop>
      <div class="panel-head"><span>📊 写作统计</span><button @click="statsOpen = false">✕</button></div>
      <WritingStatsPanel :project-id="projectId" />
    </div>
    <div v-if="ENABLE_AI && chatOpen" class="panel-popup chat-popup" @click.stop>
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
            <div class="help-row"><kbd>Ctrl+Shift+Z</kbd><span>专注模式</span></div>
            <div class="help-row"><kbd>Ctrl+/</kbd><span>本帮助</span></div>
            <div class="help-row"><kbd>Esc</kbd><span>关闭弹窗/退出专注</span></div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 新建文档弹窗 -->
    <CreateDocModal
      v-model="showCreateDocModal"
      :doc-type="createDocType"
      :folder-type-options="createDocFolderTypeOptions"
      :default-name="newDocName"
      @create="onCreateDocConfirm"
    />

    <!-- 首次进入：初始化向导（第一步首章名称 → 第二步作品相关文件） -->
    <div v-if="showSettingsInit" class="modal-overlay" @click="skipSettingsInit">
      <div class="settings-init-modal" @click.stop>
        <div class="si-steps">第 {{ settingsInitStep }} 步 / 共 2 步</div>

        <template v-if="settingsInitStep === 1">
          <h3>第一章叫什么？</h3>
          <p class="si-desc">只写章节名就好（如「雨夜」），第几章系统按位置自动排。</p>
          <input v-model="firstChapterNameInput" class="si-input" placeholder="章节名" @keyup.enter="settingsInitStep = 2" />
          <div class="si-actions">
            <button class="si-btn" @click="skipSettingsInit">跳过</button>
            <button class="si-btn primary" @click="settingsInitStep = 2">下一步</button>
          </div>
        </template>

        <template v-else>
          <h3>要为这本书建哪些作品相关文件？</h3>
          <p class="si-desc">不建也没关系，以后随时可以在左侧「作品相关」分区标题的 ＋ 补。</p>
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

    <!-- 记伏笔弹窗（选中正文右键触发，埋设章自动=当前章） -->
    <div v-if="threadModal.visible" class="modal-overlay" @click="threadModal.visible = false">
      <div class="settings-init-modal" @click.stop>
        <h3>🧵 记一条伏笔</h3>
        <p class="si-desc">埋设点自动关联当前章节（第 {{ currentChapterIndex }} 章），写到回收章时在右栏速查一键收</p>
        <label class="si-steps">标题</label>
        <input class="si-input" v-model="threadModal.title" placeholder="一句话说清这条伏笔" @keyup.enter="saveThreadFromSelection" />
        <label class="si-steps" style="margin-top:10px;display:block">类型</label>
        <div class="thread-type-chips">
          <span v-for="t in THREAD_TYPES" :key="t.value" class="thread-chip" :class="[t.value, { on: threadModal.type === t.value }]" @click="threadModal.type = t.value">{{ t.label }}</span>
        </div>
        <div class="thread-quote">「{{ threadModal.quote }}」</div>
        <div class="si-actions">
          <button class="si-btn" @click="threadModal.visible = false">取消</button>
          <button class="si-btn primary" @click="saveThreadFromSelection">记下</button>
        </div>
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

    <!-- 章节公开分享：一键直达（无弹窗）；管理走右键「停止分享」 -->

    <!-- 设定画布：全屏覆盖层（入口=右栏「设定」页签） -->
    <StoryCanvas
      v-if="canvasOpen"
      :project-id="projectId"
      :chapters="chapterList.map((d: any) => ({ id: d.id, title: chapterDisplayTitle(d, documents) }))"
      :current-chapter="currentChapterIndex"
      :get-selection="getSelectionForCanvas"
      @close="canvasOpen = false"
      @open-graph="canvasOpen = false; graphOpen = true"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { EditorContent } from '@tiptap/vue-3'
import { Editor } from '@tiptap/core'
import { Slice, Fragment } from '@tiptap/pm/model'
import StarterKit from '@tiptap/starter-kit'
import Highlight from '@tiptap/extension-highlight'
import { Table } from '@tiptap/extension-table'
import { TableRow } from '@tiptap/extension-table-row'
import { TableCell } from '@tiptap/extension-table-cell'
import { TableHeader } from '@tiptap/extension-table-header'
// ponytail: 4 fixed highlight colors, no config needed
const highlightColors = ['#ffeb3b', '#4caf50', '#2196f3', '#ff5252']
import { documentApi, projectApi, threadApi, buddyApi } from '../core/api'
import { mpPush, mpRelaunch } from '../core/mp'
import { scanSensitive } from '../core/sensitiveWords'
import { chapterDisplayTitle, stripChapterPrefix, chapterOrdinal, numToChinese } from '../core/chapterTitle'
import { editorContentFromText, stripContentChapterHeading } from '../core/editorContent'
import SearchPanel from '../components/SearchPanel.vue'
import SplitEditor from '../components/editor/SplitEditor.vue'
import PhonePreview from '../components/editor/PhonePreview.vue'
import ReviewView from '../components/ReviewView.vue'
import EditorHeader from '../components/editor/EditorHeader.vue'
import EditorSidebar from '../components/editor/EditorSidebar.vue'
import AIContextPanel from '../components/editor/AIContextPanel.vue'
import StoryCanvas from '../components/editor/StoryCanvas.vue'
import AIChatPanel from '../components/editor/AIChatPanel.vue'
import MemoryPanel from '../components/editor/MemoryPanel.vue'
import RelationshipGraph from '../components/editor/RelationshipGraph.vue'
import TrashPanel from '../components/editor/TrashPanel.vue'
import WritingStatsPanel from '../components/editor/WritingStatsPanel.vue'
import BuddyWindow from '../components/editor/BuddyWindow.vue'
import PunctBar from '../components/editor/PunctBar.vue'
import SprintPanel from '../components/editor/SprintPanel.vue'
import StickyNotes from '../components/editor/StickyNotes.vue'
import { useSprint } from '../composables/useSprint'
import CharacterCardView from '../components/editor/CharacterCardView.vue'
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

// 路由参数：新链接是 16 位 public_id，旧书签可能是数字 id。
// projectId 统一解析为内部数字 id（文档/记忆/便签等接口仍用数字）
const routeProjectParam = String(route.params.projectId || '')
let projectId = Number(routeProjectParam)
if (!Number.isFinite(projectId) || projectId <= 0) projectId = 0
try { if (projectId > 0) localStorage.setItem('xzj-active-project', String(projectId)) } catch {}

// public_id → 内部数字 id（必须在加载文档前完成）
async function resolveProjectId() {
  if (projectId > 0) return
  try {
    const res = await projectApi.get(routeProjectParam)
    const data = res.data?.data || res.data
    projectId = Number(data?.id) || 0
    try { localStorage.setItem('xzj-active-project', String(projectId)) } catch {}
  } catch (e: any) {
    if (e?.response?.status === 403 || e?.response?.status === 404) {
      toast.error('无权访问该项目')
      router.push('/bookshelf')
      throw e
    }
    throw e
  }
}

const editor = shallowRef<any>(null)
const documents = ref<any[]>([])
const activeDocId = ref<number | null>(null)
const currentDocName = ref('')
// 当前打开的文档对象（角色卡视图判断用）
const activeDoc = computed(() => documents.value.find((x: any) => x.id === activeDocId.value) || null)
// 顶栏章节名和侧栏一致：第N章 + 作者写的名字（序号自动）
const currentDocDisplayName = computed(() => {
  const d = documents.value.find((x: any) => x.id === activeDocId.value)
  if (d && d.type === 'chapter') return chapterDisplayTitle(d, documents.value)
  return currentDocName.value
})
// 虚拟章名头的「章号/章名」拆分（C 样式：章号绿色高亮）
const chapterHeadingParts = computed(() => {
  const d = activeDoc.value
  if (!d || d.type !== 'chapter') return null
  const m = /^(第[0-9零一二三四五六七八九十百千万两]+章)\s*(.*)$/.exec(String(currentDocDisplayName.value || '').trim())
  if (!m) return { num: '', name: String(currentDocDisplayName.value || '') }
  return { num: m[1], name: m[2] }
})
const projectName = ref('')

// 小程序全屏（mp-full）：编辑器顶栏隐藏，位置走 document.title → web-view 原生栏自动同步显示
// 原生栏标题居中且超长截尾（平台行为改不了）——Boss 2026-11-05：只显示「第X卷 第N章」（卷内章号与树一致=全书连续编号）
watch([projectName, currentDocDisplayName, documents, activeDocId], () => {
  const p = projectName.value
  const d = currentDocDisplayName.value
  const doc = documents.value.find((x: any) => x.id === activeDocId.value)
  if (doc?.type === 'chapter') {
    const volIdx = documents.value
      .filter((x: any) => x.type === 'volume' && !x.parent_id)
      .sort((a: any, b: any) => (a.sort_order || 0) - (b.sort_order || 0))
      .findIndex((v: any) => v.id === doc.parent_id)
    const chNo = chapterOrdinal(doc, documents.value) || 1
    document.title = volIdx >= 0
      ? `第${numToChinese(volIdx + 1)}卷 第${numToChinese(chNo)}章`
      : `第${numToChinese(chNo)}章` // 根级散章（楔子/番外等）没有卷
    return
  }
  document.title = d || p || '写作菌'
}, { immediate: true })
const loadingDocs = ref(false)
const wordCount = ref(0)
// 拼字房（单例 store）：HUD/面板消费。字数口径=按章折叠（与搭子增量同款）：
// 切章时结算旧章增量、新章以打开时的字数为基线——防止开局后打开/切换章节把整章存量算成本局新写。
const sprintSettledWords = ref(0) // 已结算的旧章增量之和
const sprintChBaseline = ref(-1) // 当前章基线；-1=待立（首次 wordCount 观测时立）
watch(activeDocId, () => {
  // openDoc 先变 activeDocId、后设 wordCount——此刻 wordCount 仍是旧章的字数
  if (sprintChBaseline.value >= 0) {
    sprintSettledWords.value += Math.max(0, wordCount.value - sprintChBaseline.value)
  }
  sprintChBaseline.value = -1
})
watch(wordCount, (v) => { if (sprintChBaseline.value < 0) sprintChBaseline.value = v })
const sprintSessionWords = computed(() =>
  sprintSettledWords.value + (sprintChBaseline.value >= 0 ? Math.max(0, wordCount.value - sprintChBaseline.value) : 0))
const sprint = useSprint()
const sprintInited = sprint.init(sprintSessionWords)
// 右栏工作台（C 方案）需要的两个桥：自检取正文纯文本 / 灵感「写进本章」插到光标
function getPlainTextForCheck(): string {
  try { return getEditorText(editor.value) } catch { return '' }
}
function insertInspToEditor(text: string) {
  try {
    editor.value?.commands?.insertContent(text + '\n')
    toast.success('已插入到光标处')
  } catch { toast.error('插入失败，请手动复制') }
}

/* ═══ 设定画布 ═══ */
const canvasOpen = ref(false)
// 画布摘录：读当前编辑器选区文字 + 所在章节
function getSelectionForCanvas(): { text: string; chapterTitle: string; chapterNo: number } | null {
  try {
    if (!editor.value) return null
    const { from, to } = editor.value.state.selection
    if (from >= to) return null
    const text = editor.value.state.doc.textBetween(from, to, '\n')
    if (!text.trim()) return null
    const d = documents.value.find((x: any) => x.id === activeDocId.value)
    return { text, chapterTitle: d ? chapterDisplayTitle(d, documents.value) : '当前章', chapterNo: currentChapterIndex.value || 1 }
  } catch { return null }
}
sprintInited.then((restored: number) => {
  // 断线/刷新恢复：本局已上报的字数作为种子，当前章基线直接以现值立住（不等 watch）
  if (restored > 0) {
    sprintSettledWords.value = restored
    sprintChBaseline.value = wordCount.value
  }
})
watch(() => sprint.running, (on) => {
  // 新一局开跑：清零重计，基线立当前值
  if (on) {
    sprintSettledWords.value = 0
    sprintChBaseline.value = wordCount.value
  }
})
const saveStatus = ref('')
// Boss 2026-11-05：小程序胶囊只报异常（保存失败/离线）——「编辑中/已保存」常态是噪音，清除
const mpSaveChip = computed(() => {
  const s = saveStatus.value
  if (!s || /^(编辑中|保存中|已保存|就绪|📡 离线读取)/.test(s)) return ''
  return ' · ' + s
})
// 手机档 H 方案：贴边把手 + 玻璃工具轨 + 显示设置远程开关
// 手机：键盘弹起时收起底部弹层（fixed bottom 元素会被键盘顶起挡住视线）
// keyboardOpen：软键盘弹起时翻章条整体隐藏——它悬在键盘上方压正文，而打字时用不到翻章
const keyboardOpen = ref(false)
// 撤销地板：本章打开时的字数——标点条撤回到此为止，防止连点把全文撤没
const openBaselineWords = ref(0)
// 正文编辑器是否聚焦（标点条出现条件之一；声明必须顶层，模板要用）
const editorFocused = ref(false)
const mpRailOpen = ref(false)
const dsRef = ref<any>(null)
const draftBanner = ref<{ docId: number; text: string } | null>(null)

// E1: 场景导航已下线（Boss 2026-10-23：没啥用，按钮+快捷键+功能全清）

// AI 全线暂下线（2026-11-09 Boss 指令：暂时不接入 AI，润色/改写/续写/自检/AI对话/记忆全藏，代码保留可恢复）
const ENABLE_AI = false
const contextMenu = ref({ visible: false, x: 0, y: 0 })
// ponytail: 右键时编辑器会失焦丢选区，打开菜单前缓存
let cachedSelection: { text: string; start: number; end: number } | null = null
const docContextMenu = ref({ visible: false, x: 0, y: 0, doc: null as any, zone: 'chapters' as 'chapters' | 'related' })
const showCreateDocModal = ref(false)
const newDocName = ref('')
const createDocType = ref<string>('chapter')
const createDocParent = ref<number | null>(null)
const createDocFolderTypeOptions = computed(() => {
  // 只有「新建设定」弹窗需要选类型；卷/章节的弹窗都是单一用途
  if (createDocType.value !== 'setting') return undefined
  // 大纲/人物/世界观/章节大纲(容器)每书一份，已存在的不再给选项——否则点了「创建」树里不多东西
  // 章纲文件走「章节大纲」文件夹内的 ＋ 新建，不在此弹窗
  const existing = new Set(documents.value.filter(d => !d.parent_id).map(d => d.type))
  return [
    { value: 'outline', label: '📋 大纲' },
    { value: 'characters', label: '👥 人物' },
    { value: 'worldbuilding', label: '🌍 世界观' },
    { value: 'chapter_outline', label: '📑 章节大纲' },
    { value: 'note', label: '📄 笔记' },
  ].filter(o => o.value === 'note' || !existing.has(o.value))
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

// A 翻章条：只算可写章节（volume 不算），按全局 sort_order 线性排序
const navChapters = computed(() => chapterList.value.filter(d => d.type === 'chapter'))
const navIndex = computed(() => {
  const idx = navChapters.value.findIndex(d => d.id === activeDocId.value)
  return idx >= 0 ? idx + 1 : 0
})
const navPrev = computed(() => (navIndex.value >= 1 ? navChapters.value[navIndex.value - 2] : null) || null)
const navNext = computed(() => (navIndex.value ? navChapters.value[navIndex.value] : null) || null)

// 最后一章的「+ 新章」：弹新建章节框，不静默建章（Boss 2026-11-06 拍板）——
// 挂在同卷下，sort_order 顺排由 createDoc 统一算；名字留空=自动编号「第N章」，建完直接开写
function createNextChapter() {
  const cur = documents.value.find(d => d.id === activeDocId.value)
  openCreateDocModal(cur?.parent_id ?? null, 'chapter')
}

// C 首次直达：空书一键建「第一章」（顺带跑默认结构初始化，保证有第一卷挂靠）
const startingFirst = ref(false)
async function startFirstChapter() {
  if (startingFirst.value) return
  startingFirst.value = true
  try {
    let vol = documents.value.find(d => d.type === 'volume')
    if (!vol) { await initDefaultStructure(); vol = documents.value.find(d => d.type === 'volume') }
    const res = await documentApi.create({
      project_id: projectId, title: '', type: 'chapter', // unnamed：显示层自动编号「第一章」
      parent_id: vol?.id || undefined, sort_order: 1,
    })
    await loadDocs()
    const nd = res.data?.data || res.data
    if (nd) openDoc(nd)
  } catch (e: any) {
    toast.error(e.response?.data?.error || '创建失败')
  } finally {
    startingFirst.value = false
  }
}

// B 长目录：手机抽屉打开 → 收起无关卷、自动滚到当前章（几百章不用手动找）
// （挂在 mobileSidebarOpen 声明后，见下方 watch）
/* 方案B速查：上一章 doc id（供右栏「上一章结尾」卡取内容）——只认真正的 chapter，
   chapterList 混着卷容器：第一章的「上一章」其实是卷，会把卷 id 当章传出去 */
const prevChapterId = computed(() => {
  const list = chapterList.value.filter((d: any) => d.type === 'chapter')
  const i = list.findIndex(d => d.id === activeDocId.value) - 1
  return i >= 0 ? list[i].id : null
})

/* ═══ A. 伏笔全套：右键记伏笔（弹窗）→ threadApi 落库 → 速查联动刷新 ═══ */
const THREAD_TYPES = [
  { value: 'foreshadowing', label: '伏笔' },
  { value: 'hook', label: '钩子' },
  { value: 'promise', label: '承诺' },
  { value: 'suspense', label: '悬念' },
] as const

const ctxHasSelection = ref(false)
const threadModal = ref({ visible: false, title: '', type: 'foreshadowing', quote: '' })

async function saveThreadFromSelection() {
  const m = threadModal.value
  const title = m.title.trim() || m.quote.slice(0, 16)
  try {
    await threadApi.create(projectId, {
      type: m.type as any,
      title,
      description: m.quote ? `「${m.quote.slice(0, 60)}${m.quote.length > 60 ? '…' : ''}」` : '',
      status: 'open',
      planted_chapter: currentChapterIndex.value || null,
    })
    threadModal.value.visible = false
    window.dispatchEvent(new CustomEvent('xzj-threads-changed'))
    toast.success(`🧵 已记下：「${title}」· 埋设于第 ${currentChapterIndex.value} 章`)
  } catch {
    toast.error('记伏笔失败，请重试')
  }
}

/* ═══ B1. 敏感词检测：编辑防抖扫描，点词走全文搜索定位 ═══ */
const sensitiveOpen = ref(false)
const sensitiveHits = ref<{ word: string; count: number }[]>([])
const sensitiveTotal = computed(() => sensitiveHits.value.reduce((s, h) => s + h.count, 0))
let sensitiveTimer: ReturnType<typeof setTimeout> | null = null

function rescanSensitive() {
  if (sensitiveTimer) clearTimeout(sensitiveTimer)
  sensitiveTimer = setTimeout(() => {
    if (!editor.value) return
    sensitiveHits.value = scanSensitive(getEditorText(editor.value))
  }, 400)
}

function jumpToSensitiveWord(word: string) {
  sensitiveOpen.value = false
  searchVisible.value = true
  onSearchQueryUpdate(word)
}

/* ═══ B2. 码字锁：目标字数内拦切章/离开编辑器 ═══ */
const wordLock = ref<{ target: number; start: number } | null>(null)
const wordLockOpen = ref(false)
const lockRemaining = computed(() =>
  wordLock.value ? Math.max(0, wordLock.value.target - Math.max(0, wordCount.value - wordLock.value.start)) : 0
)

function setWordLock(target: number) {
  wordLock.value = { target, start: wordCount.value }
  wordLockOpen.value = false
  toast.success(`🔒 码字锁已上：写满 ${target} 字前不换章（当前章 ${wordCount.value} 字起算）`)
}

function clearWordLock() {
  wordLock.value = null
  wordLockOpen.value = false
  toast.info('已解锁')
}

/* 目标卡「🔒 还差 N 字」：直接把剩余量设成码字锁冲刺 */
function onGoalLock(remaining: number) {
  if (remaining > 0) setWordLock(remaining)
}

watch(wordCount, () => {
  if (wordLock.value && lockRemaining.value <= 0) {
    const t = wordLock.value.target
    wordLock.value = null
    toast.success(`🎉 码字锁达成：+${t} 字！已自动解锁`)
  }
})

function guardWordLock(): boolean {
  if (!wordLock.value || lockRemaining.value <= 0) return true
  return window.confirm(`码字锁：还差 ${lockRemaining.value} 字解锁（目标 ${wordLock.value.target} 字）。\n确定要离开吗？`)
}

/* ═══ 手机排版预览：状态栏📱按钮 → 右侧抽屉，正文实时同步 ═══ */
const phonePreviewOpen = ref(false)
const phonePreviewText = ref('')
let phonePreviewTimer: ReturnType<typeof setTimeout> | null = null

const phonePreviewTitle = computed(() => {
  if (!activeDoc.value || activeDoc.value.type !== 'chapter') return currentDocName.value || '未命名'
  return chapterDisplayTitle(activeDoc.value, documents.value) || currentDocName.value || '未命名'
})

function syncPhonePreview() {
  if (phonePreviewTimer) clearTimeout(phonePreviewTimer)
  phonePreviewTimer = setTimeout(() => {
    if (!editor.value) return
    phonePreviewText.value = getEditorText(editor.value)
  }, 500)
}

/* ═══ B3. 听写：Web Speech API（Chrome） ═══ */
// 听写：暂时下线（微信 webview 无 Web Speech API，点了只报"不支持"误导用户；等火山 ASR 按住说话版）
const enableDictation = false
const dictating = ref(false)
let recog: any = null

function toggleDictation() {
  const SR = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition
  if (!SR) {
    toast.info('当前浏览器不支持听写，推荐用 Chrome / Edge')
    return
  }
  if (dictating.value) {
    recog?.stop()
    return
  }
  recog = new SR()
  recog.lang = 'zh-CN'
  recog.continuous = true
  recog.interimResults = false
  recog.onresult = (e: any) => {
    let text = ''
    for (let i = e.resultIndex; i < e.results.length; i++) {
      if (e.results[i].isFinal) text += e.results[i][0].transcript
    }
    if (text && editor.value) editor.value.commands.insertContent(text)
  }
  recog.onend = () => { dictating.value = false }
  recog.onerror = (e: any) => {
    if (e.error !== 'aborted' && e.error !== 'no-speech') toast.error(`听写中断：${e.error}`)
    dictating.value = false
  }
  try {
    recog.start()
    dictating.value = true
    toast.info('🎙 听写已开启，说话即入文（再点一次停止）')
  } catch {
    dictating.value = false
    toast.error('听写启动失败')
  }
}

const memoryOpen = ref(false)
const graphOpen = ref(false)
const trashOpen = ref(false)
const trashPanelRef = ref<any>(null)
const graphMaximized = ref(false)

// 通用确认弹窗
const confirmModal = ref<{ title: string; desc: string; action: () => void } | null>(null)

// ===== 章节公开分享 =====
const shareDoc = ref<any>(null)
const shareCode = ref('')
const shareLoading = ref(false)
// 树外场景（分享卡片/确认弹窗）的章节名：unnamed 章要显示自动编号「第N章」，不是空串
const docDisplayTitle = (d: any) => (d?.type === 'chapter' ? chapterDisplayTitle(d, documents.value) : (d?.title || ''))
const isMiniProgramEnv = /miniProgram/i.test(navigator.userAgent) || (window as any).__wxjs_environment === 'miniProgram'
// 小程序里预载 jweixin（返回时要用 miniProgram.navigateBack 关 web-view 层）
if (isMiniProgramEnv && !(window as any).wx) {
  const s = document.createElement('script')
  s.src = 'https://res.wx.qq.com/open/js/jweixin-1.6.0.js'
  document.head.appendChild(s)
}
const shareLink = computed(() => shareCode.value ? `${location.origin}/read/${shareCode.value}` : '')

// 一键分享：没开启过就自动开启；小程序直接拉起转发页，浏览器直接复制链接
async function openShareDoc() {
  const doc = docContextMenu.value.doc
  closeDocContextMenu()
  if (!doc) return
  shareDoc.value = doc
  if (!doc.share_code) {
    shareLoading.value = true
    try {
      const res = await documentApi.share(doc.id)
      doc.share_code = res.data?.data?.share_code || ''
    } catch (e: any) {
      toast.error(e.response?.data?.error || '开启失败')
      return
    } finally { shareLoading.value = false }
  }
  shareCode.value = doc.share_code || ''
  notifyMpShare()
  if (isMiniProgramEnv) {
    pushMpShare() // 原生分享页一键转发
    return
  }
  copyShareLink()
  toast.success('分享已开启，链接已复制')
}

async function stopShareFromMenu() {
  const doc = docContextMenu.value.doc
  closeDocContextMenu()
  if (!doc) return
  confirmModal.value = {
    title: '停止分享',
    desc: `「${docDisplayTitle(doc)}」的分享链接将立刻失效，已发出的链接也无法再打开。`,
    action: async () => {
      try {
        await documentApi.unshare(doc.id)
        doc.share_code = null
        toast.success('已停止分享')
        await loadDocs()
      } catch (e: any) {
        toast.error(e.response?.data?.error || '操作失败')
      }
    }
  }
}

function copyShareLink() {
  navigator.clipboard?.writeText(shareLink.value).then(() => toast.success('链接已复制')).catch(() => {})
}

// 通知小程序壳：转发卡片改为本章（web-view postMessage 在「转发/后退」时才送达壳）
function notifyMpShare() {
  const w = window as any
  w.wx?.miniProgram?.postMessage?.({
    data: { type: 'xzj-share', code: shareCode.value, title: docDisplayTitle(shareDoc.value), book: projectName.value, ts: Date.now() }
  })
}

// 拉起壳的原生分享页（参数带在页面路径里，卡片零竞态——比 postMessage 可靠）
function pushMpShare() {
  const q = `code=${shareCode.value}&title=${encodeURIComponent(docDisplayTitle(shareDoc.value))}&book=${encodeURIComponent(projectName.value || '')}`
  mpPush('/pages/share/share?' + q)
}
const renameModal = ref<{ docId: number; title: string } | null>(null)
const renameInputRef = ref<HTMLInputElement | null>(null)

// 顶部工作模式：计划 / 写作 / 审核
type EditorMode = 'write' | 'review' // 构思已删（2026-11-01 Boss 指令）；审核隐藏保留
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
const contextPanelOpen = ref(true) // 右栏工作台常驻展开（Boss 2026-11：桌面不要收起态）；false 仅在分屏占右缘时暂避
const referencePanelOpen = ref(false) // 分屏（对等第二页）
// 分屏关掉 → 右栏自动回来（SplitEditor @close 和 toggleReference 两条路都归这里管）
watch(referencePanelOpen, (v) => { if (!v) contextPanelOpen.value = true })
const splitW = ref(360) // 左右分屏面板宽（拖动调整，localStorage 记忆）
const splitH = ref(400) // 上下分屏面板高
try { splitH.value = Math.round(window.innerHeight * 0.45) } catch {}
const splitDir = ref<'h' | 'v'>('h') // 左右 / 上下分屏，localStorage 记忆
try { splitDir.value = localStorage.getItem('xzj-split-dir') === 'v' ? 'v' : 'h' } catch {}
const lastPane = ref<'left' | 'right'>('left') // 活动页=最后聚焦的那页，侧栏点章开进它
const splitRef = ref<any>(null)
const leftDragOver = ref(false)
// 点分隔条：两页文档互换（左右位置对调）
async function swapPanes() {
  const rightId = splitRef.value?.getSelected?.()
  const leftId = activeDocId.value
  if (!rightId || !leftId || rightId === leftId) return
  const rd = documents.value.find((d: any) => d.id === rightId)
  const ld = documents.value.find((d: any) => d.id === leftId)
  if (!rd || !ld) return
  await openDoc(rd)                 // 左页开右页的章
  splitRef.value?.openDoc(leftId)   // 右页开左页的章
  toast.success('已互换两页')
}
function toggleSplitDir() {
  splitDir.value = splitDir.value === 'h' ? 'v' : 'h'
  try { localStorage.setItem('xzj-split-dir', splitDir.value) } catch {}
}
const splitDefaultId = ref<number | null>(null) // 打开分屏时的默认章（上一章）
// 分屏打开时，容器类型点了不进右页（卷/人物容器没有正文）
const SPLIT_SKIP = ['volume', 'characters', 'outline', 'worldbuilding', 'chapter_outline']
// 侧栏点章 → 开进「活动页」（最后聚焦的那页）；点到对面正开着的章则去对面
function onSelectDoc(doc: any) {
  if (referencePanelOpen.value && lastPane.value === 'right' && doc && doc.id !== activeDocId.value
      && !SPLIT_SKIP.includes(doc.type) && doc.type !== 'character') {
    splitRef.value?.openDoc(doc.id)
    return
  }
  openDoc(doc)
}
// 左页拖放目标：侧栏拖章过来松手=在左页打开
function onLeftDragOver(e: DragEvent) {
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  leftDragOver.value = true
}
function onLeftDragLeave(e: DragEvent) {
  const rt = e.relatedTarget as Node | null
  if (rt && (e.currentTarget as Node).contains(rt)) return // 子元素间移动不算离开
  leftDragOver.value = false
}
function onDropLeft(e: DragEvent) {
  leftDragOver.value = false
  const id = Number(e.dataTransfer?.getData('text/plain'))
  if (!id) return
  const doc = documents.value.find((d: any) => d.id === id)
  if (!doc || SPLIT_SKIP.includes(doc.type)) return
  openDoc(doc)
}
function toggleReference() {
  referencePanelOpen.value = !referencePanelOpen.value
  if (referencePanelOpen.value) {
    // 分屏和右栏 AI 面板同贴右缘，同开互相盖——开分屏先收 AI 面板
    contextPanelOpen.value = false
    // 默认章：上一章（对照续写）；当前就是第一章时取下一章——只认 chapter，chapterList 里混着卷容器
    const others = chapterList.value.filter((d: any) => d.type === 'chapter' && d.id !== activeDocId.value)
    splitDefaultId.value = prevChapterId.value ?? others[0]?.id ?? null
  }
}
const focusMode = ref(false) // 专注模式：隐藏侧栏+顶栏，编辑器全屏
// ponytail-mobile: 移动端抽屉侧栏（A 方案）——显示由 CSS @media 管，这里只管开合状态
const mobileSidebarOpen = ref(false)
// B 长目录：手机抽屉打开 → 收起无关卷、自动滚到当前章（几百章不用手动找）
watch(mobileSidebarOpen, async (open) => {
  if (!open || !activeDocId.value) return
  const active = documents.value.find(d => d.id === activeDocId.value)
  const keepVol = active?.parent_id || null
  for (const d of documents.value) {
    if (d.type === 'volume') d._expanded = d.id === keepVol
  }
  await nextTick()
  setTimeout(() => {
    document.querySelector('.doc-tree .tree-row.active')?.scrollIntoView({ block: 'center' })
  }, 150) // 等 CSS transform 抽屉滑入完成再滚，否则滚的是移动前的位置
})

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

/* ═══ 今日码字上报：章内字数变化聚合成净增（切章重立基线不计入），45s 一报 → POST /buddy/report
   目标卡/统计面板从 /buddy/today 取真今日（跨刷新/跨设备一致） ═══ */
let reportLastCount = -1
let reportPending = 0
watch(activeDocId, () => { reportLastCount = -1 }) // 切章：跨章字数差不计入今日
watch(wordCount, (n) => {
  if (reportLastCount < 0) { reportLastCount = n; return }
  reportPending += n - reportLastCount
  reportLastCount = n
})
async function flushWordsReport() {
  if (!reportPending) return
  const d = reportPending
  reportPending = 0
  try {
    await buddyApi.reportWords(d)
  } catch {
    reportPending += d // 断网攒着下一拍一起报
  }
}
const reportTimer = setInterval(flushWordsReport, 45000)
onBeforeUnmount(() => clearInterval(reportTimer))
window.addEventListener('pagehide', flushWordsReport) // 关页前尽力报最后一笔（axios 同步发不保证，尽力而为）
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
  { value: 'chapter_outline', label: '📑 章节大纲', checked: true },
])
const SETTING_INIT_TYPES = ['outline', 'characters', 'worldbuilding', 'chapter_outline']

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
// 段落行号显示（默认关，localStorage 独立键）
const LN_KEY = 'xzj-line-numbers'
const lineNumbers = ref(false)
try {
  displayOverrides.value = JSON.parse(localStorage.getItem(DS_KEY) || '{}')
  delete displayOverrides.value.bottomPad // 旧版「底部留白」已并入视线位置，清掉残留
} catch {}
try { lineNumbers.value = localStorage.getItem(LN_KEY) === '1' } catch {}

function onToggleLineNumbers(on: boolean) {
  lineNumbers.value = on
  try { localStorage.setItem(LN_KEY, on ? '1' : '0') } catch {}
}

const eyeLineRatio = computed(() => (displayOverrides.value.eyeLine ?? 50) / 100) // 默认 50%=屏幕正中（Boss 定）
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
  // 手机档这些弹窗没有常驻触发钮可再点关闭，点空白处统一收
  if (sensitiveOpen.value) sensitiveOpen.value = false
  if (wordLockOpen.value) wordLockOpen.value = false
  dsRef.value?.close()
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
  try { await resolveProjectId() } catch { return }
  initEditor()
  window.addEventListener('keydown', handleGlobalKeydown)
  document.addEventListener('click', closeToolboxOnClickOutside)
  window.addEventListener('online', onOnlineStatusChange)
  window.addEventListener('offline', onOnlineStatusChange)
  if (window.visualViewport) {
    // 基线=历史最高视口：有的机型键盘分两步缩（每步<150px），逐次差值会漏判；
    // 旋转/折叠后 400ms 重立基线。iOS WKWebView / 安卓 WebView / 鸿蒙 ArkWeb（Chromium 系）均支持 visualViewport
    let vvFullH = window.visualViewport.height
    window.visualViewport.addEventListener('resize', () => {
      const vh = window.visualViewport!.height
      if (vh > vvFullH) vvFullH = vh
      keyboardOpen.value = vvFullH - vh > 150
      if (keyboardOpen.value) {
        dsRef.value?.close()
        sensitiveOpen.value = false
        wordLockOpen.value = false
        toolboxOpen.value = false
      }
    })
    window.addEventListener('orientationchange', () => setTimeout(() => {
      vvFullH = window.visualViewport!.height
      keyboardOpen.value = false
    }, 400))
  }
  loadProjectName()
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
    const res = await projectApi.get(routeProjectParam)
    const data = res.data?.data || res.data
    projectName.value = data?.name || ''
  } catch (e: any) {
    // 项目不存在或不属于当前用户：直接弹回书架，别晾一个空编辑器
    if (e?.response?.status === 403 || e?.response?.status === 404) {
      toast.error('无权访问该项目')
      router.push('/bookshelf')
      return
    }
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
  try { recog?.stop() } catch {}
  if (sensitiveTimer) clearTimeout(sensitiveTimer)
  if (phonePreviewTimer) clearTimeout(phonePreviewTimer)
  if (editor.value) editor.value.destroy()
  if (saveTimer) clearTimeout(saveTimer)
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
      StarterKit.configure({ undoRedo: { newGroupDelay: 200 } }), // 撤销分组调细：一键退一小组，不是一长串
      Highlight.configure({ multicolor: true }),
      Table.configure({ resizable: true }),
      TableRow,
      TableHeader,
      TableCell,
      AIMark,
      AIStrikethrough,
    ],
    // ponytail: 粘贴强制纯文本，去掉外部加粗/斜体等格式。
    // 多行文本按行成段（此前 insertText 整段塞进当前段落，VS Code 复制来的换行全被吃成空格，
    // 行号/段落全部失效）——首行接在光标处，其余行各成一段，行间空行保留。
    editorProps: {
      handlePaste: (view, event) => {
        const text = event.clipboardData?.getData('text/plain')
        if (!text) return false
        const lines = text.replace(/\r\n?/g, '\n').split('\n')
        while (lines.length && !lines[lines.length - 1].trim()) lines.pop()
        if (!lines.length) return true
        const { state } = view
        const paras = lines.map(l =>
          state.schema.nodes.paragraph.create(null, l.length ? [state.schema.text(l)] : undefined)
        )
        const slice = new Slice(Fragment.fromArray(paras), 1, 1)
        view.dispatch(state.tr.replaceSelection(slice))
        return true
      },
    },
    onUpdate: ({ editor: ed }) => {
      if (isSettingContent.value) return
      const plainText = getEditorText(ed)
      wordCount.value = countWords(plainText)
      saveStatus.value = '编辑中...'
      rescanSensitive()
      syncPhonePreview()
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

  // 分屏：左编辑器聚焦=左页成为活动页（侧栏点章开进它）；标点条只在正文聚焦时出现
  editor.value.on('focus', () => { lastPane.value = 'left'; editorFocused.value = true })
  editor.value.on('blur', () => { editorFocused.value = false })

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

function getEditorText(ed = editor.value) {
  if (!ed) return ''
  const blocks: string[] = []
  ed.state.doc.forEach((node: any) => {
    blocks.push(node.textContent || '')
  })
  return blocks.join('\n')
}

// 字数=去掉换行符后的字符数——回车分段不算字（否则每按一次回车 +1，空段落也 +1）
function countWords(text: string): number {
  return text.replace(/\n/g, '').length
}

// ponytail: save/load as HTML to preserve marks (highlight etc.)
function getEditorHTML(ed = editor.value) {
  return ed ? ed.getHTML() : ''
}

// —— 一键排版 ——
// 处理"内容写好了但排版乱"：
// ①去每段首尾空白（含手打全角缩进）②删空段 ③统一首行缩进两个全角空格 ④段间不留空行。
// ⑤断行合并：VS Code/记事本硬换行（一自然段拆成多行、每行一段），按"上一行是否句末标点收尾"
//    判断合并——句末标点收尾=自然段结束，否则并入同段（下行以引号/破折号开头强制新段，保护对话）。
// ⑥粘连段拆分：粘贴时换行被吃成空格的"巨型段落"（。 下一句 的形态），按 句末标点+空白 切回多段。
// ⑦markdown 标题行：段首 "# / ## / ### " 前缀（拷来的 md 源码）转成真标题节点。
// 标题(h1-h3)、表格、代码块等非段落节点原样保留；marks（高亮）通过沿用源段首个 text node 保住。
// 单个 ProseMirror 事务完成，Ctrl+Z 一步可撤销。
function oneClickFormat() {
  const ed = editor.value
  if (!ed) return
  const { state, view } = ed
  const { tr } = state
  const INDENT = '\u3000\u3000' // 两个全角空格

  // 句末标点 = 自然段结束；对话开头 = 强制新段；粘连段的"标点后空白"切分点
  const endsParagraph = (t: string) => /[。！？…；~!?~”』」）)"']/.test(t.slice(-1)) || /\.{3,6}$/.test(t)
  const startsDialogue = (t: string) => /^[“「『（《—"']/.test(t)
  // 句末标点（含 ...... 省略号）后跟空白 = 被吃掉的换行
  const SPACE_BREAK = /(?<=[。！？…；~”』」）"']|\.{3,6})[ \t]+/

  // 按"连续相邻段落"分 run（中间隔着标题/表格的段不跨 run 处理）
  const runs: { pos: number; node: any }[][] = []
  let cur: { pos: number; node: any }[] = []
  state.doc.forEach((node: any, offset: number) => {
    if (node.type.name === 'paragraph') cur.push({ pos: offset, node })
    else if (cur.length) { runs.push(cur); cur = [] }
  })
  if (cur.length) runs.push(cur)

  let changed = 0, mergedCount = 0, splitCount = 0, headingCount = 0
  for (const run of runs) {
    // 1) 展平：清洗 → 行首 # 转 heading 行标记 → 粘连段按空格切分
    type Line = { text: string; marks: any[]; heading?: number }
    const lines: Line[] = []
    for (const { node } of run) {
      const cleaned = node.textContent.replace(/^[\s\u3000]+|[\s\u3000]+$/g, '')
      if (!cleaned) continue
      const firstText = node.content.firstChild
      const marks: any[] = firstText?.marks ?? []
      // md 残迹清理：**加粗** 星号、分隔线（---/***，含段中前缀形态）
      const mdClean = (s: string) => s
        .replace(/\*\*([^*]+)\*\*/g, '$1')
        .replace(/^(-{3,}|\*{3,}|_{3,})\s+/, '')
        .trim()
      const isDivider = (s: string) => /^(-{3,}|\*{3,}|_{3,})$/.test(s)
      // markdown 标题前缀（拷来的 md 源码形态："# 第1章 标题 正文..."）
      // 标题取法：首词是「第X章/楔子/序章/番外…」且次词短且非句子 → 连次词一起当标题，防"第1章 开学第一课"被腰斩
      const m = /^(#{1,3})\s+([^\s].*)$/.exec(cleaned)
      if (m) {
        const level = m[1].length
        const tokens = m[2].split(' ').filter(Boolean)
        let title = tokens[0] || ''
        let restParts = tokens.slice(1)
        if (/^(第.{1,4}章|楔子|序章|序言|番外|尾声|终章|引子|Chapter)/.test(title)
            && restParts.length && restParts[0].length <= 12 && !/[。！？…；]/.test(restParts[0])) {
          title += ' ' + restParts[0]
          restParts = restParts.slice(1)
        }
        lines.push({ text: title, marks, heading: level })
        for (const part of restParts.join(' ').split(SPACE_BREAK)) {
          const pt = mdClean(part.replace(/^[\s\u3000]+|[\s\u3000]+$/g, ''))
          if (pt && !isDivider(pt)) lines.push({ text: pt, marks })
        }
        headingCount++
        continue
      }
      for (const part of cleaned.split(SPACE_BREAK)) {
        const pt = mdClean(part.replace(/^[\s\u3000]+|[\s\u3000]+$/g, ''))
        if (pt && !isDivider(pt)) lines.push({ text: pt, marks })
      }
    }
    splitCount += lines.length // 先记总行数，后面用 run 原段数折算
    // 2) 断行合并分组（句末标点收尾不合并；对话开头新开）
    const groups: Line[][] = []
    for (const line of lines) {
      const lastGroup = groups[groups.length - 1]
      const lastLine = lastGroup?.[lastGroup.length - 1]
      const canMerge = lastLine && !lastLine.heading
        && !endsParagraph(lastLine.text)
        && !startsDialogue(line.text)
      if (canMerge) lastGroup.push(line)
      else groups.push([line])
    }
    // 3) 生成新节点序列
    const newNodes: any[] = []
    for (const g of groups) {
      if (g.length > 1) mergedCount += g.length - 1
      const head = g[0]
      const joined = g.map(x => x.text).join('')
      if (head.heading) {
        newNodes.push(state.schema.nodes.heading.create(
          { level: head.heading }, state.schema.text(head.text, head.marks)))
      } else {
        newNodes.push(state.schema.nodes.paragraph.create(null, state.schema.text(INDENT + joined, head.marks)))
      }
    }
    // 4) 与原 run 对比，变了才整体替换（run 起止 = 首段 pos 到末段结尾）
    const startPos = run[0].pos
    const endPos = run[run.length - 1].pos + run[run.length - 1].node.nodeSize
    const oldSig = run.map(x => x.node.textContent).join('\u0001')
    const newSig = newNodes.map((n: any) => n.textContent).join('\u0001')
    if (oldSig !== newSig) {
      tr.replaceWith(startPos, endPos, newNodes)
      changed++
    }
  }
  splitCount = Math.max(0, splitCount - runs.reduce((s, r) => s + r.length, 0))

  if (changed > 0) {
    tr.setSelection(state.selection.constructor.near(tr.doc.resolve(Math.min(state.selection.from, tr.doc.content.size))))
    view.dispatch(tr)
    autoSave()
    const bits: string[] = [`已整理 ${changed} 组段落`]
    if (mergedCount > 0) bits.push(`合并 ${mergedCount} 处断行`)
    if (splitCount > 0) bits.push(`拆开 ${splitCount} 处粘连`)
    if (headingCount > 0) bits.push(`转换 ${headingCount} 个标题`)
    toast.success(bits.join(' · '))
  } else {
    toast.info('排版已经很整齐，无需整理')
  }
}

async function loadDocs() {
  loadingDocs.value = true
  const docsCacheKey = `xzj-docs-${projectId}`
  try {
    const res = await documentApi.list(String(projectId))
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

// 「初始化默认结构」幂等化（Boss 2026-10-30 报 bug：弱网下多次执行重复建文档）：
// ①单飞守卫——进行中再点直接忽略；②动手前拉最新树，已存在的（设定类按 type、卷按 type+标题）跳过；
// ③后端对设定类容器也有每书一份的幂等兜底，双保险
const inittingDefault = ref(false)
async function initDefaultStructure() {
  if (inittingDefault.value) return
  inittingDefault.value = true
  try {
    let existing: any[] = []
    try {
      const res = await documentApi.list(String(projectId))
      existing = res.data?.data || res.data || []
    } catch { /* 拉不到就当空处理，靠后端幂等兜底 */ }
    const defaults = [
      { type: 'outline', title: '大纲', sort_order: 1 },
      { type: 'characters', title: '人物设定', sort_order: 2 },
      { type: 'worldbuilding', title: '世界观', sort_order: 3 },
      { type: 'chapter_outline', title: '章节大纲', sort_order: 5 },
      { type: 'volume', title: '第一卷', sort_order: 6 },
    ]
    for (const d of defaults) {
      const dup = existing.some((x: any) => x.type === d.type
        && (d.type === 'volume' ? String(x.title) === d.title : true))
      if (dup) continue
      try {
        const res = await documentApi.create({
          project_id: projectId,
          title: d.title,
          type: d.type,
          sort_order: d.sort_order,
        })
        const created = res.data?.data
        if (created) existing.push(created)
      } catch (e) {
        console.error('创建默认结构失败', d, e)
      }
    }
  } finally {
    inittingDefault.value = false
    await loadDocs()
  }
}

async function openDoc(doc: any, opts?: { keepSearch?: boolean }) {
  // 码字锁：目标没写完，拦切章
  if (!guardWordLock()) return
  compareActive.value = false
  // ponytail-mobile: 移动端选中章节后收起抽屉
  mobileSidebarOpen.value = false
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
      // 分离方案：章节正文不存标题头（顶部虚拟头动态显示「第N章 标题」），存量标题头加载时删掉
      if (doc.type === 'chapter') content = stripContentChapterHeading(content, doc.title)
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
      wordCount.value = countWords(getEditorText())
      openBaselineWords.value = wordCount.value // 撤销地板：本章打开时的字数，撤到此为止
      rescanSensitive()
      syncPhonePreview()
      saveStatus.value = fromCache ? '📡 离线读取' : '已保存'
      setTimeout(() => { isSettingContent.value = false }, 100)

      // 续写就位（Boss：码字时视线应在屏幕中段，不在底部）——
      // 章节打开光标直达末尾续写点，并把光标行滚到视线线；不用等敲第一个键才跳
      if (doc.type === 'chapter') {
        nextTick(() => {
          try {
            editor.value?.commands.focus('end')
            typewriterScroll()
          } catch { /* 空内容无光标时跳过 */ }
        })
      }

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
  wordCount.value = countWords(getEditorText())
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

function openCreateDocModal(parentId: number | null = null, type: string = 'chapter', defaultName = '') {
  createDocParent.value = parentId
  createDocType.value = type
  newDocName.value = defaultName
  showCreateDocModal.value = true
}

// 拆分组件 CreateDocModal 触发（emit create）
async function onCreateDocConfirm(name: string, folderType?: string) {
  newDocName.value = name
  await createDoc(folderType)
}

async function createDoc(folderType?: string) {
  // 章节允许留空=unnamed 自动编号「第N章」；卷/设定/文档仍必须有名
  if (!newDocName.value.trim() && createDocType.value !== 'chapter') return
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
      title: newDocName.value.trim(),
      type: actualType,
      parent_id: createDocParent.value || undefined,
      sort_order: maxSort + 1,
    })
    showCreateDocModal.value = false
    newDocName.value = ''
    await loadDocs()
    const newDoc = res.data?.data || res.data
    if (newDoc && (actualType === 'chapter' || actualType === 'character')) {
      openDoc(newDoc) // 角色卡建完直接进卡开始写小传
    }
  } catch (e: any) {
    toast.error(e.response?.data?.error || '创建失败')
  }
}

function openDocContextMenu(e: MouseEvent, doc: any, zone?: 'chapters' | 'related') {
  docContextMenu.value = {
    visible: true,
    x: e.clientX,
    y: e.clientY,
    doc,
    // 节点右键的区由类型推导（volume/chapter=正文），空白右键由侧栏传当前 tab
    zone: doc ? (isChapterType(doc.type) ? 'chapters' : 'related') : (zone || 'chapters')
  }
  document.addEventListener('click', closeDocContextMenu, { once: true })
}

function closeDocContextMenu() {
  docContextMenu.value.visible = false
}

// 节点右键的「新建」项：卷→卷内章节；章节→同级章节；人物/角色卡→角色；其余设定→同级设定
const ctxNewItem = computed(() => {
  const doc = docContextMenu.value.doc
  if (!doc) return null
  if (doc.type === 'volume') return { label: '在此卷新建章节', parentId: doc.id as number, type: 'chapter' }
  if (doc.type === 'chapter') return { label: '新建同级章节', parentId: (doc.parent_id || null) as number | null, type: 'chapter' }
  if (doc.type === 'characters') return { label: '新建角色', parentId: doc.id as number, type: 'character' }
  if (doc.type === 'chapter_outline') return { label: '新建章纲', parentId: doc.id as number, type: 'note' }
  if (doc.type === 'character') return { label: '新建角色', parentId: (doc.parent_id || null) as number | null, type: 'character' }
  return { label: '新建设定', parentId: null as number | null, type: 'setting' }
})

function doCtxNew() {
  const item = ctxNewItem.value
  if (!item) return
  closeDocContextMenu()
  openCreateDocModal(item.parentId, item.type)
}

// 作品相关空白右键=新建设定
function newSettingFromMenu() {
  closeDocContextMenu()
  openCreateDocModal(null, 'setting')
}

// ponytail: 底部按钮固定根目录新建
function createDocAtRoot() {
  openCreateDocModal(null, 'volume')
}

// ponytail: 文章区＋号=新建卷（层级：卷→章节；章节走每卷的「+ 在卷下新建章节」）
function createVolumeAtRoot() {
  // 预填「第N卷」（N=现有卷数+1，与竞品一致的自动命名，弹窗里可改）
  const volCount = documents.value.filter(d => d.type === 'volume').length + 1
  openCreateDocModal(null, 'volume', `第${numToChinese(volCount)}卷`)
}

/* 特殊章节：楔子插根级最前 / 番外放根级最后，标题即类型名，不占章节号（编号规则见 chapterTitle.ts） */
async function createSpecialChapter(kind: '楔子' | '番外') {
  const rootSiblings = documents.value.filter((d: any) => !d.parent_id)
  const sort = kind === '楔子'
    ? rootSiblings.reduce((m: number, d: any) => Math.min(m, d.sort_order || 0), 0) - 1
    : rootSiblings.reduce((m: number, d: any) => Math.max(m, d.sort_order || 0), 0) + 1
  // 番外常有多个：第二个起自动接续编号（番外二/番外三…），作者可再改名；楔子保持原名
  let title: string = kind
  if (kind === '番外') {
    const extraCount = documents.value.filter((d: any) => d.type === 'chapter' && /^番外/.test(d.title || '')).length
    if (extraCount > 0) title = `番外${numToChinese(extraCount + 1)}`
  }
  try {
    const res = await documentApi.create({
      project_id: projectId,
      title,
      type: 'chapter',
      sort_order: sort,
    })
    await loadDocs()
    const nd = res.data?.data || res.data
    if (nd) openDoc(nd)
    toast.success(kind === '楔子' ? '楔子已插入正文最前（不占章节号）' : '番外已放到最后（不占章节号）')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '创建失败')
  }
}

// ponytail: 设定区＋号——根目录建设定（弹窗里选类型：大纲/人物/世界观等）
function createSettingAtRoot() {
  openCreateDocModal(null, 'setting')
}

// ponytail: 文件夹内+按钮，根据文件夹类型决定子文档类型
function createDocInFolder(folder: any) {
  // 人物节点：+ = 新建角色卡（起名即建，字段全在卡上就地编辑）
  if (folder.type === 'characters') {
    openCreateDocModal(folder.id, 'character')
    return
  }
  // 章节大纲节点：+ = 新建章纲文件（note 挂容器下，标题自由：卷一细纲/支线纲…）
  if (folder.type === 'chapter_outline') {
    openCreateDocModal(folder.id, 'note')
    return
  }
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

// ponytail: 角色卡——meta/名字就地编辑落库（documents.meta + title）
async function onCharMetaUpdate(m: Record<string, any>) {
  const d = activeDoc.value
  if (!d) return
  d.meta = JSON.stringify(m) // 本地先更新（树徽标/头像即时刷新）
  try {
    await documentApi.update(d.id, { meta: d.meta })
  } catch { toast.error('角色卡保存失败') }
}
async function onCharTitleUpdate(title: string) {
  const d = activeDoc.value
  if (!d) return
  d.title = title
  currentDocName.value = title
  try {
    await documentApi.update(d.id, { title })
  } catch { toast.error('名字保存失败') }
}

// ponytail: 判断是否文件夹容器——folder=设定文件夹, volume=文章文件夹, 其他设定类型也可展开
// ponytail: 设定区平铺后卷+characters（人物）是容器——角色卡挂人物下（2026-10-27）
// chapter_outline（章节大纲）2026-11-09 起也是容器——章纲文件（note）挂它下面，长篇按卷/条线拆多份
function isFolderNode(doc: any): boolean {
  return doc.type === 'volume' || doc.type === 'characters' || doc.type === 'chapter_outline'
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
  const trimmed = title.trim()
  // 章节允许清空名字=回到自动编号「第N章」（名是作者的，包括「不起名」）；卷/设定仍必须有名
  const doc = documents.value.find((d: any) => d.id === docId)
  if (!trimmed && doc?.type !== 'chapter') return
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
    desc: `确定删除「${docDisplayTitle(doc)}」？删除后可在回收站恢复。`,
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

// 拖到树空白区=移出到根级（出卷/出人物容器的唯一拖拽出口）
async function handleDropRoot(srcId: number) {
  const src = documents.value.find(d => d.id === srcId)
  if (!src || !src.parent_id) return
  const roots = documents.value.filter(d => !d.parent_id)
  const sort = roots.reduce((m, d) => Math.max(m, d.sort_order || 0), 0) + 1
  try {
    await documentApi.move(srcId, { parent_id: null, sort_order: sort })
    await loadDocs()
  } catch (e: any) {
    toast.error(e.response?.data?.error || '移动失败')
  }
}

function openContextMenu(e: MouseEvent) {
  // ponytail: 右键前缓存当前选区，右键动作后编辑器已失焦
  cachedSelection = getSelection()
  ctxHasSelection.value = !!(cachedSelection && cachedSelection.text && cachedSelection.text.trim())
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
  // 手机长按初选只有 1~2 字，用户拖手柄扩选后缓存就过期了——优先读实时选区（posAtDOM 永远新鲜），
  // 实时选区没了（点菜单已把选区点丢）才退回缓存（Boss 2026-11-05：只染最后2字的根因）
  const sel = (() => { const live = getSelection(); return live.text ? live : (cachedSelection || live) })()
  cachedSelection = null
  if (!sel.text && action !== 'paste') return
  switch (action) {
    case 'thread': {
      // 记伏笔：预填选中文本，弹确认框
      contextMenu.value.visible = false
      const quote = (sel.text || '').trim().slice(0, 80)
      threadModal.value = { visible: true, title: quote.slice(0, 16), type: 'foreshadowing', quote }
      break
    }
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
}

function goBack() {
  if (!guardWordLock()) return
  if (isMiniProgramEnv) {
    // 纯 web-view 壳（2026-11-06）：编辑器和工作台同一层，navigateBack 关无可关——
    // 有 SPA 内部历史（从工作台/书架点进来）就网页后退；深链直达编辑器就回壳首页
    if (window.history.state && window.history.state.back != null) router.back()
    else mpRelaunch('/pages/index/index')
    return
  }
  router.push('/bookshelf')
}
</script>

<style scoped src="./editor-page.css"></style>
