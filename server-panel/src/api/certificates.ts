import api from './auth'

export function getCertificate(domain: string) {
  return api.get(`/certs/${domain}`)
}

export function renewCertificate(domain: string) {
  return api.post(`/certs/${domain}/renew`)
}

export function checkCerts() {
  return api.get('/certs/check')
}
