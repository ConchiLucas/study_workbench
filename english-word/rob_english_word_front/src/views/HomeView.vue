<template>
  <div class="home-container" v-if="!showTrainingSetup">
    <div class="header">
      <div class="header-left">
        <!-- 左侧留空或放logo -->
      </div>
      <div class="header-right">
        <span class="nickname">{{ authStore.nickname }}</span>
        <button class="logout-btn" @click="handleLogout">退出</button>
      </div>
    </div>

    <div class="stats-card" @click="goMatchRecords">
      <div class="stat-item" style="display: flex; justify-content: center; align-items: center;">
        <RankBadge :level="authStore.rank" />
      </div>
      <div class="stat-item">
        <span class="label">胜场</span>
        <span class="value">{{ authStore.totalWins }}</span>
      </div>
      <div class="stat-item">
        <span class="label">总场次</span>
        <span class="value">{{ authStore.totalGames }}</span>
      </div>
      <div class="stat-item">
        <span class="label">胜率</span>
        <span class="value">{{ authStore.winRate }}%</span>
      </div>
      <div class="stat-item">
        <span class="label">经验</span>
        <span class="value">{{ authStore.exp }}</span>
      </div>
    </div>

    <div class="stats-card training-stats-card" @click="goTrainingRecords">
      <div class="stat-item">
        <span class="label">训练等级</span>
        <span class="value">{{ authStore.trainingRank }}</span>
      </div>
      <div class="stat-item">
        <span class="label">训练经验</span>
        <span class="value">{{ authStore.trainingExp }}</span>
      </div>
      <div class="stat-item">
        <span class="label">训练胜场</span>
        <span class="value">{{ authStore.trainingTotalWins }}</span>
      </div>
      <div class="stat-item">
        <span class="label">训练场次</span>
        <span class="value">{{ authStore.trainingTotalGames }}</span>
      </div>
      <div class="stat-item">
        <span class="label">训练胜率</span>
        <span class="value">{{ authStore.trainingWinRate }}%</span>
      </div>
    </div>

    <div class="match-section">
      <button class="match-btn" @click="handleMatch()" :disabled="isMatching || isMatchPending">
        开始匹配
      </button>
      <button class="solo-btn secondary-action-btn" @click="openTrainingSetup" :disabled="isMatching || isMatchPending || isSoloPending">
        难度训练
      </button>
      <button class="wrong-words-btn secondary-action-btn" @click="router.push('/wrong-words')">
        错题集
      </button>
      <p v-if="connectionNotice" class="connection-notice">{{ connectionNotice }}</p>
    </div>
  </div>

  <div class="training-setup-page" v-else>
    <FullscreenCloseButton :disabled="isSoloPending || isMatching || isMatchPending" @close="closeTrainingSetup" />

    <main class="training-setup-main">
      <section class="training-setup-panel">
        <div class="training-setup-title">
          <h1>难度训练</h1>
          <p>{{ selectedDifficulty.title }}</p>
        </div>

        <div class="training-setup-actions">
          <button class="match-btn setup-action-btn" @click="handleMatch(selectedDifficulty)" :disabled="isSoloPending || isMatching || isMatchPending">
            {{ isMatchPending ? '匹配准备中...' : '开始匹配' }}
          </button>
          <button class="difficulty-btn secondary-action-btn setup-action-btn" @click="showDifficultyPicker = true" :disabled="isSoloPending || isMatching || isMatchPending">
            难度选择
          </button>
          <button class="solo-btn secondary-action-btn setup-action-btn" @click="handleSoloTraining" :disabled="isSoloPending || isMatching || isMatchPending">
            {{ isSoloPending ? '训练准备中...' : '单人训练' }}
          </button>
          <button class="result-btn secondary-action-btn setup-action-btn" @click="router.push('/training-results')" :disabled="isSoloPending || isMatching || isMatchPending">
            查看答题结果
          </button>
          <button class="mastered-words-btn secondary-action-btn setup-action-btn" @click="router.push('/mastered-words')" :disabled="isSoloPending || isMatching || isMatchPending">
            已掌握单词
          </button>
        </div>

        <p v-if="connectionNotice" class="connection-notice">{{ connectionNotice }}</p>
      </section>
    </main>
  </div>

  <!-- 匹配中弹窗 -->
  <div class="modal-overlay" v-if="isMatching" @click.self="cancelMatch">
    <div class="modal-box">
      <div class="radar">
        <div class="radar-ring ring1"></div>
        <div class="radar-ring ring2"></div>
        <div class="radar-ring ring3"></div>
        <div class="radar-dot"></div>
      </div>
      <p class="modal-title">寻找对手中...</p>
      <p v-if="matchingDifficultyLabel" class="modal-difficulty">{{ matchingDifficultyLabel }}</p>
      <p class="modal-timer">{{ formatTime(matchSeconds) }}</p>
      <button class="cancel-btn" @click="cancelMatch">取消匹配</button>
    </div>
  </div>

  <div class="difficulty-overlay" v-if="showDifficultyPicker">
    <FullscreenCloseButton @close="closeDifficultyPicker" />
    <div class="difficulty-header">
      <button
        class="rank-difficulty-btn"
        :class="{ active: selectedDifficulty.key === rankDifficultyOption.key }"
        @click="selectRankDifficulty"
      >
        <span>{{ rankDifficultyOption.title }}</span>
        <small>{{ rankDifficultyOption.description }}</small>
      </button>
      <div>
        <h2>选择训练难度</h2>
        <p>{{ selectedDifficulty.title }}</p>
      </div>
    </div>

    <div class="difficulty-grid">
      <article
        v-for="option in difficultyOptions"
        :key="option.key"
        class="difficulty-card"
        :class="{
          active: selectedDifficulty.parentKey === option.key,
          expanded: expandedDifficultyKey === option.key
        }"
      >
        <button class="difficulty-card-button" @click="toggleDifficulty(option)">
          <span class="difficulty-chevron">{{ expandedDifficultyKey === option.key ? '⌄' : '›' }}</span>
          <span class="difficulty-title">{{ option.title }}</span>
        </button>

        <div class="difficulty-children" v-if="expandedDifficultyKey === option.key">
          <button
            v-for="child in option.children"
            :key="child.key"
            class="difficulty-child-card"
            :class="{ active: selectedDifficulty.key === child.key }"
            @click.stop="selectDifficulty(option, child)"
          >
            <span>{{ child.title }}</span>
            <small>均{{ child.avgDifficulty }}</small>
          </button>
        </div>
      </article>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useWebSocketStore } from '../stores/websocket'
