<template>
  <div class="game-container">
    <div class="game-header">
      <!-- 抢词阶段只显示倒计时和退出按钮 -->
      <template v-if="phase === 'grab'">
        <div class="header-left"></div>
        <div class="timer">{{ timeLeft }}s</div>
        <div class="header-right">
          <span class="nickname">{{ authStore.nickname }}</span>
          <button class="exit-btn" @click="goHome">退出</button>
        </div>
      </template>
      <!-- 答题阶段显示题目倒计时 -->
      <template v-else>
        <div class="header-left"></div>
        <div class="timer">{{ questionTimeLeft }}s</div>
        <div class="header-right">
          <span class="nickname">{{ authStore.nickname }}</span>
          <button class="exit-btn" @click="goHome">退出</button>
        </div>
      </template>
    </div>

    <div class="game-area">
      <!-- 抢单词阶段 -->
      <template v-if="phase === 'grab'">
        <!-- 自己抢到的单词 -->
        <div class="player-words-row">
          <span class="player-name">{{ authStore.nickname }}</span>
          <div class="player-words">
            <div v-for="word in myWords" :key="word.id" class="word-card grabbed">
              {{ word.word }}
            </div>
          </div>
          <span class="grab-count" :class="{ full: myWords.length >= maxWordsPerPlayer }">{{ myWords.length }}/{{ maxWordsPerPlayer }}</span>
        </div>

        <!-- 对手抢到的单词 -->
        <div class="player-words-row opponent">
          <span class="player-name">{{ opponent?.nickname || '等待对手' }}</span>
          <div class="player-words">
            <div v-for="word in opponentWords" :key="word.id" class="word-card grabbed opponent">
              {{ word.word }}
            </div>
          </div>
        </div>

        <div class="all-words">
          <div
            v-for="word in availableWords"
            :key="word.id"
            class="word-card"
            :class="[
              word.grabbed ? '' : getWordTierClass(word.difficulty),
              { grabbed: word.grabbed, 'opponent-grabbed': word.grabbed && word.grabbedBy !== authStore.userId, disabled: !word.grabbed && myWords.length >= maxWordsPerPlayer }
            ]"
            @click="!word.grabbed && myWords.length < maxWordsPerPlayer && grabWord(word.id)"
          >
            <template v-if="!word.grabbed">
              <div class="word-text">{{ word.word }}</div>
              <div class="word-difficulty">难度：{{ word.difficulty }}</div>
            </template>
          </div>
        </div>
      </template>

      <!-- 答题阶段（纯渲染后端推送的题目） -->
      <template v-if="phase === 'answer'">
        <div class="answer-area">
          <!-- 题号 -->
          <div class="progress">{{ questionIndex + 1 }} / {{ questionTotal }}</div>

          <div class="question-row">
            <!-- 自己信息 -->
            <div class="side-info left">
              <span class="side-nickname">{{ authStore.nickname }}</span>
              <span class="side-progress">{{ myCorrectCount }}/{{ myAnsweredCount }}</span>
              <span class="side-score">{{ myScore }}分</span>
            </div>

            <!-- 中间：题目 -->
            <div class="question-center">
              <div class="question-timer">{{ questionTimeLeft }}s</div>
              <div class="question-word">{{ questionWord }}</div>
            </div>

            <!-- 对手信息 -->
            <div class="side-info right">
              <span class="side-nickname">{{ opponent?.nickname || '对手' }}</span>
              <span class="side-progress">{{ opponentCorrectCount }}/{{ opponentAnsweredCount }}</span>
              <span class="side-score">{{ opponentScore }}分</span>
            </div>
          </div>

          <div class="options">
            <button
              v-for="(option, index) in questionOptions"
              :key="index"
              class="option-btn"
              :class="{
                selected: selectedOptionIndex === index + 1,
                correct: showResult && correctOptionIndex === index + 1,
                wrong: showResult && selectedOptionIndex === index + 1 && selectedOptionIndex !== correctOptionIndex
              }"
              @click="submitAnswer(index + 1)"
              :disabled="showResult || answerSubmitted"
            >
              {{ option }}
            </button>
          </div>
          <div v-if="showResult" class="result-feedback">
            <span v-if="lastAnswerCorrect" class="correct-text">✓ 正确!</span>
            <span v-else-if="lastAnswerTimeout" class="wrong-text">✗ 超时! 正确答案是: {{ questionOptions[correctOptionIndex - 1] }}</span>
            <span v-else class="wrong-text">✗ 错误! 正确答案是: {{ questionOptions[correctOptionIndex - 1] }}</span>
          </div>
        </div>
      </template>
    </div>

    <!-- 等待对手完成 -->
    <div class="result-overlay" v-if="waitingForOpponent && !gameOver">
      <div class="result-box waiting">
        <h2>等待对手完成...</h2>
        <div class="waiting-spinner"></div>
        <p>你的得分: {{ myScore }}分</p>
      </div>
    </div>

    <!-- 结算界面 -->
    <div class="result-overlay" v-if="gameOver">
      <FullscreenCloseButton @close="goHome" />
      <div class="result-box">
        <div class="result-header">
          <h2 :class="resultClass">{{ resultText }}</h2>
        </div>

        <div class="battle-result">
          <div class="player-card" :class="{ 'winner': myScore > opponentScore, 'loser': myScore < opponentScore }">
            <div class="player-avatar" :class="{ 'winner': myScore > opponentScore, 'loser': myScore < opponentScore }">
              <span class="avatar-icon">👤</span>
              <div v-if="myScore > opponentScore" class="crown">👑</div>
            </div>
            <div class="player-name">{{ authStore.nickname }}</div>
            <div class="player-score">{{ myScore }}</div>
            <div class="player-accuracy">{{ myCorrectCount }}/{{ myAnsweredCount }} 正确</div>
          </div>

          <div class="vs-divider"><span class="vs-text">VS</span></div>

          <div class="player-card" :class="{ 'winner': opponentScore > myScore, 'loser': opponentScore < myScore }">
            <div class="player-avatar" :class="{ 'winner': opponentScore > myScore, 'loser': opponentScore < myScore }">
              <span class="avatar-icon">👤</span>
              <div v-if="opponentScore > myScore" class="crown">👑</div>
            </div>
            <div class="player-name">{{ opponent?.nickname || '对手' }}</div>
            <div class="player-score">{{ opponentScore }}</div>
            <div class="player-accuracy">{{ opponentCorrectCount }}/{{ opponentAnsweredCount }} 正确</div>
          </div>
        </div>

        <div v-if="mode === 'solo_training'" class="training-reward">
          <span>训练经验 +{{ trainingExpChange }}</span>
          <span>训练等级 {{ trainingRank }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useWebSocketStore } from '../stores/websocket'
