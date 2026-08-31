<template>
  <div class="records-page">
    <FullscreenCloseButton v-if="!selectedRecord && !answerDetailModalVisible" @close="closePage" />
    <div class="header">
      <h1 class="title">{{ pageTitle }}</h1>
      <div class="mode-tabs">
        <button
          v-for="modeTab in modeTabs"
          :key="modeTab.value"
          :class="['mode-tab-btn', { active: recordMode === modeTab.value }]"
          @click="recordMode = modeTab.value"
        >
          {{ modeTab.label }}
        </button>
      </div>
      <div class="filter-tabs">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          :class="['tab-btn', { active: currentTab === tab.value }]"
          @click="currentTab = tab.value"
        >
          {{ tab.label }}
        </button>
      </div>
    </div>

    <div class="records-container" v-if="loading">
      <div class="skeleton-list">
        <div v-for="i in 5" :key="i" class="skeleton-item"></div>
      </div>
    </div>

    <div class="records-container" v-else-if="filteredRecords.length > 0">
      <div class="record-list">
        <div
          v-for="record in filteredRecords"
          :key="record.id"
          class="record-card"
          :class="{ 'win': isWin(record), 'loss': isLoss(record), 'draw': record.isDraw }"
          @click="showDetail(record)"
        >
          <div class="record-header">
            <div class="result-badge">
              <span v-if="isWin(record)" class="badge win">{{ isTrainingRecord(record) ? '训练胜利' : '胜利' }}</span>
              <span v-else-if="isLoss(record)" class="badge loss">{{ isTrainingRecord(record) ? '训练失败' : '失败' }}</span>
              <span v-else class="badge draw">{{ isTrainingRecord(record) ? '训练平局' : '平局' }}</span>
            </div>
            <div class="time-info">
              <span class="date">{{ formatDate(record.startTime) }}</span>
              <span class="time">{{ formatTime(record.startTime) }}</span>
            </div>
          </div>

          <div class="players-section">
            <!-- 左边：自己 -->
            <div class="player player-left">
              <div class="avatar" :class="{ 'winner': isWin(record), 'loser': isLoss(record) }">
                <span class="avatar-text">{{ getMyName(record)?.charAt(0) || '?' }}</span>
                <div v-if="isWin(record)" class="crown">👑</div>
              </div>
              <div class="player-info">
                <span class="player-name">{{ getMyName(record) }}</span>
                <span class="player-score">{{ getMyScore(record) }}分</span>
              </div>
              <div class="stats">
                <span class="correct">{{ getMyCorrectCount(record) }}/{{ getMyTotalCount(record) }}</span>
                <span class="accuracy">准确率: {{ calculateMyAccuracy(record) }}%</span>
              </div>
            </div>

            <div class="vs-divider">
              <button class="detail-btn-side left" @click.stop="showAnswerDetail(record, 'self')">
                答题详情
              </button>
              <div class="vs-center">
                <span class="vs-text">VS</span>
                <span class="duration">{{ formatDuration(record.durationSeconds) }}</span>
              </div>
              <button class="detail-btn-side right" @click.stop="showAnswerDetail(record, 'opponent')">
                答题详情
              </button>
            </div>

            <!-- 右边：对手 -->
            <div class="player player-right">
              <div class="avatar" :class="{ 'winner': isLoss(record), 'loser': isWin(record) }">
                <span class="avatar-text">{{ getOpponentName(record)?.charAt(0) || '?' }}</span>
                <div v-if="isLoss(record)" class="crown">👑</div>
              </div>
              <div class="player-info">
                <span class="player-name">{{ getOpponentName(record) }}</span>
                <span class="player-score">{{ getOpponentScore(record) }}分</span>
              </div>
              <div class="stats">
                <span class="correct">{{ getOpponentCorrectCount(record) }}/{{ getOpponentTotalCount(record) }}</span>
                <span class="accuracy">准确率: {{ calculateOpponentAccuracy(record) }}%</span>
              </div>
            </div>
          </div>

          <div class="record-footer">
            <span class="battle-time">{{ isTrainingRecord(record) ? '训练时长' : '对战时长' }}: {{ formatDuration(record.durationSeconds) }}</span>
            <span v-if="!isTrainingRecord(record)" class="match-difficulty">
              匹配难度：{{ record.matchDifficultyLabel || '段位难度' }}
            </span>
            <div v-if="isTrainingRecord(record)" class="training-meta">
              <span>{{ formatTrainingExpChange(record.trainingExpChange) }}</span>
              <span>训练等级 {{ record.trainingRankAfter || '-' }}</span>
              <span>{{ getRobotTierLabel(record.robotTier) }}</span>
              <span v-if="record.robotAptitude">资质 {{ record.robotAptitude }}</span>
              <span v-if="record.robotGrowth">成长 {{ Number(record.robotGrowth).toFixed(2) }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="pagination" v-if="total > 0">
        <label class="page-size-select">
          <span>每页</span>
          <select :value="pageSize" @change="changePageSize">
            <option v-for="size in pageSizeOptions" :key="size" :value="size">
              {{ size }} 条
            </option>
          </select>
        </label>
        <button
          class="page-btn"
          :disabled="currentPage === 1"
          @click="changePage(currentPage - 1)"
        >
          上一页
        </button>
        <div class="page-numbers">
          <button
            v-for="page in displayedPages"
            :key="page"
            :class="['page-number', { active: currentPage === page }]"
            @click="changePage(page)"
          >
            {{ page }}
          </button>
        </div>
        <button
          class="page-btn"
          :disabled="currentPage === totalPages"
          @click="changePage(currentPage + 1)"
        >
          下一页
        </button>
      </div>
    </div>

    <div class="records-container empty" v-else>
      <div class="empty-state">
        <div class="empty-icon">📊</div>
        <h3>{{ recordMode === 'solo_training' ? '暂无训练记录' : '暂无战绩' }}</h3>
        <p>{{ recordMode === 'solo_training' ? '还没有进行过难度训练，先去练一局吧。' : '还没有进行过对战，快去匹配对手吧。' }}</p>
        <button class="play-btn" @click="router.push(recordMode === 'solo_training' ? '/home' : '/game')">
          {{ recordMode === 'solo_training' ? '去训练' : '立即对战' }}
        </button>
      </div>
    </div>

    <!-- Detail Modal -->
    <Transition name="modal">
      <div class="modal-overlay" v-if="selectedRecord && !answerDetailModalVisible" @click="selectedRecord = null">
        <FullscreenCloseButton @close="selectedRecord = null" />
        <div class="modal-content" @click.stop>
          <div class="modal-header">
            <h3>{{ isTrainingRecord(selectedRecord) ? '训练详情' : '对战详情' }}</h3>
          </div>
          <div class="modal-body" v-if="selectedRecord">
            <div class="detail-players">
              <!-- 左边：自己 -->
              <div class="detail-player" :class="{ 'winner': isWin(selectedRecord), 'loser': isLoss(selectedRecord) }">
                <div class="detail-avatar" :class="{ 'winner': isWin(selectedRecord), 'loser': isLoss(selectedRecord) }">
                  <span class="avatar-text">{{ getMyName(selectedRecord)?.charAt(0) || '?' }}</span>
                  <div v-if="isWin(selectedRecord)" class="crown">👑</div>
                </div>
                <span class="name">{{ getMyName(selectedRecord) }}</span>
                <span class="score">{{ getMyScore(selectedRecord) }}</span>
              </div>
              <div class="detail-vs">VS</div>
              <!-- 右边：对手 -->
              <div class="detail-player" :class="{ 'winner': isLoss(selectedRecord), 'loser': isWin(selectedRecord) }">
                <div class="detail-avatar" :class="{ 'winner': isLoss(selectedRecord), 'loser': isWin(selectedRecord) }">
                  <span class="avatar-text">{{ getOpponentName(selectedRecord)?.charAt(0) || '?' }}</span>
                  <div v-if="isLoss(selectedRecord)" class="crown">👑</div>
                </div>
                <span class="name">{{ getOpponentName(selectedRecord) }}</span>
                <span class="score">{{ getOpponentScore(selectedRecord) }}</span>
              </div>
            </div>
            <div class="detail-stats">
              <div class="stat-row">
                <span>答题数</span>
                <span>{{ getMyTotalCount(selectedRecord) }} : {{ getOpponentTotalCount(selectedRecord) }}</span>
              </div>
              <div class="stat-row">
                <span>正确数</span>
                <span>{{ getMyCorrectCount(selectedRecord) }} : {{ getOpponentCorrectCount(selectedRecord) }}</span>
              </div>
              <div class="stat-row">
                <span>准确率</span>
                <span>{{ calculateMyAccuracy(selectedRecord) }}% : {{ calculateOpponentAccuracy(selectedRecord) }}%</span>
              </div>
              <div class="stat-row">
                <span>{{ isTrainingRecord(selectedRecord) ? '训练时长' : '对战时长' }}</span>
                <span>{{ formatDuration(selectedRecord.durationSeconds) }}</span>
              </div>
              <div class="stat-row">
                <span>开始时间</span>
                <span>{{ formatDateTime(selectedRecord.startTime) }}</span>
              </div>
              <div v-if="!isTrainingRecord(selectedRecord)" class="stat-row">
                <span>匹配难度</span>
                <span>{{ selectedRecord.matchDifficultyLabel || '段位难度' }}</span>
              </div>
              <template v-if="isTrainingRecord(selectedRecord)">
                <div class="stat-row">
                  <span>训练经验</span>
                  <span>{{ formatTrainingExpChange(selectedRecord.trainingExpChange) }}</span>
                </div>
                <div class="stat-row">
                  <span>训练等级</span>
                  <span>{{ selectedRecord.trainingRankAfter || '-' }}</span>
                </div>
                <div class="stat-row">
                  <span>机器人</span>
                  <span>{{ getRobotTierLabel(selectedRecord.robotTier) }}</span>
                </div>
                <div class="stat-row">
                  <span>机器人面板</span>
                  <span>{{ formatRobotProfile(selectedRecord) }}</span>
                </div>
              </template>
            </div>
          </div>
        </div>
      </div>
    </Transition>
    <!-- Answer Detail Modal -->
    <AnswerDetailModal
      :visible="answerDetailModalVisible"
      :record-id="selectedRecord?.id"
      :target-user-id="answerDetailTargetUserId"
      :player-name="answerDetailPlayerName"
      :initial-round="answerDetailInitialRound"
      @close="closeAnswerDetailModal"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import axios from '../api'
import AnswerDetailModal from '../components/AnswerDetailModal.vue'
import FullscreenCloseButton from '../components/FullscreenCloseButton.vue'
import { useEscapeClose } from '../composables/useEscapeClose'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const records = ref<any[]>([])
const loading = ref(true)
const currentPage = ref(1)
const pageSize = ref(20)
const pageSizeOptions = [20, 50, 100, 500]
const total = ref(0)
const currentTab = ref('all')
const recordMode = ref<'match' | 'solo_training'>(normalizeRecordMode(route.query.mode))
const selectedRecord = ref<any>(null)
const answerDetailModalVisible = ref(false)
const answerDetailTargetUserId = ref<number | undefined>(undefined)
const answerDetailPlayerName = ref('')
const answerDetailInitialRound = ref<number | undefined>(undefined)

function closePage() {
  router.push('/home')
}

useEscapeClose(() => {
  if (answerDetailModalVisible.value) closeAnswerDetailModal()
  else if (selectedRecord.value) selectedRecord.value = null
  else closePage()
})

const modeTabs: Array<{ label: string; value: 'match' | 'solo_training' }> = [
  { label: '对战记录', value: 'match' },
  { label: '训练记录', value: 'solo_training' },
]

function normalizeRecordMode(mode: unknown): 'match' | 'solo_training' {
  if (Array.isArray(mode)) {
    return mode[0] === 'solo_training' ? 'solo_training' : 'match'
  }
  return mode === 'solo_training' ? 'solo_training' : 'match'
}

const tabs = [
  { label: '全部', value: 'all' },
  { label: '胜利', value: 'win' },
  { label: '失败', value: 'loss' },
]

const pageTitle = computed(() => recordMode.value === 'solo_training' ? '训练记录' : '战绩列表')

const filteredRecords = computed(() => {
  if (currentTab.value === 'all') return records.value
  if (currentTab.value === 'win') {
    return records.value.filter(r => isWin(r))
  }
  if (currentTab.value === 'loss') {
    return records.value.filter(r => isLoss(r))
  }
  return records.value
})

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

const displayedPages = computed(() => {
  const pages: number[] = []
  const maxDisplay = 5
  let start = Math.max(1, currentPage.value - Math.floor(maxDisplay / 2))
  let end = Math.min(totalPages.value, start + maxDisplay - 1)

  if (end - start < maxDisplay - 1) {
    start = Math.max(1, end - maxDisplay + 1)
  }

  for (let i = start; i <= end; i++) {
    pages.push(i)
  }
  return pages
})

const isWin = (record: any) => {
  return record.winnerId === authStore.userId
}

const isLoss = (record: any) => {
  return record.winnerId != null && record.winnerId !== authStore.userId
}

const isTrainingRecord = (record: any) => {
  return record?.mode === 'solo_training'
}

// 判断当前用户是否是 player1
const isCurrentUserPlayer1 = (record: any) => {
  return record.player1Id === authStore.userId
}

// 获取自己的名字
const getMyName = (record: any) => {
  return isCurrentUserPlayer1(record) ? record.player1Name : record.player2Name
}

// 获取自己的分数
const getMyScore = (record: any) => {
  return isCurrentUserPlayer1(record) ? record.player1Score : record.player2Score
}

// 获取自己的正确数
const getMyCorrectCount = (record: any) => {
  return isCurrentUserPlayer1(record) ? record.player1CorrectCount : record.player2CorrectCount
}

// 获取自己的总答题数
const getMyTotalCount = (record: any) => {
  return isCurrentUserPlayer1(record) ? record.player1TotalCount : record.player2TotalCount
}

// 获取对手名字
const getOpponentName = (record: any) => {
  return isCurrentUserPlayer1(record) ? record.player2Name : record.player1Name
}

// 获取对手分数
const getOpponentScore = (record: any) => {
  return isCurrentUserPlayer1(record) ? record.player2Score : record.player1Score
}

// 获取对手正确数
const getOpponentCorrectCount = (record: any) => {
  return isCurrentUserPlayer1(record) ? record.player2CorrectCount : record.player1CorrectCount
}

// 获取对手总答题数
const getOpponentTotalCount = (record: any) => {
  return isCurrentUserPlayer1(record) ? record.player2TotalCount : record.player1TotalCount
}

const formatDate = (dateStr: string) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

const formatTime = (dateStr: string) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`
}

const formatDateTime = (dateStr: string) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`
}