import FullscreenCloseButton from '../components/FullscreenCloseButton.vue'
import RankBadge from '../components/RankBadge.vue'
import { useEscapeClose } from '../composables/useEscapeClose'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const wsStore = useWebSocketStore()
const isMatching = ref(false)
const isMatchPending = ref(false)
const isSoloPending = ref(false)
const showTrainingSetup = ref(false)
const showDifficultyPicker = ref(false)
const expandedDifficultyKey = ref('junior')
const matchSeconds = ref(0)
const matchingDifficultyLabel = ref('')
const connectionNotice = ref('')
let matchTimer: ReturnType<typeof setInterval> | null = null
let soloTrainingTimer: ReturnType<typeof setTimeout> | null = null
const TRAINING_DIFFICULTY_STORAGE_KEY = 'rob.trainingDifficulty'

const rankDifficultyOption = {
  key: 'rank_current',
  parentKey: 'rank',
  title: '段位难度',
  description: '按当前段位'
}

const difficultyOptions = [
  {
    key: 'primary',
    title: '小学英语',
    children: [
      { key: 'primary_3_1', title: '3年级上册', avgDifficulty: 71 },
      { key: 'primary_3_2', title: '3年级下册', avgDifficulty: 74 },
      { key: 'primary_4_1', title: '4年级上册', avgDifficulty: 77 },
      { key: 'primary_4_2', title: '4年级下册', avgDifficulty: 78 },
      { key: 'primary_5_1', title: '5年级上册', avgDifficulty: 74 },
      { key: 'primary_5_2', title: '5年级下册', avgDifficulty: 80 },
      { key: 'primary_6_1', title: '6年级上册', avgDifficulty: 83 },
      { key: 'primary_6_2', title: '6年级下册', avgDifficulty: 82 }
    ]
  },
  {
    key: 'junior',
    title: '初中英语',
    children: [
      { key: 'junior_7_1', title: '7年级上册', avgDifficulty: 122 },
      { key: 'junior_7_2', title: '7年级下册', avgDifficulty: 132 },
      { key: 'junior_8_1', title: '8年级上册', avgDifficulty: 139 },
      { key: 'junior_8_2', title: '8年级下册', avgDifficulty: 139 },
      { key: 'junior_9_1', title: '9年级上册', avgDifficulty: 147 }
    ]
  },
  {
    key: 'senior',
    title: '高中英语',
    children: [
      { key: 'senior_1', title: '上册', avgDifficulty: 246 },
      { key: 'senior_2', title: '下册', avgDifficulty: 259 },
      { key: 'senior_3', title: '第3册', avgDifficulty: 252 },
      { key: 'senior_4', title: '第4册', avgDifficulty: 254 },
      { key: 'senior_5', title: '第5册', avgDifficulty: 261 },
      { key: 'senior_6', title: '第6册', avgDifficulty: 272 },
      { key: 'senior_7', title: '第7册', avgDifficulty: 265 },
      { key: 'senior_8', title: '第8册', avgDifficulty: 273 },
      { key: 'senior_9', title: '第9册', avgDifficulty: 290 },
      { key: 'senior_10', title: '第10册', avgDifficulty: 292 },
      { key: 'senior_11', title: '第11册', avgDifficulty: 298 }
    ]
  },
  {
    key: 'college',
    title: '大学英语',
    children: [
      { key: 'college_cet4', title: '四级', avgDifficulty: 340 },
      { key: 'college_cet6', title: '六级', avgDifficulty: 450 }
    ]
  },
  {
    key: 'entrance',
    title: '升学考试英语',
    children: [
      { key: 'entrance_kaoyan', title: '考研', avgDifficulty: 442 }
    ]
  },
  {
    key: 'business_abroad',
    title: '商务与出国英语',
    children: [
      { key: 'business_bec', title: 'BEC', avgDifficulty: 573 },
      { key: 'business_ielts', title: '雅思', avgDifficulty: 573 },
      { key: 'business_toefl', title: '托福', avgDifficulty: 640 },
      { key: 'business_gmat', title: 'GMAT', avgDifficulty: 693 }
    ]
  },
  {
    key: 'professional',
    title: '专业英语',
    children: [
      { key: 'professional_tem4', title: '专四级', avgDifficulty: 501 },
      { key: 'professional_tem8', title: '专八级', avgDifficulty: 541 }
    ]
  },
  {
    key: 'advanced_exam',
    title: '高阶考试英语',
    children: [
      { key: 'advanced_gre', title: 'GRE', avgDifficulty: 732 },
      { key: 'advanced_sat', title: 'SAT', avgDifficulty: 747 }
    ]
  }
]
type DifficultyOption = typeof difficultyOptions[number]
type DifficultyChild = DifficultyOption['children'][number]
type SelectedDifficulty = {
  key: string
  parentKey: string
  title: string
}
const defaultDifficulty: SelectedDifficulty = {
  key: difficultyOptions[1].key,
  parentKey: difficultyOptions[1].key,
  title: difficultyOptions[1].title
}
const selectedDifficulty = ref<SelectedDifficulty>(loadSavedDifficulty())

