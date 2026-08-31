import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const userId = ref<number | null>(null)
  const username = ref('')
  const nickname = ref('')
  const rank = ref(1)
  const exp = ref(0)
  const totalWins = ref(0)
  const totalGames = ref(0)
  const trainingRank = ref(1)
  const trainingExp = ref(0)
  const trainingTotalWins = ref(0)
  const trainingTotalGames = ref(0)

  const isLoggedIn = computed(() => !!token.value)
  const winRate = computed(() => totalGames.value > 0 ? Math.round(totalWins.value / totalGames.value * 100) : 0)
  const trainingWinRate = computed(() => trainingTotalGames.value > 0 ? Math.round(trainingTotalWins.value / trainingTotalGames.value * 100) : 0)

  function setAuth(auth: any) {
    token.value = auth.token
    userId.value = auth.userId
    username.value = auth.username
    nickname.value = auth.nickname
    rank.value = auth.rank
    exp.value = auth.exp
    totalWins.value = auth.totalWins
    totalGames.value = auth.totalGames
    trainingRank.value = auth.trainingRank || 1
    trainingExp.value = auth.trainingExp || 0
    trainingTotalWins.value = auth.trainingTotalWins || 0
    trainingTotalGames.value = auth.trainingTotalGames || 0
    localStorage.setItem('token', auth.token)
    api.defaults.headers.common['Authorization'] = `Bearer ${auth.token}`
  }

  function logout() {
    token.value = ''
    userId.value = null
    username.value = ''
    nickname.value = ''
    localStorage.removeItem('token')
  }

  async function fetchUserInfo() {
    try {
      const res = await api.get('/api/user/info')
      const data = res.data
      userId.value = data.userId
      username.value = data.username
      nickname.value = data.nickname
      rank.value = data.rank
      exp.value = data.exp
      totalWins.value = data.totalWins
      totalGames.value = data.totalGames
      trainingRank.value = data.trainingRank || 1
      trainingExp.value = data.trainingExp || 0
      trainingTotalWins.value = data.trainingTotalWins || 0
      trainingTotalGames.value = data.trainingTotalGames || 0
    } catch (e) {
      console.error('Failed to fetch user info:', e)
    }
  }

  return {
    token,
    userId,
    username,
    nickname,
    rank,
    exp,
    totalWins,
    totalGames,
    trainingRank,
    trainingExp,
    trainingTotalWins,
    trainingTotalGames,
    isLoggedIn,
    winRate,
    trainingWinRate,
    setAuth,
    logout,
    fetchUserInfo
  }
})