import type { Word } from '../types/word'
import FullscreenCloseButton from '../components/FullscreenCloseButton.vue'
import { useEscapeClose } from '../composables/useEscapeClose'

const router = useRouter()
const authStore = useAuthStore()
const wsStore = useWebSocketStore()

// ==================== 状态（全部由后端驱动） ====================
const phase = ref<'grab' | 'answer'>('grab')
const timeLeft = ref(6)  // 抢词倒计时（后端推送）
const myWords = ref<Word[]>([])
const opponentWords = ref<Word[]>([])
const availableWords = ref<Word[]>([])
const myScore = ref(0)
const opponentScore = ref(0)
const opponent = ref<any>(null)
const gameOver = ref(false)
const waitingForOpponent = ref(false)
const roomId = ref<number | null>(null)
const maxWordsPerPlayer = ref(5)
const mode = ref<'match' | 'solo_training'>('match')
const trainingExpChange = ref(0)
const trainingExp = ref(0)
const trainingRank = ref(1)
const robotProfile = ref<any>(null)

// 答题状态（后端推送）
const questionWord = ref('')
const questionOptions = ref<string[]>([])
const questionTimeLeft = ref(4)
const questionIndex = ref(0)
const questionTotal = ref(0)
const selectedOptionIndex = ref(0)
const correctOptionIndex = ref(0)
const showResult = ref(false)
const lastAnswerCorrect = ref(false)
const lastAnswerTimeout = ref(false)
const answerSubmitted = ref(false)

