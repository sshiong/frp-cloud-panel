import api from './auth'

export function setCFToken(token: string, email: string) {
  return api.post('/cloudflare/token', { token, email })
}

export function getCFTokenStatus() {
  return api.get('/cloudflare/token/status')
}

export function deleteCFToken() {
  return api.delete('/cloudflare/token')
}

export function testCFToken() {
  return api.post('/cloudflare/token/test')
}
