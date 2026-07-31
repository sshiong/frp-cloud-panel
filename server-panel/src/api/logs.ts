import api from './auth'

export function getLogs(params?: any) {
  return api.get('/logs', { params })
}
