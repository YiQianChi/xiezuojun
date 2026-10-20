<template>
  <div class="chat-panel">
    <!-- 消息列表 -->
    <div ref="msgListRef" class="chat-messages">
      <div v-if="messages.length === 0" class="chat-empty">
        <div class="chat-empty-icon">💬</div>
        <p>向 AI 提问关于你的小说</p>
        <p class="hint">角色设定、情节推演、灵感碰撞…</p>
      </div>
      <div
        v-for="(msg, i) in messages"
        :key="i"
        class="chat-msg"
        :class="msg.role"
      >
        <div class="msg-bubble">{{ msg.content }}</div>
      </div>
      <!-- loading 指示 -->
      <div v-if="loading" class="chat-msg assistant">
        <div class="msg-bubble typing">
          <span></span><span></span><span></span>
        </div>
      </div>
    </div>

    <!-- 输入区 -->
    <div class="chat-input-area">
      <textarea
        ref="inputRef"
        v-model="input"
        class="chat-input"
        :placeholder="'输入消息…（Enter 发送，Shift+Enter 换行）'"
        rows="1"
        @keydown="onKeydown"
        @input="autoGrow"
      ></textarea>
      <button
        class="chat-send-btn"
        :disabled="!input.trim() || loading"
        @click="send"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, nextTick, onBeforeUnmount } from 'vue'
import { streamChat } from '../../core/sse'
import { toast } from '../../composables/useToast'

interface ChatMsg {
  role: 'user' | 'assistant'
  content: string
}

const props = defineProps<{ projectId: number }>()

const messages = ref<ChatMsg[]>([])
const input = ref('')
const loading = ref(false)
const msgListRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLTextAreaElement | null>(null)
let abortCtrl: AbortController | null = null

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}

function autoGrow() {
  const el = inputRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 120) + 'px'
}

async function scrollToBottom() {
  await nextTick()
  const el = msgListRef.value
  if (el) el.scrollTop = el.scrollHeight
}

async function send() {
  const text = input.value.trim()
  if (!text || loading.value) return

  messages.value.push({ role: 'user', content: text })
  input.value = ''
  loading.value = true
  if (inputRef.value) inputRef.value.style.height = 'auto'
  await scrollToBottom()

  // 准备发往后端的消息（包含历史）
  const apiMessages = messages.value.map(m => ({ role: m.role, content: m.content }))

  // 占位 assistant 消息，流式追加
  const aiMsg = reactive<ChatMsg>({ role: 'assistant', content: '' })
  messages.value.push(aiMsg)
  await scrollToBottom()

  abortCtrl = new AbortController()
  try {
    await streamChat(
      { project_id: String(props.projectId), messages: apiMessages },
      {
        onContent: (chunk: string) => {
          aiMsg.content += chunk
          scrollToBottom()
        },
        onDone: () => {
          loading.value = false
          abortCtrl = null
        },
      },
      abortCtrl.signal,
    )
  } catch (e: any) {
    loading.value = false
    abortCtrl = null
    // 删掉空的占位消息
    const last = messages.value[messages.value.length - 1]
    if (last && last.role === 'assistant' && last.content === '') {
      messages.value.pop()
    }
    if (e.name !== 'AbortError') {
      toast.error(e.message || 'AI 对话失败')
    }
  }
}

onBeforeUnmount(() => {
  abortCtrl?.abort()
})
</script>

<style scoped>
.chat-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  font-family: 'SF Pro Display', 'PingFang SC', -apple-system, sans-serif;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.chat-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary, #aeaeb2);
  text-align: center;
  font-size: 13px;
}
.chat-empty-icon { font-size: 32px; margin-bottom: 8px; opacity: 0.6; }
.chat-empty .hint { font-size: 11px; opacity: 0.8; margin-top: 2px; }

.chat-msg {
  display: flex;
  max-width: 100%;
}
.chat-msg.user {
  justify-content: flex-end;
}
.chat-msg.assistant {
  justify-content: flex-start;
}

.msg-bubble {
  max-width: 85%;
  padding: 8px 12px;
  border-radius: 14px;
  font-size: 13px;
  line-height: 1.6;
  word-break: break-word;
  white-space: pre-wrap;
}
.chat-msg.user .msg-bubble {
  background: #007AFF;
  color: #fff;
  border-bottom-right-radius: 4px;
}
.chat-msg.assistant .msg-bubble {
  background: #f5f5f7;
  color: #1d1d1f;
  border-bottom-left-radius: 4px;
}

/* typing indicator */
.msg-bubble.typing {
  display: flex;
  gap: 4px;
  align-items: center;
  padding: 10px 14px;
}
.msg-bubble.typing span {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #aeaeb2;
  animation: chat-typing 1.4s infinite ease-in-out both;
}
.msg-bubble.typing span:nth-child(1) { animation-delay: -0.32s; }
.msg-bubble.typing span:nth-child(2) { animation-delay: -0.16s; }
@keyframes chat-typing {
  0%, 80%, 100% { transform: scale(0.6); opacity: 0.5; }
  40% { transform: scale(1); opacity: 1; }
}

.chat-input-area {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  padding: 10px 12px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  background: var(--bg-secondary, #fff);
}

.chat-input {
  flex: 1;
  resize: none;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 10px;
  padding: 8px 10px;
  font-size: 13px;
  line-height: 1.5;
  max-height: 120px;
  outline: none;
  font-family: inherit;
  background: var(--bg-primary, #fff);
  color: var(--text-primary, #1d1d1f);
  transition: border-color 0.15s;
}
.chat-input:focus {
  border-color: #007AFF;
}
.chat-input::placeholder {
  color: var(--text-tertiary, #c7c7cc);
}

.chat-send-btn {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: none;
  background: #007AFF;
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: opacity 0.15s;
}
.chat-send-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.chat-send-btn:not(:disabled):hover {
  opacity: 0.85;
}
</style>
