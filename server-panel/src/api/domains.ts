import api from './auth'

export function getDomains() {
  return api.get('/domains')
}

export function getDomain(id: number) {
  return api.get(`/domains/${id}`)
}

export function createDomain(data: any) {
  return api.post('/domains', data)
}

export function updateDomain(id: number, data: any) {
  return api.put(`/domains/${id}`, data)
}

export function deleteDomain(id: number) {
  return api.delete(`/domains/${id}`)
}
