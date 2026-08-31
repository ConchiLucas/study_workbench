<template>
  <div class="results-page">
    <FullscreenCloseButton @close="closePage" />
    <header class="results-header">
      <div class="header-copy">
        <h1>答题结果</h1>
        <p>今天所有难度训练轮次</p>
      </div>
      <button class="refresh-btn" @click="fetchTodayResults" :disabled="loading">
        刷新
      </button>
    </header>

    <main class="results-main">
      <section class="summary-band" v-if="rounds.length > 0">
        <div class="summary-item">
          <span>训练轮次</span>
          <strong>{{ rounds.length }}</strong>
        </div>
        <div class="summary-item">
          <span>答题数</span>
          <strong>{{ totalCount }}</strong>
        </div>
        <div class="summary-item">
          <span>正确</span>
          <strong>{{ correctCount }}</strong>
        </div>
        <div class="summary-item">
          <span>准确率</span>
          <strong>{{ accuracy }}%</strong>
        </div>
        <div class="summary-item">
          <span>总得分</span>
          <strong>{{ totalScore }}</strong>
        </div>
      </section>

      <section v-if="loading" class="state-panel">
        加载中...
      </section>

      <section v-else-if="rounds.length === 0" class="state-panel">
        <h2>今天暂无答题结果</h2>
        <p>完成难度训练后，这里会按轮次显示每轮的单词答题情况。</p>
      </section>

      <section v-else class="round-list">
        <article
          v-for="round in rounds"
          :key="round.record.id"
          class="round-card"
          :class="{ correct: round.wrongCount === 0, wrong: round.wrongCount > 0, expanded: expandedRounds.has(round.record.id) }"
        >
          <button class="round-summary" @click="toggleRound(round.record.id)">
            <span class="round-index">第 {{ round.index }} 轮</span>
            <span class="round-title">{{ formatTime(round.record.startTime) }}</span>
            <span class="metric-cell difficulty-cell">{{ difficultyLabel(round.record) }}</span>
            <span class="metric-cell">单词 {{ round.details.length }}</span>
            <span class="metric-cell">正确 {{ round.correctCount }}/{{ round.details.length }}</span>
            <span class="metric-cell">准确率 {{ round.accuracy }}%</span>
            <span class="metric-cell">得分 {{ round.score }}</span>
            <span class="metric-cell">耗时 {{ formatDuration(round.record.durationSeconds) }}</span>
            <span class="expand-cell">{{ expandedRounds.has(round.record.id) ? '收起' : '展开' }}</span>
          </button>

          <div class="round-detail" v-if="expandedRounds.has(round.record.id)">
            <div class="word-grid word-head">
              <span>序号</span>
              <span>单词</span>
              <span>难度</span>
              <span>结果</span>
              <span>得分</span>
              <span>耗时</span>
              <span>正确答案</span>
            </div>
            <div
              v-for="(detail, index) in round.details"
              :key="detail.id"
              class="word-grid word-row"
              :class="{ correct: detail.isCorrect === 1, wrong: detail.isCorrect !== 1 }"
            >
              <span>{{ index + 1 }}</span>
              <strong>{{ detail.wordContent }}</strong>
              <span>{{ detail.wordDifficulty || '-' }}</span>
              <span>{{ detail.isCorrect === 1 ? '答对' : '答错' }}</span>
              <span>{{ detail.score || 0 }}</span>
              <span>{{ formatAnswerTime(detail.answerTimeMs) }}</span>
              <span>{{ correctAnswerText(detail) }}</span>
            </div>
          </div>
        </article>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'
import { useAuthStore } from '../stores/auth'
import FullscreenCloseButton from '../components/FullscreenCloseButton.vue'
import { useEscapeClose } from '../composables/useEscapeClose'

interface GameRecord {
  id: number
  startTime?: string
  durationSeconds?: number
  trainingDifficultyGroup?: string
  trainingDifficultyLevel?: string
}

interface RecordsResponse {
  records?: GameRecord[]
  pages?: number
}

interface AnswerDetail {
  id: number
  recordId: number
  userId: number
  userName: string
  roundNo: number
  wordContent: string
  wordDifficulty: number
  option1: string
  option2: string
  option3: string
  option4: string
  correctAnswerIndex: number
  selectedAnswerIndex: number | null
  isCorrect: number
  score: number
  answerTimeMs: number
}

interface TrainingRound {
  index: number
  record: GameRecord
  details: AnswerDetail[]
  correctCount: number
  wrongCount: number
  score: number
  accuracy: number
}

const router = useRouter()
const authStore = useAuthStore()
const loading = ref(false)
const rounds = ref<TrainingRound[]>([])
const expandedRounds = ref<Set<number>>(new Set())

function closePage() {
  router.push({ path: '/home', state: { openTrainingSetup: true } })
}

useEscapeClose(closePage)