const formatDuration = (seconds: number) => {
  if (!seconds) return '0秒'
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  if (mins > 0) {
    return `${mins}分${secs}秒`
  }
  return `${secs}秒`
}

// 计算自己的准确率
const calculateMyAccuracy = (record: any) => {
  const correct = getMyCorrectCount(record)
  const total = getMyTotalCount(record)
  if (total === 0) return 0
  return Math.round((correct / total) * 100)
}

// 计算对手的准确率
const calculateOpponentAccuracy = (record: any) => {
  const correct = getOpponentCorrectCount(record)
  const total = getOpponentTotalCount(record)
  if (total === 0) return 0
  return Math.round((correct / total) * 100)
}

const formatTrainingExpChange = (value: number | null | undefined) => {
  const exp = Number(value || 0)
  return `训练经验 ${exp >= 0 ? '+' : ''}${exp}`
}

const getRobotTierLabel = (tier: string | null | undefined) => {
  if (tier === 'strong') return '天才型机器人'
  if (tier === 'normal') return '稳健型机器人'
  if (tier === 'weak') return '摸鱼型机器人'
  return '训练机器人'
}

const formatRobotProfile = (record: any) => {
  const parts = []
  if (record.robotAptitude) parts.push(`资质 ${record.robotAptitude}`)
  if (record.robotGrowth) parts.push(`成长 ${Number(record.robotGrowth).toFixed(2)}`)
  return parts.length ? parts.join(' / ') : '-'
}

