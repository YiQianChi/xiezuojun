#!/usr/bin/env python3
"""mock-web.py — 本地手机端验收：serve web/dist + SPA fallback + /api 返回假 JSON。

用法: python3 scripts/mock-web.py [port]   (默认 8877)
验收页需登录态: 浏览器 console 执行 localStorage.setItem('access_token','mock')
"""
import json
import re
import sys
from http.server import HTTPServer, SimpleHTTPRequestHandler
from pathlib import Path

DIST = Path(__file__).resolve().parent.parent / 'web' / 'dist'

DOCS = [
    {"id": 1, "project_id": 1, "title": "第一卷 觉醒", "type": "volume", "parent_id": None, "status": 1, "word_count": 0},
    {"id": 2, "project_id": 1, "title": "第一章 平凡的早晨", "type": "chapter", "parent_id": 1, "status": 1, "word_count": 3200},
    {"id": 3, "project_id": 1, "title": "第二章 异变", "type": "chapter", "parent_id": 1, "status": 1, "word_count": 2800},
    {"id": 4, "project_id": 1, "title": "人物设定", "type": "characters", "parent_id": None, "status": 1, "word_count": 500},
    {"id": 5, "project_id": 1, "title": "世界观", "type": "worldbuilding", "parent_id": None, "status": 1, "word_count": 800},
]
CONTENT = '<p>' + '黎明前的黑暗最是深重，但少年知道，再过片刻，第一缕光就会翻过东边的山脊。' * 8 + '</p>' * 3

PROJECTS = [
    {"id": 1, "public_id": "mockpub0001", "user_id": "32", "name": "星海拾遗", "description": "一个关于星空与故乡的故事", "cover_url": "", "genre": "科幻", "tags": "", "status": 1, "created_at": "2026-10-01T10:00:00+08:00", "updated_at": "2026-11-04T09:00:00+08:00", "word_count": 128000},
    {"id": 2, "public_id": "mockpub0002", "user_id": "32", "name": "长安十二雨", "description": "", "cover_url": "", "genre": "古言", "tags": "", "status": 1, "created_at": "2026-09-12T10:00:00+08:00", "updated_at": "2026-11-03T21:00:00+08:00", "word_count": 45200},
    {"id": 3, "public_id": "mockpub0003", "user_id": "32", "name": "深渊回响（草稿）", "description": "悬疑长篇", "cover_url": "", "genre": "悬疑", "tags": "", "status": 1, "created_at": "2026-10-20T10:00:00+08:00", "updated_at": "2026-10-28T08:00:00+08:00", "word_count": 9800},
    {"id": 4, "public_id": "mockpub0004", "user_id": "32", "name": "短篇集·城市灯火", "description": "", "cover_url": "", "genre": "都市", "tags": "", "status": 1, "created_at": "2026-11-01T10:00:00+08:00", "updated_at": "2026-11-02T08:00:00+08:00", "word_count": 5600},
]

USER = {"id": "32", "email": "test@oneon.com.cn", "nickname": "测试菌", "avatar": "", "plan": "free", "created_at": "2026-08-01T10:00:00+08:00"}