/**
 * 处理后端 WebSocket 消息
 */
function handleMessage(message: any) {
  console.log('HomeView received:', message.type, message.data)

  switch (message.type) {
    case 'ws_connection':
      handleWsConnection(message.data?.status)
      break
    case 'state_change':
      handleStateChange(message.data.state)
      break
    case 'match_waiting':
      matchingDifficultyLabel.value = message.data?.difficultyLabel || matchingDifficultyLabel.value
      break
    case 'duplicate_login':
      stopMatchingUI()
      clearSoloTrainingTimer()
      isMatchPending.value = false
      isSoloPending.value = false
      connectionNotice.value = '当前账号的旧连接已断开，请重新点击开始训练'
      break
    case 'game_start':
    case 'game_resume':
      // 后端推送游戏开始，跳转到游戏页面
      stopMatchingUI()
      clearSoloTrainingTimer()
      isSoloPending.value = false
      showTrainingSetup.value = false
      router.push({
        path: '/game',
        state: { gameData: message.data }
      })
      break
    case 'error':
      console.error('Server error:', message.data.message)
      // 只在匹配流程中才重置 UI，避免干扰其他流程
      if (isMatching.value || isMatchPending.value) {
        stopMatchingUI()
        isMatchPending.value = false
      }
      clearSoloTrainingTimer()
      isSoloPending.value = false
      connectionNotice.value = message.data?.message || '训练启动失败，请稍后重试'
      break
  }
}

/**
 * 处理后端状态推送
 */
