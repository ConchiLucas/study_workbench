<template>
  <Transition name="modal">
    <div class="answer-detail-modal" v-if="visible" @click="close">
      <FullscreenCloseButton @close="close" />
      <div class="modal-content" @click.stop>
        <!-- 头部 -->
        <div class="modal-header">
          <h3 class="title">{{ playerName }}的答题详情</h3>
        </div>

        <!-- 统计摘要 -->
        <div class="summary-bar">
          <div class="summary-item">
            <span class="label">总得分</span>
            <span class="value">{{ totalScore }}</span>
          </div>
          <div class="summary-item">
            <span class="label">正确率</span>
            <span class="value">{{ accuracy }}%</span>
          </div>
          <div class="summary-item">
            <span class="label">正确/总题</span>
            <span class="value">{{ correctCount }}/{{ totalCount }}</span>
          </div>
        </div>

        <!-- 轮次指示器 -->
        <div class="round-indicator">
          <button
            v-for="n in totalRounds"
            :key="n"
            class="round-btn"
            :class="{ active: currentRound === n, correct: isRoundCorrect(n), wrong: isRoundWrong(n) }"
            @click="goToRound(n)"
          >
            {{ n }}
          </button>
        </div>

        <!-- 题目展示区 -->
        <div class="question-area" v-if="currentQuestion">
          <div class="question-header">
            <span class="round-label">第 {{ currentRound }} 题</span>
            <span class="time-used" v-if="currentQuestion.answerTimeMs">
              用时 {{ formatTime(currentQuestion.answerTimeMs) }}
            </span>
          </div>

          <div class="word-display">{{ currentQuestion.wordContent }}</div>

          <!-- 选项区 -->
          <div class="options-grid">
            <div
              v-for="opt in currentOptions"
              :key="opt.index"
              class="option-card"
              :class="{
                'user-selected': opt.isSelected,
                'correct': opt.isCorrect,
                'wrong': opt.isSelected && !opt.isCorrect
              }"
            >
              <span class="option-index">{{ opt.index }}.</span>
              <span class="option-text">{{ opt.text }}</span>
              <!-- 用户选择的标记 -->
              <div class="option-badge user-badge" v-if="opt.isSelected">
                <span v-if="opt.isCorrect" class="badge correct">✓</span>
                <span v-else class="badge wrong">✗</span>
              </div>
              <!-- 正确答案标记（用户没选对时才显示） -->
              <div class="option-badge correct-badge" v-else-if="opt.isCorrect">
                <span class="badge correct">✓</span>
              </div>
            </div>
          </div>

          <!-- 本题得分 -->
          <div class="question-score" :class="{ correct: currentQuestion.isCorrect, wrong: !currentQuestion.isCorrect }">
            <span v-if="currentQuestion.isCorrect">答对 +{{ currentQuestion.score }}分</span>
            <span v-else>答错 0分</span>
          </div>
        </div>

        <!-- 导航按钮 -->
        <div class="navigation">
          <button class="nav-btn" :disabled="currentRound === 1" @click="prevRound">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m15 18-6-6 6-6"/>
            </svg>
            上一题
          </button>
          <button class="nav-btn" :disabled="currentRound === totalRounds" @click="nextRound">
            下一题
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m9 18 6-6-6-6"/>
            </svg>
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import axios from '../api'
import FullscreenCloseButton from './FullscreenCloseButton.vue'

interface AnswerDetail {
  id: number
  recordId: number
  userId: number
  userName: string
  roundNo: number
  wordContent: string
  option1: string
  option2: string
  option3: string
  option4: string
  correctAnswerIndex: number  // 1-4
  selectedAnswerIndex: number | null  // 1-4，未答为null
  isCorrect: number
  score: number
  answerTimeMs: number
}

