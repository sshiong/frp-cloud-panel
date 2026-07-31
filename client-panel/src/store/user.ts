import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, getUserInfo } from '../api/auth'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('client_token') || '')
  const userInfo = ref<any>(null)
  const deviceInfo = ref<any>(null)

  const isAuthenticated = computed(() => !!token.value)

  async function login(username: string, password: string) {
    try {
      const res = await loginApi(username, password)
      token.value = res.data.token
      userInfo.value = res.data.user
      localStorage.setItem('client_token', token.value)
      return true
    } catch (error) {
      console.error('Login failed:', error)
      return false
    }
  }

  async function fetchUserInfo() {
    try {
      const res = await getUserInfo()
      userInfo.value = res.data
    } catch (error) {
      console.error('Failed to fetch user info:', error)
    }
  }

  function logout() {
    token.value = ''
    userInfo.value = null
    deviceInfo.value = null
    localStorage.removeItem('client_token')
  }

  return {
    token,
    userInfo,
    deviceInfo,
    isAuthenticated,
    login,
    fetchUserInfo,
    logout,
  }
})