function handleStateChange(state: string) {
  console.log('State changed to:', state)
  switch (state) {
    case 'IDLE':
      // 收到 IDLE 状态，如果当前正在匹配则停止匹配 UI
      // （说明后端已经取消了匹配，可能是断线重连导致的状态重置）
      stopMatchingUI()
      isMatchPending.value = false
      if (isSoloPending.value) {
        clearSoloTrainingTimer()
        isSoloPending.value = false
        connectionNotice.value = '训练未能启动，请重试'
      }
      break
    case 'MATCHING':
      // 如果后端确认是 MATCHING 状态但前端 UI 没有展示，同步展示
      if (!isMatching.value) {
        isMatching.value = true
        matchSeconds.value = 0
        matchTimer = setInterval(() => matchSeconds.value++, 1000)
      }
      isMatchPending.value = false
      connectionNotice.value = ''
      break
    case 'GRABBING':
      isMatchPending.value = false
      clearSoloTrainingTimer()
      isSoloPending.value = false
      connectionNotice.value = ''
      break
    case 'ANSWERING':
      stopMatchingUI()
      isMatchPending.value = false
      clearSoloTrainingTimer()
      isSoloPending.value = false
      connectionNotice.value = ''
      break
    case 'FINISHED':
      stopMatchingUI()
      isMatchPending.value = false
      clearSoloTrainingTimer()
      isSoloPending.value = false
      connectionNotice.value = ''
      break
  }
}

function handleWsConnection(status: string) {
  if (status === 'disconnected') {
    if (isMatching.value || isMatchPending.value) {
      stopMatchingUI()
      isMatchPending.value = false
      connectionNotice.value = '连接中断，正在重连...'
    }
    if (isSoloPending.value) {
      clearSoloTrainingTimer()
      isSoloPending.value = false
      connectionNotice.value = '连接中断，训练未启动，请稍后重试'
    }
    return
  }

  if (status === 'connected' || status === 'reconnected') {
    if (status === 'reconnected') {
      connectionNotice.value = '连接已恢复，正在同步状态...'
    }
    wsStore.send('sync_state')
  }
}

/**
 * 开始匹配 — 发送 WebSocket 消息给后端
 */
let matchFlowInProgress = false
async function handleMatch(difficulty: SelectedDifficulty = rankDifficultyOption) {
  // 双重防护：isMatching 禁用按钮 + matchFlowInProgress 防重入
  if (isMatching.value || isMatchPending.value || matchFlowInProgress) return
  matchFlowInProgress = true
  isMatchPending.value = true
  matchingDifficultyLabel.value = difficulty.title
  connectionNotice.value = ''

  try {
    // 如果 WebSocket 断开，自动重连
    const ok = await wsStore.ensureConnected()
    if (!ok) {
      isMatchPending.value = false
      alert('连接服务器失败，请刷新页面重试')
      return
    }

    const sent = wsStore.send('match_start', {
      difficultyGroup: difficulty.parentKey,
      difficultyLevel: difficulty.key
    })
    if (!sent) {
      isMatchPending.value = false
      connectionNotice.value = '匹配请求发送失败，请重试'
    }
  } finally {
    matchFlowInProgress = false
  }
}

async function handleSoloTraining() {
  if (isMatching.value || isMatchPending.value || isSoloPending.value) return
  isSoloPending.value = true
  connectionNotice.value = ''
  clearSoloTrainingTimer()
  soloTrainingTimer = setTimeout(() => {
    if (!isSoloPending.value) return
    isSoloPending.value = false
    connectionNotice.value = '训练启动超时，请重试'
    wsStore.send('sync_state')
  }, 12000)

  const ok = await wsStore.ensureConnected()
  if (!ok) {
    clearSoloTrainingTimer()
    isSoloPending.value = false
    alert('连接服务器失败，请刷新页面重试')
    return
  }

  const sent = wsStore.send('solo_training_start', {
    difficultyGroup: selectedDifficulty.value.parentKey,
    difficultyLevel: selectedDifficulty.value.key
  })
  if (!sent) {
    clearSoloTrainingTimer()
    isSoloPending.value = false
    connectionNotice.value = '训练启动请求发送失败，请重试'
  }
}

function clearSoloTrainingTimer() {
  if (soloTrainingTimer) {
    clearTimeout(soloTrainingTimer)
    soloTrainingTimer = null
  }
}

function openTrainingSetup() {
  if (isMatching.value || isMatchPending.value || isSoloPending.value) return
  showTrainingSetup.value = true
  connectionNotice.value = ''
}

function closeTrainingSetup() {
  if (isSoloPending.value || isMatching.value || isMatchPending.value) return
  showTrainingSetup.value = false
}

function closeDifficultyPicker() {
  showDifficultyPicker.value = false
}

