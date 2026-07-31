import api from './auth'

export function getDNSRecords(domain: string) {
  return api.get('/dns/records', { params: { domain } })
}

export function createDNSRecord(data: any) {
  return api.post('/dns/records', data)
}

export function updateDNSRecord(id: string, data: any) {
  return api.put(`/dns/records/${id}`, data)
}

export function deleteDNSRecord(id: string) {
  return api.delete(`/dns/records/${id}`)
}