// 进度统计（后端推送）
const myCorrectCount = ref(0)
const myAnsweredCount = ref(0)
const opponentCorrectCount = ref(0)
const opponentAnsweredCount = ref(0)

let questionCountdown: number | null = null  // 仅用于前端倒计时显示

function getWordTierClass(difficulty: number) {
  if (difficulty >= 801) return 'tier-legendary' // 金卡/传世
  if (difficulty >= 601) return 'tier-epic'      // 紫卡/史诗
  if (difficulty >= 401) return 'tier-rare'      // 蓝卡/稀有
  if (difficulty >= 201) return 'tier-uncommon'  // 绿卡/优秀
  return 'tier-common'                           // 白卡/普通
}

/**
 * 消息处理 — 前端只做渲染，不做状态决策
 */
function handleMessage(message: any) {
  console.log('GameView received:', message.type, message.data)

  switch (message.type) {
    case 'grab_result':
      // 抢词结果（后端判定）
      if (message.data.success) {
        const word = message.data.word
        const wordIndex = availableWords.value.findIndex(w => w.id === word.id)
        if (wordIndex !== -1) {
          availableWords.value[wordIndex].grabbed = true
          availableWords.value[wordIndex].grabbedBy = message.data.grabbedBy
        }
        if (message.data.grabbedBy === authStore.userId) {
          myWords.value.push(word)
        } else {
          opponentWords.value.push(word)
        }
      }
      break

    case 'grab_error':
      // 抢词失败
      break

    case 'grab_time_update':
      // 后端推送的抢词倒计时
      timeLeft.value = message.data.timeLeft
      break

    case 'grab_phase_end':
      // 抢词结束，后端分配完单词
      myWords.value = message.data.myWords || []
      phase.value = 'answer'
      break

    case 'state_change':
      console.log('State changed to:', message.data.state)
      break

    case 'next_question':
      renderQuestion(message.data)
      break

    case 'answer_result':
      renderAnswerResult(message.data)
      break

    case 'question_timeout':
      renderTimeout(message.data)
      break

    case 'opponent_progress':
      opponentCorrectCount.value = message.data.correctCount || 0
      opponentAnsweredCount.value = message.data.answeredCount || 0
      opponentScore.value = message.data.score || 0
      break

    case 'opponent_finished':
      opponentScore.value = message.data.score || opponentScore.value
      break

    case 'game_settlement':
      renderSettlement(message.data)
      break

    case 'game_resume':
      renderResume(message.data)
      break

    case 'error':
      console.error('Server error:', message.data)
      break

    default:
      break
  }
}

/**
 * 渲染后端推送的题目
 */
function renderQuestion(data: any) {
  questionWord.value = data.word
  questionOptions.value = data.options
  questionTimeLeft.value = data.timeLeft
  questionIndex.value = data.index
  questionTotal.value = data.total
  selectedOptionIndex.value = 0
  correctOptionIndex.value = 0
  showResult.value = false
  lastAnswerCorrect.value = false
  lastAnswerTimeout.value = false
  answerSubmitted.value = false
  waitingForOpponent.value = false

  startDisplayCountdown(data.timeLeft)
}

/**
 * 渲染答题结果
 */
function renderAnswerResult(data: any) {
  stopDisplayCountdown()
  selectedOptionIndex.value = data.selectedIndex
  correctOptionIndex.value = data.correctIndex
  lastAnswerCorrect.value = data.correct
  lastAnswerTimeout.value = false
  showResult.value = true
  myScore.value = data.totalScore
  if (data.correct) {
    myCorrectCount.value++
  }
  myAnsweredCount.value++

  // 如果这是最后一题，延迟1秒后显示等待条
  if (myAnsweredCount.value >= questionTotal.value) {
    setTimeout(() => {
      if (!gameOver.value) {
        waitingForOpponent.value = true
      }
    }, 1000)
  }
}