useEscapeClose(() => {
  if (showDifficultyPicker.value) {
    closeDifficultyPicker()
  } else if (showTrainingSetup.value) {
    closeTrainingSetup()
  }
})

function toggleDifficulty(option: DifficultyOption) {
  setSelectedDifficulty({
    key: option.key,
    parentKey: option.key,
    title: option.title
  })
  expandedDifficultyKey.value = expandedDifficultyKey.value === option.key ? '' : option.key
}

function selectDifficulty(option: DifficultyOption, child: DifficultyChild) {
  setSelectedDifficulty({
    key: child.key,
    parentKey: option.key,
    title: `${option.title} · ${child.title}`
  })
  showDifficultyPicker.value = false
}

function selectRankDifficulty() {
  setSelectedDifficulty({
    key: rankDifficultyOption.key,
    parentKey: rankDifficultyOption.parentKey,
    title: rankDifficultyOption.title
  })
  expandedDifficultyKey.value = ''
  showDifficultyPicker.value = false
}

function setSelectedDifficulty(next: SelectedDifficulty) {
  selectedDifficulty.value = next
  localStorage.setItem(TRAINING_DIFFICULTY_STORAGE_KEY, JSON.stringify(next))
}

function loadSavedDifficulty(): SelectedDifficulty {
  try {
    const raw = localStorage.getItem(TRAINING_DIFFICULTY_STORAGE_KEY)
    if (!raw) return defaultDifficulty
    const parsed = JSON.parse(raw) as Partial<SelectedDifficulty>
    if (!parsed.key || !parsed.parentKey || !parsed.title) return defaultDifficulty
    if (parsed.key === rankDifficultyOption.key && parsed.parentKey === rankDifficultyOption.parentKey) {
      return { key: parsed.key, parentKey: parsed.parentKey, title: parsed.title }
    }
    const parent = difficultyOptions.find(option => option.key === parsed.parentKey)
    if (!parent) return defaultDifficulty
    if (parsed.key === parent.key || parent.children.some(child => child.key === parsed.key)) {
      return { key: parsed.key, parentKey: parsed.parentKey, title: parsed.title }
    }
    return defaultDifficulty
  } catch {
    return defaultDifficulty
  }
}

/**
 * 取消匹配
 */
function cancelMatch() {
  // 先停止前端 UI
  stopMatchingUI()
  isMatchPending.value = false
  connectionNotice.value = ''

  // 尝试发送取消请求，如果 ws 断开也无妨（后端会在 channelInactive 中清理）
  if (wsStore.connected) {
    wsStore.send('match_cancel')
  }
}

function stopMatchingUI() {
  isMatching.value = false
  matchSeconds.value = 0
  matchingDifficultyLabel.value = ''
  if (matchTimer) {
    clearInterval(matchTimer)
    matchTimer = null
  }
}

function formatTime(seconds: number) {
  const m = Math.floor(seconds / 60).toString().padStart(2, '0')
  const s = (seconds % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

function handleLogout() {
  wsStore.disconnect()
  authStore.logout()
  router.push('/login')
}

function goMatchRecords() {
  router.push({ path: '/records', query: { mode: 'match' } })
}

function goTrainingRecords() {
  router.push({ path: '/records', query: { mode: 'solo_training' } })
}

onMounted(() => {
  authStore.fetchUserInfo()
  const currentHistoryState = (history.state || {}) as Record<string, unknown>
  showTrainingSetup.value = Boolean(currentHistoryState.openTrainingSetup || route.query.training === '1')
  if ('openTrainingSetup' in currentHistoryState) {
    const nextHistoryState = { ...currentHistoryState }
    delete nextHistoryState.openTrainingSetup
    history.replaceState(nextHistoryState, document.title)
  }
  wsStore.registerHandler('home', handleMessage)
  // 使用全局 WebSocket 连接
  wsStore.connect(authStore.token)
  if (wsStore.connected) {
    wsStore.send('sync_state')
  }
})

onUnmounted(() => {
  wsStore.unregisterHandler('home')
  if (matchTimer) clearInterval(matchTimer)
  clearSoloTrainingTimer()
  // 注意：不关闭 WebSocket，保持连接供 GameView 使用
})
</script>

<style scoped>
.home-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
  padding: 20px;
}

.training-setup-page {
  min-height: 100vh;
  padding: 20px;
  background:
    radial-gradient(circle at 22% 18%, rgba(20, 184, 166, 0.15), transparent 34%),
    radial-gradient(circle at 78% 26%, rgba(99, 102, 241, 0.14), transparent 30%),
    linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}

.training-setup-main {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px 80px;
}

.training-setup-panel {
  width: min(620px, 92vw);
  padding: 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 34px;
  background: rgba(15, 23, 42, 0.42);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 18px;
  box-shadow: 0 28px 80px rgba(0, 0, 0, 0.24);
}

.training-setup-title {
  text-align: center;
  color: white;
}

.training-setup-title h1 {
  margin: 0;
  font-size: 40px;
  line-height: 1.15;
}

.training-setup-title p {
  margin: 12px 0 0;
  color: rgba(255, 255, 255, 0.72);
  font-size: 20px;
  font-weight: 700;
}

.training-setup-actions {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18px;
}

.setup-action-btn {
  width: min(420px, 100%);
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px;
}

.header-left {
  flex: 1;
}

.header-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 10px;
}

