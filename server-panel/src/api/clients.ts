import api from './auth'

export function getClients(params?: any) {
  return api.get('/clients', { params })
}

export function getClient(id: number) {
  return api.get(`/clients/${id}`)
}

export function updateClient(id: number, data: any) {
  return api.put(`/clients/${id}`, data)
}

export function deleteClient(id: number) {
  return api.delete(`/clients/${id}`)
}
