import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/login'
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/LoginView.vue')
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('../views/RegisterView.vue')
  },
  {
    path: '/home',
    name: 'Home',
    component: () => import('../views/HomeView.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/game',
    name: 'Game',
    component: () => import('../views/GameView.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/records',
    name: 'Records',
    component: () => import('../views/RecordView.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/wrong-words',
    name: 'WrongWords',
    component: () => import('../views/WrongWordsView.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/mastered-words',
    name: 'MasteredWords',
    component: () => import('../views/MasteredWordsView.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/training-results',
    name: 'TrainingResults',
    component: () => import('../views/TrainingAnswerResultsView.vue'),
    meta: { requiresAuth: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore()

  if (to.meta.requiresAuth && !authStore.isLoggedIn) {
    next('/login')
  } else {
    next()
  }
})

export default router
