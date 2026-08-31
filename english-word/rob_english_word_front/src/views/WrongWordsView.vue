<template>
  <div class="wrong-page">
    <FullscreenCloseButton @close="closePage" />

    <header class="page-header">
      <h1>错题集</h1>
      <p>{{ total }} 个待复习单词 · 未完成复习的错词会持续显示</p>
    </header>

    <section class="toolbar">
      <label class="search-field">
        <span>搜索</span>
        <input v-model="keyword" type="search" placeholder="输入英文单词" />
      </label>

      <div class="toolbar-actions">
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

        <div v-if="total > 0" class="pagination">
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
        </div>
      </div>
    </section>

    <main class="wrong-container">
      <div v-if="loading" class="loading-list">
        <div v-for="i in 5" :key="i" class="skeleton-row"></div>
      </div>

      <div v-else-if="events.length > 0" class="event-table">
        <div class="event-head event-grid" aria-hidden="true">
          <span>来源单词</span>
          <span>例句</span>
          <span>答错时间</span>
          <span>入口 / 模式</span>
          <span>词库 / 难度</span>
          <span>词难度</span>
          <span>耗时</span>
          <span>正确答案</span>
        </div>

        <article v-for="item in events" :key="item.progressKey" class="event-row event-grid">
          <strong data-label="来源单词">{{ item.word }}</strong>
          <span
            class="example-cell"
            data-label="例句"
          >
            <span class="example-sentence" :title="item.exampleSentence || ''">
              <template v-if="item.exampleSentence">
                <template
                  v-for="(segment, index) in splitHighlightedSentence(item.exampleSentence, item.word)"
                  :key="`${item.progressKey}:example:${index}`"
                >
                  <mark v-if="segment.highlighted" class="example-highlight">{{ segment.text }}</mark>
                  <span v-else>{{ segment.text }}</span>
                </template>
              </template>
              <template v-else>—</template>
            </span>
          </span>
          <time data-label="答错时间">{{ formatDateTime(item.answeredAt) }}</time>
          <span data-label="入口 / 模式">{{ formatEntryMode(item) }}</span>
          <span data-label="词库 / 难度">{{ formatDifficulty(item) }}</span>
          <span data-label="词难度">{{ item.wordDifficulty ?? '-' }}</span>
          <span data-label="耗时">{{ formatCost(item.costMs) }}</span>
          <span class="correct-answer" data-label="正确答案">{{ item.correctAnswer || '-' }}</span>
        </article>
      </div>

      <div v-else class="empty-state">
        <h2>暂无待复习单词</h2>
        <p>答错后进入复习队列的单词会显示在这里，完成立即、7 天和 15 天复习后移出。</p>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import axios from '../api'
import FullscreenCloseButton from '../components/FullscreenCloseButton.vue'
import { useEscapeClose } from '../composables/useEscapeClose'
import { splitHighlightedSentence } from '../lib/highlightSentence'

type Sort = 'recent' | 'count'

interface WrongWordEvent {
  progressKey: string
  eventKey: string
  word: string
  answeredAt: string
  entry: string
  mode: string
  difficultyGroup: string
  difficultyLevel: string
  difficultyLabel: string
  wordDifficulty: number | null
  costMs: number | null
  correctAnswer: string | null
  exampleSentence: string | null
  exampleSource: 'word' | 'best_sentence' | 'none'
  sourceType: 'game' | 'cloze'
  occurrenceCount: number
  reviewStatus: 'waiting_sentence' | 'due' | 'waiting'
  reviewStage: number
  nextReviewTime: string | null
}

const difficultyNames: Record<string, string> = {
  primary: '小学英语',
  junior: '初中英语',
  senior: '高中英语',
  college: '大学英语',
  entrance: '升学考试',
  business_abroad: '商务留学',
  professional: '专业英语',
  advanced_exam: '高阶考试',
  cet4: '四级',
  college_cet4: '四级',
  cet6: '六级',
  college_cet6: '六级',
  postgraduate: '考研',
  postgraduate_english: '考研英语',
  gaokao: '高考',
  zhongkao: '中考',
}

