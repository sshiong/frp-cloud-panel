import api from './auth'

export function getMappings(params?: any) {
  return api.get('/mappings', { params })
}

export function getMapping(id: number) {
  return api.get(`/mappings/${id}`)
}

export function createMapping(data: any) {
  return api.post('/mappings', data)
}

export function updateMapping(id: number, data: any) {
  return api.put(`/mappings/${id}`, data)
}

export function deleteMapping(id: number) {
  return api.delete(`/mappings/${id}`)
}