const totalCount = computed(() => rounds.value.reduce((sum, round) => sum + round.details.length, 0))
const correctCount = computed(() => rounds.value.reduce((sum, round) => sum + round.correctCount, 0))
const totalScore = computed(() => rounds.value.reduce((sum, round) => sum + round.score, 0))
const accuracy = computed(() => totalCount.value ? Math.round(correctCount.value / totalCount.value * 100) : 0)
const difficultyGroups: Record<string, string> = {
  rank: '段位难度',
  primary: '小学英语',
  junior: '初中英语',
  senior: '高中英语',
  college: '大学英语',
  entrance: '升学考试英语',
  business_abroad: '商务与出国英语',
  professional: '专业英语',
  advanced_exam: '高阶考试英语'
}
const difficultyLevels: Record<string, string> = {
  rank_current: '段位难度',
  primary_3_1: '3年级上册',
  primary_3_2: '3年级下册',
  primary_4_1: '4年级上册',
  primary_4_2: '4年级下册',
  primary_5_1: '5年级上册',
  primary_5_2: '5年级下册',
  primary_6_1: '6年级上册',
  primary_6_2: '6年级下册',
  junior_7_1: '7年级上册',
  junior_7_2: '7年级下册',
  junior_8_1: '8年级上册',
  junior_8_2: '8年级下册',
  junior_9_1: '9年级上册',
  senior_1: '上册',
  senior_2: '下册',
  senior_3: '第3册',
  senior_4: '第4册',
  senior_5: '第5册',
  senior_6: '第6册',
  senior_7: '第7册',
  senior_8: '第8册',
  senior_9: '第9册',
  senior_10: '第10册',
  senior_11: '第11册',
  college_cet4: '四级',
  college_cet6: '六级',
  entrance_kaoyan: '考研',
  business_bec: 'BEC',
  business_ielts: '雅思',
  business_toefl: '托福',
  business_gmat: 'GMAT',
  professional_tem4: '专四级',
  professional_tem8: '专八级',
  advanced_gre: 'GRE',
  advanced_sat: 'SAT'
}

async function fetchTodayResults() {
  loading.value = true
  try {
    if (!authStore.userId) {
      await authStore.fetchUserInfo()
    }

    const records = await fetchAllTodayRecords()

    const roundResults = await Promise.all(records.map(async (record) => {
      const detailRes = await api.get('/api/game/answer-detail', {
        params: {
          recordId: record.id,
          targetUserId: authStore.userId
        }
      })
      const details: AnswerDetail[] = (detailRes.data || [])
        .sort((left: AnswerDetail, right: AnswerDetail) => left.roundNo - right.roundNo)
      const correct = details.filter(item => item.isCorrect === 1).length
      const score = details.reduce((sum, item) => sum + (item.score || 0), 0)
      return {
        index: 0,
        record,
        details,
        correctCount: correct,
        wrongCount: details.length - correct,
        score,
        accuracy: details.length ? Math.round(correct / details.length * 100) : 0
      }
    }))

    const visibleRounds = roundResults
      .filter(round => round.details.length > 0)
      .map((round, index) => ({ ...round, index: index + 1 }))

    rounds.value = visibleRounds
    expandedRounds.value = visibleRounds.length > 0 ? new Set([visibleRounds[0].record.id]) : new Set()
  } catch (error) {
    console.error('Failed to fetch training answer results:', error)
  } finally {
    loading.value = false
  }
}

async function fetchAllTodayRecords() {
  const allTodayRecords: GameRecord[] = []
  const pageSize = 100
  let currentPage = 1
  let totalPages = 1

  while (currentPage <= totalPages) {
    const recordsRes = await api.get<RecordsResponse>('/api/game/records', {
      params: { mode: 'solo_training', page: currentPage, size: pageSize }
    })
    const pageRecords = recordsRes.data?.records || []
    allTodayRecords.push(...pageRecords.filter(isTodayRecord))

    totalPages = recordsRes.data?.pages || 1
    if (pageRecords.some(record => record.startTime && isBeforeToday(record.startTime))) {
      break
    }
    currentPage += 1
  }

  return allTodayRecords
}

function isTodayRecord(record: GameRecord) {
  if (!record.startTime) return false
  const date = new Date(record.startTime)
  const now = new Date()
  return date.getFullYear() === now.getFullYear()
    && date.getMonth() === now.getMonth()
    && date.getDate() === now.getDate()
}

function isBeforeToday(value: string) {
  const date = new Date(value)
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return date < today
}

function toggleRound(recordId: number) {
  const next = new Set(expandedRounds.value)
  if (next.has(recordId)) {
    next.delete(recordId)
  } else {
    next.add(recordId)
  }
  expandedRounds.value = next
}

function correctAnswerText(detail: AnswerDetail) {
  return optionText(detail, detail.correctAnswerIndex)
}

function optionText(detail: AnswerDetail, index: number) {
  const options: Record<number, string> = {
    1: detail.option1,
    2: detail.option2,
    3: detail.option3,
    4: detail.option4
  }
  return options[index] || '-'
}

