import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8081/api/v1',
  timeout: 10000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      // Try refresh token
      localStorage.removeItem('access_token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export const authApi = {
  getMe: () => api.get('/auth/me').then((r) => r.data),
  refresh: () => api.post('/auth/refresh').then((r) => r.data),
}

export const routesApi = {
  create: (data: CreateRouteRequest) => api.post('/routes', data).then((r) => r.data),
  get: (id: string) => api.get(`/routes/${id}`).then((r) => r.data),
  list: (params?: ListRoutesParams) => api.get('/routes', { params }).then((r) => r.data),
  accept: (id: string) => api.post(`/routes/${id}/accept`).then((r) => r.data),
  cancel: (id: string) => api.post(`/routes/${id}/cancel`).then((r) => r.data),
}

export const incidentsApi = {
  create: (data: CreateIncidentRequest) => api.post('/incidents', data).then((r) => r.data),
  get: (id: string) => api.get(`/incidents/${id}`).then((r) => r.data),
  list: (params?: ListIncidentsParams) => api.get('/incidents', { params }).then((r) => r.data),
  update: (id: string, data: UpdateIncidentRequest) => api.put(`/incidents/${id}`, data).then((r) => r.data),
}

export const alertsApi = {
  list: (params?: ListAlertsParams) => api.get('/alerts', { params }).then((r) => r.data),
}

export interface CreateRouteRequest {
  origin: [number, number]
  destination: [number, number]
  vehicle_type?: string
  priority_level?: number
  return_alternatives?: boolean
  max_alternatives?: number
  profile?: string
}

export interface ListRoutesParams {
  status?: string
  limit?: number
  offset?: number
}

export interface CreateIncidentRequest {
  type: string
  severity: number
  location: [number, number]
  description?: string
  affected_segments?: number[]
}

export interface ListIncidentsParams {
  status?: string
  type?: string
  severity?: number
  bbox?: string
}

export interface UpdateIncidentRequest {
  status?: string
  severity?: number
  description?: string
}

export interface ListAlertsParams {
  active?: boolean
  severity?: string
}