const props = defineProps<{
  visible: boolean
  recordId?: number
  targetUserId?: number
  playerName?: string
  initialRound?: number
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const loading = ref(false)
const answerDetails = ref<AnswerDetail[]>([])
const currentRound = ref(1)

// 计算属性
const totalRounds = computed(() => answerDetails.value.length)
const currentQuestion = computed(() => {
  return answerDetails.value.find(q => q.roundNo === currentRound.value)
})

// 所有选项，包含状态标记
const currentOptions = computed(() => {
  if (!currentQuestion.value) return []

  const q = currentQuestion.value
  const options = [
    { index: 1, text: q.option1 },
    { index: 2, text: q.option2 },
    { index: 3, text: q.option3 },
    { index: 4, text: q.option4 }
  ]

  return options.map(opt => ({
    ...opt,
    isCorrect: opt.index === q.correctAnswerIndex,
    isSelected: opt.index === q.selectedAnswerIndex
  }))
})

const totalScore = computed(() => {
  return answerDetails.value.reduce((sum, q) => sum + (q.score || 0), 0)
})

const correctCount = computed(() => {
  return answerDetails.value.filter(q => q.isCorrect === 1).length
})

const totalCount = computed(() => answerDetails.value.length)

const accuracy = computed(() => {
  if (totalCount.value === 0) return 0
  return Math.round((correctCount.value / totalCount.value) * 100)
})

// 方法
const fetchAnswerDetails = async () => {
  if (!props.recordId || !props.targetUserId) return

  loading.value = true
  try {
    const res = await axios.get('/api/game/answer-detail', {
      params: {
        recordId: props.recordId,
        targetUserId: props.targetUserId
      }
    })
    answerDetails.value = res.data || []
    // 按轮次排序
    answerDetails.value.sort((a, b) => a.roundNo - b.roundNo)
    currentRound.value = normalizeInitialRound()
  } catch (error) {
    console.error('Failed to fetch answer details:', error)
  } finally {
    loading.value = false
  }
}

const isRoundCorrect = (round: number) => {
  const q = answerDetails.value.find(q => q.roundNo === round)
  return q?.isCorrect === 1
}

const isRoundWrong = (round: number) => {
  const q = answerDetails.value.find(q => q.roundNo === round)
  return q?.isCorrect === 0
}

const goToRound = (round: number) => {
  currentRound.value = round
}

const prevRound = () => {
  if (currentRound.value > 1) {
    currentRound.value--
  }
}

const nextRound = () => {
  if (currentRound.value < totalRounds.value) {
    currentRound.value++
  }
}

const formatTime = (ms: number) => {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

const normalizeInitialRound = () => {
  const requestedRound = Number(props.initialRound || 1)
  if (!Number.isFinite(requestedRound)) return 1
  const matched = answerDetails.value.find(q => q.roundNo === requestedRound)
  return matched ? matched.roundNo : 1
}

const close = () => {
  emit('close')
}

// 监听visible变化
watch(() => props.visible, (newVal) => {
  if (newVal) {
    fetchAnswerDetails()
  } else {
    answerDetails.value = []
    currentRound.value = 1
  }
})

watch(() => props.initialRound, () => {
  if (props.visible && answerDetails.value.length > 0) {
    currentRound.value = normalizeInitialRound()
  }
})
</script>

<style scoped>
.answer-detail-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  z-index: 2000;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 20px;
  max-width: 800px;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
}

/* 头部 */
.modal-header {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
}

.title {
  color: white;
  font-size: 20px;
  font-weight: 600;
  margin: 0;
}

/* 统计摘要 */
.summary-bar {
  display: flex;
  justify-content: center;
  gap: 40px;
  background: rgba(255, 255, 255, 0.1);
  padding: 16px 24px;
  border-radius: 16px;
  margin-bottom: 16px;
}

.summary-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.summary-item .label {
  color: rgba(255, 255, 255, 0.7);
  font-size: 12px;
}

.summary-item .value {
  color: white;
  font-size: 24px;
  font-weight: 700;
}

/* 轮次指示器 */
.round-indicator {
  display: flex;
  justify-content: center;
  gap: 8px;
  margin-bottom: 20px;
  padding: 0 10px;
}

.round-btn {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.3);
  background: rgba(255, 255, 255, 0.1);
  color: white;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.round-btn:hover {
  background: rgba(255, 255, 255, 0.2);
  transform: scale(1.1);
}

.round-btn.active {
  background: white;
  color: #667eea;
  border-color: white;
  transform: scale(1.15);
}

.round-btn.correct {
  border-color: #10b981;
  background: rgba(16, 185, 129, 0.2);
}

.round-btn.correct.active {
  background: #10b981;
  color: white;
}

.round-btn.wrong {
  border-color: #ef4444;
  background: rgba(239, 68, 68, 0.2);
}

.round-btn.wrong.active {
  background: #ef4444;
  color: white;
}

/* 题目区域 */
.question-area {
  flex: 1;
  background: white;
  border-radius: 24px;
  padding: 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  margin-bottom: 16px;
}

.question-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  max-width: 500px;
}