const router = useRouter()
const sort = ref<Sort>('recent')
const keyword = ref('')
const debouncedKeyword = ref('')
const loading = ref(false)
const events = ref<WrongWordEvent[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const pageSizeOptions = [20, 50, 100, 500]
const sortOptions: Array<{ label: string; value: Sort }> = [
  { label: '最近错误', value: 'recent' },
  { label: '错误次数', value: 'count' },
]
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

let keywordTimer: ReturnType<typeof setTimeout> | null = null

function closePage() {
  router.push('/home')
}

useEscapeClose(closePage)

async function fetchEvents() {
  loading.value = true
  try {
    const params: Record<string, string | number> = {
      sort: sort.value,
      page: currentPage.value,
      size: pageSize.value,
    }
    if (debouncedKeyword.value) params.keyword = debouncedKeyword.value

    const response = await axios.get('/api/wrong-words/events', { params })
    events.value = response.data.items || []
    total.value = response.data.total || 0
  } catch (error) {
    console.error('Failed to fetch pending wrong-word review progress:', error)
    events.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function changePage(page: number) {
  currentPage.value = page
  fetchEvents()
}

function changePageSize(event: Event) {
  pageSize.value = Number((event.target as HTMLSelectElement).value)
  currentPage.value = 1
  fetchEvents()
}

function formatDateTime(value: string | null | undefined) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  const second = String(date.getSeconds()).padStart(2, '0')
  return `${month}/${day} ${hour}:${minute}:${second}`
}

function formatEntryMode(item: WrongWordEvent) {
  const values = [item.entry, item.mode].filter((value, index, all) => value && all.indexOf(value) === index)
  return values.join(' / ') || '-'
}

function formatDifficulty(item: WrongWordEvent) {
  const group = difficultyNames[item.difficultyGroup] || item.difficultyGroup
  const fallbackLabel = difficultyNames[item.difficultyLabel] || item.difficultyLabel
  const level = difficultyNames[item.difficultyLevel] || fallbackLabel || item.difficultyLevel
  const values = [group, level].filter((value, index, all) => value && all.indexOf(value) === index)
  return values.join(' / ') || '-'
}

function formatCost(costMs: number | null) {
  if (costMs == null || costMs < 0) return '-'
  if (costMs < 1000) return `${costMs}ms`
  return `${(costMs / 1000).toFixed(1)}s`
}

watch(keyword, (value) => {
  if (keywordTimer) clearTimeout(keywordTimer)
  keywordTimer = setTimeout(() => {
    debouncedKeyword.value = value.trim()
  }, 300)
})

watch([sort, debouncedKeyword], () => {
  currentPage.value = 1
  fetchEvents()
})

onMounted(fetchEvents)
onBeforeUnmount(() => {
  if (keywordTimer) clearTimeout(keywordTimer)
})
</script>

<style scoped>
.wrong-page {
  min-height: 100vh;
  padding: 48px 24px;
  background: #101116;
  color: #f5f7fb;
}

.page-header,
.toolbar,
.wrong-container {
  width: min(1440px, 100%);
  margin-inline: auto;
}

.page-header {
  margin-bottom: 24px;
}

.page-header h1 {
  margin: 0 0 7px;
  font-size: 32px;
}

.page-header p,
.search-field {
  color: #969bac;
}

.page-header p {
  margin: 0;
}

.toolbar {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) auto;
  gap: 16px;
  align-items: end;
  margin-bottom: 18px;
}

.search-field {
  display: grid;
  gap: 8px;
  font-size: 13px;
}

.search-field input,
.page-size-select select {
  border: 1px solid #2d3039;
  background: #15161b;
  color: #f5f7fb;
}

.search-field input {
  height: 44px;
  border-radius: 8px;
  padding: 0 14px;
  outline: none;
}

.search-field input:focus {
  border-color: #8158f4;
  box-shadow: 0 0 0 3px rgba(129, 88, 244, 0.18);
}

.toolbar-actions,
.pagination,
.page-size-select {
  display: flex;
  align-items: center;
}

.toolbar-actions {
  gap: 12px;
}

.sort-segment {
  display: flex;
  padding: 4px;
  border-radius: 8px;
  background: #191b22;
}

button,
select {
  font: inherit;
}

.sort-segment button,
.pagination button {
  border: 0;
  cursor: pointer;
}

.sort-segment button {
  height: 36px;
  padding: 0 14px;
  border-radius: 6px;
  background: transparent;
  color: #969bac;
}

.sort-segment button.active {
  background: #262936;
  color: #a98aff;
}

.pagination {
  gap: 9px;
}

.pagination button {
  height: 40px;
  padding: 0 13px;
  border-radius: 8px;
  background: #191b22;
  color: #e6e8ef;
}

.pagination button:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.pagination > span {
  min-width: 48px;
  text-align: center;
  color: #b8bcc8;
  font-weight: 700;
}

.page-size-select {
  gap: 7px;
  color: #969bac;
  font-size: 13px;
}

.page-size-select select {
  height: 38px;
  border-radius: 8px;
  padding: 0 28px 0 10px;
}

.event-table {
  overflow: hidden;
  border: 1px solid #2b2e36;
  border-radius: 10px;
}

.event-grid {
  display: grid;
  grid-template-columns:
    minmax(110px, 0.8fr) minmax(260px, 2fr) minmax(130px, 0.9fr)
    minmax(150px, 1.1fr) minmax(140px, 1fr) 72px 72px
    minmax(160px, 1.2fr);
  gap: 14px;
  align-items: center;
}

.event-head {
  min-height: 44px;
  padding: 0 16px;
  background: #191a20;
  color: #858b9c;
  font-size: 12px;
  font-weight: 700;
}

.event-row {
  min-height: 66px;
  padding: 12px 16px;
  border-top: 1px solid #2b2e36;
  background: #14151a;
  color: #d7dae3;
  font-size: 14px;
}

.event-row strong {
  color: #aa8cff;
  overflow-wrap: anywhere;
}

.event-row time {
  color: #a7abb7;
  white-space: nowrap;
}

.example-sentence {
  display: -webkit-box;
  overflow: hidden;
  color: #d7dae3;
  line-height: 1.55;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.example-highlight {
  border-radius: 3px;
  padding: 0 2px;
  background: rgba(137, 92, 246, 0.18);
  color: #b99cff;
  font-weight: 700;
}

.correct-answer {
  color: #f1f2f6;
  overflow-wrap: anywhere;
}

.loading-list {
  display: grid;
  gap: 9px;
}

.skeleton-row {
  height: 66px;
  border-radius: 8px;
  background: linear-gradient(90deg, #17191f, #242630, #17191f);
  background-size: 180% 100%;
  animation: shimmer 1.2s ease-in-out infinite;
}

.empty-state {
  padding: 72px 24px;
  border: 1px dashed #343742;
  border-radius: 10px;
  background: #14151a;
  text-align: center;
}

.empty-state h2 {
  margin: 0 0 8px;
}

.empty-state p {
  margin: 0;
  color: #858b9c;
}

@keyframes shimmer {
  to {
    background-position: -180% 0;
  }
}

@media (max-width: 1279px) {
  .toolbar {
    grid-template-columns: 1fr;
  }

  .toolbar-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .sort-segment button {
    flex: 1;
  }

  .pagination {
    flex-wrap: wrap;
  }

  .event-table {
    display: grid;
    gap: 10px;
    overflow: visible;
    border: 0;
  }

  .event-head {
    display: none;
  }

  .event-row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 13px 18px;
    border: 1px solid #2b2e36;
    border-radius: 10px;
  }

  .event-row > * {
    display: grid;
    gap: 4px;
    min-width: 0;
  }

  .event-row > *::before {
    content: attr(data-label);
    color: #747989;
    font-size: 11px;
    font-weight: 700;
  }
}

@media (max-width: 560px) {
  .wrong-page {
    padding: 32px 16px;
  }

  .event-row {
    grid-template-columns: 1fr;
  }
}
</style>