const fetchRecords = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/game/records', {
      params: {
        page: currentPage.value,
        size: pageSize.value,
        mode: recordMode.value
      }
    })
    records.value = res.data.records || []
    total.value = res.data.total || 0
    await openRecordFromRoute()
  } catch (error) {
    console.error('Failed to fetch records:', error)
  } finally {
    loading.value = false
  }
}

const changePage = (page: number) => {
  currentPage.value = page
  fetchRecords()
}

const changePageSize = (event: Event) => {
  pageSize.value = Number((event.target as HTMLSelectElement).value)
  currentPage.value = 1
  fetchRecords()
}

const showDetail = (record: any) => {
  selectedRecord.value = record
}

const showAnswerDetail = (record: any, type: 'self' | 'opponent') => {
  selectedRecord.value = record
  answerDetailInitialRound.value = parseRoundQuery(route.query.round)
  if (type === 'self') {
    answerDetailTargetUserId.value = authStore.userId ?? undefined
    answerDetailPlayerName.value = getMyName(record)
  } else {
    answerDetailTargetUserId.value = isCurrentUserPlayer1(record) ? record.player2Id : record.player1Id
    answerDetailPlayerName.value = getOpponentName(record)
  }
  answerDetailModalVisible.value = true
}

const closeAnswerDetailModal = () => {
  answerDetailModalVisible.value = false
  selectedRecord.value = null
  answerDetailTargetUserId.value = undefined
  answerDetailPlayerName.value = ''
  answerDetailInitialRound.value = undefined
  if (route.query.recordId || route.query.round) {
    router.replace({ path: '/records', query: { mode: recordMode.value } })
  }
}

