import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import HomeView from './HomeView.vue'

const { authStore, router, wsStore } = vi.hoisted(() => ({
  authStore: {
    exp: 0,
    fetchUserInfo: vi.fn(),
    logout: vi.fn(),
    nickname: 'tester',
    rank: 1,
    token: 'test-token',
    totalGames: 0,
    totalWins: 0,
    trainingExp: 0,
    trainingRank: 1,
    trainingTotalGames: 0,
    trainingTotalWins: 0,
    trainingWinRate: 0,
    winRate: 0
  },
  router: { push: vi.fn() },
  wsStore: {
    connect: vi.fn(),
    connected: true,
    disconnect: vi.fn(),
    ensureConnected: vi.fn().mockResolvedValue(true),
    registerHandler: vi.fn(),
    send: vi.fn().mockReturnValue(true),
    unregisterHandler: vi.fn()
  }
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => router
}))

vi.mock('../stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('../stores/websocket', () => ({
  useWebSocketStore: () => wsStore
}))

function mountHome() {
  return mount(HomeView)
}

async function openDifficultyPicker(wrapper: ReturnType<typeof mountHome>) {
  await wrapper.get('.match-section .solo-btn').trigger('click')
  expect(wrapper.find('.training-setup-page').exists()).toBe(true)

  await wrapper.get('.difficulty-btn').trigger('click')
  expect(wrapper.find('.difficulty-overlay').exists()).toBe(true)
}

describe('HomeView difficulty picker close flow', () => {
  beforeEach(() => {
    history.replaceState({}, '')
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('returns to the training setup after clicking the difficulty picker close button', async () => {
    const wrapper = mountHome()
    await openDifficultyPicker(wrapper)

    await wrapper.get('.difficulty-overlay .fullscreen-close-button').trigger('click')

    expect(wrapper.find('.difficulty-overlay').exists()).toBe(false)
    expect(wrapper.find('.training-setup-page').exists()).toBe(true)
    expect(wrapper.find('.home-container').exists()).toBe(false)
    wrapper.unmount()
  })

  it('returns to the training setup after pressing Escape in the difficulty picker', async () => {
    const wrapper = mountHome()
    await openDifficultyPicker(wrapper)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()

    expect(wrapper.find('.difficulty-overlay').exists()).toBe(false)
    expect(wrapper.find('.training-setup-page').exists()).toBe(true)
    expect(wrapper.find('.home-container').exists()).toBe(false)
    wrapper.unmount()
  })

  it('returns to the training setup after selecting a concrete difficulty', async () => {
    const wrapper = mountHome()
    await openDifficultyPicker(wrapper)

    await wrapper.get('.difficulty-child-card').trigger('click')

    expect(wrapper.find('.difficulty-overlay').exists()).toBe(false)
    expect(wrapper.find('.training-setup-page').exists()).toBe(true)
    expect(wrapper.find('.home-container').exists()).toBe(false)
    wrapper.unmount()
  })

  it('consumes the training navigation state so a browser refresh returns home', async () => {
    history.replaceState({ openTrainingSetup: true }, '')

    const firstMount = mountHome()
    await nextTick()

    expect(firstMount.find('.training-setup-page').exists()).toBe(true)
    expect(history.state.openTrainingSetup).not.toBe(true)
    firstMount.unmount()

    const refreshedMount = mountHome()
    await nextTick()

    expect(refreshedMount.find('.home-container').exists()).toBe(true)
    expect(refreshedMount.find('.training-setup-page').exists()).toBe(false)
    refreshedMount.unmount()
  })
})