function formatAnswerTime(ms?: number) {
  if (!ms) return '-'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function formatDuration(seconds?: number) {
  if (!seconds) return '-'
  const minutes = Math.floor(seconds / 60)
  const rest = seconds % 60
  if (minutes <= 0) return `${rest}s`
  return `${minutes}m ${rest}s`
}

function formatTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day} ${hour}:${minute}`
}

function difficultyLabel(record: GameRecord) {
  const groupKey = record.trainingDifficultyGroup || ''
  const levelKey = record.trainingDifficultyLevel || ''
  if (levelKey === 'rank_current') return '段位难度'

  const groupLabel = difficultyGroups[groupKey]
  const levelLabel = difficultyLevels[levelKey]
  if (groupLabel && levelLabel) return `${groupLabel} · ${levelLabel}`
  if (levelLabel) return levelLabel
  if (groupLabel) return groupLabel
  return '段位难度'
}

onMounted(fetchTodayResults)
</script>

<style scoped>
.results-page {
  min-height: 100vh;
  padding: 28px;
  color: white;
  background:
    radial-gradient(circle at 18% 14%, rgba(20, 184, 166, 0.15), transparent 34%),
    radial-gradient(circle at 82% 24%, rgba(124, 58, 237, 0.14), transparent 30%),
    linear-gradient(135deg, #172032 0%, #192044 100%);
}

.results-header {
  min-height: 72px;
  display: grid;
  grid-template-columns: 130px 1fr 130px;
  align-items: start;
  gap: 18px;
}

.header-copy {
  text-align: center;
}

.header-copy h1 {
  margin: 0;
  font-size: 36px;
  line-height: 1.15;
}

.header-copy p {
  margin: 8px 0 0;
  color: rgba(255, 255, 255, 0.68);
  font-size: 15px;
}

.refresh-btn {
  padding: 12px 24px;
  color: white;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.16);
  border-radius: 10px;
  cursor: pointer;
  font-size: 16px;
}

.refresh-btn {
  justify-self: end;
}

.refresh-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.14);
}

.refresh-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.results-main {
  width: 100%;
  padding-top: 24px;
}

.summary-band {
  width: 100%;
  margin-bottom: 22px;
  padding: 22px 28px;
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 16px;
  background: rgba(15, 23, 42, 0.42);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 16px;
}

.summary-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  text-align: center;
}

.summary-item span {
  color: rgba(255, 255, 255, 0.62);
  font-size: 14px;
}

.summary-item strong {
  font-size: 28px;
}

.state-panel {
  width: 100%;
  padding: 80px 24px;
  text-align: center;
  background: rgba(15, 23, 42, 0.42);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 16px;
}

.state-panel h2 {
  margin: 0 0 10px;
}

.state-panel p {
  margin: 0;
  color: rgba(255, 255, 255, 0.66);
}

.round-list {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.round-card {
  width: 100%;
  overflow: hidden;
  background: rgba(15, 23, 42, 0.52);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-left: 5px solid #64748b;
  border-radius: 14px;
}

.round-card.correct {
  border-left-color: #22c55e;
}

.round-card.wrong {
  border-left-color: #ef4444;
}

.round-card.expanded {
  background: rgba(15, 23, 42, 0.68);
}

.round-summary {
  width: 100%;
  padding: 20px 24px;
  display: grid;
  grid-template-columns: 110px minmax(220px, 1.1fr) minmax(180px, 1fr) repeat(5, minmax(100px, 0.65fr)) 80px;
  align-items: center;
  gap: 16px;
  color: white;
  background: transparent;
  border: 0;
  cursor: pointer;
  text-align: left;
}

.round-index {
  color: #93c5fd;
  font-weight: 800;
}

.round-title {
  font-size: 22px;
  font-weight: 800;
}

.metric-cell,
.expand-cell {
  color: rgba(255, 255, 255, 0.76);
  font-weight: 700;
  text-align: center;
}

.expand-cell {
  color: #a5b4fc;
}

.difficulty-cell {
  color: rgba(226, 232, 240, 0.92);
}

.round-detail {
  padding: 0 24px 22px;
}

.word-grid {
  display: grid;
  grid-template-columns: 70px minmax(160px, 1.2fr) 90px 90px 90px 100px minmax(180px, 1fr);
  gap: 14px;
  align-items: center;
}

.word-head {
  padding: 12px 18px;
  color: rgba(255, 255, 255, 0.58);
  font-size: 13px;
}

.word-row {
  min-height: 58px;
  padding: 14px 18px;
  margin-top: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
}

.word-row.correct {
  background: rgba(22, 101, 52, 0.34);
  border-color: rgba(34, 197, 94, 0.76);
  box-shadow: inset 0 0 0 1px rgba(34, 197, 94, 0.16);
}

.word-row.wrong {
  background: rgba(127, 29, 29, 0.38);
  border-color: rgba(239, 68, 68, 0.78);
  box-shadow: inset 0 0 0 1px rgba(239, 68, 68, 0.16);
}

.word-row strong {
  font-size: 20px;
  overflow-wrap: anywhere;
}

@media (max-width: 900px) {
  .results-page {
    padding: 16px;
  }

  .results-header {
    grid-template-columns: 1fr;
  }

  .header-copy {
    text-align: left;
  }

  .refresh-btn {
    justify-self: start;
  }

  .summary-band {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .round-summary,
  .word-grid {
    grid-template-columns: 1fr;
  }

  .metric-cell,
  .expand-cell {
    text-align: left;
  }
}
</style>