.nickname {
  font-size: 20px;
  font-weight: bold;
  color: white;
}

.logout-btn {
  padding: 8px 20px;
  background: #ff6b6b;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.stats-card {
  display: flex;
  align-items: center;
  justify-content: space-around;
  background: rgba(255,255,255,0.1);
  border-radius: 12px;
  padding: 20px;
  margin: 40px 20px;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
}

.stats-card:hover {
  background: rgba(255,255,255,0.15);
  transform: translateY(-2px);
  box-shadow: 0 10px 30px rgba(102, 126, 234, 0.2);
}

.stats-card::after {
  content: '点击查看详情 →';
  position: absolute;
  bottom: -30px;
  left: 50%;
  transform: translateX(-50%);
  color: rgba(255, 255, 255, 0.6);
  font-size: 12px;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.stats-card:hover::after {
  opacity: 1;
}

.training-stats-card {
  margin-top: -18px;
  background: rgba(255, 255, 255, 0.08);
}

.training-stats-card::after {
  content: '点击查看训练记录 →';
}

.stat-item {
  text-align: center;
  color: white;
  flex: 1;
}

.label {
  display: block;
  font-size: 14px;
  opacity: 0.8;
}

.value {
  display: block;
  font-size: 28px;
  font-weight: bold;
  margin-top: 8px;
}

.rank-value {
  background: linear-gradient(135deg, #ffd700, #ffaa00);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.match-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-top: 60px;
}

.match-btn,
.secondary-action-btn {
  padding: 20px 80px;
  font-size: 24px;
  color: white;
  border-radius: 50px;
  cursor: pointer;
  transition: all 0.3s;
}

.match-btn {
  border: 1px solid transparent;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  box-shadow: 0 10px 30px rgba(102, 126, 234, 0.4);
}

.secondary-action-btn {
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: rgba(15, 23, 42, 0.46);
  box-shadow: none;
  color: rgba(255, 255, 255, 0.9);
}

.match-btn:hover:not(:disabled) {
  transform: scale(1.05);
}

.secondary-action-btn:hover:not(:disabled) {
  border-color: rgba(148, 163, 184, 0.38);
  background: rgba(30, 41, 59, 0.68);
  color: white;
  transform: translateY(-2px);
}

.match-btn:disabled,
.secondary-action-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.connection-notice {
  margin-top: 16px;
  color: rgba(255, 230, 120, 0.92);
  font-size: 14px;
  text-align: center;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 100;
}

.modal-box {
  background: linear-gradient(135deg, #1e2a45, #16213e);
  border: 1px solid rgba(102, 126, 234, 0.3);
  border-radius: 20px;
  padding: 48px 60px;
  text-align: center;
  box-shadow: 0 0 60px rgba(102, 126, 234, 0.2);
}

.radar {
  position: relative;
  width: 100px;
  height: 100px;
  margin: 0 auto 24px;
}

.radar-ring {
  position: absolute;
  border-radius: 50%;
  border: 2px solid rgba(102, 126, 234, 0.6);
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%) scale(0);
  animation: ping 2s ease-out infinite;
}

.ring1 { animation-delay: 0s; }
.ring2 { animation-delay: 0.6s; }
.ring3 { animation-delay: 1.2s; }

@keyframes ping {
  0%   { width: 10px; height: 10px; opacity: 1; }
  100% { width: 100px; height: 100px; opacity: 0; }
}

.radar-dot {
  position: absolute;
  width: 14px;
  height: 14px;
  background: #667eea;
  border-radius: 50%;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  box-shadow: 0 0 12px #667eea;
}

.modal-title {
  color: white;
  font-size: 20px;
  margin-bottom: 8px;
}

.modal-difficulty {
  margin: 0 0 12px;
  color: rgba(191, 219, 254, 0.92);
  font-size: 15px;
  font-weight: 700;
}

.modal-timer {
  color: #ffd700;
  font-size: 40px;
  font-weight: bold;
  font-variant-numeric: tabular-nums;
  margin-bottom: 28px;
}

.cancel-btn {
  padding: 10px 32px;
  background: transparent;
  border: 1px solid rgba(255, 107, 107, 0.6);
  color: #ff6b6b;
  border-radius: 8px;
  font-size: 15px;
  cursor: pointer;
  transition: all 0.2s;
}

.cancel-btn:hover {
  background: rgba(255, 107, 107, 0.1);
}

.difficulty-overlay {
  position: fixed;
  inset: 0;
  z-index: 4000;
  min-height: 100vh;
  padding: 32px;
  display: flex;
  flex-direction: column;
  gap: 24px;
  background:
    radial-gradient(circle at 18% 16%, rgba(56, 189, 248, 0.18), transparent 30%),
    radial-gradient(circle at 82% 28%, rgba(249, 115, 22, 0.14), transparent 28%),
    linear-gradient(135deg, #101426 0%, #16213e 100%);
}

.difficulty-header {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  color: white;
}

.difficulty-header h2 {
  margin: 0;
  font-size: 30px;
  line-height: 1.2;
}

.difficulty-header p {
  margin: 6px 0 0;
  color: rgba(255, 255, 255, 0.68);
  font-size: 15px;
  text-align: right;
}

.rank-difficulty-btn {
  position: absolute;
  left: 50%;
  top: 0;
  transform: translateX(-50%);
  min-width: 260px;
  padding: 12px 28px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  color: #edf4ff;
  background:
    linear-gradient(135deg, rgba(59, 130, 246, 0.2), rgba(20, 184, 166, 0.14)),
    rgba(18, 24, 43, 0.9);
  border: 2px solid rgba(147, 197, 253, 0.36);
  border-radius: 18px;
  cursor: pointer;
  box-shadow: 0 14px 36px rgba(59, 130, 246, 0.16);
  transition: transform 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
}

.rank-difficulty-btn:hover,
.rank-difficulty-btn.active {
  transform: translateX(-50%) translateY(-2px);
  border-color: #93c5fd;
  background:
    linear-gradient(135deg, rgba(59, 130, 246, 0.3), rgba(20, 184, 166, 0.18)),
    rgba(18, 24, 43, 0.96);
  box-shadow: 0 18px 44px rgba(59, 130, 246, 0.24);
}

.rank-difficulty-btn span {
  font-size: 24px;
  font-weight: 800;
  line-height: 1.15;
}

.rank-difficulty-btn small {
  color: rgba(255, 255, 255, 0.66);
  font-size: 13px;
  font-weight: 700;
}

.difficulty-grid {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  grid-template-rows: repeat(2, minmax(0, 1fr));
  gap: 18px;
}

.difficulty-card {
  position: relative;
  overflow: hidden;
  width: 100%;
  min-width: 0;
  min-height: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #edf4ff;
  background:
    linear-gradient(145deg, rgba(255, 255, 255, 0.09), rgba(255, 255, 255, 0.04)),
    rgba(18, 24, 43, 0.94);
  border: 2px solid rgba(255, 255, 255, 0.18);
  border-radius: 18px;
  box-shadow: 0 18px 46px rgba(0, 0, 0, 0.28);
  transition: transform 0.22s ease, border-color 0.22s ease, box-shadow 0.22s ease, background 0.22s ease;
}

.difficulty-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(96, 165, 250, 0.16), rgba(20, 184, 166, 0.1));
  opacity: 0;
  transition: opacity 0.22s ease;
}

.difficulty-card:hover,
.difficulty-card.active {
  border-color: #93c5fd;
  box-shadow: 0 20px 56px rgba(59, 130, 246, 0.24);
}

.difficulty-card:not(.expanded):hover,
.difficulty-card:not(.expanded).active {
  transform: translateY(-4px);
}

.difficulty-card:hover::before,
.difficulty-card.active::before {
  opacity: 1;
}

.difficulty-card.active {
  background:
    linear-gradient(145deg, rgba(59, 130, 246, 0.18), rgba(20, 184, 166, 0.1)),
    rgba(18, 24, 43, 0.98);
}

.difficulty-card.expanded {
  justify-content: flex-start;
}

.difficulty-card-button {
  position: relative;
  z-index: 1;
  width: 100%;
  min-height: 108px;
  padding: 24px 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: inherit;
  background: transparent;
  border: 0;
  cursor: pointer;
}

.difficulty-card:not(.expanded) .difficulty-card-button {
  flex: 1;
}

.difficulty-card.expanded .difficulty-card-button {
  justify-content: flex-start;
  min-height: 84px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.12);
}