def mock_api(path):
    """按路径模式返回 (status, json_data)。"""
    def d(x):
        return 200, {"data": x}
    if path == '/user/me':
        return d(USER)
    if path == '/user/usage':
        return d({"words_used": 188600, "words_quota": 500000, "ai_calls_today": 12, "ai_quota_daily": 50})
    if path == '/projects':
        return d(PROJECTS)
    if m := re.fullmatch(r'/projects/(\d+|mockpub\d+)', path):
        pid = 1 if not m.group(1).isdigit() else int(m.group(1))
        return d(PROJECTS[min(pid, len(PROJECTS)) - 1])
    if re.fullmatch(r'/projects/\d+/stats', path):
        return d({"chapters": 2, "words": 6000, "today_words": 1200, "streak_days": 5})
    if path == '/documents':
        return d(DOCS)
    if m := re.fullmatch(r'/documents/(\d+)', path):
        doc = next((x for x in DOCS if x['id'] == int(m.group(1))), DOCS[1])
        return d({**doc, "content": CONTENT})
    if m := re.fullmatch(r'/public/read/(\w+)', path):
        return d({"project_name": "星海拾遗", "title": "第一章 平凡的早晨", "word_count": 3200, "updated_at": "2026-11-04", "content": CONTENT})
    if path == '/memory/characters':
        return d([{"name": "林昭", "role": "主角", "summary": "十七岁，观星者"}, {"name": "沈砚", "role": "配角", "summary": "藏书楼管理员"}])
    if path == '/memory/summaries':
        return d([])
    if path == '/memory/file':
        return d({"content": ""})
    if path == '/memory/relationships':
        return d({})
    if path == '/memory/canvas':
        return d({})
    if path == '/sticky':
        return d([])
    if path.endswith('/threads'):
        return d([])
    if path == '/buddy/state':
        return d({"active": False})
    if path == '/buddy/today':
        return d({"words": 1200, "minutes": 45})
    if path == '/buddy/goal':
        return d({"daily_words": 2000})
    if path == '/buddy/session-stats':
        return d({"words": 0, "start_at": None})
    if path == '/sprint/rooms':
        return d([])
    if path == '/admin/stats/overview':
        return d({"user_count": 128, "project_count": 356, "doc_count": 8421, "word_count": 9860000, "new_users_7d": 23})
    if path == '/admin/stats/daily':
        return d([{"date": f"2026-11-{i:02d}", "active_users": 20 + i, "new_users": i % 4, "ai_calls": 100 + i * 7, "words": 30000 + i * 900} for i in range(1, 5)])
    if path == '/admin/stats/ai-usage':
        return d([{"model": "MiniMax-M3", "calls": 812, "tokens": 1230000}])
    if path == '/admin/users':
        return d([{**USER, "id": i, "email": f"user{i}@t.cn", "nickname": f"用户{i}", "plan": "free" if i % 2 else "pro", "status": 1, "created_at": "2026-10-01T10:00:00+08:00"} for i in range(1, 9)])
    if path == '/admin/projects':
        return d([{**p, "user_email": "user1@t.cn", "nickname": "用户1"} for p in PROJECTS])
    if path.endswith('/documents') and path.startswith('/admin/projects/'):
        return d(DOCS)
    if path == '/admin/announcements':
        return d([{"id": 1, "title": "公告：新版上线", "content": "修复若干问题", "created_at": "2026-11-01T10:00:00+08:00"}])
    if path == '/admin/feedbacks':
        return d([{"id": 1, "user_email": "user2@t.cn", "content": "希望增加导出功能", "status": "open", "created_at": "2026-11-02T10:00:00+08:00"}])
    # 默认兜底：GET → 空列表，其余 → 空对象
    return 200, {"data": []}


class Handler(SimpleHTTPRequestHandler):
    def __init__(self, *a, **kw):
        super().__init__(*a, directory=str(DIST), **kw)

    def _send_json(self, status, obj):
        body = json.dumps(obj).encode()
        self.send_response(status)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        path = self.path.split('?')[0]
        if path.startswith('/api/'):
            api = path[4:]
            status, obj = mock_api(api)
            return self._send_json(status, obj)
        if path != '/' and not (DIST / path.lstrip('/')).exists():
            self.path = '/index.html'  # SPA fallback
        return super().do_GET()

    def do_POST(self):
        return self._send_json(200, {"data": {}})

    def do_PATCH(self):
        return self._send_json(200, {"data": {}})

    def do_PUT(self):
        return self._send_json(200, {"data": {}})

    def do_DELETE(self):
        return self._send_json(200, {"message": "ok"})


if __name__ == '__main__':
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8877
    print(f'mock-web serving {DIST} on http://127.0.0.1:{port} (token=mock)')
    HTTPServer(('127.0.0.1', port), Handler).serve_forever()