/**
 * 渲染超时结果
 */
function renderTimeout(data: any) {
  stopDisplayCountdown()
  correctOptionIndex.value = data.correctIndex
  selectedOptionIndex.value = 0
  lastAnswerCorrect.value = false
  lastAnswerTimeout.value = true
  showResult.value = true
  myScore.value = data.totalScore
  myAnsweredCount.value++

  // 如果这是最后一题，延迟1秒后显示等待条
  if (myAnsweredCount.value >= questionTotal.value) {
    setTimeout(() => {
      if (!gameOver.value) {
        waitingForOpponent.value = true
      }
    }, 1000)
  }
}

/**
 * 渲染结算结果
 */
function renderSettlement(data: any) {
  stopDisplayCountdown()
  mode.value = data.mode === 'solo_training' ? 'solo_training' : mode.value
  trainingExpChange.value = data.trainingExpChange || 0
  trainingExp.value = data.trainingExp || trainingExp.value
  trainingRank.value = data.trainingRank || trainingRank.value
  robotProfile.value = data.robotProfile || robotProfile.value
  const p1 = data.player1
  const p2 = data.player2
  if (p1.userId === authStore.userId) {
    myScore.value = p1.score
    myCorrectCount.value = p1.correctCount
    myAnsweredCount.value = p1.totalCount
    opponentScore.value = p2.score
    opponentCorrectCount.value = p2.correctCount
    opponentAnsweredCount.value = p2.totalCount
    opponent.value = { nickname: p2.nickname }
  } else {
    myScore.value = p2.score
    myCorrectCount.value = p2.correctCount
    myAnsweredCount.value = p2.totalCount
    opponentScore.value = p1.score
    opponentCorrectCount.value = p1.correctCount
    opponentAnsweredCount.value = p1.totalCount
    opponent.value = { nickname: p1.nickname }
  }
  gameOver.value = true
  waitingForOpponent.value = false
  authStore.fetchUserInfo()
}

/**
 * 恢复游戏状态。单人训练会直接进入答题阶段，靠这里补上当前题目。
 */
function renderResume(data: any) {
  roomId.value = data.roomId
  mode.value = data.mode === 'solo_training' ? 'solo_training' : mode.value
  opponent.value = data.opponent || opponent.value
  robotProfile.value = data.robotProfile || robotProfile.value

  if (data.phase === 'grab') {
    phase.value = 'grab'
    availableWords.value = data.words || []
    timeLeft.value = data.timeLeft || 6
    maxWordsPerPlayer.value = data.maxWordsPerPlayer || maxWordsPerPlayer.value
    return
  }

  if (data.phase === 'answer') {
    phase.value = 'answer'
    myScore.value = data.myScore || 0
    myCorrectCount.value = data.myCorrectCount || 0
    myAnsweredCount.value = data.myAnsweredCount || 0
    opponentScore.value = data.opponentScore || 0
    opponentCorrectCount.value = data.opponentCorrectCount || 0
    opponentAnsweredCount.value = data.opponentAnsweredCount || 0
    questionTotal.value = data.questionTotal || 0
    questionIndex.value = data.questionIndex || 0
    waitingForOpponent.value = !!data.waitingForOpponent

    if (data.questionWord) {
      renderQuestion({
        word: data.questionWord,
        options: data.questionOptions || [],
        timeLeft: data.questionTimeLeft || 4,
        index: data.questionIndex || 0,
        total: data.questionTotal || 0
      })
    }
  }
}

/**
 * 纯显示用的倒计时（后端才是权威计时）
 */
