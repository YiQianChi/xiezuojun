/**
 * SSE streaming client for AI generate endpoint.
 * Uses fetch + ReadableStream (axios can't stream response bodies).
 *
 * Backend SSE events (gateway/api_handler.go):
 *   event: settings → { used_settings: [...] }
 *   event: content  → { content: "chunk" }
 *   event: done     → ""
 *   event: error    → { error: "msg" }
 */

export interface SSECallbacks {
  onContent?: (chunk: string) => void
  onSettings?: (settings: any[]) => void
  onDone?: () => void
}

export async function streamGenerate(
  params: {
    project_id: string
    document_id?: string
    mode: string
    content: string
  },
  cb: SSECallbacks,
  signal?: AbortSignal,
): Promise<void> {
  const token = localStorage.getItem('access_token')
  const resp = await fetch('/api/ai/generate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: 'Bearer ' + token } : {}),
    },
    body: JSON.stringify({ ...params, stream: true }),
    signal,
  })

  if (resp.status === 401) {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    if (!window.location.pathname.includes('/login')) {
      window.location.href = '/login'
    }
    throw new Error('登录已过期，请重新登录')
  }

  if (!resp.ok) {
    let msg = `HTTP ${resp.status}`
    try {
      const err = await resp.json()
      msg = err.error || msg
    } catch { /* ignore */ }
    throw new Error(msg)
  }

  const reader = resp.body?.getReader()
  if (!reader) throw new Error('浏览器不支持流式响应')

  const decoder = new TextDecoder()
  let buffer = ''

  for (;;) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })

    let sep: number
    while ((sep = buffer.indexOf('\n\n')) >= 0) {
      const rawEvent = buffer.slice(0, sep)
      buffer = buffer.slice(sep + 2)

      let event = 'message'
      let data = ''
      for (const line of rawEvent.split('\n')) {
        if (line.startsWith('event:')) event = line.slice(6).trim()
        else if (line.startsWith('data:')) data += line.slice(5).replace(/^ /, '')
      }
      if (!data) continue

      let parsed: any
      try {
        parsed = JSON.parse(data)
      } catch {
        continue
      }

      if (event === 'content' && parsed.content) {
        cb.onContent?.(parsed.content)
      } else if (event === 'settings' && parsed.used_settings) {
        cb.onSettings?.(parsed.used_settings)
      } else if (event === 'done') {
        cb.onDone?.()
        return
      } else if (event === 'error') {
        throw new Error(parsed.error || 'AI 生成失败')
      }
    }
  }

  // Stream ended without explicit done event
  cb.onDone?.()
}