.difficulty-chevron,
.difficulty-title {
  position: relative;
  z-index: 1;
}

.difficulty-chevron {
  margin-right: 18px;
  color: #a8c7ee;
  font-size: 46px;
  line-height: 1;
  font-weight: 300;
}

.difficulty-title {
  max-width: 100%;
  overflow-wrap: anywhere;
  font-size: clamp(24px, 2.15vw, 38px);
  font-weight: 800;
  letter-spacing: 0;
}

.difficulty-children {
  position: relative;
  z-index: 1;
  width: 100%;
  flex: 1;
  min-height: 0;
  padding: 14px 18px 18px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
}

.difficulty-child-card {
  width: 100%;
  min-height: 74px;
  padding: 16px 18px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  color: white;
  background: rgba(7, 10, 18, 0.72);
  border: 2px solid rgba(255, 255, 255, 0.18);
  border-radius: 16px;
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease, transform 0.2s ease;
}

.difficulty-child-card:hover,
.difficulty-child-card.active {
  border-color: #93c5fd;
  background: rgba(20, 28, 47, 0.9);
  transform: translateY(-2px);
}

.difficulty-child-card span {
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: clamp(18px, 1.45vw, 27px);
  font-weight: 800;
  letter-spacing: 0;
}

.difficulty-child-card small {
  flex: 0 0 auto;
  padding: 8px 12px;
  color: #aaa6ff;
  background: rgba(42, 48, 66, 0.86);
  border-radius: 10px;
  font-size: clamp(14px, 1.05vw, 20px);
  font-weight: 800;
}

