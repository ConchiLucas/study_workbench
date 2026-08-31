<template>
  <div class="mastery-page">
    <FullscreenCloseButton @close="closePage" />
    <header class="page-header">
      <div>
        <h1>已掌握单词</h1>
        <p>难度训练按 1 天和 7 天复习确认；匹配连续 3 次答对或挖空答对会直接进入已掌握</p>
      </div>
      <button class="refresh-btn" @click="fetchWords" :disabled="loading">刷新</button>
    </header>

    <section class="summary-band">
      <article>
        <span>全部记录</span>
        <strong>{{ total }}</strong>
      </article>
      <article>
        <span>复习中</span>
        <strong>{{ learningTotal }}</strong>
      </article>
      <article>
        <span>已掌握</span>
        <strong>{{ masteredTotal }}</strong>
      </article>
    </section>

    <section class="toolbar">
      <label class="search-field">
        <span>搜索</span>
        <input v-model="keyword" type="search" placeholder="输入英文单词或中文释义" />
      </label>

      <div class="toolbar-actions">
        <div class="sort-segment" aria-label="掌握状态">
          <button
            v-for="item in statusOptions"
            :key="item.value"
            :class="{ active: status === item.value }"
            @click="status = item.value"
          >
            {{ item.label }}
          </button>
        </div>

        <div class="sort-segment" aria-label="排序方式">
          <button
            v-for="item in sortOptions"
            :key="item.value"
            :class="{ active: sort === item.value }"
            @click="sort = item.value"
          >
            {{ item.label }}
          </button>
        </div>
      </div>
    </section>

    <main class="mastery-container">
      <div v-if="loading" class="loading-list">
        <div v-for="i in 6" :key="i" class="skeleton-row"></div>
      </div>

      <div v-else-if="words.length > 0" class="word-list">
        <article v-for="word in words" :key="word.wordId" class="word-row" :class="word.status">
          <span class="status-pill" :class="{ due: isDue(word), mastered: word.status === 'mastered' }">
            {{ stageLabel(word) }}
          </span>
          <h2>{{ word.wordContent }}</h2>
          <strong class="word-meaning">{{ word.correctMeaning || '-' }}</strong>
          <span class="progress-pill">{{ progressLabel(word) }}</span>
          <span class="time-cell">{{ timeLabel(word) }}</span>
          <span class="count-cell">累计答对 {{ word.correctCount || 0 }} 次</span>
        </article>
      </div>

      <div v-else class="empty-state">
        <h2>暂无掌握记录</h2>
        <p>完成难度训练、匹配或挖空练习并答对单词后，这里会开始记录掌握情况。</p>
      </div>
    </main>

    <section v-if="total > 0" class="pagination">
      <label class="page-size-select">
        <span>每页</span>
        <select :value="pageSize" @change="changePageSize">
          <option v-for="size in pageSizeOptions" :key="size" :value="size">
            {{ size }} 条
          </option>
        </select>
      </label>
      <button :disabled="currentPage === 1" @click="changePage(currentPage - 1)">上一页</button>
      <span>{{ currentPage }} / {{ totalPages }}</span>
      <button :disabled="currentPage === totalPages" @click="changePage(currentPage + 1)">下一页</button>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import axios from '../api'
import FullscreenCloseButton from '../components/FullscreenCloseButton.vue'
import { useEscapeClose } from '../composables/useEscapeClose'

type StatusFilter = 'all' | 'learning' | 'mastered'
type Sort = 'recent' | 'due' | 'mastered'

interface MasteredWord {
  wordId: number
  wordContent: string
  correctMeaning: string | null
  status: 'learning' | 'mastered'
  stage: number
  correctCount: number
  firstCorrectTime: string | null
  day1CorrectTime: string | null
  day7CorrectTime: string | null
  nextReviewTime: string | null
  lastCorrectTime: string | null
  masteredTime: string | null
}

const router = useRouter()
const keyword = ref('')
const debouncedKeyword = ref('')
const status = ref<StatusFilter>('all')
const sort = ref<Sort>('recent')
const loading = ref(false)
const words = ref<MasteredWord[]>([])
const total = ref(0)
const learningTotal = ref(0)
const masteredTotal = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const pageSizeOptions = [20, 50, 100, 500]

