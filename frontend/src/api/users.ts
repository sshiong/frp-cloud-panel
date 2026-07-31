import api from './auth'

export function getUsers(params?: any) {
  return api.get('/users', { params })
}

export function getUser(id: number) {
  return api.get(`/users/${id}`)
}

export function createUser(data: any) {
  return api.post('/users', data)
}

export function updateUser(id: number, data: any) {
  return api.put(`/users/${id}`, data)
}

export function updateUserStatus(id: number, status: string) {
  return api.put(`/users/${id}/status`, { status })
}

export function deleteUser(id: number) {
  return api.delete(`/users/${id}`)
}