.round-label {
  font-size: 14px;
  color: #6b7280;
  font-weight: 500;
}

.time-used {
  font-size: 13px;
  color: #9ca3af;
}

.word-display {
  font-size: 42px;
  font-weight: 700;
  color: #1f2937;
  margin: 10px 0;
}

/* 选项网格 */
.options-grid {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
  max-width: 500px;
  align-items: center;
}

.option-card {
  position: relative;
  padding: 20px 24px;
  border-radius: 16px;
  border: 2px solid #e5e7eb;
  background: #f9fafb;
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 60px;
  width: 100%;
  max-width: 400px;
  transition: all 0.2s ease;
}

.option-card.user-selected.correct {
  border-color: #10b981;
  background: #dcfce7;
}

.option-card.user-selected.wrong {
  border-color: #ef4444;
  background: #fee2e2;
}

.option-card.correct:not(.user-selected) {
  border-color: #10b981;
  background: rgba(16, 185, 129, 0.1);
}

.option-index {
  font-size: 16px;
  font-weight: 600;
  color: #9ca3af;
  min-width: 24px;
}

.option-text {
  font-size: 18px;
  font-weight: 500;
  color: #374151;
  flex: 1;
}

.option-badge {
  position: absolute;
  top: -8px;
  right: -8px;
}

.option-badge.correct-badge {
  opacity: 0.7;
}

.badge {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 700;
}

.badge.correct {
  background: #10b981;
  color: white;
}

.badge.wrong {
  background: #ef4444;
  color: white;
}

/* 题目得分 */
.question-score {
  font-size: 18px;
  font-weight: 600;
  padding: 10px 24px;
  border-radius: 20px;
}

.question-score.correct {
  color: #059669;
  background: #dcfce7;
}

.question-score.wrong {
  color: #dc2626;
  background: #fee2e2;
}

/* 导航 */
.navigation {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 20px;
}

.nav-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  background: white;
  border: none;
  color: #667eea;
  padding: 14px 24px;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.nav-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.2);
}

.nav-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* Transition */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

@media (max-width: 640px) {
  .modal-content {
    padding: 12px;
  }

  .title {
    font-size: 16px;
  }

  .summary-bar {
    gap: 20px;
    padding: 12px 16px;
  }

  .summary-item .value {
    font-size: 20px;
  }

  .word-display {
    font-size: 32px;
  }

  .options-grid {
    gap: 12px;
  }

  .option-card {
    padding: 16px 20px;
    min-height: 50px;
    max-width: 100%;
    gap: 8px;
  }

  .option-index {
    font-size: 14px;
    min-width: 20px;
  }

  .option-text {
    font-size: 16px;
  }
}

@media (max-width: 480px) {
  .modal-content { padding: 8px; }
  .title { font-size: 14px; }
  .summary-bar { gap: 12px; padding: 10px 12px; border-radius: 12px; }
  .summary-item .label { font-size: 11px; }
  .summary-item .value { font-size: 18px; }

  .round-btn { width: 30px; height: 30px; font-size: 12px; }
  .round-indicator { gap: 5px; margin-bottom: 12px; }

  .question-area { padding: 14px 10px; gap: 12px; border-radius: 16px; }
  .word-display { font-size: 26px; margin: 6px 0; }

  .option-card { padding: 12px 14px; min-height: 40px; border-radius: 10px; }
  .option-index { font-size: 13px; min-width: 18px; }
  .option-text { font-size: 14px; }

  .question-score { font-size: 14px; padding: 6px 16px; }

  .nav-btn { padding: 10px 16px; font-size: 13px; border-radius: 10px; }
  .navigation { gap: 10px; padding-bottom: 12px; }
}
</style>
