import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, getUserInfo } from '../api/auth'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const userInfo = ref<any>(null)

  const isAuthenticated = computed(() => !!token.value)

  async function login(username: string, password: string) {
    try {
      const res = await loginApi(username, password)
      token.value = res.data.token
      userInfo.value = res.data.user
      localStorage.setItem('token', token.value)
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
    localStorage.removeItem('token')
  }

  return {
    token,
    userInfo,
    isAuthenticated,
    login,
    fetchUserInfo,
    logout,
  }
})
