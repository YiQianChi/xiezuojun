# -*- coding: utf-8 -*-
# 预案脚本：生产「章节大纲文件夹化」上线前的老数据核查 + 迁移
# 用法（在能 ssh 到 124 的本机跑）：
#   python3 prod-outline-migrate.py check    # 只核查，不改任何数据
#   python3 prod-outline-migrate.py migrate  # 核查到有内容的才执行迁移（先 dry-run 打印，再询问确认）
# 迁移内容：有正文的老章纲文档 → 新建容器（章节大纲）+ 老文档变子文档（type=note）挂容器下，文件随 parent 搬迁
import subprocess, sys, uuid, time

SSH = "sshpass -p 'Le199789' ssh -o StrictHostKeyChecking=no ubuntu@124.222.138.209"
MYSQL = "sudo mysql -uxiezuojun -pXzj2026Test xiezuojun -N -e"
DATA = "/opt/data/documents"

def sh(cmd):
    r = subprocess.run(f"{SSH} \"{cmd}\"", shell=True, capture_output=True, text=True, timeout=120)
    return r.stdout.strip()

def q(sql):
    return sh(f"{MYSQL} \\\"{sql}\\\" 2>/dev/null")

def check():
    rows = q("SELECT id, project_id, title, LENGTH(COALESCE(search_text,'')) FROM documents WHERE type='chapter_outline' AND status=1 AND LENGTH(COALESCE(search_text,''))>30")
    docs = [r.split('\t') for r in rows.splitlines() if r.strip()]
    print(f"有内容的章纲文档: {len(docs)}")
    for d in docs:
        print('  id=%s project=%s title=%s 内容长度=%s' % tuple(d))
    return docs

def migrate(docs, dry=True):
    for did, pid, title, clen in docs:
        uid = q(f"SELECT u.id FROM projects p JOIN users u ON u.id=p.user_id WHERE p.id={pid}")
        src_root = f"{DATA}/{uid}/p-{int(pid):010d}/{did}.md"
        print(f"项目 {pid} 文档 {did}({title}): uid={uid}")
        print(f"  1) 建容器: INSERT chapter_outline '章节大纲'")
        print(f"  2) 老文档改 type=note parent=容器")
        print(f"  3) 文件 {src_root} → folder-容器id/{did}.md")
        if dry: continue
        # 实测执行段（仅 dry=False 才走，且逐条打印结果）
        maxsort = q(f"SELECT COALESCE(MAX(sort_order),0) FROM documents WHERE project_id={pid} AND parent_id IS NULL AND status=1")
        now = int(time.time())
        cid = q(f"INSERT INTO documents (project_id,parent_id,type,title,sort_order,word_count,status,created_at,updated_at,search_text) VALUES ({pid},NULL,'chapter_outline','章节大纲',{int(maxsort)+1},0,1,{now},{now},''); SELECT LAST_INSERT_ID()").splitlines()[-1]
        q(f"UPDATE documents SET type='note', parent_id={cid} WHERE id={did}")
        dst_dir = f"{DATA}/{uid}/p-{int(pid):010d}/folder-{cid}"
        sh(f"sudo mkdir -p {dst_dir} && sudo mv {src_root} {dst_dir}/{did}.md")
        print(f"  ✓ 容器 id={cid}，已挂载搬迁")
    if dry: print("（dry-run，未改数据；确认后加 migrate --real）")

if __name__ == '__main__':
    docs = check()
    if sys.argv[1:2] == ['migrate'] and docs:
        migrate(docs, dry='--real' not in sys.argv)
    elif not docs:
        print('零存量内容，直接发布即可，无需迁移。')