function closePage() {
  router.push({ path: '/home', state: { openTrainingSetup: true } })
}

useEscapeClose(closePage)

const statusOptions: Array<{ label: string; value: StatusFilter }> = [
  { label: '全部', value: 'all' },
  { label: '复习中', value: 'learning' },
  { label: '已掌握', value: 'mastered' },
]

const sortOptions: Array<{ label: string; value: Sort }> = [
  { label: '最近答对', value: 'recent' },
  { label: '复习到期', value: 'due' },
  { label: '掌握时间', value: 'mastered' },
]

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
let keywordTimer: ReturnType<typeof setTimeout> | null = null

async function fetchWords() {
  loading.value = true
  try {
    const params: Record<string, string | number> = {
      status: status.value,
      sort: sort.value,
      page: currentPage.value,
      size: pageSize.value,
    }
    if (debouncedKeyword.value) {
      params.keyword = debouncedKeyword.value
    }

    const res = await axios.get('/api/mastered-words', { params })
    words.value = res.data.items || []
    total.value = res.data.total || 0
    learningTotal.value = res.data.learningTotal || 0
    masteredTotal.value = res.data.masteredTotal || 0
  } catch (error) {
    console.error('Failed to fetch mastered words:', error)
  } finally {
    loading.value = false
  }
}

function changePage(page: number) {
  currentPage.value = page
  fetchWords()
}

function changePageSize(event: Event) {
  pageSize.value = Number((event.target as HTMLSelectElement).value)
  currentPage.value = 1
  fetchWords()
}

function isDue(word: MasteredWord) {
  if (word.status === 'mastered' || !word.nextReviewTime) return false
  const dueTime = new Date(word.nextReviewTime).getTime()
  return !Number.isNaN(dueTime) && dueTime <= Date.now()
}

function stageLabel(word: MasteredWord) {
  if (word.status === 'mastered') return '已掌握'
  if (!word.nextReviewTime && word.stage > 0) return `连续答对 ${Math.min(word.stage, 2)}/3`
  if (word.stage === 2) return isDue(word) ? '7天复习到期' : '等待7天复习'
  if (word.stage === 1) return isDue(word) ? '1天复习到期' : '等待1天复习'
  return '需重新开始'
}

function progressLabel(word: MasteredWord) {
  if (word.status === 'mastered') return '3/3'
  if (word.stage === 2) return '2/3'
  if (word.stage === 1) return '1/3'
  return '0/3'
}

function timeLabel(word: MasteredWord) {
  if (word.status === 'mastered') {
    return `掌握 ${formatDateTime(word.masteredTime)}`
  }
  if (word.nextReviewTime) {
    return `${isDue(word) ? '应复习' : '下次复习'} ${formatDateTime(word.nextReviewTime)}`
  }
  return `最近答对 ${formatDateTime(word.lastCorrectTime)}`
}