const parseRecordIdQuery = (value: unknown) => {
  const raw = Array.isArray(value) ? value[0] : value
  const parsed = Number(raw)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null
}

const parseRoundQuery = (value: unknown) => {
  const raw = Array.isArray(value) ? value[0] : value
  const parsed = Number(raw)
  return Number.isFinite(parsed) && parsed > 0 ? Math.floor(parsed) : undefined
}

const openRecordFromRoute = async () => {
  const recordId = parseRecordIdQuery(route.query.recordId)
  if (!recordId || answerDetailModalVisible.value) return

  let targetRecord = records.value.find(record => record.id === recordId)
  if (!targetRecord) {
    try {
      const res = await axios.get(`/api/game/records/${recordId}`)
      targetRecord = res.data
    } catch (error) {
      console.error('Failed to fetch linked record:', error)
      return
    }
  }
  if (!targetRecord) return

  selectedRecord.value = targetRecord
  answerDetailTargetUserId.value = authStore.userId ?? undefined
  answerDetailPlayerName.value = getMyName(targetRecord)
  answerDetailInitialRound.value = parseRoundQuery(route.query.round)
  answerDetailModalVisible.value = true
}

watch(currentTab, () => {
  currentPage.value = 1
})

watch(recordMode, () => {
  currentPage.value = 1
  currentTab.value = 'all'
  selectedRecord.value = null
  fetchRecords()
})

