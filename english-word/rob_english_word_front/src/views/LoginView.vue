<template>
  <div class="login-container">
    <div class="login-box">
      <h1>英语抢词大战</h1>
      <form @submit.prevent="handleLogin">
        <input v-model="form.username" placeholder="用户名" required />
        <input v-model="form.password" type="password" placeholder="密码" required />
        <button type="submit">登录</button>
      </form>
      <p class="link">
        还没有账号？<router-link to="/register">立即注册</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import api from '../api'

const router = useRouter()
const authStore = useAuthStore()

const form = ref({
  username: '',
  password: ''
})

async function handleLogin() {
  try {
    const res = await api.post('/api/auth/login', form.value)
    authStore.setAuth(res.data)
    router.push('/home')
  } catch (e: any) {
    alert(e.response?.data?.message || '登录失败')
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
.login-box {
  background: white;
  padding: 40px;
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0,0,0,0.2);
  width: 350px;
}
h1 {
  text-align: center;
  color: #333;
  margin-bottom: 30px;
}
input {
  width: 100%;
  padding: 12px;
  margin-bottom: 15px;
  border: 1px solid #ddd;
  border-radius: 6px;
  box-sizing: border-box;
}
button {
  width: 100%;
  padding: 12px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 16px;
}
button:hover {
  opacity: 0.9;
}
.link {
  text-align: center;
  margin-top: 15px;
}
a {
  color: #667eea;
}

@media (max-width: 480px) {
  .login-box {
    width: calc(100% - 40px);
    padding: 30px 20px;
    margin: 0 20px;
  }
  h1 { font-size: 22px; margin-bottom: 20px; }
  input { padding: 10px; }
  button { padding: 10px; font-size: 15px; }
}
</style>