function startDisplayCountdown(seconds: number) {
  stopDisplayCountdown()
  questionTimeLeft.value = seconds
  questionCountdown = window.setInterval(() => {
    questionTimeLeft.value--
    if (questionTimeLeft.value <= 0) {
      stopDisplayCountdown()
    }
  }, 1000)
}

function stopDisplayCountdown() {
  if (questionCountdown) {
    clearInterval(questionCountdown)
    questionCountdown = null
  }
}

/**
 * 抢词 — 发送 WebSocket 消息
 */
function grabWord(wordId: number) {
  wsStore.send('grab_word', { wordId })
}

/**
 * 提交答案 — 发送 WebSocket 消息（仅索引 1-4）
 */
function submitAnswer(optionIndex: number) {
  if (showResult.value || answerSubmitted.value) return

  answerSubmitted.value = true
  selectedOptionIndex.value = optionIndex

  wsStore.send('submit_answer', {
    roomId: roomId.value,
    selectedIndex: optionIndex
  })
}

const resultText = computed(() => {
  if (myScore.value > opponentScore.value) return '胜利！'
  if (myScore.value < opponentScore.value) return '失败'
  return '平局'
})

const resultClass = computed(() => {
  if (myScore.value > opponentScore.value) return 'win'
  if (myScore.value < opponentScore.value) return 'lose'
  return 'draw'
})

function goHome() {
  wsStore.send('go_home')
  router.push({
    path: '/home',
    state: mode.value === 'solo_training' ? { openTrainingSetup: true } : {}
  })
}

useEscapeClose(() => {
  if (gameOver.value) goHome()
})

onMounted(() => {
  // 从路由 state 恢复游戏数据
  const gameData = (history.state as any)?.gameData
  if (gameData) {
    mode.value = gameData.mode === 'solo_training' ? 'solo_training' : 'match'
    phase.value = gameData.phase === 'answer' ? 'answer' : 'grab'
    availableWords.value = gameData.words
    myWords.value = gameData.myWords || []
    opponent.value = gameData.opponent
    robotProfile.value = gameData.robotProfile || null
    timeLeft.value = gameData.timeLeft || 6
    roomId.value = gameData.roomId
    maxWordsPerPlayer.value = gameData.maxWordsPerPlayer || 5
  }

  // 使用全局 WebSocket，注册 game handler
  wsStore.registerHandler('game', handleMessage)
  if (gameData?.phase === 'answer') {
    wsStore.send('sync_state')
  }
})

onUnmounted(() => {
  stopDisplayCountdown()
  wsStore.unregisterHandler('game')
  // 注意：不关闭 WebSocket，由全局 store 管理
})
</script>

<style scoped>
.game-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}

.game-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  background: rgba(0,0,0,0.3);
  color: white;
}

.header-left { flex: 1; }

.header-right {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
}

.header-right .nickname {
  font-size: 16px;
  font-weight: bold;
  color: white;
}

.exit-btn {
  padding: 6px 16px;
  background: #ff6b6b;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.timer {
  font-size: 48px;
  font-weight: bold;
  color: #ffd700;
}

.all-words {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 15px;
  padding: 20px;
  max-width: 800px;
  margin: 0 auto;
}

