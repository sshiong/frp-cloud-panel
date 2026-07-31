import api from './auth'

export function getBackups() {
  return api.get('/backup/list')
}

export function createBackup(password: string) {
  return api.post('/backup/create', { password })
}

export function restoreBackup(filename: string, password: string) {
  return api.post('/backup/restore', { filepath: filename, password })
}

export function deleteBackup(filename: string) {
  return api.delete(`/backup/${filename}`)
}