function formatDateTime(value: string | null | undefined) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return `${date.getMonth() + 1}月${date.getDate()}日 ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

watch(keyword, (value) => {
  if (keywordTimer) clearTimeout(keywordTimer)
  keywordTimer = setTimeout(() => {
    debouncedKeyword.value = value.trim()
    currentPage.value = 1
    fetchWords()
  }, 250)
})

watch([status, sort], () => {
  currentPage.value = 1
  fetchWords()
})

onMounted(fetchWords)

onBeforeUnmount(() => {
  if (keywordTimer) clearTimeout(keywordTimer)
})
</script>

<style scoped>
.mastery-page {
  min-height: 100vh;
  padding: 32px;
  background: linear-gradient(135deg, #17283a 0%, #202347 100%);
  color: white;
}

.page-header,
.summary-band,
.toolbar,
.mastery-container,
.pagination {
  width: min(1120px, 100%);
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 24px;
}

.page-header h1 {
  margin: 0;
  font-size: 42px;
}

.page-header p {
  margin: 8px 0 0;
  color: rgba(255, 255, 255, 0.72);
  font-size: 16px;
}

.refresh-btn {
  min-width: 98px;
  padding: 14px 20px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.06);
  color: white;
  cursor: pointer;
  font-size: 16px;
  font-weight: 700;
}

.summary-band {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 18px;
}

.summary-band article {
  display: grid;
  gap: 8px;
  padding: 18px 22px;
  border: 1px solid rgba(255, 255, 255, 0.13);
  border-radius: 14px;
  background: rgba(8, 13, 27, 0.36);
}

.summary-band span,
.search-field span,
.page-size-select span {
  color: rgba(255, 255, 255, 0.58);
  font-size: 13px;
  font-weight: 700;
}

.summary-band strong {
  font-size: 32px;
}

.toolbar {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) auto;
  gap: 14px;
  align-items: end;
  margin-bottom: 18px;
}

.search-field {
  display: grid;
  gap: 8px;
}

.search-field input,
.page-size-select select {
  height: 44px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 12px;
  background: rgba(6, 10, 20, 0.42);
  color: white;
  outline: none;
}

.search-field input {
  padding: 0 14px;
}

.toolbar-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.sort-segment {
  display: inline-flex;
  padding: 4px;
  border: 1px solid rgba(255, 255, 255, 0.13);
  border-radius: 12px;
  background: rgba(8, 13, 27, 0.38);
}

.sort-segment button {
  min-width: 72px;
  padding: 9px 12px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: rgba(255, 255, 255, 0.72);
  cursor: pointer;
  font-weight: 800;
}

.sort-segment button.active {
  background: rgba(34, 197, 94, 0.22);
  color: #bbf7d0;
}

.mastery-container {
  padding: 18px;
  border: 1px solid rgba(255, 255, 255, 0.13);
  border-radius: 18px;
  background: rgba(7, 11, 23, 0.46);
}

.word-list,
.loading-list {
  display: grid;
  gap: 12px;
}

.word-row {
  display: grid;
  grid-template-columns: 132px minmax(160px, 1fr) minmax(220px, 1.2fr) 72px minmax(180px, 0.9fr) 130px;
  gap: 14px;
  align-items: center;
  min-height: 76px;
  padding: 14px 18px;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-left: 5px solid #38bdf8;
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.7);
}

.word-row.mastered {
  border-left-color: #22c55e;
}

.word-row h2 {
  margin: 0;
  font-size: 22px;
}

.word-meaning,
.time-cell,
.count-cell {
  min-width: 0;
  overflow: hidden;
  color: rgba(255, 255, 255, 0.78);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-pill,
.progress-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: fit-content;
  min-width: 82px;
  min-height: 30px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(14, 165, 233, 0.2);
  color: #bae6fd;
  font-weight: 900;
}

.status-pill.due {
  background: rgba(245, 158, 11, 0.2);
  color: #fde68a;
}

.status-pill.mastered {
  background: rgba(34, 197, 94, 0.2);
  color: #bbf7d0;
}

.progress-pill {
  min-width: 54px;
  background: rgba(148, 163, 184, 0.18);
  color: rgba(255, 255, 255, 0.78);
}

.skeleton-row {
  height: 76px;
  border-radius: 12px;
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.05), rgba(255, 255, 255, 0.12), rgba(255, 255, 255, 0.05));
  background-size: 220% 100%;
  animation: shimmer 1.2s infinite;
}

.empty-state {
  display: grid;
  place-items: center;
  min-height: 260px;
  text-align: center;
}

.empty-state h2 {
  margin: 0;
  font-size: 28px;
}

.empty-state p {
  color: rgba(255, 255, 255, 0.62);
}

.pagination {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 10px;
  margin-top: 16px;
}

.pagination button,
.pagination select {
  min-height: 38px;
  padding: 0 12px;
  border: 1px solid rgba(255, 255, 255, 0.13);
  border-radius: 10px;
  background: rgba(8, 13, 27, 0.42);
  color: white;
}

.page-size-select {
  display: flex;
  align-items: center;
  gap: 8px;
}

@keyframes shimmer {
  0% { background-position: 120% 0; }
  100% { background-position: -120% 0; }
}

@media (max-width: 820px) {
  .mastery-page {
    padding: 18px;
  }

  .page-header,
  .toolbar,
  .summary-band,
  .word-row {
    grid-template-columns: 1fr;
  }

  .page-header {
    align-items: stretch;
  }

  .toolbar-actions {
    justify-content: flex-start;
  }

  .pagination {
    justify-content: space-between;
  }
}
</style>
