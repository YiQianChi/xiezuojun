<template>
  <Teleport to="body">
    <!-- 拼字房结算榜（入口/建房/加入/大厅已深度融合进搭子悬浮窗「拼字」页签） -->
    <div v-if="sprint.showResult && sprint.room && sprint.room.state === 'finished'" class="sp-overlay solid" @click.self="sprint.showResult = false">
      <div class="sp-card">
        <div class="sp-cup">{{ iWon ? '🏆' : '🏁' }}</div>
        <div class="sp-title">{{ resultTitle }}</div>
        <div class="sp-sub">{{ sprint.room.target_label }} · 我第 {{ sprint.myRank }} 名 / {{ sprint.room.players.length }} 人</div>
        <div class="sp-members">
          <div v-for="p in sprint.room.players" :key="p.user_id" class="sp-member" :class="{ me: p.user_id === sprint.room.me }">
            <span class="sp-m-rk">{{ medal(p.rank) }}</span>
            <span class="sp-m-av">{{ sprint.avatar(p.nickname) }}</span>
            <span class="sp-m-name">{{ p.nickname }}{{ p.user_id === sprint.room.me ? '（我）' : '' }}</span>
            <span class="sp-m-wc">{{ p.words }} 字</span>
          </div>
        </div>
        <div class="sp-row3 foot">
          <button class="sp-btn green" @click="onAgain">再来一局</button>
          <button class="sp-btn line" @click="sprint.showResult = false">先去写</button>
          <button class="sp-btn ghost" @click="sprint.leave()">退出房间</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
// 「再来一局」回搭子窗拼字页签的建房表单：openPanel 触发开窗事件（表单记忆上一局配置）
import { computed } from 'vue'
import { useSprint } from '../../composables/useSprint'

const sprint = useSprint()

const iWon = computed(() => sprint.myRank === 1)
const resultTitle = computed(() => (iWon.value ? '你赢了这局！' : `🏅 ${sprint.room?.players[0]?.nickname || '菌友'} 拿下本局`))

function medal(r: number) {
  return r === 1 ? '🥇' : r === 2 ? '🥈' : r === 3 ? '🥉' : String(r)
}

function onAgain() {
  sprint.again()      // phase → create，保留上一局配置
  sprint.openPanel()  // 打开搭子窗并切「拼字」页签
}
</script>

<style scoped>
.sp-overlay {
  position: fixed; inset: 0; z-index: 950;
  background: rgba(30, 26, 16, 0.45);
  display: flex; align-items: center; justify-content: center;
  padding: 20px;
}
.sp-card {
  width: 400px; max-width: 94vw; max-height: 86vh; overflow-y: auto;
  background: var(--ed-paper, #fffdf8); border: 1px solid var(--ed-paper-line, #e6dcc6); border-radius: 20px;
  box-shadow: 0 24px 70px var(--ed-shadow, rgba(40, 30, 10, 0.35));
  padding: 24px 26px 20px; text-align: center;
}
.sp-cup { font-size: 34px; }
.sp-title { font-size: 18px; font-weight: 800; margin-top: 4px; color: var(--ed-ink, #3a352c); }
.sp-sub { font-size: 12px; color: #7a7263; margin-top: 4px; }
.sp-row3 { display: flex; gap: 8px; }
.sp-row3.foot { margin-top: 18px; }
.sp-btn {
  flex: 1; border: none; border-radius: 12px; padding: 11px 0;
  font-size: 13px; font-weight: 700; cursor: pointer;
}
.sp-btn.green { background: linear-gradient(135deg, #8fc59b, #6aa877); color: #fff; }
.sp-btn.ghost { background: rgba(129, 187, 142, 0.12); color: #5a9668; }
.sp-btn.line { background: none; border: 1px solid var(--ed-line, #e2d8c2); color: var(--ed-ink-mid, #7a7263); }
.sp-members { margin-top: 14px; text-align: left; }
.sp-member {
  display: flex; align-items: center; gap: 10px; padding: 8px 10px;
  border-radius: 11px; margin-bottom: 4px; font-size: 13px;
}
.sp-member.me { background: rgba(129, 187, 142, 0.12); }
.sp-m-av {
  width: 30px; height: 30px; border-radius: 9px; flex-shrink: 0;
  background: linear-gradient(135deg, #8fc59b, #6aa877); color: #fff;
  display: flex; align-items: center; justify-content: center;
  font-size: 13px; font-weight: 800;
}
.sp-m-name { font-weight: 700; color: var(--ed-ink, #3a352c); flex: 1; }
.sp-m-rk { width: 24px; text-align: center; font-size: 16px; }
.sp-m-wc { font-size: 15px; font-weight: 800; color: var(--ed-ink, #3a352c); font-variant-numeric: tabular-nums; }

/* 手机端：结算卡收紧（dvh 防地址栏收缩溢出） */
@media (max-width: 768px) {
  .sp-overlay { padding: 12px; }
  .sp-card { width: 100%; padding: 18px 16px 16px; max-height: 82dvh; }
  .sp-members { max-height: 46dvh; overflow-y: auto; }
}

</style>