@media (max-width: 480px) {
  .home-container { padding: 10px; }
  .training-setup-page { padding: 10px; }
  .header { padding: 10px; }
  .nickname { font-size: 16px; }

  .training-setup-main {
    min-height: 100vh;
    padding: 24px 10px 50px;
  }

  .training-setup-panel {
    padding: 28px 18px;
    gap: 24px;
    border-radius: 14px;
  }

  .training-setup-title h1 {
    font-size: 30px;
  }

  .training-setup-title p {
    font-size: 16px;
  }

  .stats-card {
    flex-wrap: wrap;
    gap: 10px;
    padding: 20px 10px;
    margin: 20px 10px;
  }
  .stat-item { min-width: 30%; }
  .value { font-size: 22px; }
  .label { font-size: 12px; }

  .match-section { margin-top: 30px; }
  .match-btn,
  .secondary-action-btn {
    padding: 16px 50px;
    font-size: 18px;
    min-width: min(320px, 92vw);
  }

  .modal-box { padding: 30px 24px; }
  .modal-timer { font-size: 32px; }
  .modal-title { font-size: 16px; }

  .difficulty-overlay {
    padding: 16px;
    gap: 14px;
  }

  .difficulty-header {
    align-items: flex-start;
    padding-top: 66px;
  }

  .difficulty-header h2 {
    font-size: 22px;
  }

  .difficulty-header p {
    font-size: 13px;
  }

  .rank-difficulty-btn {
    top: 0;
    min-width: min(220px, 58vw);
    padding: 10px 16px;
    border-radius: 14px;
  }

  .rank-difficulty-btn span {
    font-size: 18px;
  }

  .rank-difficulty-btn small {
    font-size: 12px;
  }

  .difficulty-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    grid-template-rows: repeat(4, minmax(0, 1fr));
    gap: 10px;
  }

  .difficulty-card {
    border-radius: 14px;
  }

  .difficulty-card-button {
    min-height: 78px;
    padding: 14px;
  }

  .difficulty-card.expanded .difficulty-card-button {
    min-height: 62px;
  }

  .difficulty-chevron {
    margin-right: 8px;
    font-size: 28px;
  }

  .difficulty-title {
    font-size: 18px;
  }

  .difficulty-children {
    padding: 10px;
    gap: 8px;
  }

  .difficulty-child-card {
    min-height: 52px;
    padding: 10px 12px;
    border-radius: 12px;
  }

  .difficulty-child-card span {
    font-size: 15px;
  }

  .difficulty-child-card small {
    padding: 6px 8px;
    font-size: 12px;
  }
}
</style>
