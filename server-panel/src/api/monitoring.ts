import api from './auth'

export function getRouterStats() {
  return api.get('/router/stats')
}

export function reloadCertificates() {
  return api.post('/router/reload-certs')
}

export function clearRouterCache() {
  return api.post('/router/clear-cache')
}