watch(() => route.query.mode, mode => {
  const nextMode = normalizeRecordMode(mode)
  if (nextMode !== recordMode.value) {
    recordMode.value = nextMode
  }
})

watch(() => [route.query.recordId, route.query.round], () => {
  openRecordFromRoute()
})

onMounted(async () => {
  // 确保用户信息已加载
  if (!authStore.userId && authStore.isLoggedIn) {
    await authStore.fetchUserInfo()
  }
  fetchRecords()
})
</script>

<style scoped>
.records-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.header {
  max-width: 800px;
  margin: 0 auto 20px;
}

.title {
  color: white;
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 16px;
  text-align: center;
}

.mode-tabs {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  max-width: 360px;
  margin: 0 auto 14px;
  padding: 4px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.18);
}

.mode-tab-btn {
  border: none;
  border-radius: 12px;
  padding: 10px 16px;
  background: transparent;
  color: rgba(255, 255, 255, 0.86);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mode-tab-btn.active {
  background: white;
  color: #667eea;
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.12);
}

.filter-tabs {
  display: flex;
  justify-content: center;
  gap: 12px;
}

.tab-btn {
  background: rgba(255, 255, 255, 0.2);
  border: none;
  color: white;
  padding: 10px 24px;
  border-radius: 20px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.3s ease;
}

.tab-btn:hover {
  background: rgba(255, 255, 255, 0.3);
}

.tab-btn.active {
  background: white;
  color: #667eea;
}

.records-container {
  max-width: 800px;
  margin: 0 auto;
}

