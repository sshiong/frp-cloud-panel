import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export function login(username: string, password: string) {
  return api.post('/auth/login', { username, password })
}

export function register(username: string, password: string, email: string) {
  return api.post('/auth/register', { username, password, email })
}

export function getUserInfo() {
  return api.get('/users/me')
}

export function updateUserInfo(data: any) {
  return api.put('/users/me', data)
}

export function updatePassword(oldPassword: string, newPassword: string) {
  return api.put('/users/me/password', { old_password: oldPassword, new_password: newPassword })
}

export default api