.word-card {
  padding: 20px;
  border-radius: 12px;
  text-align: center;
  cursor: pointer;
  transition: transform 0.3s, opacity 0.3s, box-shadow 0.3s;
  box-shadow: 0 4px 15px rgba(0,0,0,0.2);
  min-height: 80px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

/* 根据难道计算的卡牌稀有度反馈主题 */
.tier-common { background: linear-gradient(135deg, #f5f5f5, #e0e0e0); color: #333; border: 1px solid #ccc; }
.tier-uncommon { background: linear-gradient(135deg, #d4fc79, #96e6a1); color: #1a5611; border: 1px solid #82d88d; }
.tier-rare { background: linear-gradient(135deg, #a1c4fd, #c2e9fb); color: #00368a; border: 1px solid #7cb1f8; box-shadow: 0 4px 15px rgba(161,196,253,0.4); }
.tier-epic { background: linear-gradient(135deg, #fbc2eb, #a6c1ee); color: #4e126e; border: 1px solid #d4a5ee; box-shadow: 0 4px 15px rgba(166,193,238,0.5); }
.tier-legendary { background: linear-gradient(135deg, #ffecd2, #fcb69f); color: #9c3305; border: 2px solid #ffd700; box-shadow: 0 4px 20px rgba(252,182,159,0.6); }

/* 文字色彩继承覆盖 */
.tier-common .word-text, .tier-uncommon .word-text, .tier-rare .word-text, .tier-epic .word-text, .tier-legendary .word-text { color: inherit; }
.tier-common .word-difficulty, .tier-uncommon .word-difficulty, .tier-rare .word-difficulty, .tier-epic .word-difficulty, .tier-legendary .word-difficulty { color: inherit; font-weight: 600; }

.word-card:hover:not(.grabbed) { transform: scale(1.08) translateY(-4px); }
.word-card.grabbed { visibility: hidden; cursor: default; }

.player-words-row {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 15px;
  margin: 0 20px 15px;
  border-radius: 12px;
  background: rgba(102, 126, 234, 0.2);
}

.player-words-row.opponent { background: rgba(255, 107, 107, 0.2); }

.player-name {
  font-size: 16px;
  font-weight: bold;
  color: white;
  margin-bottom: 10px;
}

.player-words {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: center;
  align-items: center;
  min-height: 50px;
}

.player-words .word-card.grabbed {
  visibility: visible;
  background: #667eea;
  color: white;
  padding: 10px 18px;
  border-radius: 12px;
  font-size: 16px;
  min-height: auto;
  display: flex;
  align-items: center;
  justify-content: center;
}

.player-words .word-card.grabbed.opponent { background: #ff6b6b; }

.grab-count {
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.8);
  margin-top: 8px;
  padding: 4px 12px;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 12px;
}

.grab-count.full {
  color: #ffd700;
  background: rgba(255, 215, 0, 0.2);
}

.word-card.disabled {
  opacity: 0.4;
  cursor: not-allowed !important;
  pointer-events: none;
}

.word-text { font-size: 18px; font-weight: bold; }
.word-difficulty { font-size: 12px; margin-top: 5px; opacity: 0.8; }

/* Answer area */
.answer-area {
  max-width: 600px;
  margin: 40px auto;
  padding: 40px;
  background: white;
  border-radius: 16px;
  text-align: center;
  position: relative;
}

.progress {
  position: absolute;
  left: 20px;
  top: 20px;
  font-size: 18px;
  color: #666;
}

.question-timer {
  font-size: 20px;
  font-weight: bold;
  color: #ff5722;
  padding: 8px 16px;
  background: #fff3e0;
  border-radius: 20px;
  margin-bottom: 10px;
}

.question-center {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
}

.question-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  gap: 20px;
}

.side-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  min-width: 80px;
}

.side-info.left { color: #667eea; }
.side-info.right { color: #ff6b6b; }

.side-nickname {
  font-size: 14px;
  font-weight: bold;
  white-space: nowrap;
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.side-progress { font-size: 18px; font-weight: bold; }
.side-score { font-size: 14px; opacity: 0.8; }

.question-word {
  font-size: 48px;
  font-weight: bold;
  color: #333;
  text-align: center;
}

.options {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.option-btn {
  padding: 20px;
  font-size: 18px;
  border: 2px solid #e0e0e0;
  border-radius: 12px;
  background: white;
  cursor: pointer;
  transition: all 0.3s;
}

.option-btn:hover:not(:disabled) {
  border-color: #667eea;
  background: #f5f5ff;
}

.option-btn.selected { border-color: #667eea; background: #e8e8ff; }
.option-btn.correct { border-color: #4caf50; background: #e8f5e9; }
.option-btn.wrong { border-color: #f44336; background: #ffebee; }

.result-feedback { margin-top: 30px; font-size: 20px; font-weight: bold; }
.correct-text { color: #4caf50; }
.wrong-text { color: #f44336; }

/* Result overlay */
.result-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.8);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.result-box {
  background: white;
  padding: 40px;
  border-radius: 12px;
  text-align: center;
}

.result-header { margin-bottom: 30px; }
.result-header h2 { font-size: 42px; font-weight: bold; letter-spacing: 4px; margin: 0; }
.result-header h2.win { color: #4caf50; text-shadow: 0 2px 10px rgba(76, 175, 80, 0.3); }
.result-header h2.lose { color: #f44336; text-shadow: 0 2px 10px rgba(244, 67, 54, 0.3); }
.result-header h2.draw { color: #ff9800; text-shadow: 0 2px 10px rgba(255, 152, 0, 0.3); }

.battle-result {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 30px;
  margin: 30px 0;
  padding: 20px 0;
}

.player-card {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 20px;
  padding: 30px 40px;
  min-width: 150px;
  text-align: center;
  position: relative;
  transition: all 0.3s ease;
  color: white;
  box-shadow: 0 10px 40px rgba(102, 126, 234, 0.35);
}

.player-card.winner {
  transform: scale(1.1);
  box-shadow: 0 20px 60px rgba(102, 126, 234, 0.6), 0 0 30px rgba(255, 255, 255, 0.3);
  z-index: 10;
}

.player-card.winner::after {
  content: 'WINNER';
  position: absolute;
  top: -12px;
  left: 50%;
  transform: translateX(-50%);
  background: linear-gradient(135deg, #ffd700 0%, #ffb700 100%);
  color: #333;
  font-size: 12px;
  font-weight: bold;
  padding: 4px 12px;
  border-radius: 12px;
  box-shadow: 0 4px 15px rgba(255, 215, 0, 0.5);
  letter-spacing: 1px;
}

.player-card.loser {
  transform: scale(0.9);
  opacity: 0.6;
  filter: grayscale(60%);
  box-shadow: 0 5px 20px rgba(0, 0, 0, 0.2);
}

.player-card.loser::after {
  content: 'KO';
  position: absolute;
  top: -25px;
  left: 50%;
  transform: translateX(-50%) rotate(-10deg);
  font-size: 36px;
  font-weight: 900;
  color: #ff4444;
  text-shadow: 0 0 15px rgba(255, 68, 68, 0.9), 0 0 30px rgba(255, 68, 68, 0.6);
  letter-spacing: 3px;
  animation: koPulse 0.8s ease-in-out infinite;
}

@keyframes koPulse {
  0%, 100% { transform: translateX(-50%) rotate(-10deg) scale(1); opacity: 0.9; }
  50% { transform: translateX(-50%) rotate(-10deg) scale(1.15); opacity: 1; }
}

.player-avatar {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 15px;
  font-size: 28px;
  position: relative;
  width: 60px;
  height: 60px;
}

.player-avatar.winner {
  background: linear-gradient(135deg, #ffd700 0%, #ffaa00 50%, #ff8c00 100%);
  box-shadow:
    0 0 20px rgba(255, 215, 0, 0.6),
    0 0 40px rgba(255, 215, 0, 0.3),
    inset 0 0 20px rgba(255, 255, 255, 0.3);
  animation: winnerGlow 1.5s ease-in-out infinite;
  border: 3px solid #ffd700;
}

.player-avatar.winner::before {
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

.player-avatar.winner::after {
  content: '';
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
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.player-avatar.loser {
  transform: scale(0.9);
  opacity: 0.6;
  filter: grayscale(60%);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

@keyframes koPulse {
  0%, 100% { transform: translateX(-50%) rotate(-10deg) scale(1); opacity: 1; }
  50% { transform: translateX(-50%) rotate(-10deg) scale(1.1); opacity: 0.8; }
}

.crown {
  position: absolute;
  top: -8px;
  right: -4px;
  font-size: 18px;
}

.player-score { font-size: 48px; font-weight: bold; line-height: 1; }

.player-accuracy {
  font-size: 14px;
  font-weight: 600;
  margin-top: 8px;
  padding: 4px 12px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 12px;
  color: white;
}

.vs-divider { display: flex; align-items: center; justify-content: center; }
.vs-text { font-size: 24px; font-weight: bold; color: #999; letter-spacing: 2px; }

.training-reward {
  margin-top: 18px;
  display: flex;
  justify-content: center;
  gap: 16px;
  color: #0f766e;
  font-size: 15px;
  font-weight: 700;
}

.result-box.waiting {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  min-width: 300px;
}

.result-box.waiting h2 { color: white; margin-bottom: 30px; }
.result-box.waiting p { margin-top: 20px; font-size: 18px; opacity: 0.9; }

.waiting-spinner {
  width: 50px;
  height: 50px;
  border: 4px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  margin: 0 auto;
  animation: spin 1s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

/* ==================== 手机端响应式适配 ==================== */
@media (max-width: 480px) {
  /* Header */
  .game-header { padding: 10px 12px; }
  .timer { font-size: 32px; }
  .header-right .nickname { font-size: 13px; }
  .exit-btn { padding: 4px 12px; font-size: 12px; }

  /* 抢词阶段 */
  .player-words-row {
    padding: 8px 10px;
    margin: 0 8px 8px;
  }
  .player-name { font-size: 13px; margin-bottom: 6px; }
  .player-words { gap: 6px; min-height: 36px; }
  .player-words .word-card.grabbed {
    padding: 6px 12px;
    font-size: 13px;
    border-radius: 8px;
  }
  .grab-count { font-size: 12px; margin-top: 4px; padding: 2px 8px; }

  .all-words {
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
    padding: 10px 8px;
  }
  .word-card {
    padding: 12px 8px;
    min-height: 55px;
    border-radius: 8px;
  }
  .word-text { font-size: 14px; }
  .word-difficulty { font-size: 10px; margin-top: 3px; }

  /* 答题阶段 */
  .answer-area {
    margin: 12px 8px;
    padding: 16px 12px;
    border-radius: 12px;
  }
  .progress { font-size: 14px; left: 12px; top: 12px; }
  .question-timer { font-size: 16px; padding: 5px 12px; }
  .question-word { font-size: 28px; }
  .question-row { gap: 8px; }
  .side-info { min-width: 55px; gap: 2px; }
  .side-nickname { font-size: 11px; max-width: 55px; }
  .side-progress { font-size: 14px; }
  .side-score { font-size: 11px; }
  .options { gap: 8px; }
  .option-btn { padding: 12px; font-size: 14px; border-radius: 8px; }
  .result-feedback { margin-top: 15px; font-size: 16px; }

  /* 结算界面 */
  .result-box { padding: 20px 16px; border-radius: 10px; }
  .result-header { margin-bottom: 15px; }
  .result-header h2 { font-size: 28px; letter-spacing: 2px; }

  .battle-result {
    flex-direction: column;
    gap: 12px;
    margin: 15px 0;
    padding: 10px 0;
  }
  .player-card {
    padding: 16px 24px;
    min-width: auto;
    width: 100%;
    border-radius: 14px;
  }
  .player-card.winner { transform: scale(1.02); }
  .player-avatar { width: 48px; height: 48px; font-size: 22px; margin-bottom: 8px; }
  .player-score { font-size: 36px; }
  .player-accuracy { font-size: 12px; }
  .vs-divider { margin: 4px 0; }
  .vs-text { font-size: 18px; }
  .training-reward { flex-direction: column; gap: 6px; }

  /* 等待对手弹窗 */
  .result-box.waiting { min-width: auto; width: calc(100% - 40px); }
  .result-box.waiting h2 { font-size: 20px; margin-bottom: 16px; }
  .result-box.waiting p { font-size: 16px; }
}
</style>