.record-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.record-card {
  background: white;
  border-radius: 20px;
  padding: 20px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
  cursor: pointer;
  position: relative;
  overflow: hidden;
}

.record-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
}

.record-card.win::before {
  background: linear-gradient(90deg, #10b981, #34d399);
}

.record-card.loss::before {
  background: linear-gradient(90deg, #ef4444, #f87171);
}

.record-card.draw::before {
  background: linear-gradient(90deg, #6b7280, #9ca3af);
}

.record-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
}

.record-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.result-badge .badge {
  padding: 6px 16px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.badge.win {
  background: #dcfce7;
  color: #166534;
}

.badge.loss {
  background: #fee2e2;
  color: #991b1b;
}

.badge.draw {
  background: #f3f4f6;
  color: #374151;
}

.time-info {
  display: flex;
  gap: 8px;
  color: #6b7280;
  font-size: 13px;
}

.players-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 16px 0;
  border-top: 1px solid #f3f4f6;
  border-bottom: 1px solid #f3f4f6;
}

.player {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.player-left {
  flex-direction: row;
}

.player-right {
  flex-direction: row-reverse;
}

.avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.avatar.winner {
  background: linear-gradient(135deg, #ffd700 0%, #ffaa00 50%, #ff8c00 100%);
  box-shadow:
    0 0 20px rgba(255, 215, 0, 0.6),
    0 0 40px rgba(255, 215, 0, 0.3),
    inset 0 0 20px rgba(255, 255, 255, 0.3);
  animation: winnerGlow 1.5s ease-in-out infinite;
  border: 3px solid #ffd700;
}

.avatar.winner::before {
  content: '';
  position: absolute;
  top: -5px;
  left: -5px;
  right: -5px;
  bottom: -5px;
  border-radius: 50%;
  background: conic-gradient(
    from 0deg,
    transparent 0deg,
    rgba(255, 215, 0, 0.6) 60deg,
    transparent 120deg,
    rgba(255, 215, 0, 0.6) 180deg,
    transparent 240deg,
    rgba(255, 215, 0, 0.6) 300deg,
    transparent 360deg
  );
  animation: winnerRotate 3s linear infinite;
  z-index: -1;
}

.avatar.winner::after {
  content: 'WIN';
  position: absolute;
  top: -18px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 14px;
  font-weight: 900;
  color: #ffd700;
  text-shadow:
    0 0 5px rgba(255, 215, 0, 0.8),
    0 0 10px rgba(255, 215, 0, 0.6),
    0 2px 4px rgba(0, 0, 0, 0.3);
  animation: winnerPulse 0.8s ease-in-out infinite;
  letter-spacing: 1px;
}

@keyframes winnerGlow {
  0%, 100% {
    box-shadow:
      0 0 20px rgba(255, 215, 0, 0.6),
      0 0 40px rgba(255, 215, 0, 0.3),
      inset 0 0 20px rgba(255, 255, 255, 0.3);
  }
  50% {
    box-shadow:
      0 0 30px rgba(255, 215, 0, 0.9),
      0 0 60px rgba(255, 215, 0, 0.5),
      inset 0 0 30px rgba(255, 255, 255, 0.5);
  }
}

@keyframes winnerRotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes winnerPulse {
  0%, 100% {
    transform: translateX(-50%) scale(1);
    opacity: 1;
  }
  50% {
    transform: translateX(-50%) scale(1.15);
    opacity: 0.9;
  }
}

.avatar.loser {
  transform: scale(0.9);
  opacity: 0.6;
  filter: grayscale(60%);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.avatar.loser::after {
  content: 'KO';
  position: absolute;
  top: -20px;
  left: 50%;
  transform: translateX(-50%) rotate(-10deg);
  font-size: 20px;
  font-weight: 900;
  color: #ff4444;
  animation: koPulse 0.8s ease-in-out infinite;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

@keyframes koPulse {
  0%, 100% {
    transform: translateX(-50%) rotate(-10deg) scale(1);
    opacity: 1;
  }
  50% {
    transform: translateX(-50%) rotate(-10deg) scale(1.1);
    opacity: 0.8;
  }
}

.avatar-text {
  color: white;
  font-size: 18px;
  font-weight: 600;
}

.crown {
  position: absolute;
  top: -8px;
  right: -4px;
  font-size: 16px;
}

.player-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.player-name {
  font-weight: 600;
  color: #1f2937;
  font-size: 14px;
}

.player-score {
  font-size: 20px;
  font-weight: 700;
  color: #667eea;
}

.stats {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #6b7280;
}

.stats .accuracy {
  color: #667eea;
  font-weight: 500;
}

.player-left .stats {
  align-items: flex-start;
}

.player-right .stats {
  align-items: flex-end;
}

.vs-divider {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

.vs-center {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.detail-btn-side {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  color: white;
  padding: 6px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.detail-btn-side:hover {
  transform: scale(1.05);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.detail-btn-side.left {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.detail-btn-side.right {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.detail-btn-side.right:hover {
  box-shadow: 0 4px 12px rgba(245, 87, 108, 0.4);
}

.vs-text {
  font-size: 14px;
  font-weight: 700;
  color: #9ca3af;
}

.duration {
  font-size: 11px;
  color: #9ca3af;
}

.record-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-top: 16px;
  flex-wrap: wrap;
}

.battle-time {
  font-size: 13px;
  color: #6b7280;
}

.training-meta {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

.training-meta span {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 4px 9px;
  border-radius: 999px;
  background: #eef2ff;
  color: #4f46e5;
  font-size: 12px;
  font-weight: 600;
}

.detail-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  background: #f3f4f6;
  border: none;
  color: #374151;
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.detail-btn:hover {
  background: #e5e7eb;
}

/* Pagination */
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  margin-top: 32px;
}

.page-btn {
  background: rgba(255, 255, 255, 0.2);
  border: none;
  color: white;
  padding: 10px 20px;
  border-radius: 10px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s ease;
}

.page-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.3);
}

.page-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-numbers {
  display: flex;
  gap: 8px;
}

.page-size-select {
  display: flex;
  align-items: center;
  gap: 8px;
  color: rgba(255, 255, 255, 0.88);
  font-size: 14px;
  font-weight: 700;
}

.page-size-select select {
  height: 40px;
  border: none;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.2);
  color: white;
  font-weight: 700;
  padding: 0 30px 0 12px;
}

.page-size-select option {
  color: #172033;
}

.page-number {
  width: 40px;
  height: 40px;
  background: rgba(255, 255, 255, 0.2);
  border: none;
  color: white;
  border-radius: 10px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s ease;
}

.page-number:hover {
  background: rgba(255, 255, 255, 0.3);
}

.page-number.active {
  background: white;
  color: #667eea;
}

/* Empty State */
.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: white;
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 16px;
}

.empty-state h3 {
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 8px;
}

.empty-state p {
  opacity: 0.8;
  margin-bottom: 24px;
}

.play-btn {
  background: white;
  color: #667eea;
  border: none;
  padding: 14px 32px;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
}

.play-btn:hover {
  transform: scale(1.05);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
}

/* Skeleton Loading */
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.skeleton-item {
  height: 180px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 20px;
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 20px;
}

.modal-content {
  background: white;
  border-radius: 24px;
  width: 100%;
  max-width: 420px;
  overflow: hidden;
  animation: modalIn 0.3s ease;
}

@keyframes modalIn {
  from {
    opacity: 0;
    transform: scale(0.9);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid #f3f4f6;
}

.modal-header h3 {
  font-size: 18px;
  font-weight: 600;
  color: #1f2937;
}

.modal-body {
  padding: 24px;
}

.detail-players {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
  padding: 20px;
  background: #f9fafb;
  border-radius: 16px;
}

.detail-player {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.detail-player.winner .name {
  color: #059669;
}

.detail-player.winner .score {
  color: #059669;
}

.detail-player.loser .name {
  color: #6b7280;
}

.detail-player.loser .score {
  color: #9ca3af;
}

.detail-player .name {
  font-weight: 600;
  color: #374151;
}

.detail-player .score {
  font-size: 32px;
  font-weight: 700;
  color: #1f2937;
}

.detail-avatar {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  margin-bottom: 8px;
}

.detail-avatar .avatar-text {
  color: white;
  font-size: 24px;
  font-weight: 700;
}

.detail-avatar .crown {
  position: absolute;
  top: -8px;
  right: -4px;
  font-size: 20px;
}

.detail-avatar.winner {
  background: linear-gradient(135deg, #ffd700 0%, #ffaa00 50%, #ff8c00 100%);
  box-shadow:
    0 0 20px rgba(255, 215, 0, 0.6),
    0 0 40px rgba(255, 215, 0, 0.3),
    inset 0 0 20px rgba(255, 255, 255, 0.3);
  animation: winnerGlow 1.5s ease-in-out infinite;
  border: 3px solid #ffd700;
}

.detail-avatar.winner::before {
  content: '';
  position: absolute;
  top: -5px;
  left: -5px;
  right: -5px;
  bottom: -5px;
  border-radius: 50%;
  background: conic-gradient(
    from 0deg,
    transparent 0deg,
    rgba(255, 215, 0, 0.6) 60deg,
    transparent 120deg,
    rgba(255, 215, 0, 0.6) 180deg,
    transparent 240deg,
    rgba(255, 215, 0, 0.6) 300deg,
    transparent 360deg
  );
  animation: winnerRotate 3s linear infinite;
  z-index: -1;
}

.detail-avatar.winner::after {
  content: 'WIN';
  position: absolute;
  top: -22px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 16px;
  font-weight: 900;
  color: #ffd700;
  text-shadow:
    0 0 5px rgba(255, 215, 0, 0.8),
    0 0 10px rgba(255, 215, 0, 0.6),
    0 2px 4px rgba(0, 0, 0, 0.3);
  animation: winnerPulse 0.8s ease-in-out infinite;
  letter-spacing: 1px;
}

.detail-avatar.loser {
  transform: scale(0.9);
  opacity: 0.6;
  filter: grayscale(60%);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.detail-avatar.loser::after {
  content: 'KO';
  position: absolute;
  top: -24px;
  left: 50%;
  transform: translateX(-50%) rotate(-10deg);
  font-size: 22px;
  font-weight: 900;
  color: #ff4444;
  animation: koPulse 0.8s ease-in-out infinite;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.detail-vs {
  font-size: 14px;
  font-weight: 600;
  color: #9ca3af;
}

.detail-stats {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.stat-row {
  display: flex;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px solid #f3f4f6;
}

.stat-row:last-child {
  border-bottom: none;
}

.stat-row span:first-child {
  color: #6b7280;
}

.stat-row span:last-child {
  font-weight: 600;
  color: #374151;
}

/* Transitions */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

@media (max-width: 640px) {
  .records-page {
    padding: 12px;
  }

  .title {
    font-size: 22px;
  }

  .record-card {
    padding: 14px;
  }

  .players-section {
    gap: 8px;
    flex-direction: column;
    padding: 12px 0;
  }

  .player {
    flex-direction: row !important;
    width: 100%;
    justify-content: center;
    gap: 10px;
  }

  .avatar {
    width: 36px;
    height: 36px;
  }

  .avatar-text {
    font-size: 13px;
  }

  .player-score {
    font-size: 16px;
  }

  .player-name {
    font-size: 13px;
  }

  .stats {
    align-items: flex-start !important;
    font-size: 11px;
  }

  .tab-btn {
    padding: 8px 16px;
    font-size: 13px;
  }

  .mode-tabs {
    max-width: 100%;
  }

  .mode-tab-btn {
    padding: 9px 12px;
    font-size: 13px;
  }

  .vs-divider {
    flex-direction: column;
    gap: 6px;
  }

  .detail-btn-side {
    font-size: 11px;
    padding: 5px 10px;
  }

  .vs-text {
    font-size: 16px;
  }

  .record-footer {
    font-size: 12px;
  }

  .training-meta {
    justify-content: flex-start;
  }

  /* Modal */
  .modal-content {
    width: calc(100% - 24px);
    margin: 0 12px;
    padding: 20px 16px;
  }

  .detail-players {
    gap: 10px;
  }

  .detail-avatar {
    width: 40px;
    height: 40px;
  }

  .detail-vs {
    font-size: 16px;
  }

  .crown {
    font-size: 12px;
    top: -6px;
    right: -2px;
  }

  .filter-tabs {
    gap: 8px;
  }

}
</style>
